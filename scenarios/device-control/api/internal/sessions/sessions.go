package sessions

import "time"

type Session struct {
	ID         string    `json:"id"`
	DeviceID   string    `json:"device_id"`
	Actor      string    `json:"actor"`
	State      string    `json:"state"`
	LeaseToken string    `json:"lease_token"`
	KillReason string    `json:"kill_reason,omitempty"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}
