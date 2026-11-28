package businessaccount

import "time"

// Account is the LPBS-owned commercial context selected during a desktop link
// flow. Membership is authoritative: callers must not infer access from the
// account ID alone.
type Account struct {
	ID           string    `json:"id"`
	DisplayName  string    `json:"display_name"`
	BillingEmail string    `json:"billing_email"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}
