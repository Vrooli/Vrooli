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
	"sync"
	"time"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/internal/credentialspec"
	"github.com/vrooli/vrooli/internal/hostinventory"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	"github.com/vrooli/vrooli/packages/hostreq"
	readinessdomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/readiness"
)

type credentialReadiness = readinessdomain.Credential

const (
	credentialEvidenceUnavailable = "unavailable"
	credentialEvidenceUnverified  = "unverified"
	credentialStatusPending       = "pending"
)

type integrationRequirement struct {
	Connector string   `json:"connector"`
	Scopes    []string `json:"scopes,omitempty"`
	Purpose   string   `json:"purpose,omitempty"`
	Required  bool     `json:"required,omitempty"`
	Multi     bool     `json:"multi,omitempty"`
}

type readinessResponse = readinessdomain.Response

type recoveryReadiness = readinessdomain.Recovery
type recoveryGapReadiness = readinessdomain.RecoveryGap

type credentialDiagnosisResponse struct {
	Recovery recoveryReadiness `json:"recovery"`
}

type credentialConsumerInventoryCacheState struct {
	sync.Mutex
	key       string
	blockers  []completionBlocker
	ready     bool
	running   bool
	expiresAt time.Time
	lastError string
}

var credentialConsumerInventoryCache credentialConsumerInventoryCacheState

// readinessItem is a metadata-safe, actionable validation result. Its status
// is one of ready, degraded, missing, unsupported, or deferred.
type readinessItem = readinessdomain.Item

type releaseAuthorityStatus struct {
	Configured       bool   `json:"configured"`
	TrustAnchorMatch bool   `json:"trust_anchor_match"`
	Provider         string `json:"provider"`
}

type hostReadiness = readinessdomain.Host

var credentialStatusCommand = func(ctx context.Context, logicalID, field string) ([]byte, error) {
	return onboardingStatusJSON(ctx, logicalID, field)
}

var releaseAuthorityStatusCommand = func(ctx context.Context, root string) ([]byte, error) {
	command := exec.CommandContext(ctx, "vrooli", "release-authority", "status", "--format", "json")
	command.Dir = root
	return command.Output()
}

func releaseAuthorityReadiness(root string) readinessItem {
	item, _, _ := releaseAuthorityReadinessWithStatus(root)
	return item
}

func releaseAuthorityReadinessWithStatus(root string) (readinessItem, releaseAuthorityStatus, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return releaseAuthorityReadinessWithStatusContext(ctx, root)
}

