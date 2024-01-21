package engine

import (
	"fmt"
	"sync"

	"github.com/nikhilghind/mem-db/internal/btree"
)

// EngineStats holds aggregate statistics.
type EngineStats struct {
	TableCount int
	TotalKeys  int
	Tables     map[string]btree.TreeStats
}

// Engine manages multiple named tables, each backed by a B+ tree.
type Engine struct {
	mu     sync.RWMutex
	tables map[string]*btree.BPlusTree
	order  int
}

// New creates a new Engine. It creates a default table with the given name.
func New(defaultTable string, order int) *Engine {
	e := &Engine{
		tables: make(map[string]*btree.BPlusTree),
		order:  order,
	}
	e.tables[defaultTable] = btree.New(order)
	return e
}

// CreateTable creates a new table. Returns an error if it already exists.
func (e *Engine) CreateTable(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.tables[name]; ok {
		return fmt.Errorf("table %q already exists", name)
	}
	e.tables[name] = btree.New(e.order)
	return nil
}

// DropTable removes a table.
func (e *Engine) DropTable(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.tables[name]; !ok {
		return fmt.Errorf("table %q not found", name)
	}
	delete(e.tables, name)
	return nil
}

// GetTable returns a table by name.
func (e *Engine) GetTable(name string) (*btree.BPlusTree, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	t, ok := e.tables[name]
	if !ok {
		return nil, fmt.Errorf("table %q not found", name)
	}
	return t, nil
}

// Insert inserts a key-value pair into the named table.
func (e *Engine) Insert(table, key string, value []byte) error {
	t, err := e.GetTable(table)
	if err != nil {
		return err
	}
	t.Insert(key, value)
	return nil
}

// Get retrieves a value from the named table.
func (e *Engine) Get(table, key string) ([]byte, bool, error) {
	t, err := e.GetTable(table)
	if err != nil {
		return nil, false, err
	}
	v, ok := t.Get(key)
	return v, ok, nil
}

// Delete removes a key from the named table.
func (e *Engine) Delete(table, key string) (bool, error) {
	t, err := e.GetTable(table)
	if err != nil {
		return false, err
	}
	return t.Delete(key), nil
}

// RangeScan returns key-value pairs in [startKey, endKey] from the named table.
func (e *Engine) RangeScan(table, startKey, endKey string) ([]btree.KeyValue, error) {
	t, err := e.GetTable(table)
	if err != nil {
		return nil, err
	}
	return t.RangeScan(startKey, endKey), nil
}

// Stats returns aggregate statistics across all tables.
func (e *Engine) Stats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	stats := EngineStats{
		TableCount: len(e.tables),
		Tables:     make(map[string]btree.TreeStats),
	}
	for name, t := range e.tables {
		ts := t.Stats()
		stats.TotalKeys += ts.Size
		stats.Tables[name] = ts
	}
	return stats
}
