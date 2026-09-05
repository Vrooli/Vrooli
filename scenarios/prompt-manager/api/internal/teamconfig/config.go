package teamconfig

import (
	"fmt"
	"strings"
)

const (
	RuntimeModeMultiProcess  = "multi-process"
	RuntimeModeSingleProcess = "single-process"

	CoordinationPatternIndependent = "independent"
	CoordinationPatternPeer        = "peer"
	CoordinationPatternLeaderLed   = "leader-led"

	ReportingModeNone     = "none"
	ReportingModeLeader   = "leader"
	ReportingModeOrgChart = "org-chart"

	MessagingModeDisabled   = "disabled"
	MessagingModeAsyncInbox = "async-inbox"
	MessagingModeInSession  = "in-session"

	QueuePolicySerialized      = "serialized"
	QueuePolicyBoundedParallel = "bounded-parallel"
)

type Runtime struct {
	Mode string `json:"mode"`
}

type Capabilities struct {
	ShowOrgContext           bool `json:"showOrgContext"`
	InjectInbox              bool `json:"injectInbox"`
	AllowPeerTriggers        bool `json:"allowPeerTriggers"`
	ShowTaskBoardGuidance    bool `json:"showTaskBoardGuidance"`
	ShowKnowledgeLogGuidance bool `json:"showKnowledgeLogGuidance"`
	RequireHandoff           bool `json:"requireHandoff"`
}

type Coordination struct {
	Pattern       string       `json:"pattern"`
	LeadAgentID   string       `json:"leadAgentId,omitempty"`
	ReportingMode string       `json:"reportingMode"`
	MessagingMode string       `json:"messagingMode"`
	Capabilities  Capabilities `json:"capabilities"`
}

type Execution struct {
	QueuePolicy       string `json:"queuePolicy"`
	MaxConcurrentRuns int    `json:"maxConcurrentRuns"`
}

type Contract struct {
	Runtime      Runtime
	Coordination Coordination
	Execution    Execution
}

type ValidationError struct {
	Message string
}

// ValidationFinding is one independently actionable contract defect. Reads
// expose the complete set; writes still reject whenever this set is nonempty.
type ValidationFinding struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}

func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

func validationError(message string) error {
	return &ValidationError{Message: message}
}

func DefaultIndependentCapabilities() Capabilities {
	return Capabilities{
		ShowOrgContext:           false,
		InjectInbox:              false,
		AllowPeerTriggers:        false,
		ShowTaskBoardGuidance:    true,
		ShowKnowledgeLogGuidance: true,
		RequireHandoff:           true,
	}
}

func DefaultPeerCapabilities() Capabilities {
	return Capabilities{
		ShowOrgContext:           true,
		InjectInbox:              true,
		AllowPeerTriggers:        true,
		ShowTaskBoardGuidance:    true,
		ShowKnowledgeLogGuidance: true,
		RequireHandoff:           true,
	}
}

func DefaultLeaderLedCapabilities(runtimeMode string) Capabilities {
	return Capabilities{
		ShowOrgContext:           true,
		InjectInbox:              runtimeMode == RuntimeModeMultiProcess,
		AllowPeerTriggers:        false,
		ShowTaskBoardGuidance:    true,
		ShowKnowledgeLogGuidance: true,
		RequireHandoff:           true,
	}
}

func BuildCoordinationPreset(pattern, runtimeMode, leadAgentID string) (Coordination, error) {
	switch strings.TrimSpace(pattern) {
	case CoordinationPatternIndependent:
		return Coordination{
			Pattern:       CoordinationPatternIndependent,
			ReportingMode: ReportingModeNone,
			MessagingMode: MessagingModeDisabled,
			Capabilities:  DefaultIndependentCapabilities(),
		}, nil
	case CoordinationPatternPeer:
		return Coordination{
			Pattern:       CoordinationPatternPeer,
			ReportingMode: ReportingModeOrgChart,
			MessagingMode: MessagingModeAsyncInbox,
			Capabilities:  DefaultPeerCapabilities(),
		}, nil
	case CoordinationPatternLeaderLed:
		leadAgentID = strings.TrimSpace(leadAgentID)
		if leadAgentID == "" {
			return Coordination{}, fmt.Errorf("leader-led teams require leadAgentId")
		}

		messagingMode := MessagingModeAsyncInbox
		if runtimeMode == RuntimeModeSingleProcess {
			messagingMode = MessagingModeInSession
		}

		return Coordination{
			Pattern:       CoordinationPatternLeaderLed,
			LeadAgentID:   leadAgentID,
			ReportingMode: ReportingModeLeader,
			MessagingMode: messagingMode,
			Capabilities:  DefaultLeaderLedCapabilities(runtimeMode),
		}, nil
	default:
		return Coordination{}, fmt.Errorf("unsupported coordination pattern %q", pattern)
	}
}

