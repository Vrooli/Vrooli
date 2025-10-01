package main

import (
	"encoding/json"
	"time"
)

type createSessionRequest struct {
	Operator string          `json:"operator,omitempty"`
	Reason   string          `json:"reason,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
	Command  string          `json:"command,omitempty"`
	Args     []string        `json:"args,omitempty"`
	TabID    string          `json:"tabId,omitempty"`
}

type createSessionResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	Command   string    `json:"command"`
	Args      []string  `json:"args"`
}

type apiError struct {
	Message string `json:"message"`
}

type websocketEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type outputPayload struct {
	Data      string    `json:"data"`
	Encoding  string    `json:"encoding"`
	Direction string    `json:"direction"`
	Timestamp time.Time `json:"timestamp"`
}

type statusPayload struct {
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type inputPayload struct {
	Data     string `json:"data"`
	Encoding string `json:"encoding"`
}

type heartbeatPayload struct {
	Timestamp time.Time `json:"timestamp"`
}

type resizePayload struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

type sessionSummary struct {
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	ExpiresAt    time.Time `json:"expiresAt"`
	LastActivity time.Time `json:"lastActivity"`
	State        string    `json:"state"`
	Command      string    `json:"command"`
	Args         []string  `json:"args"`
}

type transcriptDirection string

const (
	directionStdout transcriptDirection = "stdout"
	directionStderr transcriptDirection = "stderr"
	directionStdin  transcriptDirection = "stdin"
	directionStatus transcriptDirection = "status"
)

type transcriptEntry struct {
	Timestamp time.Time           `json:"timestamp"`
	Direction transcriptDirection `json:"direction"`
	Data      string              `json:"data,omitempty"`
	Encoding  string              `json:"encoding,omitempty"`
	Message   string              `json:"message,omitempty"`
}
