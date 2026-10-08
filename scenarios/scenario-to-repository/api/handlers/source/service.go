package source

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-repository/v1/source"
	domain "scenario-to-repository/internal/source"
)

type Store struct {
	mu            sync.RWMutex
	db            *sql.DB
	artifacts     map[string]domain.Artifact
	distributions map[string]domain.Distribution
}

func NewStore() *Store {
	return &Store{artifacts: map[string]domain.Artifact{}, distributions: map[string]domain.Distribution{}}
}

// NewSQLiteStore persists artifact and distribution identity across process
// restarts. The in-memory constructor remains useful for isolated handler
// tests; production wiring always supplies the scenario-owned database.
func NewSQLiteStore(db *sql.DB) *Store {
	s := &Store{db: db, artifacts: map[string]domain.Artifact{}, distributions: map[string]domain.Distribution{}}
	return s
}

func (s *Store) saveArtifact(ctx context.Context, artifact domain.Artifact) error {
	payload, err := json.Marshal(artifact)
	if err != nil {
		return err
	}
	if s.db == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.artifacts[artifact.ArtifactID] = artifact
		return nil
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO source_artifacts(artifact_id,payload_json,created_at) VALUES(?,?,?) ON CONFLICT(artifact_id) DO UPDATE SET payload_json=excluded.payload_json`, artifact.ArtifactID, payload, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) loadArtifact(ctx context.Context, id string) (domain.Artifact, bool, error) {
	if s.db == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		a, ok := s.artifacts[id]
		return a, ok, nil
	}
	var payload []byte
	err := s.db.QueryRowContext(ctx, `SELECT payload_json FROM source_artifacts WHERE artifact_id=?`, id).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Artifact{}, false, nil
	}
	if err != nil {
		return domain.Artifact{}, false, err
	}
	var artifact domain.Artifact
	if err := json.Unmarshal(payload, &artifact); err != nil {
		return domain.Artifact{}, false, err
	}
	return artifact, true, nil
}

func (s *Store) saveDistribution(ctx context.Context, distribution domain.Distribution) error {
	payload, err := json.Marshal(distribution)
	if err != nil {
		return err
	}
	if s.db == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.distributions[distribution.DistributionID] = distribution
		return nil
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO source_distributions(distribution_id,payload_json,updated_at) VALUES(?,?,?) ON CONFLICT(distribution_id) DO UPDATE SET payload_json=excluded.payload_json,updated_at=excluded.updated_at`, distribution.DistributionID, payload, distribution.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

func (s *Store) loadDistribution(ctx context.Context, id string) (domain.Distribution, bool, error) {
	if s.db == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		d, ok := s.distributions[id]
		return d, ok, nil
	}
	var payload []byte
	err := s.db.QueryRowContext(ctx, `SELECT payload_json FROM source_distributions WHERE distribution_id=?`, id).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Distribution{}, false, nil
	}
	if err != nil {
		return domain.Distribution{}, false, err
	}
	var distribution domain.Distribution
	if err := json.Unmarshal(payload, &distribution); err != nil {
		return domain.Distribution{}, false, err
	}
	return distribution, true, nil
}

