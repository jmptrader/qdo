package leveldb

import (
	"errors"

	"github.com/borgenk/qdo/third_party/github.com/syndtr/goleveldb/leveldb"
	"github.com/borgenk/qdo/third_party/github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/borgenk/qdo/third_party/github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/borgenk/qdo/third_party/github.com/syndtr/goleveldb/leveldb/util"

	"github.com/borgenk/qdo/store"
)

const Name = "leveldb"

func init() {
	store.RegisterStore(Name, NewStore)
}

var (
	db *leveldb.DB
)

type Store struct {
	db *leveldb.DB
}

func NewStore(filePath string) (store.Store, error) {
	storey := Store{}
	storey.Open(filePath)
	return &storey, nil
}

func (s *Store) Open(filePath string) error {
	var err error
	s.db, err = leveldb.OpenFile(filePath, nil)
	if err != nil {
		return store.ErrOpenFile
	}
	return nil
}

func (s *Store) Close() error {
	s.db.Close()
	return nil
}

func (s *Store) Get(key []byte) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (s *Store) Put(key, val []byte) error {
	wo := &opt.WriteOptions{}
	err := s.db.Put(key, val, wo)
	if err != nil {
		return store.ErrWrite
	}
	return nil
}

func (s *Store) Delete(key []byte) error {
	return s.db.Delete(key, nil)
}

func (s *Store) NewIterator(r *store.Range) store.Iterator {
	var rr = &util.Range{}
	if r != nil {
		rr.Start = r.Start
		rr.Limit = r.Limit
	}
	return &Iterator{s.db.NewIterator(rr, nil)}
}

type Iterator struct {
	iter iterator.Iterator
}

func (i *Iterator) Seek(key []byte) {
	i.iter.Seek(key)
}

func (i *Iterator) Next() bool {
	return i.iter.Next()
}

func (i *Iterator) Prev() bool {
	return i.iter.Prev()
}

func (i *Iterator) Key() []byte {
	return i.iter.Key()
}

func (i *Iterator) Value() []byte {
	return i.iter.Value()
}

func (i *Iterator) Valid() bool {
	return i.iter.Valid()
}

func (i *Iterator) Close() {
	i.iter.Release()
}
