// capture-cohort runs the maintained native fixture against managed BAS services.
// It does not start services or certify readiness. Keep stdout machine-readable.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vrooli/browser-automation-studio/internal/capturequalification"
)

func main() {
	var cfg capturequalification.Config
	flag.StringVar(&cfg.Output, "output", "", "new directory for retained receipts")
	flag.StringVar(&cfg.ScenarioRoot, "scenario-root", "..", "BAS scenario root")
	flag.StringVar(&cfg.APIURL, "api-url", "", "managed BAS API URL")
	flag.StringVar(&cfg.DriverURL, "driver-url", "", "managed BAS driver URL")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 8*time.Minute)
	defer cancel()
	receipt, err := capturequalification.Run(ctx, cfg)
	if receipt != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"operation_id": receipt.OperationID,
			"output": cfg.Output, "attempts": len(receipt.Attempts), "errors": receipt.Errors})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
