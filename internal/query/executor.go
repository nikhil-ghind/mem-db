package query

import (
	"fmt"
	"strings"
	"time"

	"github.com/nikhilghind/mem-db/internal/btree"
	"github.com/nikhilghind/mem-db/internal/engine"
)

// QueryResult holds the result of executing a query.
type QueryResult struct {
	Success  bool
	Value    []byte
	Rows     []btree.KeyValue
	Message  string
	Duration time.Duration
}

// Executor executes parsed queries against an engine.
type Executor struct {
	engine *engine.Engine
}

// NewExecutor creates a new Executor.
func NewExecutor(e *engine.Engine) *Executor {
	return &Executor{engine: e}
}

// Execute runs a parsed Query and returns the result.
func (ex *Executor) Execute(q Query) QueryResult {
	start := time.Now()
	var result QueryResult

	switch q.Type {
	case QueryInsert:
		err := ex.engine.Insert(q.Table, q.Key, []byte(q.Value))
		if err != nil {
			result = QueryResult{Success: false, Message: err.Error()}
		} else {
			result = QueryResult{Success: true, Message: "OK"}
		}

	case QueryGet:
		val, ok, err := ex.engine.Get(q.Table, q.Key)
		if err != nil {
			result = QueryResult{Success: false, Message: err.Error()}
		} else if !ok {
			result = QueryResult{Success: false, Message: "key not found"}
		} else {
			result = QueryResult{Success: true, Value: val, Message: string(val)}
		}

	case QueryRange:
		rows, err := ex.engine.RangeScan(q.Table, q.StartKey, q.EndKey)
		if err != nil {
			result = QueryResult{Success: false, Message: err.Error()}
		} else {
			msg := fmt.Sprintf("%d rows", len(rows))
			result = QueryResult{Success: true, Rows: rows, Message: msg}
		}

	case QueryDelete:
		ok, err := ex.engine.Delete(q.Table, q.Key)
		if err != nil {
			result = QueryResult{Success: false, Message: err.Error()}
		} else if !ok {
			result = QueryResult{Success: false, Message: "key not found"}
		} else {
			result = QueryResult{Success: true, Message: "OK"}
		}

	case QueryCreateTable:
		err := ex.engine.CreateTable(q.Table)
		if err != nil {
			result = QueryResult{Success: false, Message: err.Error()}
		} else {
			result = QueryResult{Success: true, Message: fmt.Sprintf("table %q created", q.Table)}
		}

	case QueryDropTable:
		err := ex.engine.DropTable(q.Table)
		if err != nil {
			result = QueryResult{Success: false, Message: err.Error()}
		} else {
			result = QueryResult{Success: true, Message: fmt.Sprintf("table %q dropped", q.Table)}
		}

	case QueryStats:
		stats := ex.engine.Stats()
		var sb strings.Builder
		fmt.Fprintf(&sb, "tables: %d, total_keys: %d\n", stats.TableCount, stats.TotalKeys)
		for name, ts := range stats.Tables {
			fmt.Fprintf(&sb, "  %s: keys=%d height=%d order=%d\n", name, ts.Size, ts.Height, ts.Order)
		}
		result = QueryResult{Success: true, Message: sb.String()}

	default:
		result = QueryResult{Success: false, Message: "unknown query type"}
	}

	result.Duration = time.Since(start)
	return result
}

// ExecuteRaw parses and executes a raw query string.
func (ex *Executor) ExecuteRaw(input string) QueryResult {
	q, err := Parse(input)
	if err != nil {
		return QueryResult{Success: false, Message: err.Error()}
	}
	return ex.Execute(q)
}
