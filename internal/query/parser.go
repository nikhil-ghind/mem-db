package query

import (
	"fmt"
	"strings"
)

// QueryType represents the type of a parsed query.
type QueryType int

const (
	QueryInsert QueryType = iota
	QueryGet
	QueryRange
	QueryDelete
	QueryCreateTable
	QueryDropTable
	QueryStats
)

// Query is a parsed query.
type Query struct {
	Type     QueryType
	Table    string
	Key      string
	Value    string
	StartKey string
	EndKey   string
}

// Parse parses a query string into a Query struct.
// Supported formats:
//
//	INSERT <table> <key> <value...>
//	GET <table> <key>
//	RANGE <table> <startKey> <endKey>
//	DELETE <table> <key>
//	CREATE TABLE <name>
//	DROP TABLE <name>
//	STATS
func Parse(input string) (Query, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Query{}, fmt.Errorf("empty query")
	}

	parts := strings.Fields(input)
	cmd := strings.ToUpper(parts[0])

	switch cmd {
	case "INSERT":
		if len(parts) < 4 {
			return Query{}, fmt.Errorf("INSERT requires: INSERT <table> <key> <value>")
		}
		// Value is everything after table and key, to allow spaces.
		valueStart := len(parts[0]) + 1 + len(parts[1]) + 1 + len(parts[2]) + 1
		value := ""
		if valueStart <= len(input) {
			value = input[valueStart:]
		}
		return Query{Type: QueryInsert, Table: parts[1], Key: parts[2], Value: value}, nil

	case "GET":
		if len(parts) != 3 {
			return Query{}, fmt.Errorf("GET requires: GET <table> <key>")
		}
		return Query{Type: QueryGet, Table: parts[1], Key: parts[2]}, nil

	case "RANGE":
		if len(parts) != 4 {
			return Query{}, fmt.Errorf("RANGE requires: RANGE <table> <startKey> <endKey>")
		}
		return Query{Type: QueryRange, Table: parts[1], StartKey: parts[2], EndKey: parts[3]}, nil

	case "DELETE":
		if len(parts) != 3 {
			return Query{}, fmt.Errorf("DELETE requires: DELETE <table> <key>")
		}
		return Query{Type: QueryDelete, Table: parts[1], Key: parts[2]}, nil

	case "CREATE":
		if len(parts) != 3 || strings.ToUpper(parts[1]) != "TABLE" {
			return Query{}, fmt.Errorf("CREATE requires: CREATE TABLE <name>")
		}
		return Query{Type: QueryCreateTable, Table: parts[2]}, nil

	case "DROP":
		if len(parts) != 3 || strings.ToUpper(parts[1]) != "TABLE" {
			return Query{}, fmt.Errorf("DROP requires: DROP TABLE <name>")
		}
		return Query{Type: QueryDropTable, Table: parts[2]}, nil

	case "STATS":
		return Query{Type: QueryStats}, nil

	default:
		return Query{}, fmt.Errorf("unknown command: %s", cmd)
	}
}
