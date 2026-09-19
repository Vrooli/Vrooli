package readiness

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/markedrefs"

	planmodel "plan-manager/internal/planmodel"
)

type Verdict string

const (
	VerdictUnspecified Verdict = ""
	VerdictPass        Verdict = "pass"
	VerdictFail        Verdict = "fail"
	VerdictUnknown     Verdict = "unknown"
)

type Severity string

const (
	SeverityFail    Severity = "fail"
	SeverityWarning Severity = "warning"
	SeverityUnknown Severity = "unknown"
)

const (
	SourceStructure         = "structure"
	SourceQuality           = "quality"
	SourceCommandReference  = "command_reference"
	SourceContextReference  = "context_reference"
	DependencyReady         = "ready"
	DependencyUnavailable   = "unavailable"
	DependencyNotApplicable = "not_applicable"
)

type Finding struct {
	Severity         Severity
	Code             string
	Location         string
	Message          string
	Source           string
	DependencyStatus string
	CommandText      string
	Level            string
	IssueCodes       []string
	Suggestions      []string
	Guidance         []string
}

type Result struct {
	Verdict  Verdict
	Findings []Finding
	Detail   string
}

type Mode struct {
	Structure         bool
	Quality           bool
	CommandReferences bool
	ContextReferences bool
}

func DeterministicMode() Mode {
	return Mode{Structure: true, Quality: true}
}

func PreflightMode() Mode {
	return Mode{Structure: true, Quality: true, CommandReferences: true, ContextReferences: true}
}

type CommandValidator interface {
	ValidateCommandReference(context.Context, CommandRequest) (CommandResult, error)
}

type CommandRequest struct {
	CommandText string
	Qualifiers  []string
}

type CommandResult struct {
	Verdict         string
	ValidationLevel string
	Issues          []CommandIssue
	Suggestions     []string
	Guidance        []string
}

type CommandIssue struct {
	Code    string
	Message string
}

type ReferenceResolver interface {
	Resolve(ctx context.Context, ref planmodel.Reference) (planmodel.Reference, error)
}

type Options struct {
	PhaseID           string
	Mode              Mode
	CommandValidator  CommandValidator
	ReferenceResolver ReferenceResolver
}

func Evaluate(ctx context.Context, p planmodel.Plan, opts Options) Result {
	mode := opts.Mode
	var findings []Finding
	var details []string
	if mode.Structure {
		got := evaluateRelevantContextStructure(p, opts.PhaseID)
		findings = append(findings, got.Findings...)
		details = append(details, got.Detail)
	}
	if mode.Quality {
		got := evaluatePlanQuality(p, opts.PhaseID)
		findings = append(findings, got.Findings...)
		details = append(details, got.Detail)
	}
	if mode.CommandReferences {
		got := evaluatePlanCommands(ctx, p, opts.PhaseID, opts.CommandValidator)
		findings = append(findings, got.Findings...)
		details = append(details, got.Detail)
	}
	if mode.ContextReferences {
		got := evaluateContextReferences(ctx, p, opts.PhaseID, opts.ReferenceResolver)
		findings = append(findings, got.Findings...)
		details = append(details, got.Detail)
	}
	return Result{
		Verdict:  VerdictForFindings(findings),
		Findings: findings,
		Detail:   JoinDetails(details...),
	}
}

func VerdictForFindings(findings []Finding) Verdict {
	verdict := VerdictPass
	for _, finding := range findings {
		switch finding.Severity {
		case SeverityFail:
			return VerdictFail
		case SeverityUnknown, SeverityWarning:
			verdict = CombineVerdicts(verdict, VerdictUnknown)
		}
	}
	return verdict
}

func CombineVerdicts(a, b Verdict) Verdict {
	if a == VerdictFail || b == VerdictFail {
		return VerdictFail
	}
	if a == VerdictUnknown || b == VerdictUnknown || a == VerdictUnspecified || b == VerdictUnspecified {
		return VerdictUnknown
	}
	return VerdictPass
}

func JoinDetails(parts ...string) string {
	var nonEmpty []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			nonEmpty = append(nonEmpty, part)
		}
	}
	return strings.Join(nonEmpty, "\n")
}

