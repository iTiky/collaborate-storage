package storage

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/itiky/collaborate-storage/model"
)

type (
	// Item keep Storage element data.
	Item struct {
		Id        uuid.UUID
		Value     model.StorageValue
		IsDeleted bool
		UpdatedBy model.ClientId
		UpdatedAt time.Time
	}
)

// String implements stringer interface.
func (i Item) String() string {
	raw, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return fmt.Sprintf("marshal: %v", err)
	}

	return string(raw)
}

// NewStorageItem creates a new Item object (no validation as it is used internaly).
func NewStorageItem(itemId uuid.UUID, itemValue model.StorageValue, clientId model.ClientId, timestamp time.Time) *Item {
	return &Item{
		Id:        itemId,
		Value:     itemValue,
		UpdatedBy: clientId,
		UpdatedAt: timestamp,
	}
}
