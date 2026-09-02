// Package offerfacts publishes desktop observer facts to Offer Desk.
package offerfacts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	offersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type buildRecord struct {
	ScenarioName string `json:"scenario_name"`
	UpdatedAt    string `json:"updated_at"`
}

// Start starts the producer scheduler. A missing Offer Desk is a valid
// degraded configuration: the desktop ramp continues serving while facts are
// retried on the next interval.
func Start(ctx context.Context, logf func(string, ...any)) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("OFFER_DESK_API_BASE_URL")), "/")
	if baseURL == "" {
		resolved, err := discovery.ResolveScenarioURLDefault(ctx, "offer-desk")
		if err != nil {
			logf("offer fact producer unavailable", "error", err)
			return
		}
		baseURL = strings.TrimRight(strings.TrimSpace(resolved), "/")
	}
	interval := 24 * time.Hour
	if raw := strings.TrimSpace(os.Getenv("OFFER_DESK_FACT_INTERVAL")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed >= time.Minute {
			interval = parsed
		}
	}
	client := offersconnect.NewGatesServiceClient(http.DefaultClient, baseURL)
	publish := func() {
		facts, err := ReadFacts(defaultRecordsPath(), time.Now().UTC(), 30)
		if err != nil {
			logf("desktop offer fact producer skipped", "error", err)
			return
		}
		for _, fact := range facts {
			if _, err := client.AddFact(ctx, connect.NewRequest(&offersv1.AddFactRequest{Fact: fact})); err != nil {
				logf("desktop offer fact publish failed", "fact", fact.GetName(), "error", err)
				continue
			}
			logf("desktop offer fact published", "fact", fact.GetName())
		}
	}
	go func() {
		publish()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				publish()
			}
		}
	}()
}

func defaultRecordsPath() string {
	if configured := strings.TrimSpace(os.Getenv("SCENARIO_TO_DESKTOP_RECORDS_PATH")); configured != "" {
		return configured
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".vrooli", "data", "vrooli", "scenario-to-desktop", "desktop_records_v2.json")
	}
	return filepath.Join(home, ".vrooli", "data", "vrooli", "scenario-to-desktop", "desktop_records_v2.json")
}

// ReadFacts converts durable desktop build records into producer-owned facts.
// The record timestamp is retained so Offer Desk can apply its own freshness
// semantics and classify stale observations as unknown.
func ReadFacts(path string, _ time.Time, staleDays int32) ([]*offersv1.Fact, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read desktop build records: %w", err)
	}
	var records []buildRecord
	if err := json.Unmarshal(contents, &records); err != nil {
		return nil, fmt.Errorf("decode desktop build records: %w", err)
	}
	latest := map[string]time.Time{}
	for _, record := range records {
		name := strings.TrimSpace(record.ScenarioName)
		if name == "" {
			continue
		}
		updated, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(record.UpdatedAt))
		if err != nil {
			return nil, fmt.Errorf("parse desktop build timestamp for %q: %w", name, err)
		}
		if updated.After(latest[name]) {
			latest[name] = updated.UTC()
		}
	}
	if staleDays <= 0 {
		return nil, fmt.Errorf("stale days must be positive")
	}
	facts := make([]*offersv1.Fact, 0, len(latest))
	for name, updated := range latest {
		fact := &offersv1.Fact{
			Name:           "shipped_on_ramp.desktop." + name,
			Value:          1,
			ObservedAt:     timestamppb.New(updated),
			StaleAfterDays: staleDays,
			Dimension:      "producer:scenario-to-desktop",
		}
		facts = append(facts, fact)
		// Preserve the existing web-console trigger vocabulary while the
		// ramp-specific fact becomes the canonical producer observation.
		if name == "web-console" {
			alias := *fact
			alias.Name = "release_gate_passed.web-console"
			facts = append(facts, &alias)
		}
	}
	return facts, nil
}