func evaluateRelevantContextStructure(p planmodel.Plan, phaseID string) Result {
	var findings []Finding
	if phaseID == "" {
		findings = append(findings, validateContextItems("plan.relevant_context", p.RelevantContext)...)
		for _, phase := range p.Phases {
			findings = append(findings, validatePhaseContextStructure(phase)...)
		}
	} else {
		found := false
		for _, phase := range p.Phases {
			if phase.ID != phaseID {
				continue
			}
			found = true
			findings = append(findings, validatePhaseContextStructure(phase)...)
			break
		}
		if !found {
			findings = append(findings, Finding{
				Severity: SeverityUnknown,
				Source:   SourceStructure,
				Message:  fmt.Sprintf("phase %q was not found for relevant context validation", phaseID),
				Location: "phase." + phaseID,
			})
		}
	}
	return Result{Verdict: VerdictForFindings(findings), Findings: findings, Detail: findingsDetail("relevant context structure validation:", findings)}
}

func validatePhaseContextStructure(phase planmodel.Phase) []Finding {
	location := "phase." + phase.ID
	if strings.TrimSpace(phase.ID) == "" {
		location = fmt.Sprintf("phase.%d", phase.Order)
	}
	findings := validateContextItems(location+".relevant_context", phase.RelevantContext)
	if !planmodel.HasPhaseContextOrNoContextReason(phase) {
		findings = append(findings, structureFinding(location+".relevant_context", "missing_phase_context", "phase has no relevant context and no explicit NO_CONTEXT reason", "Add phase relevant context or an operator note starting with NO_CONTEXT: when no setup is useful."))
	}
	return findings
}

func validateContextItems(location string, items []planmodel.RelevantContextItem) []Finding {
	var findings []Finding
	for i, item := range items {
		itemLocation := fmt.Sprintf("%s[%d]", location, i)
		if item.Required && item.RepeatPolicy == "" {
			findings = append(findings, structureFinding(itemLocation, "missing_repeat_policy", "required context item has no repeat policy", ""))
		}
		if item.Required && !contextItemHasPayload(item) {
			findings = append(findings, structureFinding(itemLocation, "missing_context_payload", "required context item has no command, argv, target, instruction, or note payload", ""))
		}
		if item.Kind == planmodel.RelevantContextCommand || item.Kind == planmodel.RelevantContextSearch {
			if strings.TrimSpace(item.Reason) == "" {
				findings = append(findings, structureFinding(itemLocation+".reason", "missing_context_reason", "command/search context item has no reason", ""))
			}
			if strings.TrimSpace(item.Instruction) == "" {
				findings = append(findings, structureFinding(itemLocation+".instruction", "missing_context_instruction", "command/search context item has no instruction", ""))
			}
			if strings.TrimSpace(item.Command) == "" && len(item.Argv) == 0 && strings.TrimSpace(item.Target) == "" {
				findings = append(findings, structureFinding(itemLocation+".command", "missing_context_command", "command/search context item has no runnable command, argv, or target", ""))
			}
		}
	}
	return findings
}

func contextItemHasPayload(item planmodel.RelevantContextItem) bool {
	if strings.TrimSpace(item.Command) != "" || len(item.Argv) > 0 || strings.TrimSpace(item.Target) != "" {
		return true
	}
	return item.Kind == planmodel.RelevantContextNote && strings.TrimSpace(firstNonEmpty(item.Instruction, item.Label, item.Reason)) != ""
}

func structureFinding(location, code, message, guidance string) Finding {
	out := Finding{
		Severity:         SeverityFail,
		Source:           SourceStructure,
		DependencyStatus: DependencyNotApplicable,
		Code:             code,
		Location:         location,
		Message:          message,
		IssueCodes:       []string{code},
	}
	if strings.TrimSpace(guidance) != "" {
		out.Guidance = []string{guidance}
	}
	return out
}

func evaluatePlanQuality(p planmodel.Plan, phaseID string) Result {
	report := planmodel.AssessPlanQuality(p, phaseID)
	findings := make([]Finding, 0, len(report.Findings))
	for _, finding := range report.Findings {
		findings = append(findings, fromQualityFinding(finding))
	}
	return Result{Verdict: VerdictForFindings(findings), Findings: findings, Detail: findingsDetail("plan quality validation:", findings)}
}

func fromQualityFinding(finding planmodel.QualityFinding) Finding {
	severity := SeverityFail
	if finding.Severity == planmodel.QualitySeverityWarning {
		severity = SeverityWarning
	}
	guidance := []string(nil)
	if strings.TrimSpace(finding.Guidance) != "" {
		guidance = []string{finding.Guidance}
	}
	return Finding{
		Severity:         severity,
		Source:           SourceQuality,
		DependencyStatus: DependencyNotApplicable,
		Code:             finding.Code,
		Location:         finding.Location,
		Message:          finding.Message,
		IssueCodes:       []string{finding.Code},
		Guidance:         guidance,
	}
}

