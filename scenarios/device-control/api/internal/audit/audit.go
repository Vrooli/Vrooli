package audit

import "time"

type Record struct {
	ID                string    `json:"id"`
	Actor             string    `json:"actor"`
	DeviceID          string    `json:"device_id"`
	LeaseID           string    `json:"lease_id"`
	Verb              string    `json:"verb"`
	Outcome           string    `json:"outcome"`
	ProfileID         string    `json:"profile_id,omitempty"`
	Method            string    `json:"method,omitempty"`
	Attempts          int       `json:"attempts,omitempty"`
	ProviderState     string    `json:"provider_state,omitempty"`
	BeforeLockState   string    `json:"before_lock_state,omitempty"`
	AfterLockState    string    `json:"after_lock_state,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	RedactionVerified bool      `json:"redaction_verified"`
	RedactionOptedOut bool      `json:"redaction_opted_out,omitempty"`
}
