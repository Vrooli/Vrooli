package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	agentharness "github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	hookReceiptURLKey   = "VROOLI_AM_HOOK_RECEIPT_URL"
	hookReceiptTokenKey = "VROOLI_AM_HOOK_RECEIPT_TOKEN"
	hookInvocationIDKey = "VROOLI_AM_HOOK_RECEIPT_INVOCATION_ID"
	identityTokenKey    = "VROOLI_AGENT_IDENTITY_TOKEN"
	maxReceiptURLBytes  = 2048
	maxReceiptBodyBytes = 4096
	maxReceiptReplySize = 4096
	hookReceiptTimeout  = 2 * time.Second
	hookReceiptPath     = "/api/v1/runtime/hook-receipts"
)

// evaluatorReceipt contains only evaluator-owned facts and bounded run
// correlation. In particular, hook input, command arguments and decision
// reasons never cross this boundary.
type evaluatorReceipt struct {
	ReceiptID    string    `json:"receipt_id"`
	InvocationID string    `json:"invocation_id"`
	SessionID    string    `json:"session_id"`
	Runner       string    `json:"runner"`
	EvaluatorID  string    `json:"evaluator_id"`
	DurationMS   int64     `json:"duration_ms"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type hookReceiptConfig struct {
	endpoint      *url.URL
	receiptToken  string
	identityToken string
	invocationID  string
}

func loadHookReceiptConfig(lookupEnv func(string) string) (hookReceiptConfig, error) {
	if lookupEnv == nil {
		return hookReceiptConfig{}, errors.New("missing environment reader")
	}
	endpointRaw := strings.TrimSpace(lookupEnv(hookReceiptURLKey))
	if endpointRaw == "" || len(endpointRaw) > maxReceiptURLBytes {
		return hookReceiptConfig{}, errors.New("receipt endpoint missing or too long")
	}
	endpoint, err := url.ParseRequestURI(endpointRaw)
	if err != nil || endpoint == nil || endpoint.Host == "" || endpoint.User != nil || endpoint.Fragment != "" || endpoint.RawQuery != "" {
		return hookReceiptConfig{}, errors.New("receipt endpoint is malformed")
	}
	if endpoint.Scheme != "https" && endpoint.Scheme != "http" {
		return hookReceiptConfig{}, errors.New("receipt endpoint scheme is unsupported")
	}
	if endpoint.Path != hookReceiptPath {
		return hookReceiptConfig{}, errors.New("receipt endpoint path is unsupported")
	}
	if endpoint.Scheme == "http" && !isLoopbackHost(endpoint.Hostname()) {
		return hookReceiptConfig{}, errors.New("plain HTTP receipt endpoint must be loopback")
	}
	if hasControl(endpointRaw) {
		return hookReceiptConfig{}, errors.New("receipt endpoint contains invalid characters")
	}
	receiptToken := strings.TrimSpace(lookupEnv(hookReceiptTokenKey))
	identityToken := strings.TrimSpace(lookupEnv(identityTokenKey))
	invocationID := strings.TrimSpace(lookupEnv(hookInvocationIDKey))
	if !boundedSecret(receiptToken, 4096) || !boundedSecret(identityToken, 16*1024) {
		return hookReceiptConfig{}, errors.New("receipt credentials missing or malformed")
	}
	if !validUUID(invocationID) {
		return hookReceiptConfig{}, errors.New("receipt invocation identity missing or malformed")
	}
	return hookReceiptConfig{endpoint: endpoint, receiptToken: receiptToken, identityToken: identityToken, invocationID: invocationID}, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func boundedSecret(value string, max int) bool {
	return len(value) > 0 && len(value) <= max && !hasControl(value)
}

func hasControl(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

func createEvaluatorReceipt(event agentharness.ToolEvent, runner string, decision agentharness.Decision, now time.Time) (evaluatorReceipt, error) {
	if decision.Action != agentharness.ActionDeny || !validEvaluatorReference(decision.ProviderID, decision.EvaluatorID) {
		return evaluatorReceipt{}, errors.New("decision is not an evaluator-owned denial")
	}
	sessionID := ""
	if event.Context != nil {
		sessionID = strings.TrimSpace(event.Context["session_id"])
	}
	if sessionID == "" || len(sessionID) > 256 || hasControl(sessionID) {
		return evaluatorReceipt{}, errors.New("hook session identity is missing or malformed")
	}
	if !boundedIdentifier(runner, 64) || decision.EvaluationDurationMS < 0 || decision.EvaluationDurationMS > int64((10*time.Minute)/time.Millisecond) {
		return evaluatorReceipt{}, errors.New("receipt fields are out of bounds")
	}
	var idBytes [16]byte
	if _, err := rand.Read(idBytes[:]); err != nil {
		return evaluatorReceipt{}, errors.New("could not create receipt identifier")
	}
	// Set RFC 4122 version 4 and variant bits.
	idBytes[6] = idBytes[6]&0x0f | 0x40
	idBytes[8] = idBytes[8]&0x3f | 0x80
	id := fmt.Sprintf("%s-%s-%s-%s-%s", hex.EncodeToString(idBytes[0:4]), hex.EncodeToString(idBytes[4:6]), hex.EncodeToString(idBytes[6:8]), hex.EncodeToString(idBytes[8:10]), hex.EncodeToString(idBytes[10:16]))
	if now.IsZero() {
		return evaluatorReceipt{}, errors.New("receipt time is missing")
	}
	return evaluatorReceipt{
		ReceiptID: id, SessionID: sessionID, Runner: strings.TrimSpace(runner),
		EvaluatorID: decision.EvaluatorID, DurationMS: decision.EvaluationDurationMS,
		OccurredAt: now.UTC(),
	}, nil
}

func validUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	for i, r := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return false
		}
	}
	return true
}

func boundedIdentifier(value string, max int) bool {
	if value == "" || len(value) > max {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._/-", r)) {
			return false
		}
	}
	return true
}

// persistEvaluatorReceipt attempts an authenticated, bounded POST. Any
// failure is deliberately returned without affecting the hook verdict; callers
// report only a generic audit warning so credentials and endpoint details do
// not leak into provider transcripts.
func persistEvaluatorReceipt(event agentharness.ToolEvent, runner string, decision agentharness.Decision, lookupEnv func(string) string, client *http.Client, now time.Time) error {
	cfg, err := loadHookReceiptConfig(lookupEnv)
	if err != nil {
		return err
	}
	receipt, err := createEvaluatorReceipt(event, runner, decision, now)
	if err != nil {
		return err
	}
	receipt.InvocationID = cfg.invocationID
	body, err := json.Marshal(receipt)
	if err != nil || len(body) == 0 || len(body) > maxReceiptBodyBytes {
		return errors.New("receipt payload is invalid or too large")
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil || len(payload) != 7 {
		return errors.New("receipt payload is malformed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), hookReceiptTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return errors.New("could not build receipt request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(cliutil.HeaderAgentIdentityToken, cfg.identityToken)
	req.Header.Set("X-Vrooli-Hook-Receipt-Token", cfg.receiptToken)
	if client == nil {
		client = &http.Client{Timeout: hookReceiptTimeout}
	}
	// Do not forward either credential to a redirect target.
	clientCopy := *client
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := clientCopy.Do(req)
	if err != nil {
		return errors.New("receipt endpoint unavailable")
	}
	defer response.Body.Close()
	_, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, maxReceiptReplySize+1))
	if readErr != nil || response.StatusCode < 200 || response.StatusCode >= 300 {
		return errors.New("receipt endpoint did not acknowledge persistence")
	}
	return nil
}

func currentReceiptEnv(key string) string { return os.Getenv(key) }
