package coreactivation

import "github.com/vrooli/vrooli/internal/stateformat"

// workerIdentityLimit bounds the capability response. It carries the versioned
// state-format manifest, which is several times larger than the legacy digest
// fields alone; the bound is still far below a general response size.
const workerIdentityLimit = 64 << 10

// WorkerIdentity is the bounded read-only capability response of the retained
// watchdog image. Build role and frozen bytes must be checked separately.
type WorkerIdentity struct {
	Protocol                      int    `json:"protocol"`
	BuildIdentity                 string `json:"build_identity"`
	RuntimeHomeContractSHA256     string `json:"runtime_home_contract_sha256"`
	PersistentStateContractSHA256 string `json:"persistent_state_contract_sha256"`
	// StateFormat is the versioned compatibility manifest the watchdog speaks.
	// It is a pointer so a watchdog built before this field decodes as absent
	// and re-encodes canonically; such an image falls back to the identity rule
	// in ProbeWorkerIdentity.
	StateFormat *stateformat.Manifest `json:"state_format,omitempty"`
}
