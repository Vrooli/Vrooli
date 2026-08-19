package integrations

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"notification-hub/internal/hub"
)

type eventPayload struct {
	EventID          string          `json:"event_id"`
	EventType        string          `json:"event_type"`
	SourceScenario   string          `json:"source_scenario"`
	Title            string          `json:"title"`
	Body             string          `json:"body"`
	Urgency          string          `json:"urgency"`
	SensitivityLabel string          `json:"sensitivity_label"`
	Payload          json.RawMessage `json:"payload"`
}

type eventNotificationPayload struct {
	Title            string `json:"title"`
	Body             string `json:"body"`
	Message          string `json:"message"`
	Urgency          string `json:"urgency"`
	SensitivityLabel string `json:"sensitivity_label"`
}

// EventWebhook is the single sanctioned REST exception for event ingress.
// The signature covers the exact bytes that are decoded, and the event id is
// used as the notification idempotency key.
func EventWebhook(service *hub.Service, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil || strings.TrimSpace(secret) == "" || !validSignature(body, r.Header.Get("X-Vrooli-Events-Signature"), secret) {
			writeJSONError(w, http.StatusUnauthorized, "event webhook signature is invalid or unavailable")
			return
		}
		var payload eventPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "event webhook body is invalid")
			return
		}
		eventID := strings.TrimSpace(r.Header.Get("X-Vrooli-Events-Event-ID"))
		if eventID == "" {
			eventID = strings.TrimSpace(payload.EventID)
		}
		var nested eventNotificationPayload
		if len(payload.Payload) > 0 {
			_ = json.Unmarshal(payload.Payload, &nested)
		}
		if payload.Title == "" {
			payload.Title = nested.Title
		}
		if payload.Body == "" {
			payload.Body = nested.Body
			if payload.Body == "" {
				payload.Body = nested.Message
			}
		}
		if payload.Urgency == "" {
			payload.Urgency = nested.Urgency
		}
		if payload.SensitivityLabel == "" {
			payload.SensitivityLabel = nested.SensitivityLabel
		}
		if payload.Body == "" && len(payload.Payload) > 0 {
			payload.Body = string(payload.Payload)
		}
		if payload.SensitivityLabel == "" {
			payload.SensitivityLabel = "public"
		}
		if eventID == "" || payload.Body == "" {
			writeJSONError(w, http.StatusBadRequest, "event_id, body, and sensitivity_label are required")
			return
		}
		recipient := service.DefaultRecipient()
		if recipient == "" {
			recipient = payload.SourceScenario
		}
		notification, err := service.Send(r.Context(), hub.SendInput{RequestedBy: recipient, Title: payload.Title, Body: payload.Body, SensitivityLabel: payload.SensitivityLabel, IdempotencyKey: eventID, DedupeKey: payload.EventType})
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, hub.ErrInvalidArgument) {
				status = http.StatusBadRequest
			}
			writeJSONError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"notification_id": notification.ID, "event_id": eventID})
	})
}

// EnsureEventSubscription makes the optional vrooli-events integration
// durable and idempotent. It is startup reconciliation rather than a new
// control plane: the events scenario remains the owner of subscription state
// and retry delivery.
func EnsureEventSubscription(ctx context.Context, baseURL, target, pattern string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	target = strings.TrimSpace(target)
	pattern = strings.TrimSpace(pattern)
	if baseURL == "" || target == "" || pattern == "" {
		return nil
	}
	client := &http.Client{Timeout: 10 * time.Second}
	listURL := baseURL + "/api/v1/subscriptions?owner=notification-hub&pattern=" + url.QueryEscape(pattern)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("list event subscriptions: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return fmt.Errorf("list event subscriptions returned %s", response.Status)
	}
	var existing []struct {
		DeliveryTarget string `json:"delivery_target"`
		Enabled        bool   `json:"enabled"`
	}
	if err := json.NewDecoder(response.Body).Decode(&existing); err == nil {
		for _, item := range existing {
			if item.Enabled && item.DeliveryTarget == target {
				return nil
			}
		}
	}
	body, err := json.Marshal(map[string]any{"name": "notification-hub-events", "owner_scenario": "notification-hub", "event_pattern": pattern, "delivery_type": "webhook", "delivery_target": target, "enabled": true})
	if err != nil {
		return err
	}
	request, err = http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/subscriptions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err = client.Do(request)
	if err != nil {
		return fmt.Errorf("create event subscription: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return fmt.Errorf("create event subscription returned %s", response.Status)
	}
	return nil
}

func validSignature(body []byte, value, secret string) bool {
	value = strings.TrimSpace(strings.TrimPrefix(value, "sha256="))
	provided, err := hex.DecodeString(value)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	return len(provided) == len(expected) && subtle.ConstantTimeCompare(provided, expected) == 1
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
