// Package guardmodes persists audited, expiring per-check log-only modes.
// It is shared by the policy runtime and the root control-plane CLI.
package guardmodes

import "errors"

// These aliases keep the store independent from agentharness while preserving
// a stable string-based on-disk and CLI contract.
type CheckID = string
type Action = string

type HardStop string

const (
	RiskInspection        CheckID = "inspection"
	RiskFrozenReproduce   CheckID = "frozen_lockfile_reproduction"
	RiskDependencyAdd     CheckID = "dependency_addition"
	RiskDependencyUpgrade CheckID = "dependency_upgrade"
	RiskDependencyRemove  CheckID = "dependency_removal"
	RiskLifecycle         CheckID = "lifecycle_execution"
	RiskPublish           CheckID = "publish"
	RiskOpaque            CheckID = "opaque"
	RiskUnknown           CheckID = "unknown"

	ActionAllow Action = "allow"

	HardStopMoney          HardStop = "money"
	HardStopOutsideMessage HardStop = "outside_message"
	HardStopSecret         HardStop = "secret"
	HardStopOutsideDelete  HardStop = "outside_repository_deletion"
	HardStopUnknownEffect  HardStop = "unknown_protected_effect"
)

var ErrStoreBusy = errors.New("guard mode store is busy")
