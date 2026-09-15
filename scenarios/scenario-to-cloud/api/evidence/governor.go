package evidence

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	dmevidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence"
	dmevidenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence/evidencev1connect"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness/readinessv1connect"
)

// Governor is the governance owner seam (Deployment Manager). The cloud
// service reports its coverage verdict, asks for one exact review, and reads
// the review back immediately before an effect. It never approves anything
// itself and never treats a transport success as a decision.
type Governor interface {
	// ReportCoverage records the cloud coverage verdict for the profile and
	// candidate commit the identity names.
	ReportCoverage(ctx context.Context, identity ReviewIdentity, verdict *commonv1.TargetVerdict) error
	// PrepareReview asks the owner to prepare (or return) the review for the
	// exact identity and returns the owner's review reference.
	PrepareReview(ctx context.Context, identity ReviewIdentity) (*ReviewSnapshot, error)
	// GetReview reads one review by reference.
	GetReview(ctx context.Context, ref string) (*ReviewSnapshot, error)
}

// DMGovernor calls Deployment Manager over Connect.
type DMGovernor struct {
	ResolveURL func(context.Context) (string, error)
	HTTPClient *http.Client
	// Token is attached as a bearer credential; Deployment Manager admits the
	// cloud service principal for evidence reports through it.
	Token string
}

// NewDMGovernor resolves deployment-manager through scenario discovery. The
// bearer token comes from DEPLOYMENT_MANAGER_API_TOKEN or VROOLI_API_TOKEN.
func NewDMGovernor() *DMGovernor {
	token := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_API_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(os.Getenv("VROOLI_API_TOKEN"))
	}
	return &DMGovernor{
		ResolveURL: func(ctx context.Context) (string, error) {
			if configured := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_URL")); configured != "" {
				return strings.TrimRight(configured, "/"), nil
			}
			return discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")
		},
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		Token:      token,
	}
}

func (g *DMGovernor) base(ctx context.Context) (string, error) {
	if g == nil || g.ResolveURL == nil {
		return "", fmt.Errorf("governance owner is not configured")
	}
	url, err := g.ResolveURL(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve deployment-manager: %w", err)
	}
	return strings.TrimRight(url, "/"), nil
}

func (g *DMGovernor) client() *http.Client {
	if g.HTTPClient != nil {
		return g.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (g *DMGovernor) options() []connect.ClientOption {
	opts := []connect.ClientOption{connect.WithProtoJSON()}
	if strings.TrimSpace(g.Token) != "" {
		opts = append(opts, connect.WithInterceptors(bearerInterceptor{token: g.Token}))
	}
	return opts
}

type bearerInterceptor struct{ token string }

func (b bearerInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set("Authorization", "Bearer "+b.token)
		return next(ctx, req)
	}
}

func (b bearerInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (b bearerInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// ReportCoverage implements Governor.
func (g *DMGovernor) ReportCoverage(ctx context.Context, identity ReviewIdentity, verdict *commonv1.TargetVerdict) error {
	canonical, err := identity.Canonical()
	if err != nil {
		return err
	}
	base, err := g.base(ctx)
	if err != nil {
		return err
	}
	client := dmevidenceconnect.NewEvidenceServiceClient(g.client(), base, g.options()...)
	_, err = client.ReportTargetVerdict(ctx, connect.NewRequest(&dmevidencev1.ReportTargetVerdictRequest{ProfileId: canonical.ProfileID, GitCommitHash: canonical.CandidateCommit, Verdict: verdict}))
	if err != nil {
		return fmt.Errorf("report coverage verdict: %w", err)
	}
	return nil
}

func dmIdentity(i ReviewIdentity) *readinessv1.ReviewIdentity {
	return &readinessv1.ReviewIdentity{
		Scenario: i.ScenarioID, ProfileId: i.ProfileID, CandidateCommit: i.CandidateCommit, ArtifactDigest: i.ArtifactDigest,
		Targets: append([]string(nil), i.TargetSet...), Channel: i.Channel, PolicyVersion: int32(i.PolicyVersion),
		CandidateId: i.CandidateID, DestinationRevisionId: i.DestinationRevisionID, AuthorizationEpoch: i.AuthorizationEpoch,
	}
}

// PrepareReview implements Governor.
func (g *DMGovernor) PrepareReview(ctx context.Context, identity ReviewIdentity) (*ReviewSnapshot, error) {
	canonical, err := identity.Canonical()
	if err != nil {
		return nil, err
	}
	base, err := g.base(ctx)
	if err != nil {
		return nil, err
	}
	client := readinessconnect.NewReadinessServiceClient(g.client(), base, g.options()...)
	response, err := client.PrepareReview(ctx, connect.NewRequest(&readinessv1.PrepareReviewRequest{
		Scenario: canonical.ScenarioID, ProfileId: canonical.ProfileID, CandidateCommit: canonical.CandidateCommit, ArtifactDigest: canonical.ArtifactDigest,
		Targets: canonical.TargetSet, Channel: canonical.Channel, PolicyVersion: int32(canonical.PolicyVersion),
		Deliverable: "cloud publication of " + canonical.ScenarioID, Trigger: "scenario-to-cloud publication request",
		Facts:       map[string]string{"cloud_target": "true", "packaged_platform": "false"},
		CandidateId: canonical.CandidateID, DestinationRevisionId: canonical.DestinationRevisionID, AuthorizationEpoch: canonical.AuthorizationEpoch,
	}))
	if err != nil {
		return nil, fmt.Errorf("prepare readiness review: %w", err)
	}
	return snapshot(response.Msg), nil
}

// GetReview implements Governor.
func (g *DMGovernor) GetReview(ctx context.Context, ref string) (*ReviewSnapshot, error) {
	if strings.TrimSpace(ref) == "" {
		return nil, fmt.Errorf("review reference is required")
	}
	base, err := g.base(ctx)
	if err != nil {
		return nil, err
	}
	client := readinessconnect.NewReadinessServiceClient(g.client(), base, g.options()...)
	response, err := client.GetReview(ctx, connect.NewRequest(&readinessv1.GetReviewRequest{ReviewKey: ref}))
	if err != nil {
		return nil, fmt.Errorf("read readiness review: %w", err)
	}
	return snapshot(response.Msg), nil
}

func snapshot(msg *readinessv1.ReviewResponse) *ReviewSnapshot {
	if msg == nil {
		return nil
	}
	out := &ReviewSnapshot{Key: msg.GetReviewKey(), Status: msg.GetStatus(), PredecessorReleaseID: msg.GetPredecessorReleaseId(), PredecessorArtifact: msg.GetPredecessorArtifactDigest(), ApprovedBy: msg.GetApprovedBy()}
	if id := msg.GetIdentity(); id != nil {
		out.Scenario, out.ProfileID, out.CandidateCommit, out.ArtifactDigest = id.GetScenario(), id.GetProfileId(), id.GetCandidateCommit(), id.GetArtifactDigest()
		out.Targets, out.Channel, out.PolicyVersion = append([]string(nil), id.GetTargets()...), id.GetChannel(), int(id.GetPolicyVersion())
		out.CandidateID, out.DestinationRevisionID, out.AuthorizationEpoch = id.GetCandidateId(), id.GetDestinationRevisionId(), id.GetAuthorizationEpoch()
	}
	return out
}

var _ Governor = (*DMGovernor)(nil)