func evaluatePlanCommands(ctx context.Context, p planmodel.Plan, phaseID string, validator CommandValidator) Result {
	refs, err := commandRefsForScope(p, phaseID)
	if err != nil {
		findings := []Finding{{Severity: SeverityUnknown, Source: SourceCommandReference, DependencyStatus: DependencyUnavailable, Message: err.Error()}}
		return Result{Verdict: VerdictUnknown, Findings: findings, Detail: "command reference validation unknown: " + err.Error()}
	}
	if len(refs) == 0 {
		return Result{Verdict: VerdictPass}
	}
	if validator == nil {
		findings := []Finding{{Severity: SeverityUnknown, Source: SourceCommandReference, DependencyStatus: DependencyUnavailable, Code: "cli_health_unavailable", Message: "CLI Health command validator unavailable", IssueCodes: []string{"cli_health_unavailable"}}}
		return Result{Verdict: VerdictUnknown, Findings: findings, Detail: "command reference validation unknown: CLI Health command validator unavailable"}
	}
	var findings []Finding
	for _, ref := range refs {
		if !markedrefs.RequiresExistence(ref.ref) {
			continue
		}
		result, err := validator.ValidateCommandReference(ctx, CommandRequest{
			CommandText: ref.ref.Value,
			Qualifiers:  append([]string(nil), ref.ref.Qualifiers...),
		})
		if err != nil {
			findings = append(findings, Finding{
				Severity:         SeverityUnknown,
				Source:           SourceCommandReference,
				DependencyStatus: DependencyUnavailable,
				Code:             "cli_health_unavailable",
				CommandText:      ref.ref.Value,
				Message:          "CLI Health unavailable: " + err.Error(),
				Location:         ref.location,
				IssueCodes:       []string{"cli_health_unavailable"},
			})
			continue
		}
		issueCodes := commandIssueCodes(result.Issues)
		finding := Finding{
			Severity:         commandSeverity(result.Verdict),
			Source:           SourceCommandReference,
			DependencyStatus: DependencyReady,
			Code:             firstNonEmpty(issueCodes...),
			CommandText:      ref.ref.Value,
			Level:            result.ValidationLevel,
			Message:          commandResultMessage(result),
			Location:         ref.location,
			IssueCodes:       issueCodes,
			Suggestions:      append([]string(nil), result.Suggestions...),
			Guidance:         append([]string(nil), result.Guidance...),
		}
		if finding.Severity != "" {
			findings = append(findings, finding)
		}
	}
	return Result{Verdict: VerdictForFindings(findings), Findings: findings, Detail: commandFindingsDetail(findings)}
}

func commandSeverity(verdict string) Severity {
	switch strings.ToLower(verdict) {
	case "valid", "skipped":
		return ""
	case "invalid", "unsupported":
		return SeverityFail
	default:
		return SeverityUnknown
	}
}

type scopedCommandRef struct {
	ref      markedrefs.Reference
	location string
}

func commandRefsForScope(p planmodel.Plan, phaseID string) ([]scopedCommandRef, error) {
	var out []scopedCommandRef
	if phaseID == "" {
		addCommandRefs(&out, "plan.purpose", p.Purpose)
		addCommandRefs(&out, "plan.scope", p.Scope)
		addCommandRefs(&out, "plan.constraints", p.Constraints)
		addCommandRefs(&out, "plan.definition_of_done", p.DefinitionOfDone)
		addContextCommandRefs(&out, "plan.relevant_context", p.RelevantContext)
		for _, phase := range p.Phases {
			addCommandRefs(&out, "phase."+phase.ID+".intent", phase.Intent)
			for i, item := range phase.RequiredReading {
				addCommandRefs(&out, fmt.Sprintf("phase.%s.required_reading[%d]", phase.ID, i), item)
			}
			for i, item := range phase.Reminders {
				addCommandRefs(&out, fmt.Sprintf("phase.%s.reminders[%d]", phase.ID, i), item)
			}
			addCommandRefs(&out, "phase."+phase.ID+".acceptance", phase.Acceptance)
			addContextCommandRefs(&out, "phase."+phase.ID+".relevant_context", phase.RelevantContext)
		}
		return out, nil
	}
	for _, phase := range p.Phases {
		if phase.ID != phaseID {
			continue
		}
		addCommandRefs(&out, "phase."+phase.ID+".intent", phase.Intent)
		for i, item := range phase.RequiredReading {
			addCommandRefs(&out, fmt.Sprintf("phase.%s.required_reading[%d]", phase.ID, i), item)
		}
		for i, item := range phase.Reminders {
			addCommandRefs(&out, fmt.Sprintf("phase.%s.reminders[%d]", phase.ID, i), item)
		}
		addCommandRefs(&out, "phase."+phase.ID+".acceptance", phase.Acceptance)
		addContextCommandRefs(&out, "phase."+phase.ID+".relevant_context", phase.RelevantContext)
		return out, nil
	}
	return nil, fmt.Errorf("phase %q not found", phaseID)
}

