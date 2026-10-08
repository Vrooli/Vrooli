package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
)

// DefaultCloudHealthMaxAge is how old a cloud observation may be, measured on
// the DM clock from the producer's observed_at, and still gate a release.
const DefaultCloudHealthMaxAge = 120 * time.Second

// cloudHealthSchemaVersion is the only observation schema DM understands.
const cloudHealthSchemaVersion = "1"

// Stable reason codes a health verdict can carry. They are DM-owned; cloud
// error codes (deployment_not_found, deployment_selector_ambiguous, ...) pass
// through unchanged when the cloud service refuses a request.
const (
	HealthReasonTransportError      = "transport_error"
	HealthReasonMalformedReport     = "malformed_report"
	HealthReasonUnsupportedSchema   = "unsupported_schema_version"
	HealthReasonDeploymentMismatch  = "deployment_mismatch"
	HealthReasonEnvironmentMismatch = "environment_mismatch"
	HealthReasonReleaseMismatch     = "release_mismatch"
	HealthReasonStatusUnknown       = "status_unknown"
	HealthReasonStatusNotHealthy    = "status_not_healthy"
	HealthReasonFreshnessUnknown    = "freshness_unknown"
	HealthReasonStale               = "stale_observation"
	HealthReasonObservedAtMissing   = "observed_at_missing"
	HealthReasonPartialObservation  = "partial_observation"
)

// DeploymentSelector names a cloud deployment by identity facets when the
// stable id is not yet known. ScenarioID is mandatory; the client uses the
// first non-empty facet in the order Domain, Environment, Host, because the
// cloud service accepts exactly one facet beside the scenario. Environment,
// when given alongside another facet, is verified against the resolved record.
type DeploymentSelector struct {
	ScenarioID  string
	Environment string
	Domain      string
	Host        string
}

// IsZero reports whether no facet is set.
func (s DeploymentSelector) IsZero() bool {
	return strings.TrimSpace(s.ScenarioID) == "" && strings.TrimSpace(s.Environment) == "" && strings.TrimSpace(s.Domain) == "" && strings.TrimSpace(s.Host) == ""
}

// DeploymentHealthRequest asks whether one exact deployment is healthy now.
type DeploymentHealthRequest struct {
	// DeploymentID is the stable cloud deployment id. When empty, Selector
	// is resolved through the cloud service's resolve endpoint.
	DeploymentID string
	Selector     DeploymentSelector
	// ExpectedReleaseDigest, when set, must equal the observed release digest
	// (bundle sha256, with or without the "sha256:" prefix).
	ExpectedReleaseDigest string
	// MaxAge bounds the observation age on the DM clock. Zero means
	// DefaultCloudHealthMaxAge.
	MaxAge time.Duration
}

