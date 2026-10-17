package remediation

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
)

var (
	ErrAutoHealDisabled       = errors.New("remediation refused: autoHeal is disabled")
	ErrAskNotApproved         = errors.New("remediation refused: ask response is not approved")
	ErrAskVerifierUnavailable = errors.New("remediation refused: notification ask verifier is unavailable")
	ErrPreflightFailed        = errors.New("remediation refused: script preflight failed")
)

// AskApproval is the server-side result of reading the durable answer from
// notification-hub.  The HTTP caller must not be allowed to assert this value
// itself: an approved remediation is an authorization record, not a request
// field.
type AskApproval struct {
	AskID  string
	Answer string
	Actor  string
}

type AskVerifier interface {
	Verify(context.Context, string) (AskApproval, error)
}

type ScriptRunner interface {
	Preflight(context.Context, string) error
	Run(context.Context, string) (int, string, error)
}

type shellScriptRunner struct{}

func (shellScriptRunner) Preflight(ctx context.Context, path string) error {
	return exec.CommandContext(ctx, "bash", "-n", path).Run()
}

func (shellScriptRunner) Run(ctx context.Context, path string) (int, string, error) {
	output, err := exec.CommandContext(ctx, "bash", path).CombinedOutput()
	if err == nil {
		return 0, string(output), nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), string(output), err
	}
	return -1, string(output), err
}

type Authorisation struct {
	AskID               string
	IncidentID          string
	IncidentFingerprint string
	CandidateID         string
	Approved            bool
	AutoHealEnabled     bool
}

func ApprovedAsk(approval AskApproval) bool {
	return strings.EqualFold(strings.TrimSpace(approval.Answer), "approve") ||
		strings.EqualFold(strings.TrimSpace(approval.Answer), "approved") ||
		strings.EqualFold(strings.TrimSpace(approval.Answer), "yes")
}

type ExecutionResult struct {
	AskID       string `json:"askId"`
	IncidentID  string `json:"incidentId"`
	CandidateID string `json:"candidateId"`
	ScriptPath  string `json:"scriptPath"`
	ExitStatus  int    `json:"exitStatus"`
	Output      string `json:"output,omitempty"`
	Success     bool   `json:"success"`
}

func (s *Service) SetScriptRunner(runner ScriptRunner) {
	if runner != nil {
		s.runner = runner
	}
}

func (s *Service) Execute(ctx context.Context, incident incidents.Incident, candidateID string, auth Authorisation) (ExecutionResult, error) {
	if !auth.AutoHealEnabled {
		return ExecutionResult{}, ErrAutoHealDisabled
	}
	if !auth.Approved {
		return ExecutionResult{}, ErrAskNotApproved
	}
	if strings.TrimSpace(auth.AskID) == "" {
		return ExecutionResult{}, fmt.Errorf("%w: ask id is missing", ErrAskNotApproved)
	}
	if auth.IncidentID != incident.ID || auth.IncidentFingerprint != incident.Fingerprint || auth.CandidateID != candidateID {
		return ExecutionResult{}, fmt.Errorf("%w: authorization does not match incident and candidate", ErrAskNotApproved)
	}
	candidate, ok := findCandidate(incident.RemediationCandidates, candidateID)
	if !ok {
		return ExecutionResult{}, fmt.Errorf("remediation candidate %q not found", candidateID)
	}
	if len(incident.RemediationArtifacts) == 0 {
		return ExecutionResult{}, fmt.Errorf("remediation candidate %q has no generated artifact", candidateID)
	}
	var scriptPath string
	for _, artifact := range incident.RemediationArtifacts {
		if artifact.RemediationID == candidate.ID && strings.TrimSpace(artifact.Path) != "" {
			scriptPath = filepath.Join(artifact.Path, "remediation.sh")
			break
		}
	}
	if scriptPath == "" {
		return ExecutionResult{}, fmt.Errorf("remediation candidate %q has no generated script", candidateID)
	}
	runner := s.runner
	if runner == nil {
		runner = shellScriptRunner{}
	}
	if err := runner.Preflight(ctx, scriptPath); err != nil {
		return ExecutionResult{AskID: auth.AskID, IncidentID: incident.ID, CandidateID: candidateID, ScriptPath: scriptPath, ExitStatus: -1}, fmt.Errorf("%w: %v", ErrPreflightFailed, err)
	}
	exitStatus, output, runErr := runner.Run(ctx, scriptPath)
	result := ExecutionResult{AskID: auth.AskID, IncidentID: incident.ID, CandidateID: candidateID, ScriptPath: scriptPath, ExitStatus: exitStatus, Output: output, Success: runErr == nil && exitStatus == 0}
	if runErr != nil {
		return result, runErr
	}
	return result, nil
}