func BuildExecutionConfig(runtimeMode, queuePolicy string, maxConcurrentRuns int) (Execution, error) {
	if runtimeMode == RuntimeModeSingleProcess {
		return Execution{
			QueuePolicy:       QueuePolicySerialized,
			MaxConcurrentRuns: 1,
		}, nil
	}

	switch strings.TrimSpace(queuePolicy) {
	case "", QueuePolicyBoundedParallel:
		if maxConcurrentRuns < 2 {
			maxConcurrentRuns = 2
		}
		return Execution{
			QueuePolicy:       QueuePolicyBoundedParallel,
			MaxConcurrentRuns: maxConcurrentRuns,
		}, nil
	case QueuePolicySerialized:
		return Execution{
			QueuePolicy:       QueuePolicySerialized,
			MaxConcurrentRuns: 1,
		}, nil
	default:
		return Execution{}, fmt.Errorf("unsupported queue policy %q", queuePolicy)
	}
}

func Validate(contract Contract) error {
	findings := ValidateFindings(contract)
	if len(findings) == 0 {
		return nil
	}
	return validationError(findings[0].Message)
}

// ValidateFindings reports every invalid runtime/coordination combination in
// one pass. It deliberately does not short-circuit: an operator repairing a
// malformed team.json needs the whole actionable set, not a retry loop.
func ValidateFindings(contract Contract) []ValidationFinding {
	var findings []ValidationFinding
	add := func(field, message string) {
		findings = append(findings, ValidationFinding{Field: field, Message: message})
	}
	switch contract.Runtime.Mode {
	case RuntimeModeMultiProcess, RuntimeModeSingleProcess:
	default:
		add("runtime.mode", "runtime.mode must be 'multi-process' or 'single-process'")
	}

	switch contract.Coordination.Pattern {
	case CoordinationPatternIndependent, CoordinationPatternPeer, CoordinationPatternLeaderLed:
	default:
		add("coordination.pattern", "coordination.pattern must be 'independent', 'peer', or 'leader-led'")
	}

	switch contract.Coordination.ReportingMode {
	case ReportingModeNone, ReportingModeLeader, ReportingModeOrgChart:
	default:
		add("coordination.reportingMode", "coordination.reportingMode must be 'none', 'leader', or 'org-chart'")
	}

	switch contract.Coordination.MessagingMode {
	case MessagingModeDisabled, MessagingModeAsyncInbox, MessagingModeInSession:
	default:
		add("coordination.messagingMode", "coordination.messagingMode must be 'disabled', 'async-inbox', or 'in-session'")
	}

	switch contract.Execution.QueuePolicy {
	case QueuePolicySerialized, QueuePolicyBoundedParallel:
	default:
		add("execution.queuePolicy", "execution.queuePolicy must be 'serialized' or 'bounded-parallel'")
	}

	if contract.Execution.MaxConcurrentRuns < 1 {
		add("execution.maxConcurrentRuns", "execution.maxConcurrentRuns must be at least 1")
	}

	if contract.Execution.QueuePolicy == QueuePolicySerialized && contract.Execution.MaxConcurrentRuns != 1 {
		add("execution.maxConcurrentRuns", "serialized execution requires maxConcurrentRuns to equal 1")
	}

	if contract.Execution.QueuePolicy == QueuePolicyBoundedParallel && contract.Execution.MaxConcurrentRuns < 2 {
		add("execution.maxConcurrentRuns", "bounded-parallel execution requires maxConcurrentRuns to be at least 2")
	}

	switch contract.Coordination.Pattern {
	case CoordinationPatternLeaderLed:
		if contract.Coordination.LeadAgentID == "" {
			add("coordination.leadAgentId", "coordination.leadAgentId is required for leader-led teams")
		}
		if contract.Coordination.ReportingMode == ReportingModeNone {
			add("coordination.reportingMode", "leader-led teams must use reportingMode 'leader' or 'org-chart'")
		}
	case CoordinationPatternIndependent, CoordinationPatternPeer:
		if contract.Coordination.LeadAgentID != "" {
			add("coordination.leadAgentId", fmt.Sprintf("coordination.leadAgentId is only allowed for %q teams", CoordinationPatternLeaderLed))
		}
	}

	if contract.Coordination.Pattern == CoordinationPatternIndependent && contract.Coordination.ReportingMode != ReportingModeNone {
		add("coordination.reportingMode", "independent teams must use reportingMode 'none'")
	}

	if contract.Coordination.Pattern == CoordinationPatternPeer && contract.Coordination.ReportingMode == ReportingModeLeader {
		add("coordination.reportingMode", "peer teams cannot use reportingMode 'leader'")
	}

	if contract.Coordination.MessagingMode == MessagingModeInSession && contract.Runtime.Mode != RuntimeModeSingleProcess {
		add("coordination.messagingMode", "in-session messaging is only supported for single-process runtime mode")
	}

	if contract.Coordination.MessagingMode == MessagingModeDisabled && contract.Coordination.Capabilities.InjectInbox {
		add("coordination.capabilities.injectInbox", "injectInbox requires async-inbox messaging")
	}

	if contract.Coordination.Capabilities.InjectInbox && contract.Coordination.MessagingMode != MessagingModeAsyncInbox {
		add("coordination.capabilities.injectInbox", "injectInbox requires messagingMode 'async-inbox'")
	}

	if contract.Coordination.Capabilities.AllowPeerTriggers && contract.Runtime.Mode != RuntimeModeMultiProcess {
		add("coordination.capabilities.allowPeerTriggers", "allowPeerTriggers is only supported for multi-process teams")
	}

	if contract.Runtime.Mode == RuntimeModeSingleProcess {
		if contract.Coordination.Pattern != CoordinationPatternLeaderLed {
			add("coordination.pattern", "single-process runtime mode requires coordination.pattern 'leader-led'")
		}
		if contract.Coordination.MessagingMode != MessagingModeInSession {
			add("coordination.messagingMode", "single-process teams must use messagingMode 'in-session'")
		}
		if contract.Execution.QueuePolicy != QueuePolicySerialized || contract.Execution.MaxConcurrentRuns != 1 {
			add("execution", "single-process teams must use serialized execution with maxConcurrentRuns 1")
		}
		if contract.Coordination.Capabilities.InjectInbox {
			add("coordination.capabilities.injectInbox", "single-process teams cannot inject async inbox messages")
		}
		if contract.Coordination.Capabilities.AllowPeerTriggers {
			add("coordination.capabilities.allowPeerTriggers", "single-process teams cannot allow peer triggers")
		}
	}

	return findings
}

