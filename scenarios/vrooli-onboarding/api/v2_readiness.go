package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/packages/hostreq"
)

type credentialReadiness struct {
	Resource     string `json:"resource"`
	LogicalID    string `json:"logical_id"`
	Field        string `json:"field"`
	Label        string `json:"label"`
	Description  string `json:"description,omitempty"`
	ObtainURL    string `json:"obtain_url,omitempty"`
	Provisioning string `json:"provisioning,omitempty"`
	DerivedFrom  string `json:"derived_from,omitempty"`
	Required     bool   `json:"required"`
	Status       string `json:"status"`
	Detail       string `json:"detail,omitempty"`
}

type readinessCredentialDescriptor struct {
	LogicalID    string `json:"logical_id"`
	Field        string `json:"field"`
	Label        string `json:"label"`
	Description  string `json:"description"`
	ObtainURL    string `json:"obtain_url"`
	Required     bool   `json:"required"`
	Provisioning string `json:"provisioning,omitempty"`
	DerivedFrom  string `json:"derived_from,omitempty"`
}

type integrationRequirement struct {
	Connector string   `json:"connector"`
	Scopes    []string `json:"scopes,omitempty"`
	Purpose   string   `json:"purpose,omitempty"`
	Required  bool     `json:"required,omitempty"`
	Multi     bool     `json:"multi,omitempty"`
}

type readinessResponse struct {
	Status              string                `json:"status"`
	Scenarios           []string              `json:"scenarios"`
	Resources           []string              `json:"resources"`
	Credentials         []credentialReadiness `json:"credentials"`
	Hosts               []hostReadiness       `json:"hosts"`
	Integrations        []readinessItem       `json:"integrations"`
	CheckedAt           string                `json:"checked_at"`
	CredentialDiagnosis json.RawMessage       `json:"credential_diagnosis,omitempty"`
	Recovery            recoveryReadiness     `json:"recovery"`
	// Blockers are the named reasons configuration is not complete. Degraded
	// holds the optional gaps, which do not block once acknowledged.
	Blockers             []completionBlocker `json:"blockers"`
	Degraded             []completionBlocker `json:"degraded"`
	DegradedDigest       string              `json:"degraded_digest,omitempty"`
	DegradedAcknowledged bool                `json:"degraded_acknowledged"`
}

type recoveryReadiness struct {
	ReceiptExists         bool                   `json:"receipt_exists"`
	ExportedAt            string                 `json:"exported_at,omitempty"`
	EntryCount            int                    `json:"entry_count"`
	Uncovered             []string               `json:"uncovered"`
	RequiredAbsent        []string               `json:"required_absent"`
	RequiredAbsentDetails []recoveryGapReadiness `json:"required_absent_details"`
	RootCopy              json.RawMessage        `json:"root_copy,omitempty"`
	RootCopyIssues        []string               `json:"root_copy_issues"`
}

type recoveryGapReadiness struct {
	Address     string `json:"address"`
	Description string `json:"description,omitempty"`
}

type credentialDiagnosisResponse struct {
	Recovery recoveryReadiness `json:"recovery"`
}

