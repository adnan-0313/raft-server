package raft

import "errors"

// Log entries are replicated to all members of the Raft cluster
// and form the heart of the replicated state machine.
type Log struct {
	Index uint64
	Term  uint64
	Data  []byte
}

// LogStore is used to provide an interface for storing
// and retrieving logs in a durable fashion
type LogStore interface {
	// Returns the last index written. 0 for no entries.
	LastIndex() (uint64, error)

	// Gets a log entry at a given index
	GetLog(index uint64) (*Log, error)

	// Stores a log entry
	StoreLog(log *Log) error

	// Deletes a range of log entries. The range is inclusive.
	DeleteRange(min, max uint64) error

	// Empty returns true if log storage is empty
	Empty() bool
}

type LogStorage struct {
	records []*Log
}

func (l *LogStorage) Empty() bool {
	if size := len(l.records); size == 0 {
		return true
	}

	return false
}

func (l *LogStorage) LastIndex() (uint64, error) {
	if !l.Empty() {
		return l.records[len(l.records)-1].Index, nil
	}
	return 0, errors.New("no logs found")
}

func (l *LogStorage) LastTerm() (uint64, error) {
	if l.Empty() {
		return 0, errors.New("no logs found")
	}

	idx, err := l.LastIndex()
	if err != nil {
		return 0, err
	}
	logEntry, err := l.GetLog(idx)
	if err != nil {
		return 0, err
	}

	return logEntry.Term, nil
}

func (l *LogStorage) GetLog(index uint64) (*Log, error) {
	size := len(l.records)
	for i := 0; i < size; i++ {
		if l.records[i].Index == index {
			return l.records[i], nil
		}
	}

	return nil, errors.New("index does not exist in log storage")
}

func (l *LogStorage) StoreLog(log *Log) error {
	// TODO
	return nil
}

func (l *LogStorage) DeleteRange(min, max uint64) error {
	// TODO
	return nil
}
