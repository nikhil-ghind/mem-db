package config

import (
	"flag"
	"os"
)

// Config holds server configuration.
type Config struct {
	ListenAddr       string
	DefaultTableName string
	BTreeOrder       int
	Interactive      bool
}

// Load parses configuration from flags and environment variables.
func Load() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.ListenAddr, "addr", envOrDefault("MEMDB_ADDR", ":50051"), "gRPC listen address")
	flag.StringVar(&cfg.DefaultTableName, "default-table", envOrDefault("MEMDB_DEFAULT_TABLE", "default"), "default table name")
	flag.IntVar(&cfg.BTreeOrder, "order", 128, "B+ tree order (max keys per node)")
	flag.BoolVar(&cfg.Interactive, "interactive", false, "start in interactive CLI mode")
	flag.Parse()

	return cfg
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
