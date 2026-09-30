package main

// Source-distribution integration is deliberately read-only. The source-ramp
// owns closure, recipes, policy, artifacts, verification, publication state,
// and destination read-back; GCT only projects that evidence for operators.

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-repository/v1/source"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-repository/v1/source/source_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SourceDistribution struct {
	DistributionID            string `json:"distribution_id"`
	Scenario                  string `json:"scenario"`
	SourceDigest              string `json:"source_digest"`
	ClosureDigest             string `json:"closure_digest"`
	RecipeDigest              string `json:"recipe_digest"`
	PolicyDigest              string `json:"policy_digest"`
	ArtifactID                string `json:"artifact_id"`
	ArtifactDigest            string `json:"artifact_digest"`
	VerificationStatus        string `json:"verification_status"`
	VerificationReceipt       string `json:"verification_receipt"`
	DeploymentManagerDecision string `json:"deployment_manager_decision"`
	PublicationStatus         string `json:"publication_status"`
	Destination               string `json:"destination"`
	DestinationRevision       string `json:"destination_revision"`
	ReadbackReceipt           string `json:"readback_receipt"`
	DriftState                string `json:"drift_state"`
	SourceOfTruth             string `json:"source_of_truth"`
	SourceTimestamp           string `json:"source_timestamp"`
	Freshness                 string `json:"freshness"`
	UpdatedAt                 string `json:"updated_at"`
	WorkflowURL               string `json:"workflow_url"`
}

type DistributionContent struct {
	Path, SourcePath, Category, Digest string
	SizeBytes                          uint64
}
type DistributionExclusion struct{ Path, Category, SafeReason string }

type SourceDistributionDetails struct {
	Distribution          SourceDistribution      `json:"distribution"`
	Contents              []DistributionContent   `json:"contents"`
	Exclusions            []DistributionExclusion `json:"exclusions"`
	UnresolvedObligations []string                `json:"unresolved_obligations"`
	RuntimeRequirements   []string                `json:"runtime_requirements"`
	Handoff               *PublicationHandoff     `json:"handoff,omitempty"`
	Drift                 *DistributionDrift      `json:"drift,omitempty"`
	Available             bool                    `json:"available"`
	UnavailableReason     string                  `json:"unavailable_reason,omitempty"`
}

type PublicationHandoff struct {
	Status            string   `json:"status"`
	ArtifactID        string   `json:"artifact_id"`
	ArtifactDigest    string   `json:"artifact_digest"`
	Destination       string   `json:"destination"`
	HumanAction       string   `json:"human_action"`
	ReadbackOracle    string   `json:"readback_oracle"`
	Preconditions     []string `json:"preconditions"`
	ApprovalReference string   `json:"approval_reference"`
	SourceOfTruth     string   `json:"source_of_truth"`
	Freshness         string   `json:"freshness"`
}
type DistributionDrift struct {
	State                  string   `json:"state"`
	SourceChanged          bool     `json:"source_changed"`
	DestinationChanged     bool     `json:"destination_changed"`
	CurrentSourceDigest    string   `json:"current_source_digest"`
	RecordedSourceDigest   string   `json:"recorded_source_digest"`
	CurrentArtifactDigest  string   `json:"current_artifact_digest"`
	RecordedArtifactDigest string   `json:"recorded_artifact_digest"`
	Actions                []string `json:"actions"`
	SourceOfTruth          string   `json:"source_of_truth"`
	Freshness              string   `json:"freshness"`
}

type SourceDistributionReader interface {
	List(context.Context, string, string, int) ([]*sourcev1.Distribution, string, string, error)
	Get(context.Context, string) (*sourcev1.Distribution, error)
	Contents(context.Context, string) (*sourcev1.DistributionContentsResponse, error)
	Handoff(context.Context, string) (*sourcev1.PublicationHandoffResponse, error)
	Drift(context.Context, string) (*sourcev1.DistributionDriftResponse, error)
}

type sourceRampReader struct {
	baseURL string
	client  *http.Client
}

