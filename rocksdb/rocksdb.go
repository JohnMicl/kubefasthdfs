package rocksdb

import (
	"fmt"
	"kubefasthdfs/logger"
	"os"

	"github.com/tecbot/gorocksdb"
)

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	DefaultDataPartitionSize = 120 * GB
	TaskWorkerInterval       = 1
)
const (
	LRUCacheSize    = 3 << 30
	WriteBufferSize = 4 * MB
)

type RocksDBStore struct {
	dir string
	db  *gorocksdb.DB
}

func NewRocksDBStore(dir string, lruCacheSize int, writeBufferSize int) (*RocksDBStore, error) {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, err
	}
	logger.Logger.Info("rocksdb dir=" + dir)
	store := &RocksDBStore{dir: dir}
	if err := store.Open(lruCacheSize, writeBufferSize); err != nil {
		return nil, err
	}
	return store, nil
}

// Open opens the RocksDB instance.
func (rs *RocksDBStore) Open(lruCacheSize int, writeBufferSize int) error {
	basedTableOptions := gorocksdb.NewDefaultBlockBasedTableOptions()
	basedTableOptions.SetBlockCache(gorocksdb.NewLRUCache(uint64(lruCacheSize)))
	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(basedTableOptions)
	opts.SetCreateIfMissing(true)
	opts.SetWriteBufferSize(writeBufferSize)
	opts.SetMaxWriteBufferNumber(2)
	opts.SetCompression(gorocksdb.NoCompression)
	db, err := gorocksdb.OpenDb(opts, rs.dir)
	if err != nil {
		err = fmt.Errorf("action[openRocksDB],err:%v", err)
		return err
	}
	rs.db = db
	return nil
}

// Put adds a new key-value pair to the RocksDB.
func (rs *RocksDBStore) Put(key, value interface{}, isSync bool) (interface{}, error) {
	wo := gorocksdb.NewDefaultWriteOptions()
	wb := gorocksdb.NewWriteBatch()
	wo.SetSync(isSync)
	defer func() {
		wo.Destroy()
		wb.Destroy()
	}()
	wb.Put([]byte(key.(string)), value.([]byte))
	if err := rs.db.Write(wo, wb); err != nil {
		return nil, err
	}
	result := value
	return result, nil
}

// BatchPut puts the key-value pairs in batch.
func (rs *RocksDBStore) BatchPut(cmdMap map[string][]byte, isSync bool) error {
	wo := gorocksdb.NewDefaultWriteOptions()
	wo.SetSync(isSync)
	wb := gorocksdb.NewWriteBatch()
	defer func() {
		wo.Destroy()
		wb.Destroy()
	}()
	for key, value := range cmdMap {
		wb.Put([]byte(key), value)
	}
	if err := rs.db.Write(wo, wb); err != nil {
		err = fmt.Errorf("action[batchPutToRocksDB],err:%v", err)
		return err
	}
	return nil
}

// Get returns the value based on the given key.
func (rs *RocksDBStore) Get(key interface{}) (interface{}, error) {
	ro := gorocksdb.NewDefaultReadOptions()
	ro.SetFillCache(false)
	defer ro.Destroy()
	return rs.db.GetBytes(ro, []byte(key.(string)))
}

// Del deletes a key-value pair.
func (rs *RocksDBStore) Del(key interface{}, isSync bool) (result interface{}, err error) {
	ro := gorocksdb.NewDefaultReadOptions()
	wo := gorocksdb.NewDefaultWriteOptions()
	wb := gorocksdb.NewWriteBatch()
	wo.SetSync(isSync)
	defer func() {
		wo.Destroy()
		ro.Destroy()
		wb.Destroy()
	}()
	slice, err := rs.db.Get(ro, []byte(key.(string)))
	if err != nil {
		return
	}
	result = slice.Data()
	err = rs.db.Delete(wo, []byte(key.(string)))
	return
}

// RocksDBSnapshot returns the RocksDB snapshot.
func (rs *RocksDBStore) RocksDBSnapshot() *gorocksdb.Snapshot {
	return rs.db.NewSnapshot()
}

// ReleaseSnapshot releases the snapshot and its resources.
func (rs *RocksDBStore) ReleaseSnapshot(snapshot *gorocksdb.Snapshot) {
	rs.db.ReleaseSnapshot(snapshot)
}

// Iterator returns the iterator of the snapshot.
func (rs *RocksDBStore) Iterator(snapshot *gorocksdb.Snapshot) *gorocksdb.Iterator {
	ro := gorocksdb.NewDefaultReadOptions()
	ro.SetFillCache(false)
	ro.SetSnapshot(snapshot)

	return rs.db.NewIterator(ro)
}