func releaseAuthorityReadinessWithStatusContext(ctx context.Context, root string) (readinessItem, releaseAuthorityStatus, bool) {
	item := readinessItem{
		Name:        "release-authority",
		Category:    "system",
		Status:      "unsupported",
		Detail:      "release authority status is unavailable",
		Remediation: "Run `vrooli release-authority init` after the native secure store is available.",
	}
	output, err := releaseAuthorityStatusCommand(ctx, root)
	if err != nil {
		return item, releaseAuthorityStatus{}, false
	}
	var status releaseAuthorityStatus
	if err := json.Unmarshal(output, &status); err != nil {
		item.Detail = "release authority returned an invalid status response"
		return item, releaseAuthorityStatus{}, false
	}
	if !status.Configured {
		item.Status = "missing"
		item.Detail = "no managed release key is configured"
		return item, status, true
	}
	if !status.TrustAnchorMatch {
		item.Status = "degraded"
		item.Detail = "managed release key exists but the repository trust anchor is not synchronized"
		item.Remediation = "Run `vrooli release-authority init --replace-trust-anchor` after reviewing the trust-root change."
		return item, status, true
	}
	item.Status = "ready"
	item.Detail = "managed release key and repository trust anchor are synchronized"
	item.Remediation = ""
	return item, status, true
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

// credentialMetadataInventory is the fast half of the canonical projection.
// It resolves the same scoped descriptor set as readiness, but intentionally
// does not contact the credential authority. The list surface can therefore
// render declared inputs immediately while readiness enriches those rows with
// configured/unconfigured status in the background.
func credentialMetadataInventory(closure closureResult) ([]credentialReadiness, error) {
	root, err := manifestRoot()
	if err != nil {
		return nil, err
	}
	refs, err := credentialInventoryProjection(root, closure)
	if err != nil {
		return nil, err
	}
	credentials := credentialReadinessForRefs(context.Background(), refs, false)
	sortCredentialReadiness(credentials)
	return credentials, nil
}

func credentialReadinessInventoryContext(ctx context.Context, closure closureResult) ([]credentialReadiness, error) {
	root, err := manifestRoot()
	if err != nil {
		return nil, err
	}
	refs, err := credentialInventoryProjection(root, closure)
	if err != nil {
		return nil, err
	}
	credentials := credentialReadinessForRefs(ctx, refs, true)
	sortCredentialReadiness(credentials)
	return credentials, nil
}

func credentialInventoryProjection(root string, closure closureResult) ([]credentialclient.CredentialRef, error) {
	scenarioNames := make([]string, 0, len(closure.Scenarios))
	for _, member := range closure.Scenarios {
		scenarioNames = append(scenarioNames, member.Name)
	}
	resourceNames := make([]string, 0, len(closure.Resources))
	for _, member := range closure.Resources {
		resourceNames = append(resourceNames, member.Name)
	}
	refs, err := credentialclient.DescriptorsForScope(root, credentialclient.Scope{
		IncludeProject: projectScopeAvailable(),
		IncludeManaged: true,
		Scenarios:      scenarioNames,
		Resources:      resourceNames,
	})
	if err != nil {
		return nil, err
	}
	return refs, nil
}

// credentialConsumerInventoryBlockers turns only required declaration-to-use
// gaps into completion blockers. Optional gaps remain diagnostic inventory;
// they must not make an otherwise usable onboarding selection impossible.
func credentialConsumerInventoryBlockers(ctx context.Context, root string, closure closureResult) ([]completionBlocker, error) {
	scenarioNames := make([]string, 0, len(closure.Scenarios))
	for _, member := range closure.Scenarios {
		scenarioNames = append(scenarioNames, member.Name)
	}
	resourceNames := make([]string, 0, len(closure.Resources))
	for _, member := range closure.Resources {
		resourceNames = append(resourceNames, member.Name)
	}
	scope := credentialclient.Scope{
		IncludeProject: projectScopeAvailable(),
		Scenarios:      scenarioNames,
		Resources:      resourceNames,
	}
	key := root + "|" + strings.Join(scenarioNames, ",") + "|" + strings.Join(resourceNames, ",")
	now := time.Now()
	credentialConsumerInventoryCache.Lock()
	if credentialConsumerInventoryCache.key == key && credentialConsumerInventoryCache.ready && now.Before(credentialConsumerInventoryCache.expiresAt) {
		blockers := append([]completionBlocker(nil), credentialConsumerInventoryCache.blockers...)
		lastError := credentialConsumerInventoryCache.lastError
		credentialConsumerInventoryCache.Unlock()
		if lastError != "" {
			return blockers, fmt.Errorf("credential consumer inventory: %s", lastError)
		}
		return blockers, nil
	}
	if credentialConsumerInventoryCache.key == key && credentialConsumerInventoryCache.running {
		credentialConsumerInventoryCache.Unlock()
		return []completionBlocker{{
			Kind:        "readiness",
			Name:        "credential-consumer-inventory",
			Reason:      "required credential consumer inventory is still being verified",
			Remediation: "retry readiness in a moment to refresh the completion verdict",
		}}, nil
	}
	credentialConsumerInventoryCache.key = key
	credentialConsumerInventoryCache.ready = false
	credentialConsumerInventoryCache.running = true
	credentialConsumerInventoryCache.lastError = ""
	credentialConsumerInventoryCache.Unlock()

	// The source scan is intentionally detached from the request. Readiness is
	// allowed to report a stable pending blocker while the bounded diagnostic
	// refresh completes, instead of making every credentials check wait on a
	// repository-sized source walk.
	go func(cacheKey, scanRoot string, scanScope credentialclient.Scope) {
		scanCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		inventory, inventoryErr := credentialclient.DiscoverConsumerInventoryContext(scanCtx, scanRoot, scanScope)
		blockers := make([]completionBlocker, 0)
		if inventoryErr == nil {
			for _, gap := range inventory.Gaps {
				if !gap.Required {
					continue
				}
				name := strings.TrimSpace(gap.Address)
				if name == "" {
					name = strings.TrimSpace(gap.SourceRef)
				}
				if name == "" {
					name = "unresolved-consumer"
				}
				blockers = append(blockers, completionBlocker{
					Kind:        "credential-consumer",
					Name:        name,
					Reason:      firstNonEmptyString(gap.Reason, "a required credential consumer is unresolved"),
					Remediation: firstNonEmptyString(gap.Remediation, "declare the required credential consumer in its owning manifest"),
				})
			}
			sortBlockers(blockers)
		}
		credentialConsumerInventoryCache.Lock()
		defer credentialConsumerInventoryCache.Unlock()
		if credentialConsumerInventoryCache.key != cacheKey {
			return
		}
		credentialConsumerInventoryCache.blockers = blockers
		credentialConsumerInventoryCache.ready = true
		credentialConsumerInventoryCache.running = false
		credentialConsumerInventoryCache.expiresAt = time.Now().Add(2 * time.Minute)
		if inventoryErr != nil {
			credentialConsumerInventoryCache.lastError = inventoryErr.Error()
		}
	}(key, root, scope)
	return []completionBlocker{{
		Kind:        "readiness",
		Name:        "credential-consumer-inventory",
		Reason:      "required credential consumer inventory is being verified",
		Remediation: "retry readiness in a moment to refresh the completion verdict",
	}}, nil
}

func credentialReadinessForRefsContext(ctx context.Context, refs []credentialclient.CredentialRef) []credentialReadiness {
	return credentialReadinessForRefs(ctx, refs, true)
}

func credentialReadinessForRefs(ctx context.Context, refs []credentialclient.CredentialRef, probeAuthority bool) []credentialReadiness {
	items := make([]credentialReadiness, len(refs))
	if len(refs) == 0 {
		return items
	}
	for index := range refs {
		items[index] = credentialReadinessItem(refs[index], probeAuthority)
	}
	if !probeAuthority {
		return items
	}
	workers := 32
	if len(refs) < workers {
		workers = len(refs)
	}
	indices := make(chan int)
	var group sync.WaitGroup
	group.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func() {
			defer group.Done()
			for index := range indices {
				ref := refs[index]
				field := credentialDescriptorField(ref.Field)
				item := items[index]
				// Status is a metadata-only probe. A slow provider must not hold
				// the whole credentials page open; the card can show the resulting
				// unsupported/deferred state and the operator can retry.
				probeContext, cancel := context.WithTimeout(ctx, time.Second)
				type statusResult struct {
					output []byte
					err    error
				}
				resultCh := make(chan statusResult, 1)
				go func() {
					output, statusErr := credentialStatusCommand(probeContext, ref.LogicalID, field)
					resultCh <- statusResult{output: output, err: statusErr}
				}()
				var output []byte
				var statusErr error
				probeTimedOut := false
				select {
				case result := <-resultCh:
					output, statusErr = result.output, result.err
				case <-probeContext.Done():
					probeTimedOut = true
				}
				cancel()
				if probeTimedOut {
					item.Status = credentialStatusPending
					item.Detail = "credential authority is still checking this address"
					item.EvidenceDetail = "The provider did not answer within the readiness budget; retry to refresh this status."
				} else if statusErr != nil {
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
						item.EvidenceStatus = credentialEvidenceUnverified
						item.EvidenceDetail = "The value is stored, but its owning provider has not supplied verification evidence."
					}
				}
				items[index] = item
			}
		}()
	}
	for index := range refs {
		indices <- index
	}
	close(indices)
	group.Wait()
	return items
}