func addCommandRefs(out *[]scopedCommandRef, location, text string) {
	for lineNumber, line := range strings.Split(text, "\n") {
		for _, ref := range markedrefs.ParseInlineCode(line, lineNumber+1) {
			if ref.Marker == markedrefs.MarkerCLI {
				*out = append(*out, scopedCommandRef{ref: ref, location: location})
			}
		}
	}
}

func addContextCommandRefs(out *[]scopedCommandRef, location string, items []planmodel.RelevantContextItem) {
	for i, item := range items {
		itemLocation := fmt.Sprintf("%s[%d]", location, i)
		addCommandRefs(out, itemLocation+".instruction", item.Instruction)
		addCommandRefs(out, itemLocation+".reason", item.Reason)
		addCommandRefs(out, itemLocation+".command", item.Command)
		if item.Kind != planmodel.RelevantContextCommand && item.Kind != planmodel.RelevantContextSearch {
			continue
		}
		command := strings.TrimSpace(item.Command)
		if command == "" && len(item.Argv) > 0 {
			command = strings.Join(item.Argv, " ")
		}
		if command == "" {
			continue
		}
		*out = append(*out, scopedCommandRef{
			ref: markedrefs.Reference{
				Marker: markedrefs.MarkerCLI,
				Value:  command,
			},
			location: itemLocation + ".command",
		})
	}
}

func commandResultMessage(result CommandResult) string {
	var parts []string
	for _, issue := range result.Issues {
		if issue.Code != "" && issue.Message != "" {
			parts = append(parts, issue.Code+": "+issue.Message)
		} else if issue.Message != "" {
			parts = append(parts, issue.Message)
		}
	}
	for _, suggestion := range result.Suggestions {
		if suggestion != "" {
			parts = append(parts, "suggestion: "+suggestion)
		}
	}
	parts = append(parts, result.Guidance...)
	if len(parts) == 0 {
		return strings.TrimSpace(result.Verdict + " " + result.ValidationLevel)
	}
	return strings.Join(parts, "; ")
}

func evaluateContextReferences(ctx context.Context, p planmodel.Plan, phaseID string, resolver ReferenceResolver) Result {
	refs, err := contextReferencesForScope(p, phaseID)
	if err != nil {
		findings := []Finding{{Severity: SeverityUnknown, Source: SourceContextReference, DependencyStatus: DependencyUnavailable, Message: err.Error()}}
		return Result{Verdict: VerdictUnknown, Findings: findings, Detail: "relevant context reference validation unknown: " + err.Error()}
	}
	if len(refs) == 0 {
		return Result{Verdict: VerdictPass}
	}
	if resolver == nil {
		findings := []Finding{{
			Severity:         SeverityUnknown,
			Source:           SourceContextReference,
			DependencyStatus: DependencyUnavailable,
			Code:             "context_reference_resolver_unavailable",
			Message:          "reference resolver unavailable",
			Location:         "relevant_context",
			IssueCodes:       []string{"context_reference_resolver_unavailable"},
		}}
		return Result{Verdict: VerdictUnknown, Findings: findings, Detail: "relevant context reference validation unknown: reference resolver unavailable"}
	}
	var findings []Finding
	for _, ref := range refs {
		resolved, err := resolver.Resolve(ctx, ref.ref)
		if err != nil {
			findings = append(findings, Finding{
				Severity:         SeverityUnknown,
				Source:           SourceContextReference,
				DependencyStatus: DependencyUnavailable,
				Code:             "context_reference_resolver_error",
				CommandText:      ref.ref.Target,
				Message:          "reference resolver unavailable: " + err.Error(),
				Location:         ref.location,
				IssueCodes:       []string{"context_reference_resolver_error"},
			})
			continue
		}
		switch resolved.Resolution {
		case planmodel.ResolutionResolved, planmodel.ResolutionFuture:
			continue
		case planmodel.ResolutionMissing, planmodel.ResolutionUnresolved:
			findings = append(findings, Finding{
				Severity:         SeverityFail,
				Source:           SourceContextReference,
				DependencyStatus: DependencyReady,
				Code:             "context_reference_unresolved",
				CommandText:      ref.ref.Target,
				Message:          contextReferenceMessage(resolved),
				Location:         ref.location,
				IssueCodes:       []string{"context_reference_unresolved"},
			})
		default:
			findings = append(findings, Finding{
				Severity:         SeverityUnknown,
				Source:           SourceContextReference,
				DependencyStatus: DependencyReady,
				Code:             "context_reference_unknown",
				CommandText:      ref.ref.Target,
				Message:          contextReferenceMessage(resolved),
				Location:         ref.location,
				IssueCodes:       []string{"context_reference_unknown"},
			})
		}
	}
	return Result{Verdict: VerdictForFindings(findings), Findings: findings, Detail: contextReferenceFindingsDetail(findings)}
}