func (s *Store) listDistributions(ctx context.Context, scenario string, limit int) ([]domain.Distribution, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if s.db == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		out := make([]domain.Distribution, 0, len(s.distributions))
		for _, d := range s.distributions {
			if scenario == "" || d.Scenario == scenario {
				out = append(out, d)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
		if len(out) > limit {
			out = out[:limit]
		}
		return out, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT payload_json FROM source_distributions ORDER BY updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Distribution
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var d domain.Distribution
		if err := json.Unmarshal(payload, &d); err != nil {
			return nil, err
		}
		if scenario == "" || d.Scenario == scenario {
			out = append(out, d)
		}
	}
	return out, rows.Err()
}

type Handler struct {
	store     *Store
	logger    *log.Logger
	outputDir string
}

func NewHandler(store *Store, outputDir string, logger *log.Logger) *Handler {
	if store == nil {
		store = NewStore()
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{store: store, outputDir: outputDir, logger: logger}
}

func (h *Handler) AnalyzeClosure(ctx context.Context, req *connect.Request[sourcev1.AnalyzeClosureRequest]) (*connect.Response[sourcev1.ClosureResponse], error) {
	if req.Msg.GetScenario() == "" || req.Msg.GetSourceRoot() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario and source_root are required"))
	}
	closure, err := domain.AnalyzeClosure(req.Msg.GetSourceRoot(), req.Msg.GetScenario(), req.Msg.GetSourceDigest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&sourcev1.ClosureResponse{Closure: closureProto(closure)}), nil
}

func (h *Handler) AssembleExport(ctx context.Context, req *connect.Request[sourcev1.AssembleExportRequest]) (*connect.Response[sourcev1.ArtifactResponse], error) {
	if req.Msg.GetScenario() == "" || req.Msg.GetSourceRoot() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario and source_root are required"))
	}
	closure, err := domain.AnalyzeClosure(req.Msg.GetSourceRoot(), req.Msg.GetScenario(), req.Msg.GetSourceDigest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	policy := domain.DefaultPublishPolicy()
	decision, policyErr := domain.CheckPublishabilityWithPolicy(req.Msg.GetSourceRoot(), closure, policy)
	if policyErr != nil {
		return nil, connect.NewError(connect.CodeInternal, policyErr)
	}
	if !decision.Allowed {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("source distribution refused by publishability policy: %s", strings.Join(decision.Violations, "; ")))
	}
	recipe := domain.Recipe{SchemaVersion: 1, Scenario: req.Msg.GetScenario(), Mode: "buildable_source", ArchiveFormat: "tar_gzip", Deterministic: true}
	if recipeYAML := req.Msg.GetRecipeYaml(); recipeYAML != "" {
		parsed, _, parseErr := domain.ParseRecipe([]byte(recipeYAML))
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, parseErr)
		}
		if parsed.Scenario != req.Msg.GetScenario() {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("recipe scenario %q does not match requested scenario %q", parsed.Scenario, req.Msg.GetScenario()))
		}
		recipe = parsed
	}
	output := req.Msg.GetOutputPath()
	if output == "" {
		output = filepath.Join(h.outputDir, uuid.NewString()+".tar.gz")
	}
	artifact, err := domain.Assemble(req.Msg.GetSourceRoot(), output, closure, recipe)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if err := h.store.saveArtifact(ctx, artifact); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	distribution := domain.Distribution{
		DistributionID:        "distribution-" + artifact.ArtifactID,
		Scenario:              req.Msg.GetScenario(),
		SourceDigest:          artifact.SourceDigest,
		ClosureDigest:         artifact.ClosureDigest,
		RecipeDigest:          artifact.RecipeDigest,
		PolicyDigest:          policy.Digest(),
		ArtifactID:            artifact.ArtifactID,
		ArtifactDigest:        artifact.ArchiveDigest,
		VerificationStatus:    "pending",
		PublicationStatus:     "not_started",
		DriftState:            "unknown",
		SourceOfTruth:         "scenario-to-repository",
		SourceTimestamp:       time.Now().UTC(),
		Contents:              distributionContents(artifact),
		Exclusions:            distributionExclusions(closure, decision),
		UnresolvedObligations: append([]string(nil), closure.Unresolved...),
		RuntimeRequirements:   runtimeRequirements(closure),
		UpdatedAt:             time.Now().UTC(),
	}
	if err := h.store.saveDistribution(ctx, distribution); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&sourcev1.ArtifactResponse{Artifact: artifactProto(artifact)}), nil
}

