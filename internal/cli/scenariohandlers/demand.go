package scenariohandlers

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

type demandOutput struct {
	Lease demandLeaseJSON `json:"lease"`
}

type demandLeaseJSON struct {
	LeaseID       string    `json:"lease_id"`
	Scenario      string    `json:"scenario"`
	Variant       string    `json:"variant,omitempty"`
	ConsumerID    string    `json:"consumer_id"`
	Kind          string    `json:"kind"`
	RequestID     string    `json:"request_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	LastRenewedAt time.Time `json:"last_renewed_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Status        string    `json:"status"`
	StopReason    string    `json:"stop_reason,omitempty"`
	MetadataJSON  string    `json:"metadata_json,omitempty"`
}

func newDemandOutput(lease scenarioruntime.DemandLease) demandOutput {
	return demandOutput{Lease: demandLeaseJSON{
		LeaseID: lease.LeaseID, Scenario: lease.Scenario, Variant: lease.Variant,
		ConsumerID: lease.ConsumerID, Kind: lease.Kind, RequestID: lease.RequestID,
		CreatedAt: lease.CreatedAt, LastRenewedAt: lease.LastRenewedAt, ExpiresAt: lease.ExpiresAt,
		Status: lease.Status, StopReason: lease.StopReason, MetadataJSON: lease.MetadataJSON,
	}}
}

func demandHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		req, err := ParseDemandRequest(deps.Globals(ctx).JSON, args)
		if err != nil {
			return err
		}
		home, err := deps.HomeDir(ctx)
		if err != nil {
			return err
		}
		opCtx := rootcli.ResolveOperationContext(deps, ctx)
		store, err := scenarioruntime.NewSQLiteStore(opCtx, scenarioruntime.Config{HomeDir: home})
		if err != nil {
			return fmt.Errorf("open runtime registry: %w", err)
		}
		defer store.Close()
		if req.Operation == "history" {
			events, err := store.ListDemandTransitions(opCtx, req.Scenario, req.Variant, req.Limit)
			if err != nil {
				return err
			}
			return renderDemandHistory(deps.Stdout(ctx), req.JSON, events)
		}
		var lease scenarioruntime.DemandLease
		switch req.Operation {
		case "acquire":
			ttl, parseErr := demandTTL(req.TTL)
			if parseErr != nil {
				return parseErr
			}
			lease, err = store.AcquireDemandLease(opCtx, scenarioruntime.DemandLease{LeaseID: req.LeaseID, Scenario: req.Scenario, Variant: req.Variant, ConsumerID: req.Consumer, Kind: req.Kind, RequestID: req.RequestID, MetadataJSON: req.Metadata}, ttl)
		case "renew":
			ttl, parseErr := demandTTL(req.TTL)
			if parseErr != nil {
				return parseErr
			}
			lease, err = store.RenewDemandLease(opCtx, req.LeaseID, ttl)
		case "release":
			lease, err = store.ReleaseDemandLease(opCtx, req.LeaseID, req.Reason)
		}
		if err != nil {
			return err
		}
		return renderDemand(deps.Stdout(ctx), req.JSON, newDemandOutput(lease))
	}
}

func renderDemandHistory(w io.Writer, jsonOutput bool, events []scenarioruntime.DemandTransition) error {
	if events == nil {
		events = []scenarioruntime.DemandTransition{}
	}
	if jsonOutput {
		return json.NewEncoder(w).Encode(struct {
			Scope     string                             `json:"scope"`
			Retention int                                `json:"retention_per_scenario"`
			Events    []scenarioruntime.DemandTransition `json:"events"`
		}{"recent_committed_transitions", scenarioruntime.DemandAuditRetention, events})
	}
	if _, err := fmt.Fprintf(w, "Recent committed demand transitions (up to %d retained per scenario; newest first)\n", scenarioruntime.DemandAuditRetention); err != nil {
		return err
	}
	for _, event := range events {
		identity := event.LeaseID
		if identity == "" {
			identity = event.InstanceID
		}
		if _, err := fmt.Fprintf(w, "%s %s@%s %s %s\n", event.RecordedAt.Format(time.RFC3339Nano), event.Scenario, event.Variant, event.Operation, identity); err != nil {
			return err
		}
	}
	return nil
}

func demandTTL(raw string) (time.Duration, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	ttl, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid demand lease ttl: %w", err)
	}
	return ttl, nil
}

func renderDemand(w io.Writer, jsonOutput bool, out demandOutput) error {
	if jsonOutput {
		encoded, err := json.Marshal(out)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "%s\n", encoded)
		return err
	}
	_, err := fmt.Fprintf(w, "%s %s demand lease %s (expires %s)\n", out.Lease.Scenario, out.Lease.Kind, out.Lease.Status, out.Lease.ExpiresAt.Format(time.RFC3339))
	return err
}
