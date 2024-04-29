package store

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kubefasthdfs/logger"
	"kubefasthdfs/rocksdb"

	"github.com/hashicorp/raft"
	"github.com/tecbot/gorocksdb"
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
	snapshot := &fsmSnapshot{
		db:         f.rocksDBStore,
		snapshotCh: make(chan *gorocksdb.Snapshot),
	}

	go func() {
		snap := snapshot.db.RocksDBSnapshot()
		snapshot.snapshotCh <- snap
	}()

	return snapshot, nil
}

// Restore stores the key-value store to a previous state.
func (f *fsm) Restore(rc io.ReadCloser) error {
	cmdMap := make(map[string][]byte)

	// here we decode data from rc, the data type is fsmSnapShot
	decoder := gob.NewDecoder(rc)
	var snapshot fsmSnapShot
	err := decoder.Decode(&snapshot)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	// when recover, we need to get all []fsmSnapShotUint data,
	// after that in mem, put it to rocksdb to recover data
	for id := 0; id < len(snapshot.UnitList); id++ {
		logger.Logger.Info(fmt.Sprintf("resotre id=%+v, data=%+v\n", id, snapshot.UnitList[id]))
		datakv := snapshot.UnitList[id]
		cmdMap[string(datakv.Key)] = datakv.Value
	}
	f.rocksDBStore.BatchPut(cmdMap, true)
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

type fsmSnapshot struct {
	db         *rocksdb.RocksDBStore
	snapshotCh chan *gorocksdb.Snapshot
}

// Persist implements raft.FSMSnapshot.
func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	snap := <-f.snapshotCh
	iter := f.db.Iterator(snap)
	defer iter.Close()

	snapshots := fsmSnapShot{}
	for iter.SeekToFirst(); iter.Valid(); iter.Next() {
		// how to choose a better way to encode store this?
		uintkv := fsmSnapShotUint{
			Key:   iter.Key().Data(),
			Value: iter.Value().Data(),
		}
		logger.Logger.Info(fmt.Sprintf("persist write data data=%+v", uintkv))
		snapshots.UnitList = append(snapshots.UnitList, uintkv)
	}

	buffer := new(bytes.Buffer)
	encoder := gob.NewEncoder(buffer)
	err := encoder.Encode(snapshots)
	if err != nil {
		// snapshot need to release
		f.db.ReleaseSnapshot(snap)
		return errors.New("failed to snapshot, release it now")
	}
	sink.Write(buffer.Bytes())
	if err := sink.Close(); err != nil {
		return errors.New("failed close write data to snapshot")
	}
	// snapshot also need to release
	return nil
}

// Release implements raft.FSMSnapshot.
func (f *fsmSnapshot) Release() {
	// 如果snapshotCh中有未使用的RocksDB快照，释放它
	select {
	case snap := <-f.snapshotCh:
		// do release snap
		f.db.ReleaseSnapshot(snap)
	default:
	}
}

type fsmSnapShot struct {
	UnitList []fsmSnapShotUint
}

type fsmSnapShotUint struct {
	Key   []byte
	Value []byte
}