func NewSourceRampReader(baseURL string) SourceDistributionReader {
	return &sourceRampReader{baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: 15 * time.Second}}
}
func (r *sourceRampReader) rpc(ctx context.Context) (sourceconnect.SourceRepositoryServiceClient, error) {
	base := r.baseURL
	if base == "" {
		var err error
		base, err = discovery.ResolveScenarioURLDefault(ctx, "scenario-to-repository")
		if err != nil {
			return nil, err
		}
	}
	return sourceconnect.NewSourceRepositoryServiceClient(r.client, base), nil
}
func (r *sourceRampReader) List(ctx context.Context, scenario, repositoryContext string, limit int) ([]*sourcev1.Distribution, string, string, error) {
	c, err := r.rpc(ctx)
	if err != nil {
		return nil, "scenario-to-repository", "unavailable", err
	}
	resp, err := c.ListDistributions(ctx, connect.NewRequest(&sourcev1.ListDistributionsRequest{Scenario: scenario, RepositoryContext: repositoryContext, Limit: int32(limit)}))
	if err != nil {
		return nil, "scenario-to-repository", "unavailable", err
	}
	return resp.Msg.Distributions, resp.Msg.SourceOfTruth, resp.Msg.Freshness, nil
}
func (r *sourceRampReader) Get(ctx context.Context, id string) (*sourcev1.Distribution, error) {
	c, err := r.rpc(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := c.GetDistribution(ctx, connect.NewRequest(&sourcev1.GetDistributionRequest{DistributionId: id}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.Distribution, nil
}
func (r *sourceRampReader) Contents(ctx context.Context, id string) (*sourcev1.DistributionContentsResponse, error) {
	c, err := r.rpc(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := c.GetDistributionContents(ctx, connect.NewRequest(&sourcev1.GetDistributionContentsRequest{DistributionId: id}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}
func (r *sourceRampReader) Handoff(ctx context.Context, id string) (*sourcev1.PublicationHandoffResponse, error) {
	c, err := r.rpc(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := c.GetPublicationHandoff(ctx, connect.NewRequest(&sourcev1.GetPublicationHandoffRequest{DistributionId: id}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}
func (r *sourceRampReader) Drift(ctx context.Context, id string) (*sourcev1.DistributionDriftResponse, error) {
	c, err := r.rpc(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := c.GetDistributionDrift(ctx, connect.NewRequest(&sourcev1.GetDistributionDriftRequest{DistributionId: id}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (s *Server) sourceDistributionList(w http.ResponseWriter, r *http.Request) {
	if s.sourceDistributions == nil {
		NewResponse(w).ServiceUnavailable(map[string]any{"available": false, "source_of_truth": "scenario-to-repository", "unavailable_reason": "source-ramp reader is not configured"})
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	items, owner, freshness, err := s.sourceDistributions.List(r.Context(), r.URL.Query().Get("scenario"), r.URL.Query().Get("repository_context"), limit)
	if err != nil {
		NewResponse(w).ServiceUnavailable(map[string]any{"available": false, "source_of_truth": owner, "freshness": "unavailable", "unavailable_reason": safeSourceError(err)})
		return
	}
	out := make([]SourceDistribution, 0, len(items))
	for _, item := range items {
		out = append(out, mapDistribution(item, owner, freshness))
	}
	NewResponse(w).OK(map[string]any{"available": true, "source_of_truth": owner, "freshness": freshness, "distributions": out})
}

func (s *Server) sourceDistributionDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		NewResponse(w).BadRequest("distribution id is required")
		return
	}
	if s.sourceDistributions == nil {
		NewResponse(w).ServiceUnavailable(map[string]any{"available": false, "unavailable_reason": "source-ramp reader is not configured"})
		return
	}
	d, err := s.sourceDistributions.Get(r.Context(), id)
	if err != nil {
		NewResponse(w).ServiceUnavailable(map[string]any{"available": false, "distribution_id": id, "unavailable_reason": safeSourceError(err)})
		return
	}
	detail := SourceDistributionDetails{Distribution: mapDistribution(d, d.SourceOfTruth, d.Freshness), Available: true}
	if detail.Distribution.SourceOfTruth == "" {
		detail.Distribution.SourceOfTruth = "scenario-to-repository"
	}
	if contents, e := s.sourceDistributions.Contents(r.Context(), id); e == nil {
		detail.Contents, detail.Exclusions = mapContents(contents)
		detail.UnresolvedObligations = contents.UnresolvedObligations
		detail.RuntimeRequirements = contents.RuntimeRequirements
	} else {
		detail.UnavailableReason = "contents unavailable: " + safeSourceError(e)
	}
	if handoff, e := s.sourceDistributions.Handoff(r.Context(), id); e == nil {
		detail.Handoff = mapHandoff(handoff)
	} else if detail.UnavailableReason == "" {
		detail.UnavailableReason = "publication handoff unavailable: " + safeSourceError(e)
	}
	if drift, e := s.sourceDistributions.Drift(r.Context(), id); e == nil {
		detail.Drift = mapDrift(drift)
	} else if detail.UnavailableReason == "" {
		detail.UnavailableReason = "drift unavailable: " + safeSourceError(e)
	}
	NewResponse(w).OK(detail)
}

func mapDistribution(d *sourcev1.Distribution, owner, freshness string) SourceDistribution {
	if d == nil {
		return SourceDistribution{}
	}
	return SourceDistribution{DistributionID: d.DistributionId, Scenario: d.Scenario, SourceDigest: d.SourceDigest, ClosureDigest: d.ClosureDigest, RecipeDigest: d.RecipeDigest, PolicyDigest: d.PolicyDigest, ArtifactID: d.ArtifactId, ArtifactDigest: d.ArtifactDigest, VerificationStatus: d.VerificationStatus, VerificationReceipt: d.VerificationReceipt, DeploymentManagerDecision: d.DeploymentManagerDecision, PublicationStatus: d.PublicationStatus, Destination: d.DestinationReference, DestinationRevision: d.DestinationRevision, ReadbackReceipt: d.ReadbackReceipt, DriftState: d.DriftState, SourceOfTruth: firstNonEmpty(d.SourceOfTruth, owner, "scenario-to-repository"), SourceTimestamp: protoTime(d.SourceTimestamp), Freshness: firstNonEmpty(d.Freshness, freshness, "unknown"), UpdatedAt: protoTime(d.UpdatedAt), WorkflowURL: d.WorkflowUrl}
}
func protoTime(t *timestamppb.Timestamp) string {
	if t == nil {
		return ""
	}
	return t.AsTime().UTC().Format(time.RFC3339Nano)
}
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
func mapContents(c *sourcev1.DistributionContentsResponse) ([]DistributionContent, []DistributionExclusion) {
	var contents []DistributionContent
	for _, x := range c.Contents {
		contents = append(contents, DistributionContent{Path: x.Path, SourcePath: x.SourcePath, Category: x.Category, Digest: x.Digest, SizeBytes: x.SizeBytes})
	}
	var exclusions []DistributionExclusion
	for _, x := range c.Exclusions {
		exclusions = append(exclusions, DistributionExclusion{Path: x.Path, Category: x.Category, SafeReason: x.SafeReason})
	}
	return contents, exclusions
}
func mapHandoff(h *sourcev1.PublicationHandoffResponse) *PublicationHandoff {
	if h == nil {
		return nil
	}
	return &PublicationHandoff{Status: h.Status, ArtifactID: h.ArtifactId, ArtifactDigest: h.ArtifactDigest, Destination: h.Destination, HumanAction: h.HumanAction, ReadbackOracle: h.ReadbackOracle, Preconditions: h.Preconditions, ApprovalReference: h.ApprovalReference, SourceOfTruth: h.SourceOfTruth, Freshness: h.Freshness}
}
func mapDrift(d *sourcev1.DistributionDriftResponse) *DistributionDrift {
	if d == nil {
		return nil
	}
	return &DistributionDrift{State: d.State, SourceChanged: d.SourceChanged, DestinationChanged: d.DestinationChanged, CurrentSourceDigest: d.CurrentSourceDigest, RecordedSourceDigest: d.RecordedSourceDigest, CurrentArtifactDigest: d.CurrentArtifactDigest, RecordedArtifactDigest: d.RecordedArtifactDigest, Actions: d.Actions, SourceOfTruth: d.SourceOfTruth, Freshness: d.Freshness}
}
func safeSourceError(err error) string {
	if err == nil {
		return ""
	}
	return strings.ReplaceAll(strings.TrimSpace(err.Error()), "\n", " ")
}
