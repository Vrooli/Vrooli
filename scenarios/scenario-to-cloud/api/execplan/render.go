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
	Target           string             `json:"target"`
	Outcome          string             `json:"outcome"`
	Changes          []Change           `json:"changes"`
	DataEffects      []DataEffect       `json:"data_effects"`
	Downtime         Downtime           `json:"downtime"`
	RecoveryStrategy string             `json:"recovery_strategy"`
	Handoff          *Handoff           `json:"handoff,omitempty"`
	ShellPreview     []ShellPreviewLine `json:"shell_preview"`
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
	preview.Downtime = Downtime{ExpectedSeconds: totalDowntime(plan.Actions)}
	recovery := map[string]bool{}
	for _, action := range plan.Actions {
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
		if options.shell != nil {
			if cmd := options.shell(cloneAction(action)); cmd != "" {
				preview.ShellPreview = append(preview.ShellPreview, ShellPreviewLine{ActionID: action.ID, Command: cmd})
			}
		}
	}
	preview.RecoveryStrategy = strings.Join(sortedKeys(recovery), ",")
	if preview.RecoveryStrategy == "" {
		preview.RecoveryStrategy = "none_required"
	}
	return preview
}

func targetLabel(t Target) string {
	if t.MachineID != "" {
		return fmt.Sprintf("machine:%s (enrollment %d, %s)", t.MachineID, t.EnrollmentGeneration, t.Transport)
	}
	return fmt.Sprintf("%s target (enrollment %d)", t.Transport, t.EnrollmentGeneration)
}

func summarize(action Action) string {
	switch action.OwnerOperation {
	case OpHostPrepare:
		return "Ensure system packages: " + action.Inputs["packages"]
	case OpEdgeFirewallAllow:
		return "Allow inbound " + action.Inputs["protocol"] + " ports " + action.Inputs["ports"]
	case OpDataInventory:
		return "Inventory persistent data under scenarios " + action.Inputs["scenarios"]
	case OpReleaseDeliver:
		return "Deliver release artifact " + action.Inputs["artifact_ref"] + " to " + action.Inputs["destination"]
	case OpReleaseVerify:
		return "Verify delivered archive against release " + action.Inputs["release_digest"]
	case OpReleaseStage:
		return "Stage release into " + action.Inputs["release_dir"]
	case OpDataBackup:
		return "Capture a recovery point of " + action.Inputs["bindings"] + " sealed under " + action.Inputs["recovery_key_ref"] + " (" + action.Inputs["migration_posture"] + ")"
	case OpReleaseActivate:
		return "Activate release " + action.Inputs["release_digest"] + " (" + action.Inputs["strategy"] + "; runtime, then pointer)"
	case OpConfigApply:
		return "Apply " + action.Inputs["selection"] + " configuration for environment " + action.Inputs["environment"]
	case OpWorkloadStop:
		return "Stop scenario " + action.Inputs["scenario"]
	case OpEdgeRouteApply:
		return "Route " + action.Inputs["domain"] + " to upstream port " + action.Inputs["upstream_port"]
	case OpCredentialsProvision:
		return "Provision credential descriptors: " + action.Inputs["descriptors"]
	case OpRuntimeStartDeps:
		return "Start resources [" + action.Inputs["resources"] + "] and scenarios [" + action.Inputs["scenarios"] + "]"
	case OpWorkloadStart:
		return "Start scenario " + action.Inputs["scenario"] + " from the active release with ports " + action.Inputs["ports"]
	case OpVerifyReadiness:
		return "Verify readiness: " + action.Inputs["checks"]
	case OpReleaseRetainPredecesor:
		return "Retain predecessor release " + action.Inputs["predecessor_digest"] + " (rollback eligible: " + action.Inputs["rollback_eligible"] + ", " + action.Inputs["rollback_reason"] + ")"
	case OpInputResumeHandoff:
		return "Resume onboarding for missing inputs: " + action.Inputs["missing"]
	case OpEdgeRouteRetire:
		return "Remove the public route for " + action.Inputs["domain"]
	case OpGrantsRevoke:
		return "Revoke credential grants: " + action.Inputs["descriptors"]
	case OpDataRetire:
		return "Persistent data disposition: " + action.Inputs["retention_policy"] + " (" + action.Inputs["bindings"] + ")"
	case OpArtifactsRetire:
		return "Retire unreferenced release artifacts; protected: " + action.Inputs["protected"]
	}
	return action.OwnerOperation
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