func (h *Handler) VerifyExport(ctx context.Context, req *connect.Request[sourcev1.VerifyExportRequest]) (*connect.Response[sourcev1.VerificationResponse], error) {
	artifact, ok, err := h.store.loadArtifact(ctx, req.Msg.GetArtifactId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("artifact %q not found", req.Msg.GetArtifactId()))
	}
	if req.Msg.GetArchivePath() != "" {
		artifact.ArchivePath = req.Msg.GetArchivePath()
	}
	verification, _ := domain.VerifyArchive(artifact)
	distributionID := "distribution-" + artifact.ArtifactID
	if distribution, found, loadErr := h.store.loadDistribution(ctx, distributionID); loadErr == nil && found {
		distribution.VerificationStatus = verification.Status
		distribution.VerificationReceipt = verification.ReceiptDigest
		distribution.ArtifactDigest = artifact.ArchiveDigest
		distribution.DriftState = "up_to_date"
		distribution.UpdatedAt = time.Now().UTC()
		if saveErr := h.store.saveDistribution(ctx, distribution); saveErr != nil {
			return nil, connect.NewError(connect.CodeInternal, saveErr)
		}
	}
	return connect.NewResponse(&sourcev1.VerificationResponse{VerificationId: verification.VerificationID, ArtifactId: verification.ArtifactID, Status: verification.Status, GatesPassed: verification.GatesPassed, Failures: verification.Failures, ReceiptDigest: verification.ReceiptDigest, VerifiedAt: timestamp(verification.VerifiedAt)}), nil
}

func (h *Handler) PreparePublication(ctx context.Context, req *connect.Request[sourcev1.PreparePublicationRequest]) (*connect.Response[sourcev1.PublicationPreviewResponse], error) {
	if req.Msg.GetDistributionId() == "" || req.Msg.GetArtifactId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("distribution_id and artifact_id are required"))
	}
	if req.Msg.GetDestinationKind() == "" || req.Msg.GetDestinationReference() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("destination kind and reference are required"))
	}
	if strings.EqualFold(req.Msg.GetDestinationKind(), "host_api") || strings.EqualFold(req.Msg.GetDestinationKind(), "git") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("publication requires human actuation; automated destination writes are not supported"))
	}
	artifact, ok, err := h.store.loadArtifact(ctx, req.Msg.GetArtifactId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("artifact %q not found", req.Msg.GetArtifactId()))
	}
	distribution, found, err := h.store.loadDistribution(ctx, req.Msg.GetDistributionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !found || distribution.ArtifactID != artifact.ArtifactID {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("distribution is not bound to the requested artifact"))
	}
	if distribution.VerificationStatus != "passed" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("artifact verification has not passed"))
	}
	distribution.PublicationStatus = "awaiting_human"
	distribution.DestinationKind = req.Msg.GetDestinationKind()
	distribution.DestinationRef = req.Msg.GetDestinationReference()
	distribution.UpdatedAt = time.Now().UTC()
	if err := h.store.saveDistribution(ctx, distribution); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&sourcev1.PublicationPreviewResponse{DistributionId: req.Msg.GetDistributionId(), ArtifactId: req.Msg.GetArtifactId(), Status: "awaiting_human", HumanAction: "Review the exact artifact and manually publish it to the declared destination.", ReadbackOracle: "Read back destination content and compare its manifest digest and expected revision.", Preconditions: []string{"artifact verification passed", "destination is unchanged since last verification", "human publication authority is present"}}), nil
}

func (h *Handler) GetDistribution(ctx context.Context, req *connect.Request[sourcev1.GetDistributionRequest]) (*connect.Response[sourcev1.DistributionResponse], error) {
	distribution, ok, err := h.store.loadDistribution(ctx, req.Msg.GetDistributionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("distribution %q not found", req.Msg.GetDistributionId()))
	}
	return connect.NewResponse(&sourcev1.DistributionResponse{Distribution: distributionProto(distribution)}), nil
}

