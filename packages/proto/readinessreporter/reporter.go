// Package readinessreporter provides the small owner-side adapter for sending
// a Scenario Validation result to Deployment Manager's readiness ledger.
// Candidate identity remains configuration owned by the release operation; an
// owner never invents identity from a validation request.
package readinessreporter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness/readinessv1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	statusPassed      = "passed"
	statusFailed      = "failed"
	statusUnavailable = "unavailable"
)

type Client interface {
	ReportEvidence(context.Context, *connect.Request[readinessv1.ReportEvidenceRequest]) (*connect.Response[readinessv1.ReportEvidenceResponse], error)
}

type Identity struct {
	Scenario              string
	ProfileID             string
	CandidateCommit       string
	ArtifactDigest        string
	Targets               []string
	Channel               string
	PolicyVersion         int32
	CandidateID           string
	DestinationRevisionID string
	AuthorizationEpoch    uint64
}

func (i Identity) validate() error {
	if strings.TrimSpace(i.Scenario) == "" || strings.TrimSpace(i.ProfileID) == "" || strings.TrimSpace(i.CandidateCommit) == "" || strings.TrimSpace(i.ArtifactDigest) == "" || strings.TrimSpace(i.Channel) == "" || i.PolicyVersion <= 0 || len(i.Targets) == 0 {
		return errors.New("release identity is incomplete")
	}
	if (strings.TrimSpace(i.CandidateID) == "") != (strings.TrimSpace(i.DestinationRevisionID) == "") || (strings.TrimSpace(i.CandidateID) == "" && i.AuthorizationEpoch != 0) || (strings.TrimSpace(i.CandidateID) != "" && i.AuthorizationEpoch == 0) {
		return errors.New("release-bound readiness identity requires candidate, destination revision, and authorization epoch together")
	}
	return nil
}

type Reporter struct {
	client          Client
	identity        Identity
	criterionID     string
	producerBinding string
	producerVersion string
	err             error
}

type Config struct {
	BaseURL         string
	Token           string
	Identity        Identity
	CriterionID     string
	ProducerBinding string
	ProducerVersion string
	HTTPClient      *http.Client
}

func New(config Config) (*Reporter, error) {
	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	config.Token = strings.TrimSpace(config.Token)
	config.CriterionID = strings.TrimSpace(config.CriterionID)
	config.ProducerBinding = strings.TrimSpace(config.ProducerBinding)
	config.ProducerVersion = strings.TrimSpace(config.ProducerVersion)
	if config.BaseURL == "" || config.Token == "" || config.CriterionID == "" || config.ProducerBinding == "" || config.ProducerVersion == "" {
		return nil, errors.New("readiness reporter requires URL, token, criterion, binding, and producer version")
	}
	if err := config.Identity.validate(); err != nil {
		return nil, err
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	copy := *client
	copy.Transport = bearerTransport{base: client.Transport, token: config.Token}
	return &Reporter{
		client:          readinessconnect.NewReadinessServiceClient(&copy, config.BaseURL),
		identity:        config.Identity,
		criterionID:     config.CriterionID,
		producerBinding: config.ProducerBinding,
		producerVersion: config.ProducerVersion,
	}, nil
}

// NewFromEnv returns nil when the owner integration is not enabled. Once the
// token is present, all identity fields are mandatory and configuration errors
// are returned rather than disabling the callback silently.
func NewFromEnv(prefix, baseURL, criterionID, producerBinding, producerVersion string) (*Reporter, error) {
	prefix = strings.TrimRight(strings.TrimSpace(prefix), "_")
	token := os.Getenv(prefix + "_TOKEN")
	if strings.TrimSpace(token) == "" {
		return nil, nil
	}
	policyVersion, err := strconv.ParseInt(strings.TrimSpace(os.Getenv(prefix+"_POLICY_VERSION")), 10, 32)
	if err != nil || policyVersion <= 0 {
		return nil, errors.New("readiness policy version must be a positive integer")
	}
	var epoch uint64
	if raw := strings.TrimSpace(os.Getenv(prefix + "_AUTHORIZATION_EPOCH")); raw != "" {
		epoch, err = strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("readiness authorization epoch: %w", err)
		}
	}
	return New(Config{
		BaseURL: baseURL,
		Token:   token,
		Identity: Identity{
			Scenario: strings.TrimSpace(os.Getenv(prefix + "_SCENARIO")), ProfileID: strings.TrimSpace(os.Getenv(prefix + "_PROFILE_ID")),
			CandidateCommit: strings.TrimSpace(os.Getenv(prefix + "_CANDIDATE_COMMIT")), ArtifactDigest: strings.TrimSpace(os.Getenv(prefix + "_ARTIFACT_DIGEST")),
			Targets: splitTargets(os.Getenv(prefix + "_TARGETS")), Channel: strings.TrimSpace(os.Getenv(prefix + "_CHANNEL")), PolicyVersion: int32(policyVersion),
			CandidateID: strings.TrimSpace(os.Getenv(prefix + "_CANDIDATE_ID")), DestinationRevisionID: strings.TrimSpace(os.Getenv(prefix + "_DESTINATION_REVISION_ID")), AuthorizationEpoch: epoch,
		},
		CriterionID: criterionID, ProducerBinding: producerBinding, ProducerVersion: producerVersion,
	})
}