type scopedContextReference struct {
	ref      planmodel.Reference
	location string
}

func contextReferencesForScope(p planmodel.Plan, phaseID string) ([]scopedContextReference, error) {
	var out []scopedContextReference
	if phaseID == "" {
		addContextReferences(&out, "plan.relevant_context", p.RelevantContext)
		for _, phase := range p.Phases {
			addContextReferences(&out, "phase."+phase.ID+".relevant_context", phase.RelevantContext)
		}
		return out, nil
	}
	for _, phase := range p.Phases {
		if phase.ID != phaseID {
			continue
		}
		addContextReferences(&out, "phase."+phase.ID+".relevant_context", phase.RelevantContext)
		return out, nil
	}
	return nil, fmt.Errorf("phase %q not found", phaseID)
}

func addContextReferences(out *[]scopedContextReference, location string, items []planmodel.RelevantContextItem) {
	for i, item := range items {
		kind, ok := contextReferenceKind(item.Kind)
		if !ok {
			continue
		}
		target := strings.TrimSpace(item.Target)
		if target == "" {
			continue
		}
		*out = append(*out, scopedContextReference{
			ref: planmodel.Reference{
				ID:     item.ID,
				Kind:   kind,
				Target: target,
			},
			location: fmt.Sprintf("%s[%d].target", location, i),
		})
	}
}

func contextReferenceKind(kind planmodel.RelevantContextKind) (planmodel.ReferenceKind, bool) {
	switch kind {
	case planmodel.RelevantContextCodeRef:
		return planmodel.ReferenceCode, true
	case planmodel.RelevantContextDoc:
		return planmodel.ReferenceDoc, true
	case planmodel.RelevantContextReqRef:
		return planmodel.ReferenceReq, true
	default:
		return "", false
	}
}

func contextReferenceMessage(ref planmodel.Reference) string {
	if strings.TrimSpace(ref.Note) != "" {
		return ref.Note
	}
	if ref.Resolution != "" {
		return "relevant context reference " + string(ref.Resolution)
	}
	return "relevant context reference resolution unknown"
}

func commandIssueCodes(issues []CommandIssue) []string {
	var out []string
	for _, issue := range issues {
		if strings.TrimSpace(issue.Code) != "" {
			out = append(out, issue.Code)
		}
	}
	return out
}

func commandFindingsDetail(findings []Finding) string {
	if len(findings) == 0 {
		return ""
	}
	lines := []string{"command reference validation:"}
	for _, finding := range findings {
		lines = append(lines, fmt.Sprintf("%s %s (%s): %s", string(VerdictForFindings([]Finding{finding})), finding.CommandText, finding.Location, finding.Message))
	}
	return strings.Join(lines, "\n")
}

func contextReferenceFindingsDetail(findings []Finding) string {
	if len(findings) == 0 {
		return ""
	}
	lines := []string{"relevant context reference validation:"}
	for _, finding := range findings {
		lines = append(lines, fmt.Sprintf("%s (%s): %s", string(VerdictForFindings([]Finding{finding})), finding.Location, finding.Message))
	}
	return strings.Join(lines, "\n")
}

func findingsDetail(title string, findings []Finding) string {
	if len(findings) == 0 {
		return ""
	}
	lines := []string{title}
	for _, finding := range findings {
		lines = append(lines, fmt.Sprintf("%s (%s): %s", string(VerdictForFindings([]Finding{finding})), finding.Location, finding.Message))
	}
	return strings.Join(lines, "\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
