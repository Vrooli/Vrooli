package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	offersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type deploymentReportFact struct {
	GeneratedAt string `json:"generated_at"`
}

// startReleaseFactPublisher keeps the producer direction explicit: deployment
// manager observes its own durable report and pushes a fact to Offer Desk. A
// configured static URL overrides local discovery. Missing Offer Desk and
// transient publish errors are retried on the next interval and never affect
// the deployment API's serving lifecycle.
func (s *Server) startReleaseFactPublisher(ctx context.Context) {
	resolver := discovery.NewResolverWithOptionalStaticOverride(os.Getenv("OFFER_DESK_API_BASE_URL"))
	interval := releaseFactInterval()
	logger := s.Logger
	if logger == nil {
		logger = slog.Default()
	}
	publish := func() {
		baseURL, err := resolver.ResolveScenarioURLDefault(ctx, "offer-desk")
		if err != nil {
			logger.Info("release fact producer unavailable", "error", err)
			return
		}
		client := offersconnect.NewGatesServiceClient(http.DefaultClient, baseURL)
		fact, err := deploymentReportFactValue(defaultDeploymentReportPath(), time.Now().UTC(), 30*24*time.Hour)
		if err != nil {
			logger.Info("release fact producer skipped", "error", err)
			return
		}
		if _, err := client.AddFact(ctx, connect.NewRequest(&offersv1.AddFactRequest{Fact: fact})); err != nil {
			logger.Info("release fact producer publish failed", "error", err)
			return
		}
		logger.Info("release fact producer published", "fact", fact.GetName())
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

func releaseFactInterval() time.Duration {
	const defaultInterval = 24 * time.Hour
	raw := strings.TrimSpace(os.Getenv("OFFER_DESK_FACT_INTERVAL"))
	if raw == "" {
		return defaultInterval
	}
	if duration, err := time.ParseDuration(raw); err == nil && duration >= time.Minute {
		return duration
	}
	if minutes, err := strconv.Atoi(raw); err == nil && minutes >= 1 {
		return time.Duration(minutes) * time.Minute
	}
	return defaultInterval
}

func defaultDeploymentReportPath() string {
	if configured := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_REPORT_PATH")); configured != "" {
		return configured
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".vrooli", "data", "vrooli", "deployment-manager", "deployment", "deployment-report.json")
	}
	return filepath.Join(home, ".vrooli", "data", "vrooli", "deployment-manager", "deployment", "deployment-report.json")
}

func deploymentReportFactValue(path string, now time.Time, staleAfter time.Duration) (*offersv1.Fact, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read deployment report: %w", err)
	}
	var report deploymentReportFact
	if err := json.Unmarshal(contents, &report); err != nil {
		return nil, fmt.Errorf("decode deployment report: %w", err)
	}
	generated, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(report.GeneratedAt))
	if err != nil {
		return nil, fmt.Errorf("parse deployment report generated_at: %w", err)
	}
	generated = generated.UTC()
	if generated.After(now.UTC()) || now.UTC().Sub(generated) > staleAfter {
		return &offersv1.Fact{Name: "deployment_report_fresh", Value: 0, ObservedAt: timestamppb.New(generated), StaleAfterDays: int32(staleAfter / (24 * time.Hour)), Dimension: "producer:deployment-manager"}, nil
	}
	return &offersv1.Fact{Name: "deployment_report_fresh", Value: 1, ObservedAt: timestamppb.New(generated), StaleAfterDays: int32(staleAfter / (24 * time.Hour)), Dimension: "producer:deployment-manager"}, nil
}
