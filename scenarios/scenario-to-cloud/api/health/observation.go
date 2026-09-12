// Package health builds the typed deployment health observation
// (vrooli.scenario_to_cloud.v1.health.HealthObservation) from the live-state,
// DNS and TLS report plus the deployment record, and serves it over Connect.
//
// The observation is the only shape consumers may use to decide whether a
// deployment is healthy. It keeps four questions separate that the legacy
// report folded together: is the host present, is the pinned transport
// reachable, is the application ready, and is the observed release the
// expected one. It also separates the deployment verdict (status) from how
// recent the evidence is (freshness): a report can be current and unhealthy,
// or healthy and stale.
package health

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/vps"

	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/errors"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
)

// SchemaVersion is the observation schema version clients negotiate on.
const SchemaVersion = "1"

// ProducerRef identifies this producer on the wire.
const ProducerRef = "scenario-to-cloud:health:v1"

// DefaultMaxObservationAge is the policy window inside which a successful
// live-state inspection counts as CURRENT.
const DefaultMaxObservationAge = 120 * time.Second

// Stable check ids.
const (
	CheckDeploymentRecord     = "deployment_record"
	CheckHostPresence         = "host_presence"
	CheckTransportReach       = "transport_reach"
	CheckApplicationReadiness = "application_readiness"
	CheckReleaseFreshness     = "release_freshness"
	CheckEdgeDNS              = "edge_dns"
	CheckEdgeTLS              = "edge_tls"
	CheckSystemResources      = "system_resources"
)

// Evidence sources named in missing_dependencies.
const (
	DependencyLiveState = "live_state"
	DependencyEdgeDNS   = "edge_dns"
	DependencyEdgeTLS   = "edge_tls"
)

// Policy is the producer's freshness policy.
type Policy struct {
	// MaxObservationAge bounds how old a successful inspection may be and
	// still be reported CURRENT. Zero means DefaultMaxObservationAge.
	MaxObservationAge time.Duration
}

// DefaultPolicy returns the production policy.
func DefaultPolicy() Policy {
	return Policy{MaxObservationAge: DefaultMaxObservationAge}
}

func (p Policy) maxAge() time.Duration {
	if p.MaxObservationAge <= 0 {
		return DefaultMaxObservationAge
	}
	return p.MaxObservationAge
}

// Input is everything the builder needs. Report is the legacy composed
// report from vps.ComputeHealth; LiveState is the raw inspection result the
// report was built from (nil when the inspection was not attempted).
type Input struct {
	Deployment *domain.Deployment
	Report     domain.HealthResponse
	LiveState  *domain.LiveStateResult
	// Now is the producer clock used to compute freshness. Zero means
	// time.Now().
	Now time.Time
}

