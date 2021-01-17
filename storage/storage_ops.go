package storage

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/itiky/collaborate-storage/model"
)

type (
	// StorageOperation is an operation performed on Storage to update its state.
	StorageOperation interface {
		// Update the storage state
		Apply(s *Storage) *model.ListOperation
		// Only used for tests
		GetId() uuid.UUID
		// Only used for tests
		GetTimestamp() time.Time
	}

	// SetOperation implements StorageOperation interface for create/update operation.
	SetOperation struct {
		Id        uuid.UUID
		Value     model.StorageValue
		IsDeleted bool
		UpdatedBy model.ClientId
		UpdatedAt time.Time
	}

	// DeleteOperation implements StorageOperation interface for delete operation.
	DeleteOperation struct {
		Id        uuid.UUID
		DeletedBy model.ClientId
		DeletedAt time.Time
	}
)

// Apply implements StorageOperation interface.
func (o SetOperation) Apply(s *Storage) *model.ListOperation {
	return s.set(o.Id, o.Value, o.UpdatedBy, o.UpdatedAt)
}

// GetId implements StorageOperation interface.
func (o SetOperation) GetId() uuid.UUID {
	return o.Id
}

// GetTimestamp implements StorageOperation interface.
func (o SetOperation) GetTimestamp() time.Time {
	return o.UpdatedAt
}

// Apply implements StorageOperation interface.
func (o DeleteOperation) Apply(s *Storage) *model.ListOperation {
	return s.delete(o.Id, o.DeletedBy, o.DeletedAt)
}

// GetId implements StorageOperation interface.
func (o DeleteOperation) GetId() uuid.UUID {
	return o.Id
}

// GetTimestamp implements StorageOperation interface.
func (o DeleteOperation) GetTimestamp() time.Time {
	return o.DeletedAt
}

// NewSetOperation creates a valid StorageOperation object.
func NewSetOperation(itemId string, itemValue model.StorageValue, clientId model.ClientId, timestamp time.Time) (SetOperation, error) {
	id, err := uuid.Parse(itemId)
	if err != nil {
		return SetOperation{}, fmt.Errorf("%s: invalid: %w", "itemId", err)
	}
	if timestamp.IsZero() {
		return SetOperation{}, fmt.Errorf("%s: zero", "timestamp")
	}

	return SetOperation{
		Id:        id,
		Value:     itemValue,
		UpdatedBy: clientId,
		UpdatedAt: timestamp,
	}, nil
}

// NewDeleteOperation creates a valid StorageOperation object.
func NewDeleteOperation(itemId string, clientId model.ClientId, timestamp time.Time) (DeleteOperation, error) {
	id, err := uuid.Parse(itemId)
	if err != nil {
		return DeleteOperation{}, fmt.Errorf("%s: invalid: %w", "itemId", err)
	}
	if timestamp.IsZero() {
		return DeleteOperation{}, fmt.Errorf("%s: zero", "timestamp")
	}

	return DeleteOperation{
		Id:        id,
		DeletedBy: clientId,
		DeletedAt: timestamp,
	}, nil
}
