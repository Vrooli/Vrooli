package scenarios

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"
	scoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring/scoring_v1connect"
)

// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/INTEROP_AUDIT.md

const defaultCompletenessTimeout = 30 * time.Second

const (
	scsScenarioSlug          = "scenario-completeness-scoring"
	completenessPageSize     = 500
	maxCompletenessPageFetch = 500
)

// CompletenessSource provides completeness scores for scenarios.
type CompletenessSource interface {
	Scores(ctx context.Context) (map[string]int, error)
}

type scenarioURLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

// SCSCompletenessSource fetches completeness scores from ScoreService.ListScores.
type SCSCompletenessSource struct {
	timeout    time.Duration
	resolver   scenarioURLResolver
	httpClient connect.HTTPClient
}

// NewSCSCompletenessSource creates a typed scenario-completeness-scoring provider.
func NewSCSCompletenessSource(timeout time.Duration) *SCSCompletenessSource {
	if timeout <= 0 {
		timeout = defaultCompletenessTimeout
	}
	return &SCSCompletenessSource{
		timeout:    timeout,
		resolver:   discovery.NewResolver(discovery.ResolverConfig{}),
		httpClient: &http.Client{Timeout: timeout},
	}
}

// NewSCSCompletenessSourceWithDeps creates a provider with testable resolution
// and transport seams.
func NewSCSCompletenessSourceWithDeps(timeout time.Duration, resolver scenarioURLResolver, httpClient connect.HTTPClient) *SCSCompletenessSource {
	source := NewSCSCompletenessSource(timeout)
	if resolver != nil {
		source.resolver = resolver
	}
	if httpClient != nil {
		source.httpClient = httpClient
	}
	return source
}

// Scores retrieves completeness scores from persisted SCS snapshots. This path
// is intentionally read-only and never invokes fleet recomputation.
func (c *SCSCompletenessSource) Scores(ctx context.Context) (map[string]int, error) {
	if c == nil {
		return nil, fmt.Errorf("SCS completeness source is nil")
	}
	if c.timeout <= 0 {
		c.timeout = defaultCompletenessTimeout
	}
	if c.resolver == nil {
		c.resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}
	ctxWithTimeout, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	baseURL, err := c.resolver.ResolveScenarioURLDefault(ctxWithTimeout, scsScenarioSlug)
	if err != nil {
		return nil, err
	}
	client := scoringconnect.NewScoreServiceClient(c.httpClient, strings.TrimRight(baseURL, "/"))

	scores := make(map[string]int)
	pageToken := ""
	for pages := 0; pages < maxCompletenessPageFetch; pages++ {
		resp, err := client.ListScores(ctxWithTimeout, connect.NewRequest(&scoringv1.ListScoresRequest{
			SortBy:    scoringv1.ScoreSortBy_SCORE_SORT_BY_SCENARIO,
			Order:     scoringv1.SortOrder_SORT_ORDER_ASC,
			PageSize:  completenessPageSize,
			PageToken: pageToken,
		}))
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Msg == nil {
			return nil, fmt.Errorf("SCS ListScores returned no response")
		}
		for _, item := range resp.Msg.GetScores() {
			name := strings.TrimSpace(item.GetScenario())
			if name == "" {
				continue
			}
			scores[name] = clampCompletenessScore(float64(item.GetScore()))
		}
		pageToken = strings.TrimSpace(resp.Msg.GetNextPageToken())
		if pageToken == "" {
			return scores, nil
		}
	}
	return nil, fmt.Errorf("SCS ListScores exceeded %d pages", maxCompletenessPageFetch)
}

func clampCompletenessScore(raw float64) int {
	score := int(math.Round(raw))
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}
