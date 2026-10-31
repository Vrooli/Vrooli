// Package hostbroker is Bridge's unprivileged client for the setup-managed
// local privilege broker. It mirrors the versioned JSON contract instead of
// importing the root implementation across Go module boundaries.
package hostbroker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"

	platformgo "github.com/vrooli/platform-go"
)

const (
	protocolVersion = "v1"
	bridgePort      = 18767
	bridgeScenario  = "vrooli-bridge"
)

type Request struct {
	Version   string  `json:"version"`
	RequestID string  `json:"request_id"`
	Action    string  `json:"action"`
	Subject   Subject `json:"subject"`
}
type Subject struct {
	Scenario    string `json:"scenario"`
	CandidateIP string `json:"candidate_ip"`
	Port        int    `json:"port"`
}
type Result struct {
	Version   string   `json:"version"`
	RequestID string   `json:"request_id,omitempty"`
	Action    string   `json:"action,omitempty"`
	Status    string   `json:"status"`
	Code      string   `json:"code,omitempty"`
	Changed   bool     `json:"changed,omitempty"`
	Evidence  Evidence `json:"evidence,omitempty"`
}
type Evidence struct {
	Available bool   `json:"available"`
	Active    bool   `json:"active"`
	RuleFound bool   `json:"rule_found"`
	Managed   bool   `json:"managed"`
	Detail    string `json:"detail,omitempty"`
}

type Client interface {
	Call(context.Context, Request) (Result, error)
}
type SocketClient struct{ SocketPath string }

func NewSocketClient() SocketClient { return SocketClient{SocketPath: socketPath()} }
func socketPath() string {
	if path := strings.TrimSpace(os.Getenv("VROOLI_PRIVILEGE_BROKER_SOCKET")); path != "" {
		return path
	}
	return platformgo.PrivilegeBrokerSocketPath()
}

func (c SocketClient) Call(ctx context.Context, request Request) (Result, error) {
	path := c.SocketPath
	if path == "" {
		path = socketPath()
	}
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		return Result{}, fmt.Errorf("broker unavailable: %w", err)
	}
	defer conn.Close()
	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return Result{}, fmt.Errorf("write broker request: %w", err)
	}
	var result Result
	if err := json.NewDecoder(conn).Decode(&result); err != nil {
		return Result{}, fmt.Errorf("read broker response: %w", err)
	}
	if result.Version != protocolVersion {
		return Result{}, fmt.Errorf("broker protocol mismatch")
	}
	return result, nil
}

func AdmissionRequest(action, requestID, candidateIP string) Request {
	return Request{Version: protocolVersion, RequestID: requestID, Action: action, Subject: Subject{Scenario: bridgeScenario, CandidateIP: candidateIP, Port: bridgePort}}
}
