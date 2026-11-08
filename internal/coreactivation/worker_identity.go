package coreactivation

// WorkerIdentity is the bounded read-only capability response of the retained
// watchdog image. Build role and frozen bytes must be checked separately.
type WorkerIdentity struct {
	Protocol                      int    `json:"protocol"`
	BuildIdentity                 string `json:"build_identity"`
	RuntimeHomeContractSHA256     string `json:"runtime_home_contract_sha256"`
	PersistentStateContractSHA256 string `json:"persistent_state_contract_sha256"`
}
