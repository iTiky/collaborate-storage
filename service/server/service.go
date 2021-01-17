package server

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/itiky/collaborate-storage/model"
	"github.com/itiky/collaborate-storage/storage"
)

// SortedListService implements an RPC server service.
type SortedListService struct {
	// Config
	batchPeriod time.Duration
	// State
	docHistory *storage.DocumentHistory
	opsCh      chan []storage.StorageOperation
	//
	stopCh chan interface{}
}

// GetList returns a storage snapshot.
func (s *SortedListService) GetList(req model.GetListSnapshotRequest, res *model.GetListSnapshotResponse) error {
	if req.ClientId <= 0 {
		return fmt.Errorf("%s: must be GT 0", "ClientId")
	}

	version, list := s.docHistory.GetOutputSnapshot()
	res.Version = version
	res.Data = list

	return nil
}

// GetListUpdates returns model.ListOperation objects for client to apply on a local snapshot in order to upgrade it.
func (s *SortedListService) GetListUpdates(req model.GetListUpdatesRequest, res *model.GetListUpdatesResponse) error {
	start := time.Now()

	version, listOps := s.docHistory.GetOutputDiffWithLatest(req.Version)
	res.Version = version
	res.Operations = listOps

	go monitor.DiffRequestServed(time.Since(start))

	return nil
}

// UpdateList receives the storage update operations and pushes them to the queue.
func (s *SortedListService) UpdateList(req model.UpdateListRequest, res *model.UpdateListResponse) error {
	now := time.Now().UTC()

	// Input validation
	storageOps := make([]storage.StorageOperation, 0, len(req.Operations))
	for i, reqOp := range req.Operations {
		switch reqOp.Type {
		case model.InsertOperationType:
			storageOp, err := storage.NewSetOperation(reqOp.Id, reqOp.Value, req.ClientId, now)
			if err != nil {
				return fmt.Errorf("updateOperation[%d] (%s): %w", i, reqOp.Type, err)
			}
			storageOps = append(storageOps, storageOp)
		case model.UpdateOperationType:
			storageOp, err := storage.NewSetOperation(reqOp.Id, reqOp.Value, req.ClientId, now)
			if err != nil {
				return fmt.Errorf("updateOperation[%d] (%s): %w", i, reqOp.Type, err)
			}
			storageOps = append(storageOps, storageOp)
		case model.DeleteOperationType:
			storageOp, err := storage.NewDeleteOperation(reqOp.Id, req.ClientId, now)
			if err != nil {
				return fmt.Errorf("updateOperation[%d] (%s): %w", i, reqOp.Type, err)
			}
			storageOps = append(storageOps, storageOp)
		default:
			return fmt.Errorf("unsupported updateOperation: %s", reqOp.Type)
		}
	}

	s.opsCh <- storageOps

	return nil
}

// Start starts the service worker.
func (s *SortedListService) Start() {
	if s.stopCh != nil {
		return
	}
	s.stopCh = make(chan interface{})

	monitor.Start()
	go s.worker()
}

// Stop stops the service worker.
func (s *SortedListService) Stop() {
	if s.stopCh == nil {
		return
	}

	close(s.stopCh)
	monitor.Stop()
}

// worker does the actual job.
func (s *SortedListService) worker() {
	log.Println("SortedListService: start")

	stOpsQueue := make([]storage.StorageOperation, 0)

	handleCh := time.Tick(s.batchPeriod)
	for {
		select {
		case <-s.stopCh:
			// Service stop
			log.Println("SortedListService: stop")
			return
		case stOps := <-s.opsCh:
			// Push storage operations to the queue
			stOpsQueue = append(stOpsQueue, stOps...)
		case <-handleCh:
			// Start handling the queued operations
			sort.Slice(stOpsQueue, func(i, j int) bool {
				return stOpsQueue[i].GetTimestamp().Before(stOpsQueue[j].GetTimestamp())
			})

			s.docHistory.AddVersion(stOpsQueue...)
			go monitor.OpsHandled(len(stOpsQueue))

			stOpsQueue = make([]storage.StorageOperation, 0)
		}
	}
}

// NewSortedListService creates a new SortedListService object.
func NewSortedListService(chSize int, batchPeriod time.Duration, filePath string) (*SortedListService, error) {
	if chSize < 0 {
		return nil, fmt.Errorf("%s: must be GTE 0", "chSize")
	}
	if batchPeriod <= 0 {
		return nil, fmt.Errorf("%s: must be GT 0", "batchPeriod")
	}

	docHistory, err := storage.NewDocHistoryFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("storage.NewDocHistoryFromFile: %w", err)
	}

	return &SortedListService{
		docHistory:  docHistory,
		opsCh:       make(chan []storage.StorageOperation, chSize),
		batchPeriod: batchPeriod,
	}, nil
}
