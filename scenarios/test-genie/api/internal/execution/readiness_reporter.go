package execution

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"test-genie/internal/orchestrator"

	"github.com/vrooli/api-core/discovery"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness/readinessv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	suiteStateCriterion  = "suite-state-known"
	suiteStateBinding    = "test-genie.runs.report"
	regressionCriterion  = "test-regression-proof"
	regressionBinding    = "test-genie.runs.compare-release"
	performanceCriterion = "performance-capacity-proven"
	performanceBinding   = "test-genie.performance.readiness"
)

// ReleaseComparison is the bounded result needed to attribute Test Genie's
// existing predecessor comparison to a release candidate. The comparator owns
// the comparison algorithm; this package only decides whether its result is
// strong enough to satisfy the readiness criterion.
type ReleaseComparison struct {
	Verdict       string
	Behavior      string
	Coverage      string
	Compatibility string
	Provenance    string
}

type ReleaseComparator interface {
	CompareRelease(context.Context, string, string, string) (ReleaseComparison, error)
}

// DeploymentManagerReadinessReporter sends only evidence that Test Genie can
// prove from its terminal result and its durable predecessor comparison.
type DeploymentManagerReadinessReporter struct {
	client     readinessReporterClient
	now        func() time.Time
	comparator ReleaseComparator
}

type readinessReporterClient interface {
	ReportEvidence(context.Context, *connect.Request[readinessv1.ReportEvidenceRequest]) (*connect.Response[readinessv1.ReportEvidenceResponse], error)
}

