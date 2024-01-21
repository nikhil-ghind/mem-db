package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"google.golang.org/grpc"

	"github.com/nikhilghind/mem-db/internal/config"
	"github.com/nikhilghind/mem-db/internal/engine"
	"github.com/nikhilghind/mem-db/internal/query"
	"github.com/nikhilghind/mem-db/internal/server"
)

func main() {
	cfg := config.Load()

	eng := engine.New(cfg.DefaultTableName, cfg.BTreeOrder)
	log.Printf("memdb engine initialized (default table: %q, order: %d)", cfg.DefaultTableName, cfg.BTreeOrder)

	if cfg.Interactive {
		runInteractive(eng)
		return
	}

	// gRPC server mode.
	gs := grpc.NewServer()
	srv := server.NewMemDBServer(eng)

	// Graceful shutdown on SIGTERM/SIGINT.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigCh
		log.Printf("received %v, shutting down gracefully...", sig)
		gs.GracefulStop()
	}()

	if err := srv.StartWithServer(cfg.ListenAddr, gs); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func runInteractive(eng *engine.Engine) {
	executor := query.NewExecutor(eng)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("memdb interactive mode. Type queries or 'quit' to exit.")
	fmt.Println("Examples:")
	fmt.Println("  INSERT default mykey myvalue")
	fmt.Println("  GET default mykey")
	fmt.Println("  RANGE default a z")
	fmt.Println("  DELETE default mykey")
	fmt.Println("  CREATE TABLE users")
	fmt.Println("  STATS")
	fmt.Println()

	for {
		fmt.Print("memdb> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.ToLower(line) == "quit" || strings.ToLower(line) == "exit" {
			fmt.Println("bye")
			return
		}

		result := executor.ExecuteRaw(line)
		if result.Success {
			fmt.Printf("OK (%s)\n", result.Duration)
			if result.Message != "" && result.Message != "OK" {
				fmt.Println(result.Message)
			}
			for _, row := range result.Rows {
				fmt.Printf("  %s = %s\n", row.Key, string(row.Value))
			}
		} else {
			fmt.Printf("ERR: %s\n", result.Message)
		}
	}
}
