// Package tidinessprovider is the client-side seam for the tidiness-manager
// validation authority. The control plane transports the result; it does not
// reproduce the scanner locally.
package tidinessprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/vrooli/internal/tuning"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tidiness-manager/v1/validation"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const ScenarioName = "tidiness-manager"

const (
	TargetID       = "internal"
	TargetRoot     = "internal"
	TargetToolGlob = "internal/tools/*"
	TargetSafeGlob = "internal/safeguards/*"
)

var ErrUnavailable = errors.New("tidiness-manager unavailable")

type URLResolver func(context.Context, string) (string, error)

type Finding struct {
	Code        string
	Severity    string
	Location    string
	Message     string
	Remediation string
}

type Result struct {
	Status   string
	Findings []Finding
}

type Client interface {
	Validate(context.Context, string) (Result, error)
}

type Provider struct {
	ResolveURL URLResolver
	HTTPClient *http.Client
	Timeout    time.Duration
}

func NewDefault() Provider {
	return Provider{ResolveURL: discovery.ResolveScenarioURLDefault}
}

func (p Provider) Validate(ctx context.Context, root string) (Result, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return Result{}, fmt.Errorf("%w: repository root is required", ErrUnavailable)
	}
	resolveURL := p.ResolveURL
	if resolveURL == nil {
		resolveURL = discovery.ResolveScenarioURLDefault
	}
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = tuning.ExtendedOperationTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	baseURL, err := resolveURL(callCtx, ScenarioName)
	if err != nil {
		return Result{}, fmt.Errorf("%w: resolve %s: %v", ErrUnavailable, ScenarioName, err)
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return Result{}, fmt.Errorf("%w: %s returned an empty API URL", ErrUnavailable, ScenarioName)
	}
	httpClient := p.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	client := scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)
	response, err := client.ValidateTarget(callCtx, connect.NewRequest(&scenariovalidationv1.ValidateTargetRequest{
		Target: &commonv1.ValidationTarget{
			Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE,
			Id:   TargetID,
			Root: TargetRoot,
		},
		Path:    filepath.Join(root, TargetRoot),
		Exclude: []string{TargetToolGlob, TargetSafeGlob},
	}))
	if err != nil {
		return Result{}, fmt.Errorf("%w: validate project: %v", ErrUnavailable, err)
	}
	if response == nil || response.Msg == nil {
		return Result{}, fmt.Errorf("%w: %s returned an empty validation response", ErrUnavailable, ScenarioName)
	}
	return resultFromResponse(response.Msg), nil
}

func resultFromResponse(response *scenariovalidationv1.ValidateTargetResponse) Result {
	result := Result{Status: response.GetStatus().String()}
	if native := unmarshalNative(response.GetNativeDetail()); native != nil {
		for _, finding := range append(native.GetFindings(), native.GetViolations()...) {
			if finding == nil {
				continue
			}
			code := finding.GetRuleId()
			if code == "" {
				code = finding.GetCategory()
			}
			appendUniqueFinding(&result, Finding{
				Code:        code,
				Severity:    finding.GetSeverity(),
				Location:    finding.GetFilePath(),
				Message:     finding.GetDescription(),
				Remediation: firstNonEmpty(finding.GetRemediation(), finding.GetRecommendedRemediation()),
			})
		}
	}
	if assessment := response.GetAssessment(); assessment != nil {
		for _, finding := range assessment.GetFindings() {
			if finding == nil {
				continue
			}
			appendUniqueFinding(&result, Finding{
				Code:        finding.GetCode(),
				Severity:    finding.GetSeverity(),
				Location:    finding.GetLocation(),
				Message:     finding.GetMessage(),
				Remediation: finding.GetRemediation(),
			})
		}
	}
	return result
}

func appendUniqueFinding(result *Result, finding Finding) {
	for _, existing := range result.Findings {
		if existing == finding {
			return
		}
	}
	result.Findings = append(result.Findings, finding)
}

func unmarshalNative(detail *anypb.Any) *validationv1.TidinessScanResponse {
	if detail == nil {
		return nil
	}
	native := &validationv1.TidinessScanResponse{}
	if err := anypb.UnmarshalTo(detail, native, proto.UnmarshalOptions{}); err != nil {
		return nil
	}
	return native
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
