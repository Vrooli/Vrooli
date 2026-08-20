package incidents

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

type Store interface {
	UpsertIncident(ctx context.Context, input UpsertInput) (*Incident, error)
	ListIncidents(ctx context.Context, filters ListFilters) (*ListResponse, error)
	GetIncident(ctx context.Context, id string) (*Incident, error)
	ListIncidentObservations(ctx context.Context, incidentID string, limit int) ([]Observation, error)
	UpdateIncidentStatus(ctx context.Context, incidentID string, status Status, note string) (*Incident, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) UpsertFromCheckResult(ctx context.Context, result checks.Result) (*Incident, bool, error) {
	if result.Status == checks.StatusOK {
		return nil, false, s.resolveRecoveredForCheck(ctx, result)
	}
	if result.Status == checks.StatusNotApplicable {
		return nil, false, nil
	}
	rule, ok := classifyResult(result)
	if !ok {
		return nil, false, nil
	}
	observedAt := result.Timestamp
	if observedAt.IsZero() {
		observedAt = s.now()
	}
	input := UpsertInput{
		Fingerprint:     rule.fingerprint,
		Type:            rule.incidentType,
		Severity:        rule.severity,
		Title:           rule.title,
		Summary:         result.Message,
		ObservedAt:      observedAt.UTC(),
		BootID:          stringDetail(result.Details, "bootId"),
		SourceCheckID:   result.CheckID,
		Evidence:        boundedEvidence(result.Details),
		Recommendations: stringSliceDetail(result.Details, "recommendations"),
	}
	enrichInputFromResult(&input, result)
	if err := s.supersedePriorIncidents(ctx, result.CheckID, input.Fingerprint); err != nil {
		return nil, false, err
	}
	incident, err := s.store.UpsertIncident(ctx, input)
	if err != nil {
		return nil, true, err
	}
	return incident, true, nil
}

func (s *Service) supersedePriorIncidents(ctx context.Context, checkID, fingerprint string) error {
	if checkID == "" || fingerprint == "" {
		return nil
	}
	resp, err := s.store.ListIncidents(ctx, ListFilters{Status: StatusOpen, Limit: 200})
	if err != nil || resp == nil {
		return err
	}
	for _, incident := range resp.Incidents {
		if incident.Fingerprint == fingerprint || !incidentFromCheck(incident, checkID) {
			continue
		}
		if _, err := s.store.UpdateIncidentStatus(ctx, incident.ID, StatusResolved,
			fmt.Sprintf("superseded by newer evidence from %s (%s)", checkID, fingerprint)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) UpsertFromCheckResults(ctx context.Context, results []checks.Result) (int, error) {
	count := 0
	for _, result := range results {
		_, created, err := s.UpsertFromCheckResult(ctx, result)
		if err != nil {
			return count, err
		}
		if created {
			count++
		}
	}
	return count, nil
}

// OpenHealIncident records the durable operator-facing escalation once a
// recovery action has crossed the configured consecutive-failure threshold.
// The fingerprint is stable for a check/action pair, so later ticks coalesce
// into one incident while preserving the latest error in its evidence.
func (s *Service) OpenHealIncident(ctx context.Context, checkID, actionID, lastError string, consecutiveFailures int) error {
	observedAt := s.now().UTC()
	returnErr := error(nil)
	_, returnErr = s.store.UpsertIncident(ctx, UpsertInput{
		Fingerprint:   Fingerprint(string(TypeAutohealFailure), "heal", checkID, actionID),
		Type:          TypeAutohealFailure,
		Severity:      SeverityCritical,
		Title:         fmt.Sprintf("Auto-heal escalation: %s", checkID),
		Summary:       fmt.Sprintf("auto-heal action %s for check %s failed repeatedly: %s", actionID, checkID, lastError),
		ObservedAt:    observedAt,
		SourceCheckID: checkID,
		Evidence: map[string]any{
			"checkId":             checkID,
			"actionId":            actionID,
			"lastError":           lastError,
			"consecutiveFailures": consecutiveFailures,
		},
		Recommendations: []string{"Inspect the recorded auto-heal error and restore the check's dependency before retrying."},
	})
	return returnErr
}

// ResolveHealIncident closes the matching open escalation after a later
// successful heal. It deliberately only resolves incidents carrying the same
// action evidence, leaving unrelated findings for the check untouched.
func (s *Service) ResolveHealIncident(ctx context.Context, checkID, actionID string) error {
	resp, err := s.store.ListIncidents(ctx, ListFilters{Status: StatusOpen, Type: TypeAutohealFailure, Limit: 200})
	if err != nil || resp == nil {
		return err
	}
	for _, incident := range resp.Incidents {
		if !incidentFromCheck(incident, checkID) || stringDetail(incident.Evidence, "actionId") != actionID {
			continue
		}
		if _, err := s.store.UpdateIncidentStatus(ctx, incident.ID, StatusResolved,
			fmt.Sprintf("auto-resolved: %s action %s succeeded", checkID, actionID)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) resolveRecoveredForCheck(ctx context.Context, result checks.Result) error {
	if result.CheckID == "" {
		return nil
	}
	resp, err := s.store.ListIncidents(ctx, ListFilters{Status: StatusOpen, Limit: 200})
	if err != nil || resp == nil {
		return err
	}
	for _, incident := range resp.Incidents {
		if !incidentFromCheck(incident, result.CheckID) || !autoResolvableIncident(incident) {
			continue
		}
		_, err := s.store.UpdateIncidentStatus(ctx, incident.ID, StatusResolved,
			fmt.Sprintf("auto-resolved: %s reported OK", result.CheckID))
		if err != nil {
			return err
		}
	}
	return nil
}

type incidentRule struct {
	incidentType Type
	severity     Severity
	title        string
	fingerprint  string
}

func classifyResult(result checks.Result) (incidentRule, bool) {
	if result.Status == checks.StatusOK {
		return incidentRule{}, false
	}
	severity := SeverityWarning
	if result.Status == checks.StatusCritical {
		severity = SeverityCritical
	}
	if strings.HasPrefix(result.CheckID, "host-") {
		return incidentRule{
			incidentType: TypeHostIntegrity,
			severity:     severity,
			title:        hostFindingTitle(result),
			fingerprint:  Fingerprint(string(TypeHostIntegrity), result.CheckID, stableEvidenceDimension(result)),
		}, true
	}
	switch result.CheckID {
	case "infra-rdp":
		// Only credential and session faults become incidents. A daemon that is
		// merely stopped is already covered by the restart action and does not
		// need an operator-facing record.
		state := stringDetail(result.Details, "credentialState")
		if state != "empty" && state != "unreadable" && boolDetail(result.Details, "sessionAvailable") {
			return incidentRule{}, false
		}
		return incidentRule{
			incidentType: TypeHostIntegrity,
			severity:     severity,
			title:        "Remote desktop is not serviceable",
			fingerprint: Fingerprint(string(TypeHostIntegrity), result.CheckID, state,
				stringDetail(result.Details, "credentialModel")),
		}, true
	case "system-boot-history":
		return incidentRule{
			incidentType: TypeUncleanBoot,
			severity:     severity,
			title:        "Unclean boot history detected",
			fingerprint:  Fingerprint(string(TypeUncleanBoot), result.CheckID, stringDetail(result.Details, "latestUncleanBootId")),
		}, true
	case "system-pstore-evidence":
		if boolDetail(result.Details, "coverageGap") {
			return incidentRule{
				incidentType: TypeAutohealFailure,
				severity:     SeverityWarning,
				title:        "Crash artifact coverage gap detected",
				fingerprint:  Fingerprint(string(TypeAutohealFailure), result.CheckID, stringDetail(result.Details, "coverageGapReason")),
			}, true
		}
		if intDetail(result.Details, "pmsgCount") > 0 && intDetail(result.Details, "dmesgCount") == 0 && intDetail(result.Details, "consoleCount") == 0 {
			return incidentRule{incidentType: TypeHostIntegrity, severity: SeverityWarning, title: "Userspace pstore artifacts detected", fingerprint: Fingerprint(string(TypeHostIntegrity), result.CheckID, "pmsg")}, true
		}
		return incidentRule{incidentType: TypeUncleanBoot, severity: severity, title: "Kernel crash artifacts detected", fingerprint: Fingerprint(string(TypeUncleanBoot), result.CheckID)}, true
	case "system-panic-evidence":
		if boolDetail(result.Details, "coverageGap") {
			return incidentRule{
				incidentType: TypeAutohealFailure,
				severity:     SeverityWarning,
				title:        "Kernel panic evidence coverage gap detected",
				fingerprint:  Fingerprint(string(TypeAutohealFailure), result.CheckID, stringDetail(result.Details, "coverageGapReason")),
			}, true
		}
		// Identify the panic by what faulted, not by when. A recurring driver
		// bug is one incident whose evidence keeps updating; a genuinely
		// different panic gets its own record. Fingerprinting on the timestamp
		// instead would turn one defect into an unbounded stream of incidents.
		return incidentRule{
			incidentType: TypeUncleanBoot,
			severity:     severity,
			title:        "Kernel panic captured by kdump",
			fingerprint: Fingerprint(string(TypeUncleanBoot), result.CheckID,
				stringDetail(result.Details, "latestReason"), stringDetail(result.Details, "latestComm")),
		}, true
	case "system-mce-recent":
		if boolDetail(result.Details, "coverageGap") {
			return incidentRule{
				incidentType: TypeAutohealFailure,
				severity:     SeverityWarning,
				title:        "MCE telemetry coverage gap detected",
				fingerprint:  Fingerprint(string(TypeAutohealFailure), result.CheckID, stringDetail(result.Details, "coverageGapReason")),
			}, true
		}
		return incidentRule{incidentType: TypeHostIntegrity, severity: severity, title: "Recent machine-check evidence detected", fingerprint: Fingerprint(string(TypeHostIntegrity), result.CheckID)}, true
	default:
		incidentType := TypeAutohealFailure
		title := "Autoheal check failed: " + result.CheckID
		if strings.HasPrefix(result.CheckID, "scenario-") {
			incidentType = TypeScenarioFailure
			title = "Scenario failure: " + strings.TrimPrefix(result.CheckID, "scenario-")
		} else if strings.HasPrefix(result.CheckID, "resource-") {
			incidentType = TypeResourceFailure
			title = "Resource failure: " + strings.TrimPrefix(result.CheckID, "resource-")
		}
		return incidentRule{
			incidentType: incidentType,
			severity:     severity,
			title:        title,
			fingerprint:  Fingerprint(string(incidentType), result.CheckID, stableEvidenceDimension(result)),
		}, true
	}
}

func hostFindingTitle(result checks.Result) string {
	if message := strings.TrimSpace(result.Message); message != "" {
		return message
	}
	return "Host finding: " + result.CheckID
}

func stableEvidenceDimension(result checks.Result) string {
	for _, key := range []string{
		"evidenceKind", "findingKey", "coverageGapReason", "latestUncleanBootId",
		"statusReason", "packageManager", "serviceStatus", "scenarioStatus", "statusText",
	} {
		if value := stringDetail(result.Details, key); value != "" {
			return key + "=" + value
		}
	}
	if kind := firstEvidenceKindName(result.Details); kind != "" {
		return "evidence=" + kind
	}
	return "message=" + strings.TrimSpace(result.Message)
}

func firstEvidenceKindName(details map[string]any) string {
	items := evidenceItemsForResult(checks.Result{Details: details}, SeverityWarning)
	if len(items) == 0 {
		return ""
	}
	return items[0].Kind
}

func incidentFromCheck(incident Incident, checkID string) bool {
	for _, existing := range incident.SourceCheckIDs {
		if existing == checkID {
			return true
		}
	}
	return false
}

func autoResolvableIncident(incident Incident) bool {
	if incident.Status != StatusOpen {
		return false
	}
	switch incident.Type {
	case TypeHostIntegrity, TypeAutohealFailure:
		return true
	default:
		return false
	}
}

func boundedEvidence(details map[string]any) map[string]any {
	if details == nil {
		return nil
	}
	out := map[string]any{}
	for key, value := range details {
		if key == "recommendations" {
			continue
		}
		out[key] = value
	}
	return out
}

func stringDetail(details map[string]any, key string) string {
	if details == nil {
		return ""
	}
	if value, ok := details[key].(string); ok {
		return value
	}
	return ""
}

func stringSliceDetail(details map[string]any, key string) []string {
	if details == nil {
		return nil
	}
	raw, ok := details[key]
	if !ok {
		return nil
	}
	switch values := raw.(type) {
	case []string:
		return values
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func boolDetail(details map[string]any, key string) bool {
	if details == nil {
		return false
	}
	value, _ := details[key].(bool)
	return value
}

func intDetail(details map[string]any, key string) int {
	if details == nil {
		return 0
	}
	switch value := details[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func enrichInputFromResult(input *UpsertInput, result checks.Result) {
	if input == nil {
		return
	}
	input.Diagnosis = diagnosisForResult(result)
	input.Confidence = confidenceForResult(result)
	input.EvidenceItems = evidenceItemsForResult(result, input.Severity)
	input.CorroborationNeeded = stringSliceDetail(result.Details, "corroborationNeeded")
	input.SafeActions = stringSliceDetail(result.Details, "safeActions")
	input.OperatorActions = stringSliceDetail(result.Details, "operatorActions")
	input.RollbackOrFallback = stringSliceDetail(result.Details, "rollbackOrFallback")
	input.PostChecks = stringSliceDetail(result.Details, "postChecks")
	input.RemediationCandidates = remediationCandidatesForResult(result)
}

func diagnosisForResult(result checks.Result) string {
	switch result.CheckID {
	case "infra-rdp":
		if !boolDetail(result.Details, "sessionAvailable") {
			return "Remote desktop has no graphical session to share"
		}
		if boolDetail(result.Details, "lockedKeyringPosture") {
			return "Remote desktop credentials are unreadable because GDM autologin never unlocked the login keyring"
		}
		return "Remote desktop is running but cannot authenticate any client"
	case "host-runtime-integrity":
		return "Host runtime exists but cannot communicate with its backing driver or daemon"
	case "host-device-driver-binding":
		return "Important host device is present without an active kernel driver binding"
	case "host-kernel-module-drift":
		if hasEvidenceKind(result.Details, "missing_nvidia_module_package") {
			return "NVIDIA driver stack unavailable for the running kernel"
		}
		return "Running kernel and installed module/package inventory do not fully agree"
	case "host-kernel-error-signals":
		if hasEvidenceKind(result.Details, "data_fabric_sync_flood") {
			return "Kernel reported a data fabric sync flood reset event"
		}
		return "Kernel logged recent host or device error signals"
	case "system-boot-history":
		return "Recent boot history indicates unclean shutdown or reset"
	case "system-pstore-evidence":
		if boolDetail(result.Details, "coverageGap") {
			return "Persistent crash artifact coverage is incomplete because no readable pstore source is available"
		}
		if intDetail(result.Details, "pmsgCount") > 0 && intDetail(result.Details, "dmesgCount") == 0 && intDetail(result.Details, "consoleCount") == 0 {
			return "Userspace pstore messages were found without kernel crash dumps"
		}
		return "Persistent kernel crash artifacts were found"
	case "system-mce-recent":
		if boolDetail(result.Details, "coverageGap") {
			return "Machine-check telemetry coverage is incomplete"
		}
		return "Recent machine-check evidence was found"
	default:
		return ""
	}
}

func confidenceForResult(result checks.Result) string {
	if result.Status == checks.StatusCritical {
		return "high"
	}
	if result.Status == checks.StatusWarning {
		return "medium"
	}
	return ""
}

func evidenceItemsForResult(result checks.Result, severity Severity) []EvidenceItem {
	details := result.Details
	raw, ok := details["evidence"].([]map[string]any)
	if !ok {
		if values, ok := details["evidence"].([]any); ok {
			for _, value := range values {
				if item, ok := value.(map[string]any); ok {
					raw = append(raw, item)
				}
			}
		}
	}
	if len(raw) == 0 {
		return nil
	}
	items := make([]EvidenceItem, 0, len(raw))
	for i, value := range raw {
		kind := fmt.Sprintf("%v", value["kind"])
		if kind == "<nil>" || strings.TrimSpace(kind) == "" {
			kind = "check_evidence"
		}
		itemSeverity := severity
		if s, ok := value["severity"].(string); ok && ValidSeverity(s) && s != "" {
			itemSeverity = Severity(s)
		}
		items = append(items, EvidenceItem{
			ID:                    fmt.Sprintf("%s-%d", result.CheckID, i+1),
			Kind:                  kind,
			Severity:              itemSeverity,
			Summary:               evidenceSummary(kind, value),
			Source:                result.CheckID,
			BootID:                stringDetail(details, "bootId"),
			Data:                  value,
			PlatformApplicability: platformApplicability(details),
		})
	}
	return items
}

func evidenceSummary(kind string, data map[string]any) string {
	switch kind {
	case "missing_nvidia_module_package":
		return fmt.Sprintf("Expected NVIDIA module package is missing for running kernel: %v", data["expectedPackage"])
	case "nvidia_kernel_package_state":
		return "NVIDIA package state was collected for the running kernel"
	case "data_fabric_sync_flood":
		return "Kernel reported an uncorrected data fabric sync flood event"
	case "runtime_not_callable":
		return "Runtime command exists but is not callable"
	case "unbound_capability_device":
		return "Capability device has no active kernel driver binding"
	default:
		return strings.ReplaceAll(kind, "_", " ")
	}
}

func platformApplicability(details map[string]any) string {
	if details == nil {
		return ""
	}
	if unsupported, ok := details["unsupportedCapabilities"]; ok && fmt.Sprintf("%v", unsupported) != "[]" {
		return "unsupported"
	}
	return "applicable"
}

func hasEvidenceKind(details map[string]any, kind string) bool {
	for _, item := range evidenceItemsForResult(checks.Result{Details: details}, SeverityWarning) {
		if item.Kind == kind {
			return true
		}
	}
	return false
}

func remediationCandidatesForResult(result checks.Result) []RemediationCandidate {
	if evidence, ok := firstEvidenceKind(result.Details, "runtime_not_callable"); ok {
		service := strings.TrimSpace(fmt.Sprintf("%v", evidence["service"]))
		applicability := "applicable"
		if service == "" || service == "<nil>" {
			applicability = "needs_corroboration"
		}
		return []RemediationCandidate{{
			ID: "operator-runtime-restart", Title: "Review a platform-native runtime restart", Applicability: applicability,
			Platforms: []string{"linux", "macos", "windows"}, RequiresOperator: true, RequiresPrivilege: true,
			RiskLevel: "moderate", TemplateID: "operator-runtime-restart",
			PreflightChecks: []string{"confirm the affected service identity", "confirm the platform-native action", "obtain operator approval"},
			ArtifactPolicy:  "generate_only_under_user_state", PostChecks: []string{"re-run the originating check"},
			DecisionPrompt: "Should the operator review the generated platform-native restart instructions?",
		}}
	}
	if result.CheckID != "host-kernel-module-drift" {
		return nil
	}
	evidence, ok := firstEvidenceKind(result.Details, "missing_nvidia_module_package")
	if !ok {
		return nil
	}
	applicability := "applicable"
	if platformApplicability(result.Details) == "unsupported" {
		applicability = "unsupported"
	} else if strings.TrimSpace(fmt.Sprintf("%v", evidence["expectedPackage"])) == "" ||
		strings.TrimSpace(fmt.Sprintf("%v", evidence["runningKernel"])) == "" {
		applicability = "needs_corroboration"
	} else if !candidateAvailable(evidence["candidate"]) {
		applicability = "blocked"
	}
	return []RemediationCandidate{{
		ID:                "ubuntu-nvidia-kernel-module-mismatch",
		Title:             "Install matching NVIDIA kernel module package for the running kernel",
		Applicability:     applicability,
		Platforms:         []string{"linux", "ubuntu", "debian"},
		RequiresOperator:  true,
		RequiresPrivilege: true,
		RiskLevel:         "moderate",
		TemplateID:        "ubuntu-nvidia-kernel-module-mismatch",
		PreflightChecks: []string{
			"confirm running kernel and expected linux-modules-nvidia package",
			"confirm apt candidate exists",
			"run apt install simulation before mutation",
		},
		Simulation:     "apt-get -s install <expected-package>",
		ArtifactPolicy: "generate_only_under_user_state",
		RollbackOrFallback: []string{
			"boot a kernel with an already-installed matching NVIDIA module package if the install fails",
			"do not remove kernels or driver packages as part of this remediation",
		},
		PostChecks: []string{
			"nvidia-smi",
			"lsmod | grep '^nvidia'",
			"lspci -nnk | grep -A3 -i nvidia",
			"vrooli-autoheal incidents latest --json",
		},
		DecisionPrompt: "Should the operator run the generated NVIDIA kernel-module repair script?",
	}}
}

func firstEvidenceKind(details map[string]any, kind string) (map[string]any, bool) {
	if details == nil {
		return nil, false
	}
	switch values := details["evidence"].(type) {
	case []map[string]any:
		for _, item := range values {
			if fmt.Sprintf("%v", item["kind"]) == kind {
				return item, true
			}
		}
	case []any:
		for _, value := range values {
			item, ok := value.(map[string]any)
			if ok && fmt.Sprintf("%v", item["kind"]) == kind {
				return item, true
			}
		}
	}
	return nil, false
}

func candidateAvailable(candidate any) bool {
	if candidate == nil {
		return false
	}
	if values, ok := candidate.(map[string]any); ok {
		if available, ok := values["available"].(bool); ok {
			return available
		}
		if available, ok := values["Available"].(bool); ok {
			return available
		}
	}
	value := reflect.ValueOf(candidate)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return false
		}
		value = value.Elem()
	}
	if value.Kind() == reflect.Struct {
		field := value.FieldByName("Available")
		return field.IsValid() && field.Kind() == reflect.Bool && field.Bool()
	}
	return false
}
