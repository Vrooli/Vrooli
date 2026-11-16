package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	securityhealthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1"
	securityhealthconnect "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/securityhealthv1connect"
)

type candidateSecurityRPC interface {
	ScanCandidate(context.Context, *connect.Request[securityhealthv1.ScanCandidateRequest]) (*connect.Response[securityhealthv1.ScanCandidateResponse], error)
}

type CandidateSecurityFinding struct {
	RuleID      string
	Severity    string
	Title       string
	Description string
	FilePath    string
}

type CandidateSecurityResult struct {
	Complete    bool
	FailureCode string
	Findings    []CandidateSecurityFinding
	FixtureOnly bool
}

type CandidateSecurityScanner interface {
	Scan(context.Context, map[string][]byte) (CandidateSecurityResult, error)
}

const candidateSecurityTimeout = 30 * time.Second

// CandidateSecurityClient forwards the exact staged blobs to Security Health,
// which owns Gitleaks parsing and normalized redaction.
type CandidateSecurityClient struct {
	resolveURL func(context.Context) (string, error)
	makeClient func(string) securityhealthconnect.CandidateScannerServiceClient
}

func NewCandidateSecurityClient(timeout time.Duration) *CandidateSecurityClient {
	if timeout <= 0 {
		timeout = candidateSecurityTimeout
	}
	httpClient := &http.Client{Timeout: timeout}
	return &CandidateSecurityClient{
		resolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "security-health")
		},
		makeClient: func(baseURL string) securityhealthconnect.CandidateScannerServiceClient {
			return securityhealthconnect.NewCandidateScannerServiceClient(httpClient, baseURL)
		},
	}
}

func (c *CandidateSecurityClient) Scan(ctx context.Context, files map[string][]byte) (CandidateSecurityResult, error) {
	if c == nil || c.resolveURL == nil || c.makeClient == nil {
		return CandidateSecurityResult{}, fmt.Errorf("candidate security scanner is unavailable")
	}
	baseURL, err := c.resolveURL(ctx)
	if err != nil {
		return CandidateSecurityResult{}, fmt.Errorf("resolve security-health service: %w", err)
	}
	resp, err := c.makeClient(baseURL).ScanCandidate(ctx, connect.NewRequest(&securityhealthv1.ScanCandidateRequest{Files: files}))
	if err != nil {
		return CandidateSecurityResult{}, fmt.Errorf("candidate security scan request failed")
	}
	if resp == nil || resp.Msg == nil {
		return CandidateSecurityResult{}, fmt.Errorf("candidate security scan returned no response")
	}
	result := CandidateSecurityResult{Complete: resp.Msg.GetComplete(), FailureCode: resp.Msg.GetFailureCode()}
	for _, finding := range resp.Msg.GetFindings() {
		result.Findings = append(result.Findings, CandidateSecurityFinding{
			RuleID: finding.GetRuleId(), Severity: finding.GetSeverity(), Title: finding.GetTitle(),
			Description: finding.GetDescription(), FilePath: finding.GetFilePath(),
		})
	}
	return result, nil
}

var _ candidateSecurityRPC = (securityhealthconnect.CandidateScannerServiceClient)(nil)
