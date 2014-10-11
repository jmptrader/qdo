package store

import (
	"errors"
)

var (
	ErrOpenFile       = errors.New("error open file")
	ErrWrite          = errors.New("error write")
	ErrNotImplemented = errors.New("error not implemented")
)

var stores = map[string]StoreConstructor{}

type Store interface {
	Open(filePath string) error
	Put(key, val []byte) error
	Delete(key []byte) error
	Close() error
	NewIterator(r *Range) Iterator
}

type Iterator interface {
	Seek([]byte)
	Next() bool
	Prev() bool

	Key() []byte
	Value() []byte
	Valid() bool

	Close()
}

type Range struct {
	Start []byte
	Limit []byte
}

type StoreConstructor func(string) (Store, error)

func RegisterStore(name string, NewStore StoreConstructor) {
	stores[name] = NewStore
}

func GetStoreConstructor(name string, filepath string) (Store, error) {
	return stores[name](filepath)
}