func NewDeploymentManagerReadinessReporter(baseURL, token string, client *http.Client) (*DeploymentManagerReadinessReporter, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	token = strings.TrimSpace(token)
	if baseURL == "" || token == "" {
		return nil, errors.New("deployment-manager readiness reporter requires base URL and bearer token")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	client.Transport = bearerTransport{base: client.Transport, token: token}
	return &DeploymentManagerReadinessReporter{
		client: readinessconnect.NewReadinessServiceClient(client, baseURL),
		now:    time.Now,
	}, nil
}

// SetReleaseComparator wires Test Genie's own durable comparison service after
// runtime construction. Keeping this seam separate avoids making the DM
// transport responsible for inventing predecessor evidence.
func (r *DeploymentManagerReadinessReporter) SetReleaseComparator(comparator ReleaseComparator) {
	if r != nil {
		r.comparator = comparator
	}
}

// NewConfiguredDeploymentManagerReadinessReporter enables the integration only
// when its server-side credential is provisioned. An absent credential leaves
// the normal Test Genie run path usable and lets readiness fail closed.
func NewConfiguredDeploymentManagerReadinessReporter(ctx context.Context) (*DeploymentManagerReadinessReporter, error) {
	token := strings.TrimSpace(os.Getenv("TEST_GENIE_DEPLOYMENT_MANAGER_READINESS_TOKEN"))
	if token == "" {
		return nil, nil
	}
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")
	if err != nil {
		return nil, fmt.Errorf("resolve deployment-manager: %w", err)
	}
	return NewDeploymentManagerReadinessReporter(baseURL, token, nil)
}

func (r *DeploymentManagerReadinessReporter) Report(ctx context.Context, req orchestrator.SuiteExecutionRequest, result *orchestrator.SuiteExecutionResult) error {
	if r == nil || r.client == nil {
		return errors.New("readiness reporter is not configured")
	}
	if req.ReleaseIdentity == nil {
		return errors.New("release identity is required for readiness reporting")
	}
	identity := req.ReleaseIdentity
	if strings.TrimSpace(identity.ProfileID) == "" || strings.TrimSpace(identity.CandidateCommit) == "" || strings.TrimSpace(identity.ArtifactDigest) == "" || strings.TrimSpace(identity.Channel) == "" || identity.PolicyVersion <= 0 || len(identity.Targets) == 0 {
		return errors.New("release identity is incomplete for readiness reporting")
	}
	if result == nil || strings.TrimSpace(result.RunID) == "" {
		return errors.New("terminal suite result is required for readiness reporting")
	}
	status := "failed"
	if result.Verdict == orchestrator.SuiteVerdictPass {
		status = "passed"
	}
	observedAt := result.CompletedAt.UTC()
	if observedAt.IsZero() {
		observedAt = r.now().UTC()
	}
	if err := r.reportEvidence(ctx, identity, req, result.RunID, suiteStateCriterion, suiteStateBinding, status, observedAt, fmt.Sprintf("terminal run %s verdict=%s phases=%d", result.RunID, result.Verdict, len(result.Phases))); err != nil {
		return err
	}
	for _, phase := range result.Phases {
		if strings.EqualFold(strings.TrimSpace(phase.Name), "performance") {
			performanceStatus := "failed"
			if strings.EqualFold(strings.TrimSpace(phase.Status), "passed") {
				performanceStatus = "passed"
			}
			if err := r.reportEvidence(ctx, identity, req, result.RunID, performanceCriterion, performanceBinding, performanceStatus, observedAt, fmt.Sprintf("terminal run %s performance phase status=%s", result.RunID, phase.Status)); err != nil {
				return err
			}
			break
		}
	}
	if predecessor := strings.TrimSpace(identity.PredecessorRunID); predecessor != "" {
		if r.comparator == nil {
			return errors.New("release predecessor comparison is not configured")
		}
		comparison, err := r.comparator.CompareRelease(ctx, identityScenario(req, identity), predecessor, result.RunID)
		if err != nil {
			return fmt.Errorf("compare release predecessor %s with run %s: %w", predecessor, result.RunID, err)
		}
		status := "failed"
		if strings.EqualFold(strings.TrimSpace(comparison.Verdict), "clean") {
			status = "passed"
		}
		detail := fmt.Sprintf("predecessor=%s current=%s verdict=%s behavior=%s coverage=%s compatibility=%s provenance=%s", predecessor, result.RunID, comparison.Verdict, comparison.Behavior, comparison.Coverage, comparison.Compatibility, comparison.Provenance)
		return r.reportEvidence(ctx, identity, req, result.RunID, regressionCriterion, regressionBinding, status, observedAt, detail)
	}
	return nil
}

func (r *DeploymentManagerReadinessReporter) reportEvidence(ctx context.Context, identity *orchestrator.ReleaseIdentity, req orchestrator.SuiteExecutionRequest, runID, criterion, binding, status string, observedAt time.Time, detail string) error {
	_, err := r.client.ReportEvidence(ctx, connect.NewRequest(&readinessv1.ReportEvidenceRequest{
		Scenario:              identityScenario(req, identity),
		ProfileId:             identity.ProfileID,
		CandidateCommit:       identity.CandidateCommit,
		ArtifactDigest:        identity.ArtifactDigest,
		Targets:               append([]string(nil), identity.Targets...),
		Channel:               identity.Channel,
		PolicyVersion:         int32(identity.PolicyVersion),
		CriterionId:           criterion,
		ProducerBinding:       binding,
		ProducerVersion:       "test-genie",
		Status:                status,
		ObservedAt:            timestamppb.New(observedAt),
		EvidenceReference:     "test-genie:run:" + runID,
		Detail:                detail,
		CandidateId:           identity.CandidateID,
		DestinationRevisionId: identity.DestinationRevisionID,
		AuthorizationEpoch:    identity.AuthorizationEpoch,
	}))
	return err
}

func identityScenario(req orchestrator.SuiteExecutionRequest, identity *orchestrator.ReleaseIdentity) string {
	if identity != nil && strings.TrimSpace(req.ScenarioName) != "" {
		name := strings.TrimSpace(req.ScenarioName)
		if scenario, ok := strings.CutPrefix(name, "scenario:"); ok {
			return scenario
		}
		return name
	}
	return "unknown"
}

type bearerTransport struct {
	base  http.RoundTripper
	token string
}

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return base.RoundTrip(clone)
}