func credentialReadinessItem(ref credentialclient.CredentialRef, probeAuthority bool) credentialReadiness {
	item := credentialReadiness{
		Version: ref.Version, Resource: ref.Resource, LogicalID: ref.LogicalID, Field: credentialDescriptorField(ref.Field),
		Owner: ref.Owner, SourceRef: ref.SourceRef, Kind: ref.Kind, ConsumerRefs: append([]string(nil), ref.ConsumerRefs...),
		Provider: ref.Provider, AppliesWhen: ref.AppliesWhen, RequirementGroup: ref.RequirementGroup,
		CompanionSettings: append([]string(nil), ref.CompanionSettings...), CompanionCredentials: append([]string(nil), ref.CompanionCredentials...), AcquisitionRef: ref.AcquisitionRef,
		VerificationRef: ref.VerificationRef, RecoveryRef: ref.RecoveryRef, HelpRef: ref.HelpRef, EvidencePolicy: ref.EvidencePolicy, ProviderVersion: ref.ProviderVersion,
		MigrationDiagnostics: append([]credentialspec.MigrationDiagnostic(nil), ref.MigrationDiagnostics...),
		Provenance:           readinessCredentialProvenance(ref.Provenance), Label: ref.Label, Description: ref.Description,
		ObtainURL: ref.ObtainURL, Required: ref.Required, Provisioning: ref.Provisioning, DerivedFrom: ref.DerivedFrom,
		Status: credentialStatusPending, EvidenceStatus: credentialEvidenceUnavailable,
	}
	if probeAuthority {
		item.Status = "unconfigured"
		item.EvidenceDetail = "No stored value is available to verify."
	} else {
		item.Detail = "Credential storage status is checked by readiness."
		item.EvidenceDetail = "Credential storage status is checked by readiness."
	}
	return item
}

