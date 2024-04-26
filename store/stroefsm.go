package store

import (
	"encoding/json"
	"fmt"
	"io"
	"kubefasthdfs/logger"

	"github.com/hashicorp/raft"
)

type fsm Store

// Apply applies a Raft log entry to the key-value store.
func (f *fsm) Apply(l *raft.Log) interface{} {
	var c command
	if err := json.Unmarshal(l.Data, &c); err != nil {
		panic(fmt.Sprintf("failed to unmarshal command: %s", err.Error()))
	}

	switch c.Op {
	case "set":
		return f.applySet(c.Key, c.Value)
	case "delete":
		return f.applyDelete(c.Key)
	default:
		panic(fmt.Sprintf("unrecognized command op: %s", c.Op))
	}
}

// Snapshot returns a snapshot of the key-value store.
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	return nil, nil
}

// Restore stores the key-value store to a previous state.
func (f *fsm) Restore(rc io.ReadCloser) error {
	return nil
}

func (f *fsm) applySet(key, value string) interface{} {
	result, err := f.rocksDBStore.Put(key, []byte(value), true)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("failed to put value in rocksdb %+v\n", err))
		return nil
	}
	return result
}

func (f *fsm) applyDelete(key string) interface{} {
	result, err := f.rocksDBStore.Del(key, true)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("failed to delete key=%+v in rocksdb %+v\n", key, err))
		return nil
	}
	return result
}
