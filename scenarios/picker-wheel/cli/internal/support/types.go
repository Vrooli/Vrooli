package support

import "time"

// Option mirrors the api Option shape: a labeled slice of the wheel with an
// optional color and weight used by the server's weighted-random selection.
type Option struct {
	Label  string  `json:"label"`
	Color  string  `json:"color,omitempty"`
	Weight float64 `json:"weight,omitempty"`
}

// Wheel mirrors the shape returned by /api/wheels and /api/wheels/{id}.
type Wheel struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Options     []Option  `json:"options"`
	Theme       string    `json:"theme,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	TimesUsed   int       `json:"times_used,omitempty"`
}

// SpinResult mirrors the response shape from POST /api/spin and entries from
// GET /api/history.
type SpinResult struct {
	Result    string    `json:"result"`
	WheelID   string    `json:"wheel_id,omitempty"`
	SessionID string    `json:"session_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
