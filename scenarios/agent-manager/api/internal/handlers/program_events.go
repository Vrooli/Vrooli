package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/adapters/database"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
	"google.golang.org/protobuf/proto"
)

// ProgramEventWebhook receives the signed webhook envelope emitted by
// vrooli-events. Event IDs are the durable idempotency key.
func ProgramEventWebhook(db *database.DB, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil || strings.TrimSpace(secret) == "" || !validProgramEventSignature(body, r.Header.Get("X-Vrooli-Events-Signature"), secret) {
			http.Error(w, "event webhook signature is invalid or unavailable", http.StatusUnauthorized)
			return
		}
		var envelope struct {
			EventID   string          `json:"event_id"`
			EventType string          `json:"event_type"`
			Source    string          `json:"source_scenario"`
			Payload   json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			http.Error(w, "invalid webhook body", http.StatusBadRequest)
			return
		}
		eventID := strings.TrimSpace(r.Header.Get("X-Vrooli-Events-Event-ID"))
		if eventID == "" {
			eventID = strings.TrimSpace(envelope.EventID)
		}
		if eventID == "" {
			http.Error(w, "event_id is required", http.StatusBadRequest)
			return
		}
		program, payload, err := decodeProgramEvent(envelope.Payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if db == nil {
			http.Error(w, "event store unavailable", http.StatusServiceUnavailable)
			return
		}
		result, err := db.ExecContext(r.Context(), `INSERT OR IGNORE INTO program_runtime_events (event_id,event_type,source_scenario,program_id,session_id,kind,payload_json,received_at) VALUES (?,?,?,?,?,?,?,?)`, eventID, envelope.EventType, envelope.Source, program.GetProgramId(), program.GetSessionId(), program.GetKind().String(), string(payload), time.Now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			http.Error(w, "store event: "+err.Error(), http.StatusInternalServerError)
			return
		}
		changed, _ := result.RowsAffected()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"event_id": eventID, "duplicate": changed == 0})
	})
}

func decodeProgramEvent(raw json.RawMessage) (*telemetryv1.ProgramEvent, []byte, error) {
	var payload struct {
		Encoding string `json:"encoding"`
		Data     string `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil || payload.Encoding != "base64" || payload.Data == "" {
		return nil, nil, fmt.Errorf("program event payload must be a base64 envelope")
	}
	decoded, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		return nil, nil, fmt.Errorf("decode program event payload: %w", err)
	}
	event := new(telemetryv1.ProgramEvent)
	if err := proto.Unmarshal(decoded, event); err != nil {
		return nil, nil, fmt.Errorf("unmarshal program event payload: %w", err)
	}
	return event, decoded, nil
}

func validProgramEventSignature(body []byte, value, secret string) bool {
	value = strings.TrimPrefix(strings.TrimSpace(value), "sha256=")
	provided, err := hex.DecodeString(value)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	return len(provided) == len(expected) && subtle.ConstantTimeCompare(provided, expected) == 1
}
