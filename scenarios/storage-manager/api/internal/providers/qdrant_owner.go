package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// QdrantGenerationReader is the read-only boundary between storage-manager
// accounting and the scenario that owns Qdrant generation semantics.
// storage-manager never receives filesystem paths or deletion authority.
type QdrantGenerationReader interface {
	InspectQdrantGenerations(context.Context) (QdrantGenerationInspection, error)
}

type QdrantGenerationInspection struct {
	Owner              string             `json:"owner"`
	Namespace          string             `json:"namespace"`
	Alias              string             `json:"alias"`
	ActiveCollection   string             `json:"activeCollection"`
	SnapshotIdentity   string             `json:"snapshotIdentity"`
	ObservedAt         time.Time          `json:"observedAt"`
	Generations        []QdrantGeneration `json:"generations"`
	Protected          []string           `json:"protected"`
	Quarantined        []string           `json:"quarantined"`
	Eligible           []string           `json:"eligible"`
	LogicalBytes       int64              `json:"logicalBytes"`
	PhysicalBytesKnown bool               `json:"physicalBytesKnown"`
}

type QdrantGeneration struct {
	Metadata       QdrantGenerationMetadata `json:"metadata"`
	CollectionName string                   `json:"collectionName"`
	State          string                   `json:"state"`
	Lease          QdrantLease              `json:"lease"`
	Points         int                      `json:"points"`
	Bytes          int64                    `json:"bytes"`
	CleanupOutcome string                   `json:"cleanupOutcome"`
	UpdatedAt      time.Time                `json:"updatedAt"`
}

type QdrantGenerationMetadata struct {
	GenerationID    string    `json:"generationId"`
	CreatedAt       time.Time `json:"createdAt"`
	Owner           string    `json:"owner"`
	Namespace       string    `json:"namespace"`
	Alias           string    `json:"alias"`
	ContentIdentity string    `json:"contentIdentity"`
	LeaseID         string    `json:"leaseId"`
	LeaseExpiresAt  time.Time `json:"leaseExpiresAt"`
}

type QdrantLease struct {
	ID        string    `json:"id"`
	Holder    string    `json:"holder"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// HTTPQdrantGenerationReader delegates inspection to the owning scenario.
// The token is supplied by deployment configuration and is never included in
// the returned report.
type HTTPQdrantGenerationReader struct {
	ResolveURL func(context.Context, string) (string, error)
	HTTPClient *http.Client
	Token      func() string
	Scenario   string
}

func NewHTTPQdrantGenerationReader(resolveURL func(context.Context, string) (string, error), client *http.Client, token func() string) *HTTPQdrantGenerationReader {
	return &HTTPQdrantGenerationReader{ResolveURL: resolveURL, HTTPClient: client, Token: token, Scenario: "agent-manager"}
}

func (r *HTTPQdrantGenerationReader) InspectQdrantGenerations(ctx context.Context) (QdrantGenerationInspection, error) {
	if r == nil || r.ResolveURL == nil {
		return QdrantGenerationInspection{}, fmt.Errorf("qdrant owner reader unavailable")
	}
	token := ""
	if r.Token != nil {
		token = strings.TrimSpace(r.Token())
	}
	if token == "" {
		return QdrantGenerationInspection{}, fmt.Errorf("qdrant owner control token unavailable")
	}
	base, err := r.ResolveURL(ctx, r.Scenario)
	if err != nil {
		return QdrantGenerationInspection{}, fmt.Errorf("resolve qdrant owner: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/api/v1/conversation-search/generations", nil)
	if err != nil {
		return QdrantGenerationInspection{}, err
	}
	request.Header.Set("X-Search-Control-Token", token)
	client := r.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return QdrantGenerationInspection{}, fmt.Errorf("qdrant owner inspection: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		return QdrantGenerationInspection{}, fmt.Errorf("qdrant owner inspection returned HTTP %d", response.StatusCode)
	}
	var inspection QdrantGenerationInspection
	if err := json.NewDecoder(response.Body).Decode(&inspection); err != nil {
		return QdrantGenerationInspection{}, fmt.Errorf("decode qdrant owner inspection: %w", err)
	}
	return inspection, nil
}
