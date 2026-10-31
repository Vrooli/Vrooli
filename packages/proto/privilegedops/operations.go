// Package privilegedops is the single vocabulary for commands that may cross
// the node's privilege boundary. Keeping the names beside the wire enum makes
// the control plane and helper use the same closed set; adding a name to only
// one side is impossible without changing this package and its tests.
package privilegedops

import (
	"fmt"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
)

const (
	BreakGlassAudience = "vrooli:uninstall"
	BreakGlassScope    = "vrooli:uninstall"

	Provision              = "provision"
	InventoryInstallation  = "inventory_installation"
	PlanUninstall          = "plan_uninstall"
	ProvisionBreakGlass    = "provision_break_glass"
	IssueCleanupCapability = "issue_cleanup_capability"
	ApplyFrozenPlan        = "apply_frozen_plan"
	VerifyResult           = "verify_result"
	RotateBreakGlass       = "rotate_break_glass"
	ResetBreakGlass        = "reset_break_glass"
)

// Capability is the shared vocabulary used by onboarding reports and the
// Bridge readiness projection. These values describe what a node can accept
// or what the control plane has established; they are observations, not
// authorization grants. Authorization remains the separately approved scope
// list on the node record.
type Capability struct {
	Name  string
	Label string
}

const (
	CapabilityAgentPresence         = "agent.presence"
	CapabilityRuntime               = "runtime"
	CapabilityProvisioning          = "provision"
	CapabilitySSHManagement         = "ssh.management"
	CapabilityCleanupPlanning       = "cleanup.plan"
	CapabilityCleanupApplication    = "cleanup.apply"
	CapabilityTargetBoundBreakGlass = "break-glass.target-bound"
)

// OnboardingCapabilities is the single capability inventory shared by the
// onboarding report and Bridge readiness policy. Keep the order stable: it is
// operator-facing and makes additions visible in reviews and tests.
var onboardingCapabilities = []Capability{
	{Name: CapabilityAgentPresence, Label: "agent presence"},
	{Name: CapabilityRuntime, Label: "runtime"},
	{Name: CapabilityProvisioning, Label: "provisioning"},
	{Name: CapabilitySSHManagement, Label: "SSH management"},
	{Name: CapabilityCleanupPlanning, Label: "cleanup planning"},
	{Name: CapabilityCleanupApplication, Label: "cleanup application"},
	{Name: CapabilityTargetBoundBreakGlass, Label: "target-bound break-glass"},
}

// OnboardingCapabilities returns a copy so callers cannot mutate the shared
// vocabulary and make a later readiness report order-dependent.
func OnboardingCapabilities() []Capability {
	return append([]Capability(nil), onboardingCapabilities...)
}

func CapabilityNames() []string {
	capabilities := OnboardingCapabilities()
	names := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		names = append(names, capability.Name)
	}
	return names
}

var names = map[channelv1.PrivilegedOperation]string{
	channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_PROVISION:                Provision,
	channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_INVENTORY_INSTALLATION:   InventoryInstallation,
	channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_PLAN_UNINSTALL:           PlanUninstall,
	channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_PROVISION_BREAK_GLASS:    ProvisionBreakGlass,
	channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_ISSUE_CLEANUP_CAPABILITY: IssueCleanupCapability,
	channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_APPLY_FROZEN_PLAN:        ApplyFrozenPlan,
	channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_VERIFY_RESULT:            VerifyResult,
	channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_ROTATE_BREAK_GLASS:       RotateBreakGlass,
	channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_RESET_BREAK_GLASS:        ResetBreakGlass,
}

var enums = func() map[string]channelv1.PrivilegedOperation {
	out := make(map[string]channelv1.PrivilegedOperation, len(names))
	for operation, name := range names {
		out[name] = operation
	}
	return out
}()

// Name returns the canonical operation name. Unknown enum values are retained
// in the returned string so refusal messages identify the exact value.
func Name(operation channelv1.PrivilegedOperation) string {
	if name, ok := names[operation]; ok {
		return name
	}
	return fmt.Sprintf("enum:%d", operation)
}

// Parse converts a canonical operation name to the wire enum.
func Parse(name string) (channelv1.PrivilegedOperation, bool) {
	operation, ok := enums[name]
	return operation, ok
}

// Accepted returns a copy of the closed vocabulary for validation and tests.
func Accepted() map[string]struct{} {
	out := make(map[string]struct{}, len(enums))
	for name := range enums {
		out[name] = struct{}{}
	}
	return out
}
