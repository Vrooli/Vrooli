package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"storage-manager/internal/cleanup"
)

// ProviderFinding is the small, serialized join contract supplied by the
// control-plane classifier. Keeping this scenario module independent of the
// root internal package preserves Go's module boundary while still refusing
// to infer ownership locally.
type ProviderFinding struct {
	Kind, Name, Class, Reason string
	Evidence                  []string
	Resident, Swapped         uint64
	Finding                   bool
}

// UndeclaredWorkloadProvider is deliberately narrow: it can only dispose of
// names supplied by an authoritative historical-declaration resolver. An
// arbitrary container is never inferred to be Vrooli residue from its name.
type UndeclaredWorkloadProviderConfig struct {
	HistoricalNames map[string]string // name -> evidence
	// Findings is the control-plane classification seam. When supplied, the
	// provider never re-infers ownership from a name; it previews only
	// abandoned findings and refuses declared or unmanaged workloads.
	Findings  []ProviderFinding
	Posture   string
	Saturated func(context.Context) (bool, error)
}

type UndeclaredWorkloadProvider struct {
	meta       cleanup.ProviderMetadata
	runner     cleanup.ProcessRunner
	historical map[string]string
	findings   []ProviderFinding
	posture    string
	saturated  func(context.Context) (bool, error)
}

func NewUndeclaredWorkloadProvider(runner cleanup.ProcessRunner, cfg UndeclaredWorkloadProviderConfig) *UndeclaredWorkloadProvider {
	return &UndeclaredWorkloadProvider{
		meta: cleanup.ProviderMetadata{
			ID: "undeclared-workload", Name: "Abandoned undeclared workloads", Version: "v1",
			OwnerScenario: "storage-manager", SafetyTier: cleanup.SafetyTierConditional,
			DefaultMode: cleanup.ProviderModeDisabled, DefaultApproval: cleanup.ApprovalModeOperator,
			SupportedPlatforms: []string{"linux", "darwin", "windows"},
			IrreversibleEffects: []string{
				"stops and removes an abandoned container",
				"disables and removes an abandoned service unit or scheduled task",
				"deletes an abandoned Vrooli binary",
			},
			TestSubstitute: "fake-process-runner-and-declaration-resolver",
		}, runner: runner, historical: cfg.HistoricalNames, findings: append([]ProviderFinding(nil), cfg.Findings...), posture: cfg.Posture, saturated: cfg.Saturated,
	}
}

func (p *UndeclaredWorkloadProvider) Metadata() cleanup.ProviderMetadata { return p.meta }

func (p *UndeclaredWorkloadProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	preview, err := p.Preview(ctx, cleanup.PreviewRequest{Scope: req.Scope, Policy: req.Policy})
	if err != nil {
		return cleanup.Estimate{}, err
	}
	return cleanup.Estimate{ProviderID: p.meta.ID, ProviderVersion: p.meta.Version, EstimatedBytes: sumPreviewBytes(preview.Items), ItemCount: len(preview.Items), RequiresApproval: true, BlockedReason: preview.BlockedReason, ObservedAt: req.Scope.Now}, nil
}

func (p *UndeclaredWorkloadProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	out := cleanup.Preview{ProviderID: p.meta.ID, ProviderVersion: p.meta.Version}
	if !req.Policy.Enabled {
		// Conditional providers remain disabled for apply, but their preview is
		// still read-only and useful for operator review. The orchestrator
		// refuses any apply whose policy is disabled.
		out.BlockedReason = "provider disabled by policy"
	}
	if p.runner == nil {
		if out.BlockedReason == "" {
			out.BlockedReason = "workload enumeration seam unavailable"
		}
		return out, nil
	}
	if len(p.findings) > 0 {
		for _, finding := range p.findings {
			if finding.Class != "abandoned" || !finding.Finding || !p.allowedByPosture(finding) {
				continue
			}
			out.Items = append(out.Items, cleanup.PreviewItem{ID: stableItemID(p.meta.ID, finding.Name), Path: finding.Name, Description: fmt.Sprintf("abandoned %s %s: %s", finding.Kind, finding.Name, strings.Join(finding.Evidence, "; ")), Action: disposalAction(finding.Kind), SafetyTier: p.meta.SafetyTier, Bytes: int64(finding.Resident + finding.Swapped)})
		}
		return out, nil
	}
	result, err := p.runner.Run(ctx, cleanup.ProcessCommand{Name: "docker", Args: []string{"ps", "-a", "--format", "{{json .}}"}})
	if err != nil {
		return out, err
	}
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var item struct{ Name, Names, Image, State string }
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			out.Warnings = append(out.Warnings, "docker enumeration line unreadable")
			continue
		}
		if item.Name == "" {
			item.Name = item.Names
		}
		evidence, ok := p.historical[item.Name]
		if !ok {
			continue
		}
		if p.posture == "vrooli_only" && !pathEvidence(evidence) {
			out.Warnings = append(out.Warnings, item.Name+": abandoned evidence lacks a manifest or scenario path")
			continue
		}
		out.Items = append(out.Items, cleanup.PreviewItem{ID: stableItemID(p.meta.ID, item.Name), Path: item.Name, Description: fmt.Sprintf("abandoned container %s (%s): %s", item.Name, item.Image, evidence), Action: disposalAction("container"), SafetyTier: p.meta.SafetyTier})
	}
	return out, nil
}

