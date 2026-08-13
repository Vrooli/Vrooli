package execution

import (
	"device-control/internal/evidence"
	"device-control/internal/sessions"
)

type Step struct {
	ID                   string         `json:"id"`
	Kind                 string         `json:"kind"`
	RequiredCapabilities []string       `json:"required_capabilities"`
	Target               string         `json:"target"`
	TimeoutMS            int64          `json:"timeout_ms"`
	Arguments            map[string]any `json:"arguments"`
}
type Flow struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Steps                  []Step `json:"steps"`
	Transport              string `json:"transport,omitempty"`
	AllowUnredactedCapture bool   `json:"allow_unredacted_capture"`
	SuppressActuation      bool   `json:"suppress_actuation,omitempty"`
}
type GapReport struct {
	Runnable bool     `json:"runnable"`
	Gaps     []string `json:"gaps"`
	Warnings []string `json:"warnings"`
}
type Chapter struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Disposition string `json:"disposition"`
	Message     string `json:"message"`
}
type Resolution struct {
	Target     string  `json:"target"`
	Rung       string  `json:"rung"`
	Confidence float64 `json:"confidence"`
}
type RunResult struct {
	RunID            string                      `json:"run_id"`
	Disposition      string                      `json:"disposition"`
	Chapters         []Chapter                   `json:"chapters"`
	Resolutions      []Resolution                `json:"resolutions"`
	Evidence         []evidence.Reference        `json:"evidence"`
	Restoration      []sessions.RestorationEvent `json:"restoration,omitempty"`
	Incomplete       bool                        `json:"incomplete,omitempty"`
	DisconnectReason string                      `json:"disconnect_reason,omitempty"`
	DisconnectStep   string                      `json:"disconnect_step,omitempty"`
}
