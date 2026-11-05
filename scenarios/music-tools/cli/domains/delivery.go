package domains

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"music-tools/cli/internal/delivery"
	"os"
	"os/signal"
	"syscall"
)

func arrange(args []string) error {
	fs := flag.NewFlagSet("delivery arrange", flag.ContinueOnError)
	recipe := fs.String("recipe", "", "JSON sample arrangement recipe")
	out := fs.String("out", "", "New output directory; previous takes are never overwritten")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *recipe == "" || *out == "" {
		return fmt.Errorf("--recipe and --out are required")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	receipt, err := delivery.Arrange(ctx, *recipe, *out)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(raw))
	return nil
}