func (p *UndeclaredWorkloadProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != p.meta.Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider version mismatch: got %q want %q", req.ProviderVersion, p.meta.Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("apply requires idempotency key")
	}
	if req.ApprovalMode != cleanup.ApprovalModeOperator {
		return cleanup.ApplyResult{ProviderID: p.meta.ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"operator approval required"}}, nil
	}
	if p.runner == nil {
		return cleanup.ApplyResult{ProviderID: p.meta.ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"workload disposal seam unavailable"}}, nil
	}
	if p.saturated != nil {
		saturated, err := p.saturated(ctx)
		if err != nil {
			return cleanup.ApplyResult{}, err
		}
		if saturated {
			return cleanup.ApplyResult{ProviderID: p.meta.ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"host saturated; disposal is braked"}}, nil
		}
	}
	result := cleanup.ApplyResult{ProviderID: p.meta.ID}
	for _, item := range req.Preview.Items {
		if !p.allowedPath(item.Path) {
			result.SkippedItems = append(result.SkippedItems, item.ID)
			continue
		}
		command, ok := p.disposalCommand(item)
		if !ok {
			result.SkippedItems = append(result.SkippedItems, item.ID)
			result.Warnings = append(result.Warnings, item.ID+": unsupported workload disposal action")
			continue
		}
		if _, err := p.runner.Run(ctx, command); err != nil {
			result.SkippedItems = append(result.SkippedItems, item.ID)
			result.Warnings = append(result.Warnings, item.ID+": disposal failed")
			continue
		}
		result.Applied = true
		result.AppliedItems = append(result.AppliedItems, item.Path)
	}
	return result, nil
}

func (p *UndeclaredWorkloadProvider) allowedByPosture(f ProviderFinding) bool {
	if p.posture != "vrooli_only" {
		return true
	}
	for _, evidence := range f.Evidence {
		if pathEvidence(evidence) {
			return true
		}
	}
	return false
}

func pathEvidence(evidence string) bool {
	evidence = strings.ToLower(strings.TrimSpace(evidence))
	return strings.Contains(evidence, "manifest") || strings.Contains(evidence, "scenario/") || strings.Contains(evidence, "resource/") || strings.Contains(evidence, "agent-experiments/")
}

func (p *UndeclaredWorkloadProvider) allowedPath(path string) bool {
	if len(p.findings) > 0 {
		for _, finding := range p.findings {
			if finding.Name == path && finding.Class == "abandoned" && finding.Finding && p.allowedByPosture(finding) {
				return true
			}
		}
		return false
	}
	evidence, ok := p.historical[path]
	return ok && (p.posture != "vrooli_only" || pathEvidence(evidence))
}

func (p *UndeclaredWorkloadProvider) Verify(ctx context.Context, req cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	if p.runner == nil {
		return cleanup.VerifyResult{Verified: false, Message: "workload disposal seam unavailable"}, nil
	}
	if len(req.ApplyResult.AppliedItems) == 0 {
		return cleanup.VerifyResult{Verified: len(req.ApplyResult.SkippedItems) == 0, Message: "no workload disposal was applied"}, nil
	}
	for _, item := range req.ApplyResult.AppliedItems {
		action := p.actionForName(item)
		if action == "docker-remove-abandoned-workload" {
			if err := p.verifyContainer(ctx, item); err != nil {
				return cleanup.VerifyResult{Verified: false, Message: err.Error()}, nil
			}
			continue
		}
		command, ok := verificationCommand(action, item)
		if !ok {
			return cleanup.VerifyResult{Verified: false, Message: "cannot verify disposed workload: unsupported action for " + item}, nil
		}
		if _, runErr := p.runner.Run(ctx, command); runErr == nil {
			return cleanup.VerifyResult{Verified: false, Message: "disposed workload still appears in its native registry: " + item}, nil
		}
	}
	return cleanup.VerifyResult{Verified: true, Message: fmt.Sprintf("verified %d disposed workload(s) are absent from native registries", len(req.ApplyResult.AppliedItems))}, nil
}

