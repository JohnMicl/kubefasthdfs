package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"kubefasthdfs/logger"
	"net"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/raft"
)

const (
	retainSnapshotCount = 2
	raftTimeout         = 10 * time.Second
)

type Store struct {
	RaftDir  string
	RaftBind string
	raft     *raft.Raft // The consensus mechanism
	inmem    bool
	mu       sync.Mutex
	m        map[string]string // The key-value store for the system.
}

// NewStore returns a new Store.
func NewStore(inmem bool) *Store {
	return &Store{
		inmem: inmem,
		m:     make(map[string]string),
	}
}

func (s *Store) Open(enableSingle bool, localID string) error {
	// Setup Raft configuration.
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(localID)

	// Setup Raft communication.
	addr, err := net.ResolveTCPAddr("tcp", s.RaftBind)
	if err != nil {
		return err
	}
	transport, err := raft.NewTCPTransport(s.RaftBind, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return err
	}

	// Create the snapshot store. This allows the Raft to truncate the log.
	snapshots, err := raft.NewFileSnapshotStore(s.RaftDir, retainSnapshotCount, os.Stderr)
	if err != nil {
		return fmt.Errorf("file snapshot store: %s", err)
	}

	// Create the log store and stable store.
	var logStore raft.LogStore
	var stableStore raft.StableStore
	if s.inmem {
		logStore = raft.NewInmemStore()
		stableStore = raft.NewInmemStore()
	} else {
		return errors.New("only mem store now, others not support")
	}

	// Instantiate the Raft systems.
	ra, err := raft.NewRaft(config, (*fsm)(s), logStore, stableStore, snapshots, transport)
	if err != nil {
		return fmt.Errorf("new raft: %s", err)
	}
	s.raft = ra

	if enableSingle {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		ra.BootstrapCluster(configuration)
	}

	return nil
}

func (s *Store) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.m[key], nil
}

func (s *Store) Set(key, value string) error {
	c := &command{
		Op:    "set",
		Key:   key,
		Value: value,
	}
	return s.applycmd(c)
}

// Delete deletes the given key.
func (s *Store) Delete(key string) error {
	c := &command{
		Op:  "delete",
		Key: key,
	}
	return s.applycmd(c)
}

func (s *Store) applycmd(c *command) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	f := s.raft.Apply(b, raftTimeout)
	return f.Error()
}

func (s *Store) Join(nodeID, addr string) error {
	logger.Logger.Info(fmt.Sprintf("received join request for remote node %s at %s\n", nodeID, addr))

	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		logger.Logger.Error(fmt.Sprintf("failed to get raft configuration: %v\n", err))
		return err
	}

	for _, srv := range configFuture.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if srv.Address == raft.ServerAddress(addr) && srv.ID == raft.ServerID(nodeID) {
				logger.Logger.Info(fmt.Sprintf("node %s at %s already member of cluster, ignoring join request\n", nodeID, addr))
				return nil
			}

			future := s.raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, addr, err)
			}
		}
	}

	f := s.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if f.Error() != nil {
		return f.Error()
	}
	logger.Logger.Info(fmt.Sprintf("node %s at %s joined successfully\n", nodeID, addr))
	return nil
}
