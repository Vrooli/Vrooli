package monetization

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// JourneyOperation is the provider-neutral operation vocabulary shared by
// scenario-to-desktop and LPBS. New operations must be added deliberately so
// evidence remains comparable across consumers.
type JourneyOperation string

const (
	JourneySignInSharedSession JourneyOperation = "signin_shared_session"
	JourneySecondAppResolves   JourneyOperation = "second_app_resolves"
	JourneyTamperedClassA      JourneyOperation = "tampered_class_a"
	JourneyClassBLocal         JourneyOperation = "class_b_local"
	JourneyOfflineClassB       JourneyOperation = "offline_class_b"
	JourneyOfflineDegrades     JourneyOperation = "offline_gate_degrades"
	JourneyOutboxDrainsOnce    JourneyOperation = "outbox_drains_once"
	JourneyExpiredLease        JourneyOperation = "expired_lease_falls_back"
)

type JourneyObservation struct {
	Operation JourneyOperation `json:"operation"`
	Observed  string           `json:"observed"`
	Route     string           `json:"route"`
}

type JourneyProbe struct {
	BaseURL    string
	HTTPClient *http.Client
}

func (p JourneyProbe) Run(ctx context.Context, operation JourneyOperation) (JourneyObservation, error) {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base == "" {
		return JourneyObservation{}, fmt.Errorf("journey probe base URL is required")
	}
	if strings.TrimSpace(string(operation)) == "" {
		return JourneyObservation{}, fmt.Errorf("journey operation is required")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/internal/monetization/journey?operation="+url.QueryEscape(string(operation)), nil)
	if err != nil {
		return JourneyObservation{}, fmt.Errorf("create journey probe request: %w", err)
	}
	client := p.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return JourneyObservation{}, fmt.Errorf("run journey probe: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return JourneyObservation{}, fmt.Errorf("journey probe returned status %d", response.StatusCode)
	}
	var result struct {
		Observed string `json:"observed"`
		Route    string `json:"route"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return JourneyObservation{}, fmt.Errorf("decode journey probe: %w", err)
	}
	if strings.TrimSpace(result.Observed) == "" || strings.TrimSpace(result.Route) == "" {
		return JourneyObservation{}, fmt.Errorf("journey probe returned incomplete observation")
	}
	return JourneyObservation{Operation: operation, Observed: result.Observed, Route: result.Route}, nil
}