// Build composes the typed observation. It is pure: no I/O, no clock other
// than Input.Now.
func Build(in Input, policy Policy) *healthv1.HealthObservation {
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()

	obs := &healthv1.HealthObservation{
		ProducerRef: ProducerRef,
	}
	if in.Deployment != nil {
		obs.DeploymentId = in.Deployment.ID
		obs.TargetId = targetID(in.Deployment, in.Report)
		obs.ObservedReleaseDigest = releaseDigest(in.Deployment)
		obs.ObservedConfigurationDigest = configurationDigest(in.Deployment)
	}

	inspected := vps.LiveStateReachable(in.LiveState)
	observedAt := observationTime(in.LiveState, now)
	obs.ObservedAt = timestamppb.New(observedAt)

	sections := sectionsByCategory(in.Report)
	var missing []string

	// host_presence and transport_reach come from the live-state inspection.
	switch {
	case in.LiveState == nil:
		missing = append(missing, DependencyLiveState)
		obs.Checks = append(obs.Checks,
			check(CheckHostPresence, healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE, "live_state_unavailable", "Live-state inspection was not attempted"),
			check(CheckTransportReach, healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE, "transport_unobserved", "Transport identity was not observed"))
	case !inspected:
		missing = append(missing, DependencyLiveState)
		detail := "Target unreachable over SSH"
		if strings.TrimSpace(in.LiveState.Error) != "" {
			detail = "Target unreachable: " + strings.TrimSpace(in.LiveState.Error)
		}
		obs.Checks = append(obs.Checks,
			check(CheckHostPresence, healthv1.CheckStatus_CHECK_STATUS_FAILED, "ssh_unreachable", detail),
			check(CheckTransportReach, healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE, "transport_unobserved", "Transport identity could not be verified because the host was unreachable"))
	default:
		obs.Checks = append(obs.Checks, check(CheckHostPresence, healthv1.CheckStatus_CHECK_STATUS_PASSED, "", "Target answered the live-state inspection"))
		obs.Checks = append(obs.Checks, transportCheck(sections["ssh"]))
	}

	obs.Checks = append(obs.Checks, deploymentRecordCheck(sections["deployment"]))
	obs.Checks = append(obs.Checks, applicationCheck(sections["processes"]))
	obs.Checks = append(obs.Checks, releaseFreshnessCheck(in.Report.Freshness))

	dnsCheck, dnsMissing := edgeCheck(CheckEdgeDNS, sections["dns"], "dns_unavailable", "dns_not_evaluated", "dns_mismatch", "dns_proxied")
	if dnsMissing {
		missing = append(missing, DependencyEdgeDNS)
	}
	obs.Checks = append(obs.Checks, dnsCheck)

	tlsCheck, tlsMissing := edgeCheck(CheckEdgeTLS, sections["tls"], "tls_cert", "tls_not_evaluated", "tls_invalid", "tls_expiring")
	if tlsMissing {
		missing = append(missing, DependencyEdgeTLS)
	}
	obs.Checks = append(obs.Checks, tlsCheck)

	obs.Checks = append(obs.Checks, systemCheck(sections["system"]))

	obs.Status = statusFor(in.Report.Health, inspected)
	obs.Freshness = freshnessFor(inspected, observedAt, now, policy)
	obs.Partial = len(missing) > 0
	obs.MissingDependencies = missing
	obs.NextActions = nextActions(in.Report.Recommendations)
	return obs
}

var jsonOptions = protojson.MarshalOptions{UseProtoNames: true, EmitDefaultValues: true}

