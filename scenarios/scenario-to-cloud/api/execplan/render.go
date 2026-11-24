package execplan

import (
	"fmt"
	"sort"
	"strings"
)

// Change is one reviewable entry of a preview.
type Change struct {
	ActionID     string `json:"action_id"`
	Operation    string `json:"operation"`
	Effect       string `json:"effect"`
	Capability   string `json:"capability"`
	Summary      string `json:"summary"`
	Verification string `json:"verification"`
	Recovery     string `json:"recovery"`
	Retry        string `json:"retry"`
	CancelPoint  bool   `json:"cancel_point"`
}

// DataEffect names a persistent-data set the plan touches or protects.
type DataEffect struct {
	ActionID string `json:"action_id"`
	Subject  string `json:"subject"`
	Effect   string `json:"effect"`
}

// ShellPreviewLine is a derived convenience: what an operator would type to
// do the same thing by hand. It is never consumed by execution.
type ShellPreviewLine struct {
	ActionID string `json:"action_id"`
	Command  string `json:"command"`
}

// Preview is the review surface rendered from a plan.
type Preview struct {
	Target           string       `json:"target"`
	Outcome          string       `json:"outcome"`
	Changes          []Change     `json:"changes"`
	DataEffects      []DataEffect `json:"data_effects"`
	Downtime         Downtime     `json:"downtime"`
	RecoveryStrategy string       `json:"recovery_strategy"`
	Handoff          *Handoff     `json:"handoff,omitempty"`
	// Advisories are non-blocking findings the operator should read before
	// approving. They are the review's warning channel: before it existed,
	// a preview could only block or stay silent.
	Advisories   []Advisory         `json:"advisories,omitempty"`
	ShellPreview []ShellPreviewLine `json:"shell_preview"`
}

// ShellRenderer derives a display command for an action. Returning "" omits
// the line. The renderer receives a copy of the action and cannot alter it.
type ShellRenderer func(action Action) string

// RenderOption customises Render.
type RenderOption func(*renderOptions)

type renderOptions struct {
	shell ShellRenderer
}

// WithShellRenderer attaches a shell renderer for the convenience preview.
func WithShellRenderer(r ShellRenderer) RenderOption {
	return func(o *renderOptions) { o.shell = r }
}

// Render derives the preview from the action graph. It reads only the plan.
func Render(plan *Plan, opts ...RenderOption) Preview {
	options := renderOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	preview := Preview{
		Changes:      []Change{},
		DataEffects:  []DataEffect{},
		ShellPreview: []ShellPreviewLine{},
	}
	if plan == nil {
		return preview
	}
	preview.Target = targetLabel(plan.Target)
	preview.Outcome = plan.Outcome
	preview.Handoff = plan.Handoff
	preview.Advisories = plan.Advisories
	preview.Downtime = Downtime{ExpectedSeconds: totalDowntime(plan.Actions)}
	recovery := map[string]bool{}
	for _, action := range plan.Actions {
		appendRenderedAction(&preview, action, options.shell, recovery)
	}
	preview.RecoveryStrategy = recoveryStrategy(recovery)
	return preview
}

func appendRenderedAction(preview *Preview, action Action, shell ShellRenderer, recovery map[string]bool) {
	preview.Changes = append(preview.Changes, Change{
		ActionID:     action.ID,
		Operation:    action.OwnerOperation,
		Effect:       action.Effect,
		Capability:   action.RequiredCapability,
		Summary:      summarize(action),
		Verification: action.Verification,
		Recovery:     action.Recovery,
		Retry:        action.Retry,
		CancelPoint:  action.CancelPoint,
	})
	if action.Downtime != nil && preview.Downtime.Reason == "" {
		preview.Downtime.Reason = action.Downtime.Reason
	}
	if action.Recovery != "" && action.Recovery != "none_required" {
		recovery[action.Recovery] = true
	}
	preview.DataEffects = append(preview.DataEffects, dataEffects(action)...)
	appendShellPreview(preview, shell, action)
}

func appendShellPreview(preview *Preview, shell ShellRenderer, action Action) {
	if shell == nil {
		return
	}
	if cmd := shell(cloneAction(action)); cmd != "" {
		preview.ShellPreview = append(preview.ShellPreview, ShellPreviewLine{ActionID: action.ID, Command: cmd})
	}
}

func recoveryStrategy(recovery map[string]bool) string {
	strategy := strings.Join(sortedKeys(recovery), ",")
	if strategy == "" {
		return "none_required"
	}
	return strategy
}

func targetLabel(t Target) string {
	if t.MachineID != "" {
		return fmt.Sprintf("machine:%s (enrollment %d, %s)", t.MachineID, t.EnrollmentGeneration, t.Transport)
	}
	if t.ComputeManagerID != "" {
		return fmt.Sprintf("compute-manager:%s (%s)", t.ComputeManagerID, t.Transport)
	}
	return fmt.Sprintf("%s target (enrollment %d)", t.Transport, t.EnrollmentGeneration)
}

func summarize(action Action) string {
	for _, summarizeGroup := range []func(Action) (string, bool){
		summarizeInfrastructure,
		summarizeRelease,
		summarizeData,
		summarizeEdge,
		summarizeRuntime,
		summarizeCredentials,
	} {
		if summary, ok := summarizeGroup(action); ok {
			return summary
		}
	}
	return action.OwnerOperation
}

