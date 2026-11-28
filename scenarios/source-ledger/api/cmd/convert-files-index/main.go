// Command convert-files-index enables incremental auto-vacuum on Source
// Ledger's stopped live research index. It holds the lifecycle lock for the
// entire operation and reports the measured SQLite maintenance result as JSON.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"source-ledger/internal/ledgerdb"
)

func main() {
	apply := flag.Bool("apply", false, "enable incremental auto-vacuum on the stopped live research index")
	flag.Parse()
	if !*apply || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/convert-files-index --apply")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	result, err := ledgerdb.ConvertFilesIndex(ctx)
	stop()
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if encodeErr := enc.Encode(result); encodeErr != nil {
		fmt.Fprintln(os.Stderr, "encode file-index conversion receipt:", encodeErr)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