// MarshalJSON encodes an observation with proto field names, enum names and
// default values present, so JSON consumers see every mandatory field.
func MarshalJSON(obs *healthv1.HealthObservation) (json.RawMessage, error) {
	if obs == nil {
		return nil, fmt.Errorf("observation is nil")
	}
	raw, err := jsonOptions.Marshal(obs)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

// Response wraps an observation in the versioned envelope.
func Response(obs *healthv1.HealthObservation) *healthv1.GetHealthObservationResponse {
	return &healthv1.GetHealthObservationResponse{SchemaVersion: SchemaVersion, Observation: obs}
}

// MarshalResponseJSON encodes the versioned envelope for the REST surface.
func MarshalResponseJSON(obs *healthv1.HealthObservation) (json.RawMessage, error) {
	if obs == nil {
		return nil, fmt.Errorf("observation is nil")
	}
	raw, err := jsonOptions.Marshal(Response(obs))
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

// --- helpers ---

func targetID(dep *domain.Deployment, report domain.HealthResponse) string {
	if key := dep.Target.Key(); key != "" {
		return key
	}
	if host := strings.TrimSpace(report.Host); host != "" {
		return "host:" + host
	}
	return ""
}

// releaseDigest is "sha256:<bundle_sha256>" until the release domain supplies
// a release digest (plan phase 10).
func releaseDigest(dep *domain.Deployment) string {
	if dep.BundleSHA256 == nil {
		return ""
	}
	return NormalizeDigest(*dep.BundleSHA256)
}

// NormalizeDigest lowercases a hex digest and ensures the "sha256:" prefix.
// An empty input stays empty.
func NormalizeDigest(digest string) string {
	digest = strings.ToLower(strings.TrimSpace(digest))
	if digest == "" {
		return ""
	}
	if strings.Contains(digest, ":") {
		return digest
	}
	return "sha256:" + digest
}

// configurationDigest is a digest over the stored manifest JSON until the
// release domain supplies a configuration digest.
func configurationDigest(dep *domain.Deployment) string {
	if len(dep.Manifest) == 0 {
		return ""
	}
	sum := sha256.Sum256(dep.Manifest)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// observationTime is the producer time of the live-state inspection. When
// the inspection did not succeed, the attempt time (now) is recorded and
// freshness is UNKNOWN, so the value is never read as a successful reading.
func observationTime(live *domain.LiveStateResult, now time.Time) time.Time {
	if vps.LiveStateReachable(live) {
		if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(live.Timestamp)); err == nil {
			return ts.UTC()
		}
	}
	return now
}

func sectionsByCategory(report domain.HealthResponse) map[string]*domain.HealthSection {
	out := make(map[string]*domain.HealthSection, len(report.Sections))
	for i := range report.Sections {
		sec := &report.Sections[i]
		out[sec.Category] = sec
	}
	return out
}

func findCheck(sec *domain.HealthSection, id string) *domain.HealthCheck {
	if sec == nil {
		return nil
	}
	for i := range sec.Checks {
		if sec.Checks[i].ID == id {
			return &sec.Checks[i]
		}
	}
	return nil
}

func check(id string, status healthv1.CheckStatus, reason, detail string) *healthv1.HealthCheck {
	return &healthv1.HealthCheck{Id: id, Status: status, ReasonCode: reason, Detail: detail}
}

func mapCheckStatus(s domain.HealthCheckStatus) healthv1.CheckStatus {
	switch s {
	case domain.HealthCheckPass:
		return healthv1.CheckStatus_CHECK_STATUS_PASSED
	case domain.HealthCheckWarn:
		return healthv1.CheckStatus_CHECK_STATUS_WARNED
	case domain.HealthCheckFail, domain.HealthCheckError:
		return healthv1.CheckStatus_CHECK_STATUS_FAILED
	case domain.HealthCheckSkip:
		return healthv1.CheckStatus_CHECK_STATUS_SKIPPED
	default:
		return healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE
	}
}

func transportCheck(ssh *domain.HealthSection) *healthv1.HealthCheck {
	c := findCheck(ssh, "ssh_key_auth")
	if c == nil {
		return check(CheckTransportReach, healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE, "transport_unobserved", "Transport identity was not reported")
	}
	reason := ""
	switch c.Status {
	case domain.HealthCheckWarn:
		reason = "transport_not_pinned"
	case domain.HealthCheckFail, domain.HealthCheckError:
		reason = "transport_unauthorized"
	}
	return check(CheckTransportReach, mapCheckStatus(c.Status), reason, c.Message)
}

func deploymentRecordCheck(sec *domain.HealthSection) *healthv1.HealthCheck {
	c := findCheck(sec, "deployment_status")
	if c == nil {
		return check(CheckDeploymentRecord, healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE, "record_unavailable", "Deployment record status was not reported")
	}
	reason := ""
	if c.Status != domain.HealthCheckPass {
		reason = "record_" + strings.TrimSpace(c.Details["status"])
	}
	return check(CheckDeploymentRecord, mapCheckStatus(c.Status), reason, c.Message)
}

func applicationCheck(sec *domain.HealthSection) *healthv1.HealthCheck {
	if sec == nil || findCheck(sec, "processes_unavailable") != nil {
		return check(CheckApplicationReadiness, healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE, "process_state_unavailable", "Process state could not be observed")
	}
	var failing []string
	for _, c := range sec.Checks {
		if c.Status == domain.HealthCheckFail || c.Status == domain.HealthCheckError {
			failing = append(failing, c.Message)
		}
	}
	switch {
	case len(failing) > 0:
		return check(CheckApplicationReadiness, healthv1.CheckStatus_CHECK_STATUS_FAILED, "process_not_running", strings.Join(failing, "; "))
	case sec.WarnCount > 0:
		return check(CheckApplicationReadiness, healthv1.CheckStatus_CHECK_STATUS_WARNED, "process_degraded", sec.Title)
	default:
		return check(CheckApplicationReadiness, healthv1.CheckStatus_CHECK_STATUS_PASSED, "", sec.Title)
	}
}

func releaseFreshnessCheck(f *domain.FreshnessStatus) *healthv1.HealthCheck {
	if f == nil {
		return check(CheckReleaseFreshness, healthv1.CheckStatus_CHECK_STATUS_SKIPPED, "freshness_not_evaluated", "Release parity was not evaluated")
	}
	switch f.Status {
	case domain.FreshnessCurrent:
		return check(CheckReleaseFreshness, healthv1.CheckStatus_CHECK_STATUS_PASSED, "", f.Summary)
	case domain.FreshnessOutdated:
		return check(CheckReleaseFreshness, healthv1.CheckStatus_CHECK_STATUS_WARNED, "release_outdated", f.Summary)
	default:
		return check(CheckReleaseFreshness, healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE, "release_parity_unknown", f.Summary)
	}
}

// edgeCheck folds a DNS or TLS section into one check. skipID is the check
// id the legacy report emits when the evaluation was not run; the second
// return value reports that case so it is listed as a missing dependency.
func edgeCheck(id string, sec *domain.HealthSection, skipID, notEvaluated, failReason, warnReason string) (*healthv1.HealthCheck, bool) {
	if sec == nil {
		return check(id, healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE, notEvaluated, "Not evaluated"), true
	}
	if c := findCheck(sec, skipID); c != nil && c.Status == domain.HealthCheckSkip {
		return check(id, healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE, notEvaluated, c.Message), true
	}
	var details []string
	for _, c := range sec.Checks {
		if c.Status == domain.HealthCheckFail || c.Status == domain.HealthCheckError || c.Status == domain.HealthCheckWarn {
			details = append(details, c.Message)
		}
	}
	switch sec.Status {
	case domain.HealthCheckFail, domain.HealthCheckError:
		return check(id, healthv1.CheckStatus_CHECK_STATUS_FAILED, failReason, strings.Join(details, "; ")), false
	case domain.HealthCheckWarn:
		return check(id, healthv1.CheckStatus_CHECK_STATUS_WARNED, warnReason, strings.Join(details, "; ")), false
	case domain.HealthCheckPass:
		return check(id, healthv1.CheckStatus_CHECK_STATUS_PASSED, "", sec.Title), false
	default:
		return check(id, healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE, notEvaluated, sec.Title), true
	}
}

func systemCheck(sec *domain.HealthSection) *healthv1.HealthCheck {
	if sec == nil || findCheck(sec, "system_unavailable") != nil {
		return check(CheckSystemResources, healthv1.CheckStatus_CHECK_STATUS_UNAVAILABLE, "system_metrics_unavailable", "System metrics could not be observed")
	}
	var details []string
	for _, c := range sec.Checks {
		if c.Status != domain.HealthCheckPass {
			details = append(details, c.Message)
		}
	}
	switch sec.Status {
	case domain.HealthCheckFail, domain.HealthCheckError:
		return check(CheckSystemResources, healthv1.CheckStatus_CHECK_STATUS_FAILED, "resource_exhausted", strings.Join(details, "; "))
	case domain.HealthCheckWarn:
		return check(CheckSystemResources, healthv1.CheckStatus_CHECK_STATUS_WARNED, "resource_pressure", strings.Join(details, "; "))
	default:
		return check(CheckSystemResources, healthv1.CheckStatus_CHECK_STATUS_PASSED, "", sec.Title)
	}
}

// statusFor maps the legacy overall level onto the typed verdict. A failed
// inspection is UNKNOWN unless the record itself is in a terminal bad state.
func statusFor(level domain.HealthLevel, inspected bool) healthv1.HealthStatus {
	switch level {
	case domain.HealthHealthy:
		if !inspected {
			return healthv1.HealthStatus_HEALTH_STATUS_UNKNOWN
		}
		return healthv1.HealthStatus_HEALTH_STATUS_HEALTHY
	case domain.HealthDegraded:
		if !inspected {
			return healthv1.HealthStatus_HEALTH_STATUS_UNKNOWN
		}
		return healthv1.HealthStatus_HEALTH_STATUS_DEGRADED
	case domain.HealthUnhealthy, domain.HealthFailed, domain.HealthStopped:
		return healthv1.HealthStatus_HEALTH_STATUS_UNHEALTHY
	default: // pending, starting, unknown
		return healthv1.HealthStatus_HEALTH_STATUS_UNKNOWN
	}
}

func freshnessFor(inspected bool, observedAt, now time.Time, policy Policy) healthv1.Freshness {
	if !inspected || observedAt.IsZero() {
		return healthv1.Freshness_FRESHNESS_UNKNOWN
	}
	age := now.Sub(observedAt)
	if age < 0 {
		age = 0
	}
	if age <= policy.maxAge() {
		return healthv1.Freshness_FRESHNESS_CURRENT
	}
	return healthv1.Freshness_FRESHNESS_STALE
}

func nextActions(recs []domain.Recommendation) []*errorsv1.NextAction {
	var out []*errorsv1.NextAction
	for _, rec := range recs {
		if strings.TrimSpace(rec.Command) == "" {
			continue
		}
		out = append(out, &errorsv1.NextAction{
			Owner:     "scenario-to-cloud",
			Kind:      "command",
			Reference: rec.Command,
			Label:     rec.Summary,
		})
	}
	return out
}
