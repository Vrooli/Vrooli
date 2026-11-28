// Package claims provides CLI access to the test-genie playbooks
// concurrency-guard claims endpoints.
package claims

import "time"

// Claim mirrors the JSON shape returned by /api/v1/playbooks/claims.
type Claim struct {
	ScenarioName string    `json:"scenario_name"`
	RunID        string    `json:"run_id"`
	Mode         string    `json:"mode"`
	StartedBy    string    `json:"started_by"`
	AcquiredAt   time.Time `json:"acquired_at"`
	HeartbeatAt  time.Time `json:"heartbeat_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	Alive        bool      `json:"alive"`
}

// listResponse wraps the list payload.
type listResponse struct {
	Claims []Claim `json:"claims"`
}

// getResponse wraps the get payload. Claim is nil when no claim is held.
type getResponse struct {
	Claim *Claim `json:"claim"`
}

// releaseResponse wraps the force-release payload.
type releaseResponse struct {
	Released Claim `json:"released"`
}
