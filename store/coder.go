package store

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"

	"github.com/hashicorp/raft"
)

func encode2Bytes(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	switch data.(type) {
	case *raft.Log:
		err := enc.Encode(data)
		return buf.Bytes(), err
	case *uint64:
		err := enc.Encode(data)
		return buf.Bytes(), err
	default:
		return nil, fmt.Errorf("unsupported type")
	}
}

func decodeFromBytes(data []byte, target interface{}) error {
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	switch target.(type) {
	case *raft.Log:
		return dec.Decode(target)
	case *uint64:
		return dec.Decode(target)
	default:
		return fmt.Errorf("unsupported type")
	}
}

func uint642byte(data uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(data))
	return buf
}