func summarizeInfrastructure(action Action) (string, bool) {
	switch action.OwnerOperation {
	case OpHostPrepare:
		return "Ensure system packages: " + action.Inputs["packages"], true
	case OpEdgeFirewallAllow:
		return "Allow inbound " + action.Inputs["protocol"] + " ports " + action.Inputs["ports"], true
	default:
		return "", false
	}
}

func summarizeRelease(action Action) (string, bool) {
	switch action.OwnerOperation {
	case OpReleaseDeliver:
		return "Deliver release artifact " + action.Inputs["artifact_ref"] + " to " + action.Inputs["destination"], true
	case OpReleaseVerify:
		return "Verify delivered archive against release " + action.Inputs["release_digest"], true
	case OpReleaseStage:
		return "Stage release into " + action.Inputs["release_dir"], true
	case OpReleaseActivate:
		return "Activate release " + action.Inputs["release_digest"] + " (" + action.Inputs["strategy"] + "; runtime, then pointer)", true
	case OpReleaseRetainPredecesor:
		return "Retain predecessor release " + action.Inputs["predecessor_digest"] + " (rollback eligible: " + action.Inputs["rollback_eligible"] + ", " + action.Inputs["rollback_reason"] + ")", true
	case OpArtifactsRetire:
		return "Retire unreferenced release artifacts; protected: " + action.Inputs["protected"], true
	default:
		return "", false
	}
}

func summarizeData(action Action) (string, bool) {
	switch action.OwnerOperation {
	case OpDataInventory:
		return "Inventory persistent data under scenarios " + action.Inputs["scenarios"], true
	case OpDataBackup:
		return "Capture a recovery point of " + action.Inputs["bindings"] + " sealed under " + action.Inputs["recovery_key_ref"] + " (" + action.Inputs["migration_posture"] + ")", true
	case OpDataSchedule:
		return "Reconcile target-local DBM backup plan for " + action.Inputs["scenario"] + " to destination " + action.Inputs["destination"], true
	case OpDataRetire:
		return "Persistent data disposition: " + action.Inputs["retention_policy"] + " (" + action.Inputs["bindings"] + ")", true
	default:
		return "", false
	}
}

func summarizeEdge(action Action) (string, bool) {
	switch action.OwnerOperation {
	case OpEdgeRouteApply:
		return "Route " + action.Inputs["domain"] + " to upstream port " + action.Inputs["upstream_port"], true
	case OpEdgeDNSEnsure:
		return "Ensure " + action.Inputs["record_type"] + " " + action.Inputs["hostname"] + " via DNS profile " + action.Inputs["provider_profile"], true
	case OpEdgeRouteRetire:
		return "Remove the public route for " + action.Inputs["domain"], true
	default:
		return "", false
	}
}

func summarizeRuntime(action Action) (string, bool) {
	switch action.OwnerOperation {
	case OpConfigApply:
		return "Apply " + action.Inputs["selection"] + " configuration for environment " + action.Inputs["environment"], true
	case OpWorkloadStop:
		return "Stop scenario " + action.Inputs["scenario"], true
	case OpRuntimeStartDeps:
		return "Start resources [" + action.Inputs["resources"] + "] and scenarios [" + action.Inputs["scenarios"] + "]", true
	case OpWorkloadStart:
		return "Start scenario " + action.Inputs["scenario"] + " from the active release with ports " + action.Inputs["ports"], true
	case OpVerifyReadiness:
		return "Verify readiness: " + action.Inputs["checks"], true
	case OpInputResumeHandoff:
		return "Resume onboarding for missing inputs: " + action.Inputs["missing"], true
	default:
		return "", false
	}
}

func summarizeCredentials(action Action) (string, bool) {
	switch action.OwnerOperation {
	case OpCredentialsProvision:
		return "Provision credential descriptors: " + action.Inputs["descriptors"], true
	case OpGrantsRevoke:
		return "Revoke credential grants: " + action.Inputs["descriptors"], true
	default:
		return "", false
	}
}

func dataEffects(action Action) []DataEffect {
	var out []DataEffect
	add := func(list, effect string) {
		for _, subject := range strings.Split(list, ",") {
			if subject = strings.TrimSpace(subject); subject != "" {
				out = append(out, DataEffect{ActionID: action.ID, Subject: subject, Effect: effect})
			}
		}
	}
	switch action.OwnerOperation {
	case OpDataInventory:
		add(action.Inputs["bindings"], "inventoried")
		add(action.Inputs["legacy_preserve"], "inventoried")
	case OpDataBackup:
		add(action.Inputs["bindings"], "recovery_point_captured")
		add(action.Inputs["legacy_preserve"], "recovery_point_captured")
	case OpReleaseActivate:
		add(action.Inputs["data_bindings"], "bound_to_persistent_data")
		add(action.Inputs["legacy_carry"], "carried_forward")
	case OpDataRetire:
		add(action.Inputs["bindings"], "retention_"+action.Inputs["retention_policy"])
		add(action.Inputs["legacy_preserve"], "retention_"+action.Inputs["retention_policy"])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Subject < out[j].Subject })
	return out
}

func cloneAction(action Action) Action {
	clone := action
	clone.Inputs = make(map[string]string, len(action.Inputs))
	for k, v := range action.Inputs {
		clone.Inputs[k] = v
	}
	clone.DependsOn = append([]string(nil), action.DependsOn...)
	if action.Downtime != nil {
		d := *action.Downtime
		clone.Downtime = &d
	}
	return clone
}
