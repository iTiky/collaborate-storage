package client

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"syscall"
	"time"

	"github.com/itiky/collaborate-storage/model"
)

type Client struct {
	// Config
	id         model.ClientId // unique ID
	opsSendDur time.Duration  // storage updates send period
	opsSendMax int            // max number of storage updates per request
	pollDur    time.Duration  // snapshot update polling duration
	// State
	sendOps         map[string]bool   // keeps send operations which are not yet visible to client
	snapshotVersion int               // current snapshot version
	snapshotData    model.StorageList // current snapshot data
	//
	rpcClient *rpc.Client
	stopCh    chan interface{}
}

// String implements the stringer interface.
func (c *Client) String() string {
	return fmt.Sprintf("Client (%d)", c.id)
}

// Start starts the Client worker.
func (c *Client) Start() {
	if c.stopCh != nil {
		return
	}
	c.stopCh = make(chan interface{})

	monitor.Start()
	go c.worker()
}

// Stop stops the Client worker.
func (c *Client) Stop() {
	if c.stopCh == nil {
		return
	}

	close(c.stopCh)
	monitor.Stop()
}

// worker does the actual job.
func (c *Client) worker() {
	log.Printf("%s: start", c.String())
	log.Printf("%s: opsSendDur: %v", c.String(), c.opsSendDur)
	log.Printf("%s: opsSendMax: %v", c.String(), c.opsSendMax)
	log.Printf("%s: pollDur:    %v", c.String(), c.pollDur)

	if err := c.initSnapshot(); err != nil {
		log.Fatalf("%s: snapshot initialization: %v", c.String(), err)
	}

	sendCh := time.Tick(c.opsSendDur)
	pollCh := time.Tick(c.pollDur)
	for {
		select {
		case <-sendCh:
			// Send storage operations
			if err := c.sendUpdates(); err != nil {
				log.Fatalf("%s: sending updates: %v", c.String(), err)
			}
		case <-pollCh:
			// Update the local snapshot
			if err := c.pollUpdates(); err != nil {
				log.Fatalf("%s: polling updates: %v", c.String(), err)
			}
		case <-c.stopCh:
			// Stop the client
			log.Printf("%s: stop", c.String())
			c.rpcClient.Close()
			return
		}
	}
}

// NewClient creates a new Client object.
func NewClient(id model.ClientId, opsSendDur, pollDur time.Duration, opsSendMax int, serverUrl string) (*Client, error) {
	const (
		numOfRetries     = 120
		retryFallbackDur = 500 * time.Millisecond
	)

	if opsSendDur <= 0 {
		return nil, fmt.Errorf("%s: must be GT 0", "opsSendDur")
	}
	if pollDur <= 0 {
		return nil, fmt.Errorf("%s: must be GT 0", "pollDur")
	}
	if opsSendMax < 1 {
		return nil, fmt.Errorf("%s: must be GTE 1", "opsSendMax")
	}

	c := Client{
		id: id,
		//
		opsSendDur: opsSendDur,
		opsSendMax: opsSendMax,
		pollDur:    pollDur,
		//
		sendOps: make(map[string]bool),
	}

	for retry := 0; retry < numOfRetries; retry++ {
		client, err := rpc.Dial("tcp", serverUrl)
		if err == nil {
			c.rpcClient = client
			break
		}

		if netErr, ok := err.(*net.OpError); ok {
			if sysErr, ok := netErr.Err.(*os.SyscallError); ok {
				if sysErr.Err == syscall.ECONNREFUSED {
					time.Sleep(retryFallbackDur)
					continue
				}
			}
		}

		return nil, fmt.Errorf("rpc.Dial(%s): %w", serverUrl, err)
	}
	if c.rpcClient == nil {
		return nil, fmt.Errorf("RPC connection failed after %d retries with %v fallback", numOfRetries, retryFallbackDur)
	}

	return &c, nil
}
