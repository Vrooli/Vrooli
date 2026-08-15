// Package capacitysync keeps the reranker's resident VRAM claim alive.
//
// The managed-service lifecycle admits the initial claim, but capacity claims
// expire unless their owner heartbeats them. This companion is deliberately
// small: model selection and broker actuation remain owned by the reranker CLI
// and capacity broker, while this process only reclaims a missing claim and
// heartbeats the current one.
package capacitysync

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	resourceName    = "reranker"
	defaultInterval = 15 * time.Second
)

type Handlers struct {
	Stdout   io.Writer
	Stderr   io.Writer
	Exec     func(context.Context, string, ...string) ([]byte, error)
	Interval time.Duration
}

type claim struct {
	ClaimID    string `json:"claim_id"`
	OwnerID    string `json:"owner_id"`
	Generation int64  `json:"generation"`
}

func Default() *Handlers {
	return &Handlers{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Exec: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output()
		},
	}
}

func Command(h *Handlers) func([]string) error {
	if h == nil {
		h = Default()
	}
	return func(args []string) error {
		fs := flag.NewFlagSet("capacity-sync", flag.ContinueOnError)
		fs.SetOutput(h.Stderr)
		interval := fs.Duration("interval", h.interval(), "heartbeat interval")
		once := fs.Bool("once", false, "run one claim reconciliation and exit")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
		}
		if *once {
			h.syncOnce(context.Background())
			return nil
		}
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		fmt.Fprintf(h.Stdout, "reranker capacity-sync: heartbeating every %s\n", *interval)
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()
		h.syncOnce(ctx)
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				h.syncOnce(ctx)
			}
		}
	}
}

func (h *Handlers) syncOnce(ctx context.Context) {
	active := h.activeClaim(ctx)
	if active == nil {
		// The lifecycle admission path owns the manifest-derived values and is
		// normally responsible for creating this claim. A missing claim can occur
		// after expiry or manual ledger cleanup; re-admit through the control plane
		// so the manifest remains the single source of policy truth.
		_, _ = h.exec(ctx, "vrooli", "resource", "start", resourceName, "--json")
		return
	}
	_, _ = h.exec(ctx, "vrooli", "capacity", "heartbeat",
		"--claim-id", active.ClaimID,
		"--generation", strconv.FormatInt(active.Generation, 10), "--json")
}

func (h *Handlers) activeClaim(ctx context.Context) *claim {
	out, err := h.exec(ctx, "vrooli", "capacity", "list", "--owner", resourceName, "--active", "--json")
	if err != nil {
		return nil
	}
	var payload struct {
		Claims []claim `json:"claims"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil
	}
	for i := range payload.Claims {
		if payload.Claims[i].OwnerID == resourceName && payload.Claims[i].ClaimID != "" {
			return &payload.Claims[i]
		}
	}
	return nil
}

func (h *Handlers) exec(ctx context.Context, name string, args ...string) ([]byte, error) {
	if h.Exec == nil {
		return nil, fmt.Errorf("capacity-sync executor is not configured")
	}
	return h.Exec(ctx, name, args...)
}

func (h *Handlers) interval() time.Duration {
	if h.Interval > 0 {
		return h.Interval
	}
	return defaultInterval
}