// CloudNextAction is a typed remedy the cloud service attached to a verdict.
type CloudNextAction struct {
	Owner     string `json:"owner,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Reference string `json:"reference,omitempty"`
	Label     string `json:"label,omitempty"`
}

// CloudHealthResult is the DM verdict over one cloud observation. Healthy is
// true only when every gate in evaluateCloudHealthObservation passed.
type CloudHealthResult struct {
	Healthy               bool              `json:"healthy"`
	ReasonCode            string            `json:"reason_code,omitempty"`
	Details               string            `json:"details,omitempty"`
	DeploymentID          string            `json:"deployment_id,omitempty"`
	Status                string            `json:"status,omitempty"`
	Freshness             string            `json:"freshness,omitempty"`
	ObservedAt            time.Time         `json:"observed_at,omitempty"`
	ObservedReleaseDigest string            `json:"observed_release_digest,omitempty"`
	NextActions           []CloudNextAction `json:"next_actions,omitempty"`
}

// Explain renders the verdict for a step message.
func (r *CloudHealthResult) Explain() string {
	if r == nil {
		return "no result"
	}
	var parts []string
	if r.ReasonCode != "" {
		parts = append(parts, r.ReasonCode)
	}
	if r.Details != "" {
		parts = append(parts, r.Details)
	}
	for _, action := range r.NextActions {
		label := strings.TrimSpace(action.Label)
		ref := strings.TrimSpace(action.Reference)
		switch {
		case label != "" && ref != "":
			parts = append(parts, "next: "+label+" ("+ref+")")
		case ref != "":
			parts = append(parts, "next: "+ref)
		case label != "":
			parts = append(parts, "next: "+label)
		}
	}
	return strings.Join(parts, "; ")
}

func unhealthy(reason, details string) *CloudHealthResult {
	return &CloudHealthResult{Healthy: false, ReasonCode: reason, Details: details}
}

// CheckDeploymentHealth resolves the exact deployment, fetches its typed
// observation and applies the fail-closed gates. Transport failures and
// cloud refusals are verdicts (Healthy=false with a reason), not Go errors;
// an error is returned only for an unusable request.
func (c *HTTPCloudHealthClient) CheckDeploymentHealth(ctx context.Context, request DeploymentHealthRequest) (*CloudHealthResult, error) {
	deploymentID, refusal, err := c.resolveCloudDeployment(ctx, request)
	if err != nil {
		return nil, err
	}
	if refusal != nil {
		return refusal, nil
	}
	body, status, err := c.cloudGet(ctx, "/api/v1/deployments/"+url.PathEscape(deploymentID)+"/health/observation")
	if err != nil {
		result := unhealthy(HealthReasonTransportError, err.Error())
		result.DeploymentID = deploymentID
		return result, nil
	}
	if status != http.StatusOK {
		result := cloudRefusal(status, body)
		result.DeploymentID = deploymentID
		return result, nil
	}
	result := evaluateCloudHealthObservation(body, request, deploymentID, time.Now())
	return result, nil
}

// resolveCloudDeployment returns the stable deployment id for the request,
// resolving the selector through the cloud service when needed. The second
// return value is a verdict when the cloud service refused the selector.
func (c *HTTPCloudHealthClient) resolveCloudDeployment(ctx context.Context, request DeploymentHealthRequest) (string, *CloudHealthResult, error) {
	if id := strings.TrimSpace(request.DeploymentID); id != "" {
		return id, nil, nil
	}
	sel := request.Selector
	if strings.TrimSpace(sel.ScenarioID) == "" {
		return "", nil, fmt.Errorf("cloud health request needs a deployment id or a scenario selector")
	}
	query := url.Values{}
	query.Set("scenario", strings.TrimSpace(sel.ScenarioID))
	switch {
	case strings.TrimSpace(sel.Domain) != "":
		query.Set("domain", strings.TrimSpace(sel.Domain))
	case strings.TrimSpace(sel.Environment) != "":
		query.Set("environment", strings.TrimSpace(sel.Environment))
	case strings.TrimSpace(sel.Host) != "":
		query.Set("host", strings.TrimSpace(sel.Host))
	default:
		return "", nil, fmt.Errorf("cloud health selector for %s needs a domain, environment or host", sel.ScenarioID)
	}
	body, status, err := c.cloudGet(ctx, "/api/v1/deployments/resolve?"+query.Encode())
	if err != nil {
		return "", unhealthy(HealthReasonTransportError, "resolve deployment: "+err.Error()), nil
	}
	if status != http.StatusOK {
		return "", cloudRefusal(status, body), nil
	}
	var resolved struct {
		SchemaVersion string `json:"schema_version"`
		Ref           struct {
			ID          string `json:"id"`
			ScenarioID  string `json:"scenario_id"`
			Environment string `json:"environment"`
		} `json:"ref"`
	}
	if err := json.Unmarshal(body, &resolved); err != nil || strings.TrimSpace(resolved.Ref.ID) == "" {
		return "", unhealthy(HealthReasonMalformedReport, "resolve endpoint returned no deployment identity"), nil
	}
	if resolved.SchemaVersion != cloudHealthSchemaVersion {
		return "", unhealthy(HealthReasonUnsupportedSchema, "resolve schema_version "+resolved.SchemaVersion), nil
	}
	if resolved.Ref.ScenarioID != "" && resolved.Ref.ScenarioID != strings.TrimSpace(sel.ScenarioID) {
		return "", unhealthy(HealthReasonDeploymentMismatch, fmt.Sprintf("resolved scenario %q is not %q", resolved.Ref.ScenarioID, sel.ScenarioID)), nil
	}
	if env := strings.TrimSpace(sel.Environment); env != "" && resolved.Ref.Environment != "" && resolved.Ref.Environment != env {
		result := unhealthy(HealthReasonEnvironmentMismatch, fmt.Sprintf("resolved deployment %s is environment %q, expected %q", resolved.Ref.ID, resolved.Ref.Environment, env))
		result.DeploymentID = resolved.Ref.ID
		return "", result, nil
	}
	return resolved.Ref.ID, nil, nil
}

// evaluateCloudHealthObservation is the single fail-closed interpretation of
// a cloud observation body. Every gate names its reason so a refused
// promotion is explainable.
func evaluateCloudHealthObservation(body []byte, request DeploymentHealthRequest, deploymentID string, now time.Time) *CloudHealthResult {
	var envelope healthv1.GetHealthObservationResponse
	// Unknown fields are forward-compatible (DiscardUnknown); unknown
	// mandatory enum values are not, so they are rejected before decoding
	// because protojson would silently discard them to UNSPECIFIED.
	if err := rejectUnknownEnumValues(body); err != nil {
		result := unhealthy(HealthReasonMalformedReport, err.Error())
		result.DeploymentID = deploymentID
		return result
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, &envelope); err != nil {
		result := unhealthy(HealthReasonMalformedReport, "observation did not decode: "+err.Error())
		result.DeploymentID = deploymentID
		return result
	}
	obs := envelope.GetObservation()
	result := &CloudHealthResult{
		DeploymentID:          deploymentID,
		Status:                obs.GetStatus().String(),
		Freshness:             obs.GetFreshness().String(),
		ObservedReleaseDigest: obs.GetObservedReleaseDigest(),
	}
	for _, action := range obs.GetNextActions() {
		result.NextActions = append(result.NextActions, CloudNextAction{Owner: action.GetOwner(), Kind: action.GetKind(), Reference: action.GetReference(), Label: action.GetLabel()})
	}
	if obs.GetObservedAt() != nil {
		result.ObservedAt = obs.GetObservedAt().AsTime()
	}
	fail := func(reason, details string) *CloudHealthResult {
		result.Healthy = false
		result.ReasonCode = reason
		result.Details = details
		return result
	}
	if envelope.GetSchemaVersion() != cloudHealthSchemaVersion {
		return fail(HealthReasonUnsupportedSchema, "observation schema_version "+envelope.GetSchemaVersion())
	}
	if obs == nil || strings.TrimSpace(obs.GetDeploymentId()) == "" {
		return fail(HealthReasonMalformedReport, "observation carries no deployment identity")
	}
	if obs.GetDeploymentId() != deploymentID {
		return fail(HealthReasonDeploymentMismatch, fmt.Sprintf("observation is about deployment %q, expected %q", obs.GetDeploymentId(), deploymentID))
	}
	switch obs.GetStatus() {
	case healthv1.HealthStatus_HEALTH_STATUS_HEALTHY:
	case healthv1.HealthStatus_HEALTH_STATUS_UNSPECIFIED, healthv1.HealthStatus_HEALTH_STATUS_UNKNOWN:
		return fail(HealthReasonStatusUnknown, "deployment health is "+obs.GetStatus().String()+": "+failingChecks(obs))
	default:
		return fail(HealthReasonStatusNotHealthy, "deployment health is "+obs.GetStatus().String()+": "+failingChecks(obs))
	}
	switch obs.GetFreshness() {
	case healthv1.Freshness_FRESHNESS_CURRENT:
	case healthv1.Freshness_FRESHNESS_STALE:
		return fail(HealthReasonStale, "producer reports the observation as stale")
	default:
		return fail(HealthReasonFreshnessUnknown, "producer could not establish observation freshness")
	}
	if obs.GetObservedAt() == nil || !obs.GetObservedAt().IsValid() || result.ObservedAt.IsZero() {
		return fail(HealthReasonObservedAtMissing, "observation has no observed_at")
	}
	maxAge := request.MaxAge
	if maxAge <= 0 {
		maxAge = DefaultCloudHealthMaxAge
	}
	if age := now.Sub(result.ObservedAt); age > maxAge {
		return fail(HealthReasonStale, fmt.Sprintf("observation is %s old, older than %s", age.Truncate(time.Second), maxAge))
	}
	if expected := normalizeDigest(request.ExpectedReleaseDigest); expected != "" {
		observed := normalizeDigest(obs.GetObservedReleaseDigest())
		if observed != expected {
			return fail(HealthReasonReleaseMismatch, fmt.Sprintf("observed release %q is not the expected %q", observed, expected))
		}
	}
	if obs.GetPartial() {
		return fail(HealthReasonPartialObservation, "observation is partial; missing "+strings.Join(obs.GetMissingDependencies(), ", "))
	}
	result.Healthy = true
	return result
}

// rejectUnknownEnumValues fails when a mandatory status vocabulary carries a
// value this consumer does not know. A newer producer vocabulary must never
// read as UNSPECIFIED-and-ignored on the promotion path.
func rejectUnknownEnumValues(body []byte) error {
	var raw struct {
		Observation struct {
			Status    json.RawMessage `json:"status"`
			Freshness json.RawMessage `json:"freshness"`
			Checks    []struct {
				ID     string          `json:"id"`
				Status json.RawMessage `json:"status"`
			} `json:"checks"`
		} `json:"observation"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("observation did not decode: %w", err)
	}
	if err := knownEnum("status", raw.Observation.Status, healthv1.HealthStatus_value); err != nil {
		return err
	}
	if err := knownEnum("freshness", raw.Observation.Freshness, healthv1.Freshness_value); err != nil {
		return err
	}
	for _, c := range raw.Observation.Checks {
		if err := knownEnum("checks["+c.ID+"].status", c.Status, healthv1.CheckStatus_value); err != nil {
			return err
		}
	}
	return nil
}

