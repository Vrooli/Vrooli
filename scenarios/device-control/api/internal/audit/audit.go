package audit

import "time"

type Record struct {
	ID                string    `json:"id"`
	Actor             string    `json:"actor"`
	DeviceID          string    `json:"device_id"`
	LeaseID           string    `json:"lease_id"`
	Verb              string    `json:"verb"`
	Outcome           string    `json:"outcome"`
	CreatedAt         time.Time `json:"created_at"`
	RedactionVerified bool      `json:"redaction_verified"`
	RedactionOptedOut bool      `json:"redaction_opted_out,omitempty"`
}
