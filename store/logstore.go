package store

import (
	"bytes"
	"encoding/binary"

	"github.com/hashicorp/raft"
)

const (
	RAFT_LOG_PREFIX = "raft_log_"
)

// FirstIndex returns the first index written. 0 for no entries.
func (s *Store) FirstIndex() (uint64, error) {
	it := s.rocksDBStore.LastestIterator()
	defer it.Close()

	// Find the first key
	it.Seek([]byte(RAFT_LOG_PREFIX))

	if bytes.HasPrefix(it.Key().Data(), []byte(RAFT_LOG_PREFIX)) {
		return binary.BigEndian.Uint64(it.Key().Data()[len(RAFT_LOG_PREFIX):]), nil
	}

	return 0, nil
}

// LastIndex returns the last index written. 0 for no entries.
func (s *Store) LastIndex() (uint64, error) {
	it := s.rocksDBStore.LastestIterator()
	defer it.Close()

	it.Seek([]byte(RAFT_LOG_PREFIX))

	var lastIndex uint64 = 0
	for it.Valid() {
		key := it.Key().Data()
		if !bytes.HasPrefix(key, []byte(RAFT_LOG_PREFIX)) {
			break
		}
		// 解析索引
		indexbyte := (key)[len(RAFT_LOG_PREFIX):]
		if len(indexbyte) == 8 {
			index := binary.BigEndian.Uint64(indexbyte)
			if index > lastIndex {
				lastIndex = index
			}
		}
		it.Next()
	}

	return lastIndex, nil
}

// GetLog gets a log entry at a given index.
func (s *Store) GetLog(index uint64, log *raft.Log) error {
	key := RAFT_LOG_PREFIX + string(uint642byte(index))
	data, err := s.rocksDBStore.Get(key)
	if err != nil {
		return err
	}

	// here we use json to recover data.
	// I don't know what's the best way to encode, decode struct
	return decodeFromBytes(data.([]byte), log)
}

// StoreLog stores a log entry.
func (s *Store) StoreLog(log *raft.Log) error {
	data, err := encode2Bytes(log)
	if err != nil {
		return nil
	}
	key := RAFT_LOG_PREFIX + string(uint642byte(log.Index))
	_, err = s.rocksDBStore.Put(key, data, false)
	return err
}

// StoreLogs stores multiple log entries. By default the logs stored may not be contiguous with previous logs (i.e. may have a gap in Index since the last log written).
// If an implementation can't tolerate this it may optionally implement `MonotonicLogStore` to indicate that this is not allowed.
// This changes Raft's behaviour after restoring a user snapshot to remove all previous logs instead of relying on a "gap" to signal the discontinuity between logs before the snapshot and logs after.
func (s *Store) StoreLogs(logs []*raft.Log) error {
	cmdMap := make(map[string][]byte, len(logs))

	for _, log := range logs {
		valuedata, err := encode2Bytes(log)
		if err != nil {
			return nil
		}
		key := RAFT_LOG_PREFIX + string(uint642byte(log.Index))
		cmdMap[key] = valuedata
	}

	return s.rocksDBStore.BatchPut(cmdMap, false)
}

// DeleteRange deletes a range of log entries. The range is inclusive.
func (s *Store) DeleteRange(min, max uint64) error {
	for i := min; i <= max; i++ {
		key := RAFT_LOG_PREFIX + string(uint642byte(i))
		_, err := s.rocksDBStore.Del(key, false)
		if err != nil {
			return err
		}
	}
	return nil
}
