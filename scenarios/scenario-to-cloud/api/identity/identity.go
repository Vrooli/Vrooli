// Package identity owns the canonical cloud identities: deployment, target,
// release and operation references. Every other scenario-to-cloud package
// consumes these types instead of copying DTO fields. Wire shapes live in
// vrooli.scenario_to_cloud.v1.identity; these are the domain mirrors.
package identity

import "strings"

// Transport values a target binding may select. There is no fallback between
// them: a revoked bridge enrollment is a typed failure, never an SSH retry.
const (
	TransportBridge = "bridge"
	TransportSSH    = "ssh"
)

// DefaultEnvironment is the environment assigned to records that predate the
// environment column.
const DefaultEnvironment = "production"

// DeploymentRef is the stable identity of one installed scenario on one target
// in one environment. ID never changes for the life of the record.
type DeploymentRef struct {
	ID          string    `json:"id"`
	ScenarioID  string    `json:"scenario_id"`
	Environment string    `json:"environment"`
	Target      TargetRef `json:"target"`
}

// TargetRef composes the Bridge machine identity with the transport used to
// reach it. MachineID and NodeID are empty for SSH-only targets that have not
// been enrolled with the Bridge.
type TargetRef struct {
	MachineID            string        `json:"machine_id,omitempty"`
	NodeID               string        `json:"node_id,omitempty"`
	EnrollmentGeneration uint64        `json:"enrollment_generation,omitempty"`
	Transport            string        `json:"transport"`
	Locator              TargetLocator `json:"locator"`
}

// TargetLocator is mutable reachability metadata. It is never identity.
type TargetLocator struct {
	Host    string `json:"host,omitempty"`
	Port    int    `json:"port,omitempty"`
	User    string `json:"user,omitempty"`
	Workdir string `json:"workdir,omitempty"`
}

// ReleaseRef identifies an immutable release by content digests.
type ReleaseRef struct {
	Digest              string `json:"digest"`
	ProvenanceRef       string `json:"provenance_ref,omitempty"`
	ClosureDigest       string `json:"closure_digest,omitempty"`
	ConfigurationDigest string `json:"configuration_digest,omitempty"`
}

// OperationRef identifies one durable operation admitted against a
// deployment. Fence is the deployment fence captured at acquisition.
type OperationRef struct {
	ID         string `json:"id"`
	RequestKey string `json:"request_key"`
	PlanDigest string `json:"plan_digest"`
	Fence      uint64 `json:"fence"`
}

// Key returns the denormalised uniqueness key for a target. A Bridge-enrolled
// machine is keyed by its machine id; an SSH-only target is keyed by its host.
// The prefixes keep the two namespaces from colliding in the unique index
// (scenario_id, environment, target_key).
func (t TargetRef) Key() string {
	if machine := strings.TrimSpace(t.MachineID); machine != "" {
		return "machine:" + machine
	}
	if host := strings.TrimSpace(t.Locator.Host); host != "" {
		return "host:" + host
	}
	return ""
}

// IsZero reports whether no identity or locator has been bound.
func (t TargetRef) IsZero() bool {
	return t.MachineID == "" && t.NodeID == "" && t.EnrollmentGeneration == 0 &&
		t.Transport == "" && t.Locator == TargetLocator{}
}

// NormalizeEnvironment maps an empty environment to the default so callers and
// storage agree on one spelling.
func NormalizeEnvironment(environment string) string {
	environment = strings.TrimSpace(environment)
	if environment == "" {
		return DefaultEnvironment
	}
	return environment
}