// readinessItem is a metadata-safe, actionable validation result. Its status
// is one of ready, degraded, missing, unsupported, or deferred.
type readinessItem struct {
	Name        string `json:"name"`
	Category    string `json:"category,omitempty"`
	Status      string `json:"status"`
	Detail      string `json:"detail,omitempty"`
	Remediation string `json:"remediation,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type releaseAuthorityStatus struct {
	Configured       bool   `json:"configured"`
	TrustAnchorMatch bool   `json:"trust_anchor_match"`
	Provider         string `json:"provider"`
}

type hostReadiness struct {
	readinessItem
	Kind     string `json:"kind"`
	Required bool   `json:"required"`
}

var credentialStatusCommand = func(ctx context.Context, logicalID, field string) ([]byte, error) {
	return onboardingStatusJSON(ctx, logicalID, field)
}

var releaseAuthorityStatusCommand = func(ctx context.Context, root string) ([]byte, error) {
	command := exec.CommandContext(ctx, "vrooli", "release-authority", "status", "--format", "json")
	command.Dir = root
	return command.Output()
}

func releaseAuthorityReadiness(root string) readinessItem {
	item := readinessItem{
		Name:        "release-authority",
		Category:    "system",
		Status:      "unsupported",
		Detail:      "release authority status is unavailable",
		Remediation: "Run `vrooli release-authority init` after the native secure store is available.",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := releaseAuthorityStatusCommand(ctx, root)
	if err != nil {
		return item
	}
	var status releaseAuthorityStatus
	if err := json.Unmarshal(output, &status); err != nil {
		item.Detail = "release authority returned an invalid status response"
		return item
	}
	if !status.Configured {
		item.Status = "missing"
		item.Detail = "no managed release key is configured"
		return item
	}
	if !status.TrustAnchorMatch {
		item.Status = "degraded"
		item.Detail = "managed release key exists but the repository trust anchor is not synchronized"
		item.Remediation = "Run `vrooli release-authority init --replace-trust-anchor` after reviewing the trust-root change."
		return item
	}
	item.Status = "ready"
	item.Detail = "managed release key and repository trust anchor are synchronized"
	item.Remediation = ""
	return item
}

func selectedScenarioModels() ([]ScenarioReadModel, error) {
	models, err := loadScenarioReadModels()
	if err != nil {
		return nil, err
	}
	selected := make([]ScenarioReadModel, 0, len(models))
	for _, model := range models {
		if model.Enabled {
			selected = append(selected, model)
		}
	}
	return selected, nil
}

func loadCredentialReadiness(resource string) ([]credentialReadiness, error) {
	root, err := manifestRoot()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(root, "resources", resource, "resource.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var manifest struct {
		Credentials struct {
			Descriptors []readinessCredentialDescriptor `json:"descriptors"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode %s credential descriptors: %w", resource, err)
	}
	return credentialReadinessForDescriptors(resource, manifest.Credentials.Descriptors), nil
}

