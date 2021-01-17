package client

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"github.com/itiky/collaborate-storage/model"
)

// initSnapshot fetches the initial snapshot version.
func (c *Client) initSnapshot() error {
	req := model.GetListSnapshotRequest{
		ClientId: c.id,
	}
	res := model.GetListSnapshotResponse{}

	opStart := time.Now()
	if err := c.rpcClient.Call("SortedListService.GetList", req, &res); err != nil {
		return fmt.Errorf("rpc: %w", err)
	}
	opDur := time.Since(opStart)

	c.snapshotVersion = res.Version
	c.snapshotData = res.Data

	log.Printf("%s: initial snapshot v%d received: %d items within %v", c.String(), res.Version, len(res.Data), opDur)

	return nil
}

// sendUpdates creates random storage operations and sends them.
func (c *Client) sendUpdates() error {
	sendOpsPrevLen := len(c.sendOps)

	getIdFromSnapshot := func() string {
		itemIdx := rand.Intn(len(c.snapshotData))
		return c.snapshotData[itemIdx].Id
	}
	getNewValue := func() model.StorageValue {
		return model.StorageValue(rand.Int31())
	}

	sendN := rand.Intn(c.opsSendMax) + 1
	sendOps := make([]model.OperationRequest, 0, sendN)
	for i := 0; i < sendN; i++ {
		sendOp := model.OperationRequest{}

		switch rand.Intn(3) {
		case 0:
			sendOp.Type = model.InsertOperationType

			sendOp.Id = uuid.New().String()
			sendOp.Value = getNewValue()
		case 1:
			sendOp.Type = model.UpdateOperationType

			sendOp.Id = getIdFromSnapshot()
			sendOp.Value = getNewValue()
		case 2:
			sendOp.Type = model.DeleteOperationType

			// Check if this delete is not a duplicate
			for {
				itemId := getIdFromSnapshot()
				found := false
				for _, op := range sendOps {
					if op.Type == model.DeleteOperationType && op.Id == itemId {
						found = true
						break
					}
				}
				if !found {
					sendOp.Id = itemId
					break
				}
			}
		}

		sendOps = append(sendOps, sendOp)
		c.sendOps[c.reqOperationToMatchStr(sendOp)] = true
	}

	req := model.UpdateListRequest{
		ClientId:   c.id,
		Version:    c.snapshotVersion,
		Operations: sendOps,
	}

	opStart := time.Now()
	if err := c.rpcClient.Call("SortedListService.UpdateList", req, nil); err != nil {
		return fmt.Errorf("rpc: %w", err)
	}
	opDur := time.Since(opStart)

	log.Printf("%s: [%v] updates send: %d ops", c.String(), opDur, len(sendOps))

	// Update stats
	monitor.UpdatesSend(len(sendOps), opDur)
	if sendOpsPrevLen == 0 {
		monitor.ConsistencyReset(opStart)
	}

	return nil
}

// pollUpdates requests a new snapshot version (if exists) and update the local state.
func (c *Client) pollUpdates() error {
	req := model.GetListUpdatesRequest{
		Version: c.snapshotVersion,
	}
	res := model.GetListUpdatesResponse{}

	opStart := time.Now()
	if err := c.rpcClient.Call("SortedListService.GetListUpdates", req, &res); err != nil {
		return fmt.Errorf("rpc: %w", err)
	}

	if res.Version == c.snapshotVersion {
		return nil
	}

	newSnapshot, err := model.ApplyListOperations(c.snapshotData, res.Operations...)
	if err != nil {
		log.Fatalf("model.ApplyListOperations: %v", err)
	}
	opStop := time.Now()
	opDur := opStop.Sub(opStart)

	c.snapshotVersion = res.Version
	c.snapshotData = newSnapshot

	for _, listOp := range res.Operations {
		listOpStr := c.listOperationToMatchStr(listOp)
		if c.sendOps[listOpStr] {
			delete(c.sendOps, listOpStr)
		}
	}
	log.Printf("%s: [%v] snapshot updated to v%d: %d ops (%d unhandled)", c.String(), opDur, res.Version, len(res.Operations), len(c.sendOps))

	// Update stats
	monitor.UpdatesReceived(len(res.Operations), opDur)
	if len(c.sendOps) == 0 {
		monitor.ConsistencyAchieved(opStop)
	}

	return nil
}

// reqOperationToMatchStr builds a string representation of model.OperationRequest (used for c.sendOps matching).
func (c *Client) reqOperationToMatchStr(op model.OperationRequest) string {
	switch op.Type {
	case model.InsertOperationType:
		return fmt.Sprintf("%s: %s -> %d", op.Type, op.Id, op.Value)
	case model.UpdateOperationType:
		return fmt.Sprintf("%s: %s -> %d", op.Type, op.Id, op.Value)
	case model.DeleteOperationType:
		return fmt.Sprintf("%s: %s", op.Type, op.Id)
	}

	return ""
}

// listOperationToMatchStr builds a string representation of model.ListOperation (used for c.sendOps matching).
func (c *Client) listOperationToMatchStr(op model.ListOperation) string {
	switch op.Type {
	case model.InsertOperationType:
		return fmt.Sprintf("%s: %s -> %d", op.Type, op.Id, op.Value)
	case model.UpdateOperationType:
		return fmt.Sprintf("%s: %s -> %d", op.Type, op.Id, op.Value)
	case model.DeleteOperationType:
		return fmt.Sprintf("%s: %s", op.Type, op.Id)
	}

	return ""
}