func (h *Handler) ListDistributions(ctx context.Context, req *connect.Request[sourcev1.ListDistributionsRequest]) (*connect.Response[sourcev1.ListDistributionsResponse], error) {
	distributions, err := h.store.listDistributions(ctx, req.Msg.GetScenario(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &sourcev1.ListDistributionsResponse{SourceOfTruth: "scenario-to-repository", ObservedAt: timestamp(time.Now().UTC()), Freshness: "current"}
	for _, d := range distributions {
		out.Distributions = append(out.Distributions, distributionProto(d))
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) GetDistributionContents(ctx context.Context, req *connect.Request[sourcev1.GetDistributionContentsRequest]) (*connect.Response[sourcev1.DistributionContentsResponse], error) {
	d, ok, err := h.store.loadDistribution(ctx, req.Msg.GetDistributionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("distribution %q not found", req.Msg.GetDistributionId()))
	}
	out := &sourcev1.DistributionContentsResponse{DistributionId: d.DistributionID, SourceOfTruth: d.SourceOfTruth, ObservedAt: timestamp(time.Now().UTC()), Freshness: "current", UnresolvedObligations: d.UnresolvedObligations, RuntimeRequirements: d.RuntimeRequirements}
	for _, c := range d.Contents {
		out.Contents = append(out.Contents, &sourcev1.DistributionContent{Path: c.Path, SourcePath: c.SourcePath, Category: c.Category, Digest: c.Digest, SizeBytes: uint64(c.SizeBytes)})
	}
	for _, e := range d.Exclusions {
		out.Exclusions = append(out.Exclusions, &sourcev1.DistributionExclusion{Path: e.Path, Category: e.Category, SafeReason: e.SafeReason})
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) GetPublicationHandoff(ctx context.Context, req *connect.Request[sourcev1.GetPublicationHandoffRequest]) (*connect.Response[sourcev1.PublicationHandoffResponse], error) {
	d, ok, err := h.store.loadDistribution(ctx, req.Msg.GetDistributionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("distribution %q not found", req.Msg.GetDistributionId()))
	}
	return connect.NewResponse(&sourcev1.PublicationHandoffResponse{DistributionId: d.DistributionID, ArtifactId: d.ArtifactID, ArtifactDigest: d.ArtifactDigest, Destination: d.DestinationRef, Status: d.PublicationStatus, HumanAction: "Review the exact artifact and manually publish it through the destination owner.", ReadbackOracle: "Destination read-back must match the artifact manifest and expected revision.", Preconditions: []string{"artifact verification passed", "Deployment Manager approval is present", "human publication authority is present"}, ApprovalReference: d.DeploymentDecision, SourceOfTruth: firstNonEmpty(d.SourceOfTruth, "scenario-to-repository"), ObservedAt: timestamp(time.Now().UTC()), Freshness: "current"}), nil
}

func (h *Handler) GetDistributionDrift(ctx context.Context, req *connect.Request[sourcev1.GetDistributionDriftRequest]) (*connect.Response[sourcev1.DistributionDriftResponse], error) {
	d, ok, err := h.store.loadDistribution(ctx, req.Msg.GetDistributionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("distribution %q not found", req.Msg.GetDistributionId()))
	}
	state := d.DriftState
	if state == "" {
		state = "unknown"
	}
	return connect.NewResponse(&sourcev1.DistributionDriftResponse{DistributionId: d.DistributionID, State: state, CurrentSourceDigest: d.SourceDigest, RecordedSourceDigest: d.SourceDigest, CurrentArtifactDigest: d.ArtifactDigest, RecordedArtifactDigest: d.ArtifactDigest, SourceOfTruth: firstNonEmpty(d.SourceOfTruth, "scenario-to-repository"), ObservedAt: timestamp(time.Now().UTC()), Freshness: "current"}), nil
}

func closureProto(c domain.Closure) *sourcev1.SourceClosure {
	out := &sourcev1.SourceClosure{SourceDigest: c.SourceDigest, Scenario: c.Scenario, ClosureDigest: c.ClosureDigest, Unresolved: c.Unresolved}
	for _, n := range c.Nodes {
		out.Nodes = append(out.Nodes, &sourcev1.ClosureNode{Id: n.ID, Kind: n.Kind})
	}
	for _, f := range c.Files {
		out.Files = append(out.Files, &sourcev1.SourceFile{SourcePath: f.SourcePath, ExportPath: f.ExportPath, Sha256: f.SHA256, Mode: f.Mode, ReasonRefs: f.Reasons})
	}
	return out
}

func artifactProto(a domain.Artifact) *sourcev1.Artifact {
	out := &sourcev1.Artifact{ArtifactId: a.ArtifactID, SourceDigest: a.SourceDigest, RecipeDigest: a.RecipeDigest, ClosureDigest: a.ClosureDigest, ManifestDigest: a.ManifestDigest, ArchiveDigest: a.ArchiveDigest, ArchivePath: a.ArchivePath, Status: a.Status}
	for _, f := range a.Files {
		out.Files = append(out.Files, &sourcev1.ArtifactManifestEntry{Path: f.Path, Sha256: f.SHA256, Mode: f.Mode, SourcePath: f.SourcePath, SizeBytes: uint64(f.SizeBytes)})
	}
	return out
}

func distributionProto(d domain.Distribution) *sourcev1.Distribution {
	return &sourcev1.Distribution{DistributionId: d.DistributionID, Scenario: d.Scenario, SourceDigest: d.SourceDigest, ClosureDigest: d.ClosureDigest, RecipeDigest: d.RecipeDigest, PolicyDigest: d.PolicyDigest, ArtifactId: d.ArtifactID, ArtifactDigest: d.ArtifactDigest, VerificationStatus: d.VerificationStatus, VerificationReceipt: d.VerificationReceipt, DeploymentManagerDecision: d.DeploymentDecision, PublicationStatus: d.PublicationStatus, DestinationKind: d.DestinationKind, DestinationReference: d.DestinationRef, LastVerifiedRevision: d.DestinationRevision, UpdatedAt: timestamp(d.UpdatedAt), SourceOfTruth: d.SourceOfTruth, SourceTimestamp: timestamp(d.SourceTimestamp), Freshness: "current", DriftState: d.DriftState, DestinationRevision: d.DestinationRevision, ReadbackReceipt: d.ReadbackReceipt, WorkflowUrl: d.WorkflowURL}
}

func distributionContents(a domain.Artifact) []domain.DistributionContent {
	out := make([]domain.DistributionContent, 0, len(a.Files))
	for _, f := range a.Files {
		out = append(out, domain.DistributionContent{Path: f.Path, SourcePath: f.SourcePath, Category: contentCategory(f.Path), Digest: f.SHA256, SizeBytes: f.SizeBytes})
	}
	return out
}
func contentCategory(path string) string {
	lower := strings.ToLower(filepath.ToSlash(path))
	if strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".txt") {
		return "documentation"
	}
	if strings.HasPrefix(lower, "generated/") || strings.Contains(lower, "/generated/") {
		return "generated"
	}
	if strings.HasPrefix(lower, "shared/") || strings.Contains(lower, "/shared/") {
		return "shared"
	}
	return "application"
}
func distributionExclusions(c domain.Closure, p domain.PolicyDecision) []domain.DistributionExclusion {
	out := make([]domain.DistributionExclusion, 0, len(p.Excluded))
	for _, path := range p.Excluded {
		out = append(out, domain.DistributionExclusion{Path: path, Category: "private material", SafeReason: "Excluded by source-ramp privacy policy"})
	}
	for _, f := range c.Files {
		if privateSourcePath(f.SourcePath) {
			out = append(out, domain.DistributionExclusion{Path: f.SourcePath, Category: "private material", SafeReason: "Excluded by source-ramp privacy policy"})
		}
	}
	return out
}
func privateSourcePath(path string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	return strings.HasPrefix(filepath.Base(lower), ".env") || strings.HasSuffix(lower, ".pem") || strings.HasSuffix(lower, ".key") || strings.HasSuffix(lower, ".p12") || strings.Contains(lower, ".vrooli/plan-artifacts/")
}
func runtimeRequirements(c domain.Closure) []string {
	out := []string{}
	for _, n := range c.Nodes {
		if n.Kind == "runtime_requirement" {
			out = append(out, n.ID)
		}
	}
	return out
}
func timestamp(t time.Time) *timestamppb.Timestamp { return timestamppb.New(t) }

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
