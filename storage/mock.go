package storage

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/itiky/collaborate-storage/model"
)

// GenAndSaveInitialStorage generates random storage objects and saves it to file system.
func GenAndSaveInitialStorage(filePath string, storageSize int) error {
	if storageSize <= 0 {
		return fmt.Errorf("%s: must be GT 0", "storageSize")
	}

	log.Printf("Creating and sorting objects...")
	objs := newStorageMockObjs(storageSize, time.Now())

	log.Printf("GOB marshal...")
	objsRaw := new(bytes.Buffer)
	if err := gob.NewEncoder(objsRaw).Encode(objs); err != nil {
		return fmt.Errorf("GOB marshal: %w", err)
	}

	log.Printf("Saving file...")
	if err := ioutil.WriteFile(filePath, objsRaw.Bytes(), 0644); err != nil {
		return fmt.Errorf("write to file (%s): %w", filePath, err)
	}

	log.Printf("Done")

	return nil
}

// NewDocHistoryFromFile builds the DocumentHistory object with a single version (v0) from the file.
func NewDocHistoryFromFile(filePath string) (*DocumentHistory, error) {
	log.Printf("Reading file...")
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file (%s): %w", filePath, err)
	}

	log.Printf("GOB unmarshal...")
	objs := make([]Item, 0)
	buf := bytes.NewBuffer(data)
	if err := gob.NewDecoder(buf).Decode(&objs); err != nil {
		return nil, fmt.Errorf("GOB unmarshal: %w", err)
	}

	log.Printf("Storage creation...")
	storage := newStorageFromObjs(objs)

	log.Printf("DocHistory creation...")
	docHistory := NewDocumentHistory()
	docHistory.storage = storage
	docHistory.documents = append(docHistory.documents, Document{
		Version:          0,
		InputOperations:  nil,
		OutputOperations: nil,
	})

	log.Printf("Storage created: %d items", len(docHistory.storage.idDataMatch))

	return docHistory, nil
}

// newStorageMockObjs builds mocks storage objects.
func newStorageMockObjs(n int, now time.Time) []Item {
	objs := make([]Item, 0, n)
	for i := 0; i < n; i++ {
		objs = append(objs, newStorageMockObj(now))
	}

	sort.Slice(objs, func(i, j int) bool {
		return objs[i].Value < objs[j].Value
	})

	return objs
}

// newStorageMockObj builds mocks storage object.
func newStorageMockObj(now time.Time) Item {
	return Item{
		Id:        uuid.New(),
		Value:     model.StorageValue(rand.Int31()),
		IsDeleted: false,
		UpdatedBy: 0,
		UpdatedAt: now,
	}
}

// newStorageFromObjs builds the Storage object from storage items.
func newStorageFromObjs(objs []Item) *Storage {
	s := NewStorage()

	s.list = make([]*Item, 0, len(objs))
	for idx := 0; idx < len(objs); idx++ {
		item := &objs[idx]
		itemIdStr := item.Id.String()
		s.idDataMatch[itemIdStr] = item
		s.list = append(s.list, item)
	}

	return s
}