func loadScenarioCredentialReadiness(scenario string) ([]credentialReadiness, error) {
	root, err := manifestRoot()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(root, "scenarios", scenario, ".vrooli", "service.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var manifest struct {
		Credentials struct {
			Descriptors []readinessCredentialDescriptor `json:"descriptors"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode %s scenario credential descriptors: %w", scenario, err)
	}
	return credentialReadinessForDescriptors(scenario, manifest.Credentials.Descriptors), nil
}

func loadIntegrationReadiness(path, owner string) ([]readinessItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var manifest struct {
		Integrations []integrationRequirement `json:"integrations"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode %s integration declarations: %w", owner, err)
	}
	items := make([]readinessItem, 0, len(manifest.Integrations))
	for _, integration := range manifest.Integrations {
		connector := strings.TrimSpace(integration.Connector)
		if connector == "" {
			return nil, fmt.Errorf("%s declares an integration without a connector", owner)
		}
		name := owner + "/" + connector
		detail := "Connection setup is deferred until the integration capability is available."
		if integration.Purpose != "" {
			detail = integration.Purpose + " Connection setup is deferred until the integration capability is available."
		}
		if len(integration.Scopes) > 0 {
			detail += " Requested scopes: " + strings.Join(integration.Scopes, ", ") + "."
		}
		if integration.Multi {
			detail += " Multiple connections may be bound."
		}
		items = append(items, readinessItem{
			Name:        name,
			Category:    "integration",
			Status:      "deferred",
			Detail:      detail,
			Remediation: "Configure the declared connection when the integration capability is available.",
			Required:    integration.Required,
		})
	}
	return items, nil
}

func credentialReadinessForDescriptors(owner string, descriptors []readinessCredentialDescriptor) []credentialReadiness {
	items := make([]credentialReadiness, 0, len(descriptors))
	for _, descriptor := range descriptors {
		field := strings.TrimSpace(descriptor.Field)
		if field == "" {
			field = "value"
		}
		item := credentialReadiness{Resource: owner, LogicalID: descriptor.LogicalID, Field: field, Label: descriptor.Label, Description: descriptor.Description, ObtainURL: descriptor.ObtainURL, Required: descriptor.Required, Provisioning: descriptor.Provisioning, DerivedFrom: descriptor.DerivedFrom, Status: "unconfigured"}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		output, statusErr := credentialStatusCommand(ctx, descriptor.LogicalID, field)
		cancel()
		if statusErr != nil {
			item.Status = "unsupported"
			item.Detail = "native credential authority unavailable"
		} else {
			var status struct {
				Configured bool `json:"configured"`
			}
			if err := json.Unmarshal(output, &status); err != nil {
				item.Status = "unsupported"
				item.Detail = "credential authority returned an invalid status response"
			} else if status.Configured {
				item.Status = "configured"
			}
		}
		items = append(items, item)
	}
	return items
}

// buildReadinessResponse computes the whole readiness verdict.
//
// It is a function rather than handler-local code because the completion gate
// needs the same verdict the operator sees. Computing it twice from two code
// paths is how the wizard and the marker came to disagree in the first place.
func buildReadinessResponse(ctx context.Context) (readinessResponse, error) {
	// One probe per request, before any credential status is read, so a store
	// unlocked after this process started is observed on this request rather
	// than after a restart.
	recheckCredentialAuthority()
	models, err := selectedScenarioModels()
	if err != nil {
		return readinessResponse{}, err
	}
	root, err := manifestRoot()
	if err != nil {
		return readinessResponse{}, err
	}
	resourceSet := map[string]struct{}{}
	response := readinessResponse{Status: "ready", Scenarios: make([]string, 0, len(models)), Credentials: []credentialReadiness{}, Hosts: []hostReadiness{}, Integrations: []readinessItem{}, Recovery: recoveryReadiness{Uncovered: []string{}, RequiredAbsent: []string{}, RequiredAbsentDetails: []recoveryGapReadiness{}, RootCopyIssues: []string{}}, Blockers: []completionBlocker{}, Degraded: []completionBlocker{}, CheckedAt: operatorStateNow().UTC().Format(time.RFC3339)}
	for _, model := range models {
		response.Scenarios = append(response.Scenarios, model.Name)
		scenarioCredentials, err := loadScenarioCredentialReadiness(model.Name)
		if err != nil {
			return readinessResponse{}, err
		}
		response.Credentials = append(response.Credentials, scenarioCredentials...)
		integrationItems, integrationErr := loadIntegrationReadiness(filepath.Join(root, "scenarios", model.Name, ".vrooli", "service.json"), model.Name)
		if integrationErr != nil {
			return readinessResponse{}, integrationErr
		}
		response.Integrations = append(response.Integrations, integrationItems...)
		for _, resource := range model.Resources {
			resourceSet[resource] = struct{}{}
		}
	}
	for resource := range resourceSet {
		response.Resources = append(response.Resources, resource)
		credentials, err := loadCredentialReadiness(resource)
		if err != nil {
			return readinessResponse{}, err
		}
		response.Credentials = append(response.Credentials, credentials...)
		integrationItems, integrationErr := loadIntegrationReadiness(filepath.Join(root, "resources", resource, "resource.json"), resource)
		if integrationErr != nil {
			return readinessResponse{}, integrationErr
		}
		response.Integrations = append(response.Integrations, integrationItems...)
	}
	projectCredentials, projectErr := projectCredentialReadiness()
	if projectErr != nil {
		return readinessResponse{}, projectErr
	}
	response.Credentials = append(response.Credentials, projectCredentials...)
	sort.Strings(response.Scenarios)
	sort.Strings(response.Resources)
	sortCredentialReadiness(response.Credentials)
	sort.Slice(response.Integrations, func(i, j int) bool {
		return response.Integrations[i].Name < response.Integrations[j].Name
	})
	state, err := loadOperatorStateFor(ctx)
	if err != nil {
		return readinessResponse{}, err
	}
	allModels, err := loadScenarioReadModels()
	if err != nil {
		return readinessResponse{}, err
	}
	hostModels, err := hostRequirementScenarioModels(root, allModels, state)
	if err != nil {
		return readinessResponse{}, err
	}
	hosts, err := deriveV2HostRequirements(root, state, hostModels)
	if err != nil {
		return readinessResponse{}, err
	}
	for _, tool := range hosts.Tools {
		response.Hosts = append(response.Hosts, inspectToolReadiness(tool))
	}
	for _, safeguard := range hosts.Safeguards {
		response.Hosts = append(response.Hosts, inspectSafeguardReadiness(root, safeguard))
	}
	sort.Slice(response.Hosts, func(i, j int) bool {
		return response.Hosts[i].Kind+response.Hosts[i].Name < response.Hosts[j].Kind+response.Hosts[j].Name
	})
	for _, credential := range response.Credentials {
		switch {
		case credential.Status == "unsupported":
			response.Status = lessReady(response.Status, "unsupported")
		case credential.Required && operatorSuppliedCredential(credential) && credential.Status != "configured":
			response.Status = lessReady(response.Status, "missing")
		case !credential.Required && credential.Status != "configured":
			response.Status = lessReady(response.Status, "degraded")
		}
	}
	doctorCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	if output, err := credentialDoctorCommand(doctorCtx); err == nil && json.Valid(output) {
		response.CredentialDiagnosis = append(response.CredentialDiagnosis[:0], output...)
		var diagnosis credentialDiagnosisResponse
		if json.Unmarshal(output, &diagnosis) == nil {
			response.Recovery = diagnosis.Recovery
			if response.Recovery.Uncovered == nil {
				response.Recovery.Uncovered = []string{}
			}
			if response.Recovery.RequiredAbsent == nil {
				response.Recovery.RequiredAbsent = []string{}
			}
			if response.Recovery.RequiredAbsentDetails == nil {
				response.Recovery.RequiredAbsentDetails = []recoveryGapReadiness{}
			}
			if response.Recovery.RootCopyIssues == nil {
				response.Recovery.RootCopyIssues = []string{}
			}
			if len(response.Recovery.RequiredAbsent) > 0 {
				response.Status = lessReady(response.Status, "missing")
			} else if len(response.Recovery.Uncovered) > 0 || len(response.Recovery.RootCopyIssues) > 0 {
				response.Status = lessReady(response.Status, "degraded")
			}
		}
	}
	cancel()
	for _, host := range response.Hosts {
		response.Status = lessReady(response.Status, host.Status)
	}
	releaseItem := releaseAuthorityReadiness(root)
	response.Integrations = append(response.Integrations, releaseItem)
	response.Status = lessReady(response.Status, releaseItem.Status)
	assessment := assessCompletion(response, nil)
	response.Blockers = assessment.Blockers
	response.Degraded = assessment.Degraded
	response.DegradedDigest = assessment.DegradedDigest
	response.DegradedAcknowledged = degradedAcknowledgementMatches(state, assessment.DegradedDigest)
	return response, nil
}

func (s *Server) handleV2Readiness(w http.ResponseWriter, r *http.Request) {
	response, err := buildReadinessResponse(r.Context())
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func inspectToolReadiness(item hostItem) hostReadiness {
	result := hostReadiness{readinessItem: readinessItem{Name: item.Name, Status: "deferred", Detail: item.Notes, Remediation: "Select this optional tool in onboarding when its capability is needed."}, Kind: "tool", Required: item.Required}
	selected := item.Status == "required" || item.Status == "opted_in"
	if !selected {
		return result
	}
	if !hostPlatformSupported(item.Platforms) {
		result.Status = "unsupported"
		result.Detail = "This tool is not declared for the current operating system."
		result.Remediation = "Use a supported deployment target or deselect the optional capability."
		return result
	}
	if len(item.Commands) == 0 {
		result.Status = "unsupported"
		result.Detail = "The tool manifest has no executable probe command."
		result.Remediation = "Add a declared command to the tool manifest before enabling this capability."
		return result
	}
	if _, err := exec.LookPath(item.Commands[0]); err != nil {
		result.Status = "missing"
		result.Detail = "The selected tool was not found on this host."
		result.Remediation = "Install the declared host tool, then rerun validation."
		return result
	}
	result.Status = "ready"
	result.Detail = "The declared executable is available on this host."
	result.Remediation = ""
	return result
}

func inspectSafeguardReadiness(root string, item hostItem) hostReadiness {
	result := hostReadiness{readinessItem: readinessItem{Name: item.Name, Status: "deferred", Detail: item.Notes, Remediation: "Select this optional safeguard only after reviewing its host-change risk."}, Kind: "safeguard", Required: item.Required}
	selected := item.Status == "required" || item.Status == "opted_in"
	if !selected {
		return result
	}
	if !hostPlatformSupported(item.Platforms) {
		result.Status = "unsupported"
		result.Detail = "This safeguard is not declared for the current operating system."
		result.Remediation = "Use a supported deployment target or deselect the optional safeguard."
		return result
	}
	data, err := hostManifest(root, "safeguards", item.Name)
	if err != nil {
		result.Status = "unsupported"
		result.Detail = "The safeguard manifest could not be resolved."
		result.Remediation = "Repair the safeguard declaration before continuing."
		return result
	}
	var manifest struct {
		Handler           string `json:"handler"`
		VerificationCheck struct {
			Files []string `json:"files"`
		} `json:"verificationCheck"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		result.Status = "unsupported"
		result.Detail = "The safeguard manifest could not be decoded."
		result.Remediation = "Repair the safeguard declaration before continuing."
		return result
	}
	if len(manifest.VerificationCheck.Files) == 0 {
		if strings.TrimSpace(manifest.Handler) != "" {
			// A handler-owned safeguard has no file this process can stat, but
			// that never made it unknowable: the control plane owns a read-only
			// inspection half that is separate from Apply by interface. Asking
			// it is the whole check. Deferring to the apply outcome instead
			// told the operator a safeguard was uncheckable while the answer
			// was one unprivileged call away.
			return observeHandlerSafeguard(root, item)
		}
		result.Status = "unsupported"
		result.Detail = "The safeguard has no declarative verification probe and no handler."
		result.Remediation = "Add a verification check or a handler before enabling this safeguard."
		return result
	}
	for _, declared := range manifest.VerificationCheck.Files {
		path, resolveErr := resolveSafeguardVerificationPath(declared)
		if resolveErr != nil {
			result.Status = "unsupported"
			result.Detail = "A declared verification path could not be resolved on this host."
			result.Remediation = "Correct the safeguard's verification path declaration."
			return result
		}
		if _, err := os.Stat(path); err != nil {
			result.Status = "missing"
			result.Detail = "The safeguard has not been applied on this host."
			result.Remediation = "Apply the declared safeguard with the required host privilege, then rerun validation."
			return result
		}
	}
	result.Status = "ready"
	result.Detail = "The declared safeguard verification files are present."
	result.Remediation = ""
	return result
}

// observeHandlerSafeguard reports a handler-owned safeguard through the control
// plane's read-only observation boundary. The boundary never calls Apply, so
// this stays safe to run before the operator has consented to anything.
//
// The mapping is deliberately not collapsed into ready/missing. A safeguard
// that applied but needs a reboot has not taken effect, and a probe that could
// not run is not a host fault; flattening either into "missing" would report a
// gap the operator cannot act on and would send them to fix a healthy host.
func observeHandlerSafeguard(root string, item hostItem) hostReadiness {
	result := hostReadiness{readinessItem: readinessItem{Name: item.Name}, Kind: "safeguard", Required: item.Required}

	observed, err := hostreq.ObserveSafeguard(root, item.Name, nil)
	if err != nil {
		result.Status = "deferred"
		result.Detail = "The control plane could not sample this safeguard: " + err.Error()
		result.Remediation = "Apply the selection; the handler reports the outcome for this item."
		return result
	}

	result.Detail = lastNote(observed.Notes)

	switch observed.ExecutionState {
	case "already_present", "applied", "installed":
		result.Status = "ready"
		if result.Detail == "" {
			result.Detail = "The control-plane handler reports this safeguard is in place."
		}
	case "pending", "would_apply", "would_install":
		result.Status = "missing"
		if result.Detail == "" {
			result.Detail = "The control-plane handler reports this safeguard is not in place."
		}
		// The detail above is the handler's own reason, and it is often more
		// specific than "apply this" -- a missing credential, an absent device.
		// The remediation must not talk past it.
		result.Remediation = "Apply the selection to let the handler make this change; when the reason above names a missing input, supply that first."
	case "reboot_required":
		result.Status = "degraded"
		if result.Detail == "" {
			result.Detail = "This safeguard is configured but does not take effect until the host reboots."
		}
		result.Remediation = "Reboot the host to activate this safeguard."
	case "manual_action_required":
		result.Status = "missing"
		if result.Detail == "" {
			result.Detail = "This safeguard requires an action Vrooli will not take on the operator's behalf."
		}
		result.Remediation = "Complete the manual step this safeguard declares, then rerun validation."
	case "unsupported", "not_applicable":
		result.Status = "unsupported"
		if result.Detail == "" {
			result.Detail = "This safeguard does not apply to this host."
		}
		result.Remediation = "Deselect this safeguard, or run on a host it supports."
	default:
		// "failed" and any state added later. An inspection that did not reach
		// a verdict must not be rendered as one.
		result.Status = "deferred"
		if result.Detail == "" {
			result.Detail = "The control-plane handler did not reach a verdict for this safeguard."
		}
		result.Remediation = "Apply the selection; the handler reports the outcome for this item."
	}
	return result
}

// lastNote returns the most specific line a handler emitted. Handler notes are
// ordered general-to-specific, with the deciding observation last.
func lastNote(notes []string) string {
	for index := len(notes) - 1; index >= 0; index-- {
		if trimmed := strings.TrimSpace(notes[index]); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// resolveSafeguardVerificationPath expands the portable tokens a safeguard
// manifest is allowed to use in a verification path.
//
// Safeguard manifests declare user-scoped paths as $USER_HOME/... so one
// declaration serves every account and every platform. Reading such a path
// literally reports every user-scoped safeguard as unapplied, which is a false
// negative on a required item — and a completion gate built on a false
// negative would block a host that is in fact correctly configured. The
// expansion uses the same api-core storage resolver the rest of the repository
// uses, so onboarding does not invent a second path vocabulary.
func resolveSafeguardVerificationPath(declared string) (string, error) {
	declared = strings.TrimSpace(declared)
	if declared == "" {
		return "", fmt.Errorf("safeguard verification path is empty")
	}
	return storage.ResolvePortablePath("safeguard verification", storage.PortablePath{Value: declared}, storage.HostPlatform(), storage.PlatformSeams{})
}

// sortCredentialReadiness keeps one ordering for every credential surface, so
// the project scope lands in the same place on the API, the UI, and the CLI.
// The address is part of the key because owner and field alone are not unique.
func sortCredentialReadiness(items []credentialReadiness) {
	sort.Slice(items, func(i, j int) bool {
		left := items[i].Resource + "\x00" + items[i].LogicalID + "\x00" + items[i].Field
		right := items[j].Resource + "\x00" + items[j].LogicalID + "\x00" + items[j].Field
		return left < right
	})
}

func lessReady(current, candidate string) string {
	priority := map[string]int{"ready": 0, "deferred": 0, "degraded": 1, "missing": 2, "unsupported": 3}
	if priority[candidate] > priority[current] {
		return candidate
	}
	return current
}
