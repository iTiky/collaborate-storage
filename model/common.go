package model

type (
	ClientId uint32

	StorageValue int32
)

type OperationType string

const (
	InsertOperationType OperationType = "insert"
	UpdateOperationType OperationType = "update"
	DeleteOperationType OperationType = "delete"
)
