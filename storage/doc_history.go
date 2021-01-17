package storage

import (
	"sync"

	"github.com/itiky/collaborate-storage/model"
)

type (
	// DocumentHistory keeps the document history alongside cache used to client requests.
	DocumentHistory struct {
		sync.RWMutex
		// List of document versions
		documents []Document
		// Latest version storage state
		storage *Storage
		// The current document version
		latestVersion int
	}

	Document struct {
		Version int
		// Storage operations to apply on previous document version in order to upgrade it
		InputOperations []StorageOperation
		// Client model.StorageList operations to apply in order to upgrade it
		OutputOperations []model.ListOperation
	}
)

// AddVersion adds a new Document version caching input/output operations.
func (h *DocumentHistory) AddVersion(stOps ...StorageOperation) {
	if len(stOps) == 0 {
		return
	}

	h.Lock()
	defer h.Unlock()

	// Update the storage state
	listOps := h.storage.ApplyOperations(stOps...)

	// Add a new document version
	stOpsCopy := make([]StorageOperation, len(stOps))
	copy(stOpsCopy, stOps)
	newDoc := Document{
		InputOperations:  stOpsCopy,
		OutputOperations: listOps,
	}

	historyLen := len(h.documents)
	if historyLen > 0 {
		newDoc.Version = h.documents[historyLen-1].Version + 1
	}

	h.documents = append(h.documents, newDoc)

	// Update the version
	h.latestVersion = len(h.documents) - 1
}

// RemoveVersion removes an existing version.
// All the client must be notified to redownload the latest version (as it might change).
// TODO: that requires rebuilding the cache for [version:] documents
func (h *DocumentHistory) RemoveVersion(version int) {
	panic("not implemented")
}

// GetOutputDiffWithLatest returns snapshot version and model.ListOperation objects
// for client to apply on a local snapshot in order to upgrade it to the latest one.
func (h *DocumentHistory) GetOutputDiffWithLatest(version int) (int, []model.ListOperation) {
	h.RLock()
	defer h.RUnlock()

	startVersion := version + 1
	if !h.IsVersionValid(startVersion) {
		return version, nil
	}

	diffOps := make([]model.ListOperation, 0)
	for i := startVersion; i <= h.latestVersion; i++ {
		diffOps = append(diffOps, h.documents[i].OutputOperations...)
	}

	return h.latestVersion, diffOps
}

// GetOutputSnapshot returns latest snapshot version and data.
// Action is performed for new client connections in order to get the local snapshot.
func (h *DocumentHistory) GetOutputSnapshot() (int, model.StorageList) {
	h.Lock()
	defer h.Unlock()

	return h.latestVersion, h.storage.Export()
}

// BuildStorage builds a Storage snapshot for the specified version.
// Makes possible to build a snapshot for all previous document versions.
func (h *DocumentHistory) BuildStorage(version int) *Storage {
	if !h.IsVersionValid(version) {
		return nil
	}

	storage := NewStorage()
	for i := 0; i <= version; i++ {
		docOps := h.documents[i].InputOperations
		storage.ApplyOperations(docOps...)
	}

	return storage
}

// IsVersionValid checks if document version exists.
func (h *DocumentHistory) IsVersionValid(version int) bool {
	return version < len(h.documents)
}

// NewDocumentHistory creates a new empty DocumentHistory object.
func NewDocumentHistory() *DocumentHistory {
	return &DocumentHistory{
		documents: make([]Document, 0),
		storage:   NewStorage(),
	}
}