func readinessCredentialProvenance(values []credentialclient.CredentialProvenance) []readinessdomain.CredentialProvenance {
	result := make([]readinessdomain.CredentialProvenance, 0, len(values))
	for _, value := range values {
		consumers := make([]readinessdomain.CredentialConsumerProvenance, 0, len(value.Consumers))
		for _, consumer := range value.Consumers {
			consumers = append(consumers, readinessdomain.CredentialConsumerProvenance{
				LogicalID: consumer.LogicalID, AddressPattern: consumer.AddressPattern, Field: consumer.Field, Kind: consumer.Kind,
				Consumer: consumer.Consumer, SourceRef: consumer.SourceRef, Required: consumer.Required, Reason: consumer.Reason,
				Tiers: append([]string(nil), consumer.Tiers...),
			})
		}
		result = append(result, readinessdomain.CredentialProvenance{
			Version: value.Version, Owner: value.Owner, SourceRef: value.SourceRef, Kind: value.Kind, Provider: value.Provider, AppliesWhen: value.AppliesWhen, RequirementGroup: value.RequirementGroup, ConsumerRefs: append([]string(nil), value.ConsumerRefs...),
			CompanionSettings: append([]string(nil), value.CompanionSettings...), CompanionCredentials: append([]string(nil), value.CompanionCredentials...), AcquisitionRef: value.AcquisitionRef, VerificationRef: value.VerificationRef, RecoveryRef: value.RecoveryRef, HelpRef: value.HelpRef, EvidencePolicy: value.EvidencePolicy, ProviderVersion: value.ProviderVersion,
			Env: value.Env, Label: value.Label,
			Description: value.Description, ObtainURL: value.ObtainURL, Provisioning: value.Provisioning,
			DerivedFrom: value.DerivedFrom, Required: value.Required, Consumers: consumers,
		})
	}
	return result
}

// buildReadinessResponse computes the whole readiness verdict.
//
// It is a function rather than handler-local code because the completion gate
// needs the same verdict the operator sees. Computing it twice from two code
// paths is how the wizard and the marker came to disagree in the first place.
func buildReadinessResponse(ctx context.Context) (readinessResponse, error) {
	return buildReadinessResponseForTarget(ctx, "local")
}