func CoordinationSkillID(contract Contract) string {
	switch contract.Coordination.Pattern {
	case CoordinationPatternLeaderLed:
		return "team-coordination-leader-led"
	case CoordinationPatternPeer:
		return "team-coordination-peer"
	default:
		return "team-coordination-independent"
	}
}

func UsesSingleProcessInterop(contract Contract) bool {
	return contract.Runtime.Mode == RuntimeModeSingleProcess && contract.Coordination.Pattern == CoordinationPatternLeaderLed
}

func TeamTriggerTargetsLead(contract Contract) bool {
	return UsesSingleProcessInterop(contract)
}

func UsesClaudeCodeRunner(contract Contract) bool {
	return contract.Runtime.Mode == RuntimeModeSingleProcess
}

func MessagingUsesAsyncInbox(contract Contract) bool {
	return contract.Coordination.MessagingMode == MessagingModeAsyncInbox
}

func MessagingUsesInSession(contract Contract) bool {
	return contract.Coordination.MessagingMode == MessagingModeInSession
}

func MessagingEnabled(contract Contract) bool {
	return contract.Coordination.MessagingMode != MessagingModeDisabled
}

func ShouldInjectInbox(contract Contract) bool {
	return contract.Coordination.Capabilities.InjectInbox && MessagingUsesAsyncInbox(contract)
}

func ShouldShowOrgContext(contract Contract) bool {
	return contract.Coordination.Capabilities.ShowOrgContext
}

func ShouldShowTaskBoardGuidance(contract Contract) bool {
	return contract.Coordination.Capabilities.ShowTaskBoardGuidance
}

func ShouldShowKnowledgeLogGuidance(contract Contract) bool {
	return contract.Coordination.Capabilities.ShowKnowledgeLogGuidance
}

func RequiresHandoff(contract Contract) bool {
	return contract.Coordination.Capabilities.RequireHandoff
}

func AllowsPeerTriggers(contract Contract) bool {
	return contract.Coordination.Capabilities.AllowPeerTriggers
}
