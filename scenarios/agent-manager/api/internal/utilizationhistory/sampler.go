package utilizationhistory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runtimeactivity"
	"agent-manager/internal/usageattribution"
)

type ActivityReader interface {
	Read(context.Context) runtimeactivity.Snapshot
}
type RunConfigReader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Sampler struct {
	activity   ActivityReader
	runs       RunConfigReader
	store      *Repository
	interval   time.Duration
	Now        func() time.Time
	mu         sync.Mutex
	lastPruned time.Time
}

func NewSampler(activity ActivityReader, runs RunConfigReader, store *Repository, interval time.Duration) *Sampler {
	if interval <= 0 {
		interval = time.Minute
	}
	return &Sampler{activity: activity, runs: runs, store: store, interval: interval, Now: time.Now}
}

func (s *Sampler) Start(ctx context.Context) {
	if s == nil {
		return
	}
	// Take a sample immediately, then keep the source alive independent of reads.
	s.capture(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.capture(ctx)
		}
	}
}

func (s *Sampler) Capture(ctx context.Context) error {
	if s == nil || s.activity == nil || s.runs == nil || s.store == nil {
		return fmt.Errorf("utilization sampler dependencies unavailable")
	}
	snapshot := s.activity.Read(ctx)
	sample := Sample{ObservedAt: snapshot.ObservedAt.UTC(), Complete: snapshot.Complete, ProviderRunIDs: map[string][]string{}, UnmappedRunIDs: []string{}, Unknown: append([]string(nil), snapshot.Unknown...)}
	if sample.ObservedAt.IsZero() && s.Now != nil {
		sample.ObservedAt = s.Now().UTC()
	}
	if len(snapshot.PhysicalRunIDs) > 0 {
		placeholders := make([]string, len(snapshot.PhysicalRunIDs))
		args := make([]any, len(snapshot.PhysicalRunIDs))
		for i, id := range snapshot.PhysicalRunIDs {
			placeholders[i] = "?"
			args[i] = id
		}
		// Attached/imported runs may have no resolved Agent Manager config, but
		// their durable harness_kind still identifies the provider family. Keep
		// resolved_config.runnerType authoritative when present; only fall back
		// to the recorded harness kind when it is absent.
		query := `SELECT id,resolved_config,COALESCE(harness_kind,'') FROM runs WHERE id IN (` + strings.Join(placeholders, ",") + `)`
		rows, err := s.runs.QueryContext(ctx, query, args...)
		if err != nil {
			sample.Complete = false
			sample.Unknown = append(sample.Unknown, "resolve provider for physical run identities: "+err.Error())
		} else {
			seen := map[string]bool{}
			for rows.Next() {
				var id string
				var raw []byte
				var harness string
				if err := rows.Scan(&id, &raw, &harness); err != nil {
					sample.Complete = false
					sample.Unknown = append(sample.Unknown, "read physical run provider identity: "+err.Error())
					continue
				}
				seen[id] = true
				var cfg domain.RunConfig
				if len(raw) > 0 {
					if err := json.Unmarshal(raw, &cfg); err != nil {
						sample.UnmappedRunIDs = append(sample.UnmappedRunIDs, id)
						continue
					}
				}
				runner := strings.TrimSpace(string(cfg.RunnerType))
				if runner == "" {
					runner = strings.TrimSpace(harness)
				}
				provider := usageattribution.ProviderForConfiguredRun(runner, cfg.Model)
				if provider == "" {
					sample.UnmappedRunIDs = append(sample.UnmappedRunIDs, id)
					continue
				}
				sample.ProviderRunIDs[provider] = append(sample.ProviderRunIDs[provider], id)
			}
			if err := rows.Err(); err != nil {
				sample.Complete = false
				sample.Unknown = append(sample.Unknown, "finish physical run provider lookup: "+err.Error())
			}
			_ = rows.Close()
			for _, id := range snapshot.PhysicalRunIDs {
				if !seen[id] {
					sample.Complete = false
					sample.Unknown = append(sample.Unknown, "physical run identity missing from durable run store: "+id)
				}
			}
		}
	}
	for provider := range sample.ProviderRunIDs {
		sortStrings(sample.ProviderRunIDs[provider])
	}
	sortStrings(sample.UnmappedRunIDs)
	if err := s.store.Record(ctx, sample); err != nil {
		return fmt.Errorf("record physical utilization sample: %w", err)
	}
	s.mu.Lock()
	shouldPrune := s.lastPruned.IsZero() || sample.ObservedAt.Sub(s.lastPruned) >= 6*time.Hour
	if shouldPrune {
		if err := s.store.Prune(ctx, sample.ObservedAt.Add(-70*24*time.Hour)); err == nil {
			s.lastPruned = sample.ObservedAt
		}
	}
	s.mu.Unlock()
	return nil
}

func (s *Sampler) capture(ctx context.Context) {
	if err := s.Capture(ctx); err != nil && ctx.Err() == nil {
		log.Printf("[utilizationhistory] physical utilization sample failed: %v", err)
	}
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