func buildReadinessResponseForTarget(ctx context.Context, target string) (readinessResponse, error) {
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
	closure, err := resolveClosure(root, models)
	if err != nil {
		return readinessResponse{}, err
	}
	// These checks are independent of the descriptor projection and each other.
	// Start them before the credential probes so a slow native store or doctor
	// cannot add its timeout after the credential inventory has finished.
	probeCtx, probeCancel := context.WithCancel(ctx)
	defer probeCancel()
	type doctorResult struct {
		output []byte
		err    error
	}
	doctorCh := make(chan doctorResult, 1)
	go func() {
		doctorCtx, cancel := context.WithTimeout(probeCtx, 2*time.Second)
		defer cancel()
		output, doctorErr := credentialDoctorCommand(doctorCtx)
		doctorCh <- doctorResult{output: output, err: doctorErr}
	}()
	releaseCh := make(chan struct {
		item               readinessItem
		authority          releaseAuthorityStatus
		authorityAvailable bool
	}, 1)
	go func() {
		releaseCtx, cancel := context.WithTimeout(probeCtx, 2*time.Second)
		defer cancel()
		item, authority, available := releaseAuthorityReadinessWithStatusContext(releaseCtx, root)
		releaseCh <- struct {
			item               readinessItem
			authority          releaseAuthorityStatus
			authorityAvailable bool
		}{item: item, authority: authority, authorityAvailable: available}
	}()
	storeCh := make(chan hostReadiness, 1)
	go func() {
		// The native store has its own owner/collection probe budgets. Keep the
		// onboarding request shorter than the previous five-second umbrella so
		// a wedged keyring becomes an explicit status instead of holding the
		// entire readiness response open.
		storeCtx, cancel := context.WithTimeout(probeCtx, 2*time.Second)
		defer cancel()
		store := hostinventory.CredentialStoreStatus(storeCtx)
		storeStatus := store.State
		if storeStatus == "" {
			storeStatus = "unknown"
		}
		storeCh <- hostReadiness{Item: readinessItem{
			Name: "credential_store", Category: "system", Status: storeStatus,
			Detail: store.Reason, Remediation: "Run `vrooli credentials keyring unlock` and retry configuration.", Required: true,
		}, Kind: "credential_store", Required: true}
	}()
	type credentialResult struct {
		items []credentialReadiness
		err   error
	}
	credentialCh := make(chan credentialResult, 1)
	go func() {
		items, credentialErr := credentialReadinessInventoryContext(probeCtx, closure)
		credentialCh <- credentialResult{items: items, err: credentialErr}
	}()
	type consumerInventoryResult struct {
		blockers []completionBlocker
		err      error
	}
	consumerInventoryCh := make(chan consumerInventoryResult, 1)
	go func() {
		inventoryCtx, cancel := context.WithTimeout(probeCtx, 5*time.Second)
		defer cancel()
		blockers, inventoryErr := credentialConsumerInventoryBlockers(inventoryCtx, root, closure)
		consumerInventoryCh <- consumerInventoryResult{blockers: blockers, err: inventoryErr}
	}()
	response := readinessResponse{Target: strings.TrimSpace(target), Status: "ready", Scenarios: make([]string, 0, len(models)), Credentials: []credentialReadiness{}, Hosts: []hostReadiness{}, Integrations: []readinessItem{}, Recovery: recoveryReadiness{Uncovered: []string{}, RequiredAbsent: []string{}, RequiredAbsentDetails: []recoveryGapReadiness{}, RootCopyIssues: []string{}}, Blockers: []completionBlocker{}, Degraded: []completionBlocker{}, CheckedAt: operatorStateNow().UTC().Format(time.RFC3339), ExpiresAt: operatorStateNow().UTC().Add(5 * time.Minute).Format(time.RFC3339)}
	for _, model := range models {
		response.Scenarios = append(response.Scenarios, model.Name)
		integrationItems, integrationErr := loadIntegrationReadiness(filepath.Join(root, "scenarios", model.Name, ".vrooli", "service.json"), model.Name)
		if integrationErr != nil {
			return readinessResponse{}, integrationErr
		}
		response.Integrations = append(response.Integrations, integrationItems...)
	}
	for _, member := range closure.Resources {
		resource := member.Name
		response.Resources = append(response.Resources, resource)
		integrationItems, integrationErr := loadIntegrationReadiness(filepath.Join(root, "resources", resource, "resource.json"), resource)
		if integrationErr != nil {
			return readinessResponse{}, integrationErr
		}
		response.Integrations = append(response.Integrations, integrationItems...)
	}
	sort.Strings(response.Scenarios)
	sort.Strings(response.Resources)
	sort.Slice(response.Integrations, func(i, j int) bool {
		return response.Integrations[i].Name < response.Integrations[j].Name
	})
	state, err := loadOperatorStateFor(ctx)
	if err != nil {
		return readinessResponse{}, err
	}
	response.ConfigurationRevision = strings.TrimSpace(state.UpdatedAt)
	if response.ConfigurationRevision == "" {
		response.ConfigurationRevision = strings.TrimSpace(state.Version)
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
	// Credential delivery is only finishable when the node's own authority can
	// accept a value. Keep this as a named host fact so a locked login
	// collection is actionable rather than being misreported as a generic
	// credential question.
	storeFact := <-storeCh
	response.Hosts = append(response.Hosts, storeFact)
	if storeFact.Item.Status == "locked" || storeFact.Item.Status == "unresponsive" {
		response.Status = lessReady(response.Status, "missing")
	}
	credentials := <-credentialCh
	if credentials.err != nil {
		return readinessResponse{}, credentials.err
	}
	consumerInventory := <-consumerInventoryCh
	if consumerInventory.err != nil {
		// Inventory is a completion gate, but a source scan failure should remain
		// an actionable readiness verdict rather than turning the whole endpoint
		// into an opaque 500 response.
		response.Blockers = append(response.Blockers, completionBlocker{
			Kind:        "readiness",
			Name:        "credential-consumer-inventory",
			Reason:      "required credential consumer inventory could not be verified",
			Remediation: "resolve the inventory scan condition, then retry readiness",
		})
	} else {
		response.Blockers = append(response.Blockers, consumerInventory.blockers...)
	}
	response.Credentials = credentials.items
	sortCredentialReadiness(response.Credentials)
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
	doctor := <-doctorCh
	if doctor.err == nil && json.Valid(doctor.output) {
		response.CredentialDiagnosis = append(response.CredentialDiagnosis[:0], doctor.output...)
		var diagnosis credentialDiagnosisResponse
		if json.Unmarshal(doctor.output, &diagnosis) == nil {
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
	for _, host := range response.Hosts {
		response.Status = lessReady(response.Status, host.Status)
	}
	release := <-releaseCh
	if release.authorityAvailable {
		response.ManagedKeyConfigured = release.authority.Configured
		response.TrustAnchorMatch = release.authority.TrustAnchorMatch
	}
	response.Integrations = append(response.Integrations, release.item)
	response.Status = lessReady(response.Status, release.item.Status)
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
	result := hostReadiness{Item: readinessItem{Name: item.Name, Status: "deferred", Detail: item.Notes, Remediation: "Choose this optional tool in onboarding when its capability is needed."}, Kind: "tool", Required: item.Required}
	selected := item.Status == "required" || item.Status == "opted_in"
	if !selected {
		return result
	}
	if !hostPlatformSupported(item.Platforms) {
		result.Status = "not_applicable"
		result.Detail = "This tool is not declared for the current operating system."
		result.Remediation = "This capability is not applicable on the current operating system."
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
	result := hostReadiness{Item: readinessItem{Name: item.Name, Status: "deferred", Detail: item.Notes, Remediation: "Review and opt into this optional safeguard only after considering its host-change risk."}, Kind: "safeguard", Required: item.Required}
	selected := item.Status == "required" || item.Status == "opted_in"
	if !selected {
		return result
	}
	if !hostPlatformSupported(item.Platforms) {
		result.Status = "not_applicable"
		result.Detail = "This safeguard is not declared for the current operating system."
		result.Remediation = "This safeguard is not applicable on the current operating system."
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
	result := hostReadiness{Item: readinessItem{Name: item.Name}, Kind: "safeguard", Required: item.Required}

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
	case "unsupported":
		result.Status = "unsupported"
		if result.Detail == "" {
			result.Detail = "This safeguard does not apply to this host."
		}
		result.Remediation = "Deselect this safeguard, or run on a host it supports."
	case "not_applicable":
		result.Status = "not_applicable"
		if result.Detail == "" {
			result.Detail = "This safeguard does not apply to this host."
		}
		result.Remediation = "No action is required on this operating system."
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
	priority := map[string]int{"ready": 0, "deferred": 0, "not_applicable": 0, "degraded": 1, "missing": 2, "unsupported": 3}
	if priority[candidate] > priority[current] {
		return candidate
	}
	return current
}