func splitTargets(value string) []string {
	var out []string
	for _, target := range strings.Split(value, ",") {
		if target = strings.TrimSpace(target); target != "" {
			out = append(out, target)
		}
	}
	return out
}

func (r *Reporter) Report(ctx context.Context, scenario string, response *scenariovalidationv1.ValidateScenarioResponse, observedAt time.Time) error {
	if r == nil {
		return nil
	}
	if r.err != nil {
		return r.err
	}
	if strings.TrimSpace(scenario) != r.identity.Scenario {
		return fmt.Errorf("validation scenario %q does not match configured release scenario %q", scenario, r.identity.Scenario)
	}
	if response == nil {
		return errors.New("validation response is required")
	}
	if observedAt.IsZero() {
		return errors.New("observation time is required")
	}
	status, err := Status(response.GetStatus())
	if err != nil {
		return err
	}
	detail := fmt.Sprintf("%s validation status=%s", r.producerVersion, response.GetStatus().String())
	if classification := strings.TrimSpace(response.GetFailureClassification()); classification != "" {
		detail += " classification=" + classification
	}
	return r.ReportStatus(ctx, scenario, status, detail, fmt.Sprintf("%s:validation:%s:%s", r.producerVersion, scenario, observedAt.UTC().Format(time.RFC3339Nano)), observedAt)
}

// ReportStatus lets an owner publish evidence from a domain operation that
// does not implement the shared Scenario Validation RPC. The owner still
// supplies the exact configured release identity and an attributable effect
// reference; an absent or failed callback remains an error to the caller.
func (r *Reporter) ReportStatus(ctx context.Context, scenario, status, detail, evidenceReference string, observedAt time.Time) error {
	if r == nil {
		return nil
	}
	if r.err != nil {
		return r.err
	}
	if strings.TrimSpace(scenario) != r.identity.Scenario {
		return fmt.Errorf("operation scenario %q does not match configured release scenario %q", scenario, r.identity.Scenario)
	}
	if _, err := validStatus(status); err != nil {
		return err
	}
	if observedAt.IsZero() || strings.TrimSpace(evidenceReference) == "" {
		return errors.New("status evidence requires observation time and reference")
	}
	_, err := r.client.ReportEvidence(ctx, connect.NewRequest(&readinessv1.ReportEvidenceRequest{
		Scenario: strings.TrimSpace(scenario), ProfileId: r.identity.ProfileID, CandidateCommit: r.identity.CandidateCommit, ArtifactDigest: r.identity.ArtifactDigest,
		Targets: append([]string(nil), r.identity.Targets...), Channel: r.identity.Channel, PolicyVersion: r.identity.PolicyVersion,
		CriterionId: r.criterionID, ProducerBinding: r.producerBinding, ProducerVersion: r.producerVersion, Status: strings.TrimSpace(status),
		ObservedAt: timestamppb.New(observedAt.UTC()), EvidenceReference: strings.TrimSpace(evidenceReference),
		Detail: detail, CandidateId: r.identity.CandidateID, DestinationRevisionId: r.identity.DestinationRevisionID, AuthorizationEpoch: r.identity.AuthorizationEpoch,
	}))
	if err != nil {
		return fmt.Errorf("report readiness: %w", err)
	}
	return nil
}

func Status(status scenariovalidationv1.ValidationStatus) (string, error) {
	switch status {
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED:
		return statusPassed, nil
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
		scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED:
		return statusFailed, nil
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR,
		scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_SKIPPED,
		scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED:
		return statusUnavailable, nil
	default:
		return "", fmt.Errorf("unsupported validation status %q", status.String())
	}
}

func validStatus(status string) (string, error) {
	status = strings.TrimSpace(status)
	switch status {
	case statusPassed, statusFailed, statusUnavailable:
		return status, nil
	default:
		return "", fmt.Errorf("unsupported readiness status %q", status)
	}
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