func (p *UndeclaredWorkloadProvider) verifyContainer(ctx context.Context, name string) error {
	result, err := p.runner.Run(ctx, cleanup.ProcessCommand{Name: "docker", Args: []string{"ps", "-a", "--format", "{{json .}}"}})
	if err != nil {
		return fmt.Errorf("could not re-enumerate workloads after disposal: %w", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		var item struct{ Name, Names string }
		if json.Unmarshal([]byte(line), &item) == nil {
			observed := item.Name
			if observed == "" {
				observed = item.Names
			}
			if observed == name {
				return fmt.Errorf("disposed workload still enumerates: %s", name)
			}
		}
	}
	return nil
}

func disposalAction(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "service-unit":
		return "disable-remove-service-unit"
	case "scheduled-task":
		return "remove-scheduled-task"
	case "binary":
		return "remove-abandoned-binary"
	default:
		return "docker-remove-abandoned-workload"
	}
}

func (p *UndeclaredWorkloadProvider) disposalCommand(item cleanup.PreviewItem) (cleanup.ProcessCommand, bool) {
	switch item.Action {
	case "docker-remove-abandoned-workload":
		return cleanup.ProcessCommand{Name: "docker", Args: []string{"rm", "-f", item.Path}}, true
	case "disable-remove-service-unit":
		switch runtime.GOOS {
		case "linux":
			return cleanup.ProcessCommand{Name: "systemctl", Args: []string{"--user", "disable", "--now", item.Path}}, true
		case "darwin":
			return cleanup.ProcessCommand{Name: "launchctl", Args: []string{"bootout", "gui/" + fmt.Sprint(os.Getuid()), item.Path}}, true
		case "windows":
			return cleanup.ProcessCommand{Name: "sc.exe", Args: []string{"delete", item.Path}}, true
		}
	case "remove-scheduled-task":
		if runtime.GOOS == "windows" {
			return cleanup.ProcessCommand{Name: "schtasks.exe", Args: []string{"/Delete", "/TN", item.Path, "/F"}}, true
		}
	case "remove-abandoned-binary":
		if runtime.GOOS == "windows" {
			return cleanup.ProcessCommand{Name: "powershell.exe", Args: []string{"-NoProfile", "-Command", "Remove-Item -LiteralPath '" + strings.ReplaceAll(item.Path, "'", "''") + "' -Force"}}, true
		}
		return cleanup.ProcessCommand{Name: "rm", Args: []string{"-f", "--", item.Path}}, true
	}
	return cleanup.ProcessCommand{}, false
}

func (p *UndeclaredWorkloadProvider) actionForName(name string) string {
	for _, finding := range p.findings {
		if finding.Name == name {
			return disposalAction(finding.Kind)
		}
	}
	return disposalAction("container")
}

func verificationCommand(action, path string) (cleanup.ProcessCommand, bool) {
	switch action {
	case "disable-remove-service-unit":
		switch runtime.GOOS {
		case "linux":
			return cleanup.ProcessCommand{Name: "systemctl", Args: []string{"--user", "is-enabled", path}}, true
		case "darwin":
			return cleanup.ProcessCommand{Name: "launchctl", Args: []string{"print", "gui/" + fmt.Sprint(os.Getuid()) + "/" + path}}, true
		case "windows":
			return cleanup.ProcessCommand{Name: "sc.exe", Args: []string{"query", path}}, true
		}
	case "remove-scheduled-task":
		if runtime.GOOS == "windows" {
			return cleanup.ProcessCommand{Name: "schtasks.exe", Args: []string{"/Query", "/TN", path}}, true
		}
	case "remove-abandoned-binary":
		if runtime.GOOS == "windows" {
			return cleanup.ProcessCommand{Name: "powershell.exe", Args: []string{"-NoProfile", "-Command", "if (Test-Path -LiteralPath '" + strings.ReplaceAll(path, "'", "''") + "') { exit 1 }"}}, true
		}
		return cleanup.ProcessCommand{Name: "test", Args: []string{"-e", path}}, true
	}
	return cleanup.ProcessCommand{}, false
}
