package storage

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/itiky/collaborate-storage/model"
)

const BenchStorageSize = 1000000

// Test adds/removes item to/from the storage and checks data integrity.
func Test_Storage_Sorting(t *testing.T) {
	storage := NewStorage()

	isSorted := func(comment string) {
		t.Logf("%s:\n%s", comment, storage.String())

		require.GreaterOrEqual(t, len(storage.idDataMatch), len(storage.list), "list/dataMap length mismatch")

		prevValue := model.StorageValue(math.MinInt32)
		for _, item := range storage.list {
			require.LessOrEqual(t, prevValue, item.Value, "item.Value check")
			require.False(t, item.IsDeleted, "item isDeleted")

			prevValue = item.Value
		}
	}

	// add a few items
	{
		newValue := model.StorageValue(5)
		storage.set(uuid.New(), newValue, 0, time.Time{})
		isSorted(fmt.Sprintf("Adding %d", newValue))

		newValue = model.StorageValue(1)
		storage.set(uuid.New(), newValue, 0, time.Time{})
		isSorted(fmt.Sprintf("Adding %d", newValue))

		newValue = model.StorageValue(10)
		storage.set(uuid.New(), newValue, 0, time.Time{})
		isSorted(fmt.Sprintf("Adding %d", newValue))

		newValue = model.StorageValue(8)
		storage.set(uuid.New(), newValue, 0, time.Time{})
		isSorted(fmt.Sprintf("Adding %d", newValue))

		newValue = model.StorageValue(-1)
		storage.set(uuid.New(), newValue, 0, time.Time{})
		isSorted(fmt.Sprintf("Adding %d", newValue))
	}

	// remove a few items
	{
		idx := 0
		storage.delete(storage.list[idx].Id, 0, time.Time{})
		isSorted(fmt.Sprintf("Removing [%d]", idx))

		idx = 3
		storage.delete(storage.list[idx].Id, 0, time.Time{})
		isSorted(fmt.Sprintf("Removing [%d]", idx))

		idx = 1
		storage.delete(storage.list[idx].Id, 0, time.Time{})
		isSorted(fmt.Sprintf("Removing [%d]", idx))

		idx = 1
		storage.delete(storage.list[idx].Id, 0, time.Time{})
		isSorted(fmt.Sprintf("Removing [%d]", idx))

		idx = 0
		storage.delete(storage.list[idx].Id, 0, time.Time{})
		isSorted(fmt.Sprintf("Removing [%d]", idx))

		require.Len(t, storage.idDataMatch, 5)
		for _, item := range storage.idDataMatch {
			require.True(t, item.IsDeleted)
		}
	}
}

// Test applies StorageOperation and checks that returned model.ListOperation objects can build an equal model.StorageList.
func Test_Storage_ModelList(t *testing.T) {
	storage := NewStorage()
	var modelList model.StorageList
	now := time.Now()

	newInsertOp := func() SetOperation {
		op, err := NewSetOperation(uuid.New().String(), model.StorageValue(rand.Int31()), 0, now)
		require.NoError(t, err)
		return op
	}

	newUpdateOp := func(id string) SetOperation {
		op, err := NewSetOperation(id, model.StorageValue(rand.Int31()), 0, now)
		require.NoError(t, err)
		return op
	}

	newDeleteOp := func(id string) DeleteOperation {
		op, err := NewDeleteOperation(id, 0, now)
		require.NoError(t, err)
		return op
	}

	checkLists := func(comment string, storageList []*Item, modelList model.StorageList) {
		t.Log(comment)
		t.Logf("StorageList:\n%s", storage)
		t.Logf("ModelList:\n%s", modelList)

		require.Len(t, modelList, len(storageList), "len mismatch")
		for i := 0; i < len(modelList); i++ {
			storageItem := storage.list[i]
			modelItem := modelList[i]
			require.Equal(t, storageItem.Id.String(), modelItem.Id, "item[%d].Id", i)
			require.Equal(t, storageItem.Value, modelItem.Value, "item[%d].Value", i)
		}
	}

	// initial inserts
	storageOps1 := []StorageOperation{
		newInsertOp(),
		newInsertOp(),
		newInsertOp(),
		newInsertOp(),
		newInsertOp(),
	}
	{
		listOps := storage.ApplyOperations(storageOps1...)
		list, err := model.ApplyListOperations(modelList, listOps...)
		require.NoError(t, err)
		checkLists("storageOps1", storage.list, list)
		modelList = list
	}

	// insert, update, delete
	storageOps2 := []StorageOperation{
		newInsertOp(),
		newUpdateOp(storageOps1[0].GetId().String()),
		newDeleteOp(storageOps1[4].GetId().String()),
	}
	{
		listOps := storage.ApplyOperations(storageOps2...)
		list, err := model.ApplyListOperations(modelList, listOps...)
		require.NoError(t, err)
		checkLists("storageOps2", storage.list, list)
		modelList = list
	}

	// update twice
	storageOps3 := []StorageOperation{
		newUpdateOp(storageOps1[1].GetId().String()),
		newUpdateOp(storageOps1[1].GetId().String()),
	}
	{
		listOps := storage.ApplyOperations(storageOps3...)
		list, err := model.ApplyListOperations(modelList, listOps...)
		require.NoError(t, err)
		checkLists("storageOps3", storage.list, list)
		modelList = list
	}

	// delete twice
	storageOps4 := []StorageOperation{
		newDeleteOp(storageOps2[0].GetId().String()),
		newDeleteOp(storageOps2[0].GetId().String()),
	}
	{
		listOps := storage.ApplyOperations(storageOps4...)
		list, err := model.ApplyListOperations(modelList, listOps...)
		require.NoError(t, err)
		checkLists("storageOps4", storage.list, list)
		modelList = list
	}
}

func Benchmark_Storage_Insert(b *testing.B) {
	now := time.Now()
	s := newStorageFromObjs(newStorageMockObjs(BenchStorageSize, now))
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		obj := newStorageMockObj(now)
		s.set(obj.Id, obj.Value, obj.UpdatedBy, obj.UpdatedAt)
	}
}

func Benchmark_Storage_Update(b *testing.B) {
	now := time.Now()
	s := newStorageFromObjs(newStorageMockObjs(BenchStorageSize, now))
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		obj := s.list[rand.Intn(BenchStorageSize)]
		s.set(obj.Id, model.StorageValue(rand.Int31()), obj.UpdatedBy, obj.UpdatedAt)
	}
}

func Benchmark_Storage_Delete(b *testing.B) {
	now := time.Now()
	s := newStorageFromObjs(newStorageMockObjs(BenchStorageSize, now))
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		if len(s.list) == 0 {
			return
		}
		obj := s.list[rand.Intn(len(s.list))]
		s.delete(obj.Id, obj.UpdatedBy, obj.UpdatedAt)
	}
}
