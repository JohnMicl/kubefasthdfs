package store

import (
	"fmt"
	"kubefasthdfs/logger"
)

const (
	RAFT_STABLE_PREFIX = "raft_stable_"
)

// Get implements raft.StableStore.
func (s *Store) Get(key []byte) ([]byte, error) {
	value, err := s.rocksDBStore.Get(RAFT_STABLE_PREFIX + string(key))
	if err != nil {
		return nil, err
	}
	return value.([]byte), nil
}

// Set implements raft.StableStore.
func (s *Store) Set(key []byte, val []byte) error {
	_, err := s.rocksDBStore.Put(RAFT_STABLE_PREFIX+string(key), val, false)
	return err
}

// GetUint64 implements raft.StableStore.
func (s *Store) GetUint64(key []byte) (uint64, error) {
	value, err := s.rocksDBStore.Get(RAFT_STABLE_PREFIX + string(key))
	if err != nil {
		// 未找到，返回index0
		logger.Logger.Info(fmt.Sprintf("get unit64 %+v, %+v, err=%+v\n", string(key), string(value.([]byte)), err))
		return 0, nil
	}
	var data uint64
	if len(value.([]byte)) > 0 {
		err = decodeFromBytes(value.([]byte), &data)
		if err != nil {
			logger.Logger.Info(fmt.Sprintf("decode failed err=%+v\n", err))
			return 0, err
		}
	}
	return data, nil
}

// SetUint64 implements raft.StableStore.
func (s *Store) SetUint64(key []byte, val uint64) error {
	data, err := encode2Bytes(&val)
	if err != nil {
		return err
	}
	_, err = s.rocksDBStore.Put(RAFT_STABLE_PREFIX+string(key), data, false)
	return err
}
