package effortauthority

// RunInput is the canonical signed native CreateRun input. Native admission
// refuses extra overrides for effort callers; it resolves the existing profile
// and pins its full executable policy separately. TaskDigest pins task contents.
type RunInput struct {
	SerialPayloadDigest string            `json:"serialPayloadDigest,omitempty"`
	TaskID              string            `json:"taskId"`
	TaskDigest          string            `json:"taskDigest"`
	Profile             string            `json:"profile"`
	ParentRunID         string            `json:"parentRunId,omitempty"`
	IdempotencyKey      string            `json:"idempotencyKey"`
	Tag                 string            `json:"tag"`
	Environment         map[string]string `json:"environment,omitempty"`
}