func knownEnum(field string, raw json.RawMessage, values map[string]int32) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		var number int32
		if json.Unmarshal(raw, &number) == nil {
			for _, v := range values {
				if v == number {
					return nil
				}
			}
		}
		return fmt.Errorf("unknown %s value %s", field, string(raw))
	}
	if _, ok := values[name]; !ok {
		return fmt.Errorf("unknown %s value %q", field, name)
	}
	return nil
}

func failingChecks(obs *healthv1.HealthObservation) string {
	var parts []string
	for _, c := range obs.GetChecks() {
		switch c.GetStatus() {
		case healthv1.CheckStatus_CHECK_STATUS_PASSED, healthv1.CheckStatus_CHECK_STATUS_SKIPPED:
			continue
		}
		part := c.GetId() + "=" + strings.TrimPrefix(c.GetStatus().String(), "CHECK_STATUS_")
		if c.GetReasonCode() != "" {
			part += "(" + c.GetReasonCode() + ")"
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return "no failing checks reported"
	}
	return strings.Join(parts, ", ")
}

// normalizeDigest lowercases a digest and gives a bare hex value the
// "sha256:" prefix so receipt and observation digests compare equal.
func normalizeDigest(digest string) string {
	digest = strings.ToLower(strings.TrimSpace(digest))
	if digest == "" || strings.Contains(digest, ":") {
		return digest
	}
	return "sha256:" + digest
}

// cloudRefusal turns a non-200 cloud response into a verdict carrying the
// cloud service's stable error code and next action when the body is typed.
func cloudRefusal(status int, body []byte) *CloudHealthResult {
	var typed struct {
		Error struct {
			Code       string           `json:"code"`
			Message    string           `json:"message"`
			NextAction *CloudNextAction `json:"next_action"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &typed); err == nil && strings.TrimSpace(typed.Error.Code) != "" {
		result := unhealthy(typed.Error.Code, typed.Error.Message)
		if typed.Error.NextAction != nil {
			result.NextActions = []CloudNextAction{*typed.Error.NextAction}
		}
		return result
	}
	return unhealthy(HealthReasonTransportError, fmt.Sprintf("status %d: %s", status, strings.TrimSpace(string(body))))
}

func (c *HTTPCloudHealthClient) cloudGet(ctx context.Context, path string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}
	return body, resp.StatusCode, nil
}

// cloudManifestSelector derives the deployment selector from a cloud
// manifest: scenario id, environment, edge domain and VPS host.
func cloudManifestSelector(raw json.RawMessage) DeploymentSelector {
	if len(raw) == 0 {
		return DeploymentSelector{}
	}
	var envelope struct {
		Environment string `json:"environment"`
		Scenario    struct {
			ID string `json:"id"`
		} `json:"scenario"`
		Edge struct {
			Domain string `json:"domain"`
		} `json:"edge"`
		Target struct {
			VPS struct {
				Host string `json:"host"`
			} `json:"vps"`
		} `json:"target"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return DeploymentSelector{}
	}
	return DeploymentSelector{
		ScenarioID:  strings.TrimSpace(envelope.Scenario.ID),
		Environment: strings.TrimSpace(envelope.Environment),
		Domain:      strings.TrimSpace(envelope.Edge.Domain),
		Host:        strings.TrimSpace(envelope.Target.VPS.Host),
	}
}
