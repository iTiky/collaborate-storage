package storage

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/itiky/collaborate-storage/model"
)

type (
	// Storage keeps Item elements alongside the sorted list view.
	// Storage implements the "soft delete" methodology.
	Storage struct {
		list        []*Item
		idDataMatch map[string]*Item
	}
)

// String implements stringer interface.
func (s *Storage) String() string {
	str := strings.Builder{}
	for i, item := range s.list {
		str.WriteString(fmt.Sprintf("- [%d] %d (%s)\n", i, item.Value, item.Id))
	}

	return str.String()
}

// Export builds a model.StorageList slice (snapshot).
func (s *Storage) Export() model.StorageList {
	list := make(model.StorageList, 0, len(s.list))
	for _, item := range s.list {
		list = append(list, model.ListItem{
			Id:    item.Id.String(),
			Value: item.Value,
		})
	}

	return list
}

// ApplyOperations updates storage state with StorageOperation list and returns list operations performed.
func (s *Storage) ApplyOperations(ops ...StorageOperation) []model.ListOperation {
	listOps := make([]model.ListOperation, 0, len(ops))

	for _, op := range ops {
		if op == nil {
			continue
		}

		if listOp := op.Apply(s); listOp != nil {
			listOps = append(listOps, *listOp)
		}
	}

	return listOps
}

// set creates a new / updates an existing Item while updating the sorted list index state.
func (s *Storage) set(itemId uuid.UUID, itemValue model.StorageValue, clientId model.ClientId, timestamp time.Time) *model.ListOperation {
	itemIdStr := itemId.String()

	item, found := s.idDataMatch[itemIdStr]
	if !found {
		// Add a new Item
		item = NewStorageItem(itemId, itemValue, clientId, timestamp)
		s.idDataMatch[itemIdStr] = item

		// Insert
		itemIdxToInsert := s.findItemIdxLTTarget(item)
		s.list = append(s.list, nil)
		copy(s.list[itemIdxToInsert+1:], s.list[itemIdxToInsert:])
		s.list[itemIdxToInsert] = item

		return &model.ListOperation{
			Type:  model.InsertOperationType,
			Id:    itemId.String(),
			Index: itemIdxToInsert,
			Value: itemValue,
		}
	}

	// Update an existing item (that might break the sorting, so we have to cut/insert)
	// Cut
	itemIdxToCut := s.findItemIdx(item)
	s.list = append(s.list[:itemIdxToCut], s.list[itemIdxToCut+1:]...)

	// Update
	item.Value = itemValue
	item.UpdatedBy, item.UpdatedAt = clientId, timestamp

	// Insert
	itemIdxToInsert := s.findItemIdxLTTarget(item)
	s.list = append(s.list, nil)
	copy(s.list[itemIdxToInsert+1:], s.list[itemIdxToInsert:])
	s.list[itemIdxToInsert] = item

	return &model.ListOperation{
		Type:     model.UpdateOperationType,
		Id:       itemIdStr,
		Index:    itemIdxToCut,
		NewIndex: itemIdxToInsert,
		Value:    itemValue,
	}
}

// delete deletes an existing Item while updating the sorted list index state.
func (s *Storage) delete(itemId uuid.UUID, clientId model.ClientId, timestamp time.Time) *model.ListOperation {
	itemIdStr := itemId.String()

	item, found := s.idDataMatch[itemIdStr]
	if !found || item.IsDeleted {
		return nil
	}

	// Mark as deleted
	item.IsDeleted = true
	item.UpdatedBy, item.UpdatedAt = clientId, timestamp

	// Cut
	itemIdx := s.findItemIdx(item)
	s.list = append(s.list[:itemIdx], s.list[itemIdx+1:]...)

	return &model.ListOperation{
		Type:  model.DeleteOperationType,
		Id:    itemIdStr,
		Index: itemIdx,
	}
}

// findItemIdxLTTarget used by set/delete funcs: returns the the leftmost item index in the sorted list.
func (s *Storage) findItemIdxLTTarget(item *Item) int {
	return sort.Search(len(s.list), func(i int) bool {
		return s.list[i].Value >= item.Value
	})
}

// findItemIdx used by set/delete funcs: returns the specified item index.
// Panics on failure (should not happen).
func (s *Storage) findItemIdx(item *Item) int {
	itemIdxPrev := s.findItemIdxLTTarget(item)
	if itemIdxPrev == len(s.list) {
		panic("item not found: LT target")
	}

	for i := itemIdxPrev; i < len(s.list); i++ {
		if s.list[i].Id == item.Id {
			return i
		}
	}
	panic("item not found: by id")

	return -1
}

// NewStorage creates a new Storage object.
func NewStorage() *Storage {
	return &Storage{
		idDataMatch: make(map[string]*Item),
	}
}
