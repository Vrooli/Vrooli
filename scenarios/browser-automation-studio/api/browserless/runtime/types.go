package runtime

// ExecutionResponse mirrors the payload returned by Browserless execution script.
type ExecutionResponse struct {
    Success         bool         `json:"success"`
    Error           string       `json:"error,omitempty"`
    Steps           []StepResult `json:"steps"`
    TotalDurationMs int          `json:"totalDurationMs,omitempty"`
}

// StepResult captures the outcome of a single instruction.
type StepResult struct {
    Index            int    `json:"index"`
    NodeID           string `json:"nodeId"`
    Type             string `json:"type"`
    StepName         string `json:"stepName,omitempty"`
    Success          bool   `json:"success"`
    Error            string `json:"error,omitempty"`
    Stack            string `json:"stack,omitempty"`
    DurationMs       int    `json:"durationMs,omitempty"`
    ScreenshotBase64 string `json:"screenshotBase64,omitempty"`
}
