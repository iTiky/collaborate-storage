package model

import (
	"fmt"
	"strings"
)

type (
	// StorageList is a light variation of storage items (storage view) used by client.
	StorageList []ListItem

	ListItem struct {
		Id    string
		Value StorageValue
	}

	// ListOperation is an operation performed on StorageList on the client side.
	ListOperation struct {
		Type     OperationType
		Index    int
		NewIndex int
		Id       string
		Value    StorageValue
	}
)

// String implements the stringer interface.
func (l StorageList) String() string {
	str := strings.Builder{}
	for i, item := range l {
		str.WriteString(fmt.Sprintf("- [%d] %d (%s)\n", i, item.Value, item.Id))
	}

	return str.String()
}

// ApplyListOperations upgrades the input StorageList to a new version using ListOperation objects.
func ApplyListOperations(l StorageList, ops ...ListOperation) (StorageList, error) {
	for i, op := range ops {
		switch op.Type {

		case InsertOperationType:
			if op.Index < 0 {
				return nil, fmt.Errorf("op[%d] (%s): index: must be GTE 0", i, op.Type)
			}
			if op.Index > len(l) {
				return nil, fmt.Errorf("op[%d] (%s): index: must be LTE than StorageList length", i, op.Type)
			}

			// Insert
			l = append(l, ListItem{})
			copy(l[op.Index+1:], l[op.Index:])
			l[op.Index] = ListItem{
				Id:    op.Id,
				Value: op.Value,
			}

		case UpdateOperationType:
			if op.Index < 0 {
				return nil, fmt.Errorf("op[%d] (%s): index: must be GTE 0", i, op.Type)
			}
			if op.Index >= len(l) {
				return nil, fmt.Errorf("op[%d] (%s): index: must be LT than StorageList length", i, op.Type)
			}

			if op.NewIndex < 0 {
				return nil, fmt.Errorf("op[%d] (%s): newIndex: must be GTE 0", i, op.Type)
			}
			if op.NewIndex >= len(l) {
				return nil, fmt.Errorf("op[%d] (%s): newIndex: must be LT than StorageList length", i, op.Type)
			}

			// Cut and insert
			id := l[op.Index].Id
			l = append(l[:op.Index], l[op.Index+1:]...)
			l = append(l, ListItem{})
			copy(l[op.NewIndex+1:], l[op.NewIndex:])
			l[op.NewIndex] = ListItem{
				Id:    id,
				Value: op.Value,
			}

		case DeleteOperationType:
			if op.Index < 0 {
				return nil, fmt.Errorf("op[%d] (%s): index: must be GTE 0", i, op.Type)
			}
			if op.Index >= len(l) {
				return nil, fmt.Errorf("op[%d] (%s): index: must be LT than StorageList length", i, op.Type)
			}

			// Cut
			l = append(l[:op.Index], l[op.Index+1:]...)

		default:
			return nil, fmt.Errorf("op[%d] (%s): unknown type", i, op.Type)

		}
	}

	return l, nil
}
