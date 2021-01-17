package model

// Get the latest list snapshot RPC request.
type (
	GetListSnapshotRequest struct {
		// ClientID
		ClientId ClientId
	}

	GetListSnapshotResponse struct {
		// Snapshot version
		Version int
		// Snapshot data
		Data StorageList
	}
)

// Update the list RPC request.
type (
	UpdateListRequest struct {
		// Update source
		ClientId ClientId
		// Client snapshot version
		Version int
		// Update operations
		Operations []OperationRequest
	}

	OperationRequest struct {
		Type  OperationType
		Id    string
		Value StorageValue
	}

	UpdateListResponse struct{}
)

// Get snapshot update operation to bump local snapshot version.
type (
	GetListUpdatesRequest struct {
		// Local snapshot version
		Version int
	}

	GetListUpdatesResponse struct {
		// Snapshot version
		Version int
		// Operations to apply in order to upgrade GetListUpdatesRequest.Version tot Version
		Operations []ListOperation
	}
)
