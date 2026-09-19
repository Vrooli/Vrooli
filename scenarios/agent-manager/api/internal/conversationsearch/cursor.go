package conversationsearch

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const cursorVersion = 1

type cursorCodec struct{ key []byte }

type cursorEnvelope struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

type cursorPayload struct {
	Version     int        `json:"version"`
	Fingerprint string     `json:"fingerprint"`
	Sort        SearchSort `json:"sort"`
	Score       float64    `json:"score,omitempty"`
	OccurredAt  time.Time  `json:"occurred_at"`
	DocumentID  string     `json:"document_id"`
}

func newCursorCodec(key []byte) (cursorCodec, error) {
	if len(key) < 32 {
		return cursorCodec{}, errors.New("cursor signing key must contain at least 32 bytes")
	}
	return cursorCodec{key: append([]byte(nil), key...)}, nil
}

func (c cursorCodec) encode(payload cursorPayload) (string, error) {
	payload.Version = cursorVersion
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode cursor payload: %w", err)
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(raw)
	signature := c.sign(encodedPayload)
	envelope, err := json.Marshal(cursorEnvelope{Payload: encodedPayload, Signature: signature})
	if err != nil {
		return "", fmt.Errorf("encode cursor envelope: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(envelope), nil
}

func (c cursorCodec) decode(encoded, fingerprint string, sort SearchSort) (cursorPayload, error) {
	rawEnvelope, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return cursorPayload{}, errors.New("page_cursor: invalid encoding")
	}
	var envelope cursorEnvelope
	if err := json.Unmarshal(rawEnvelope, &envelope); err != nil || envelope.Payload == "" || envelope.Signature == "" {
		return cursorPayload{}, errors.New("page_cursor: invalid envelope")
	}
	if !hmac.Equal([]byte(envelope.Signature), []byte(c.sign(envelope.Payload))) {
		return cursorPayload{}, errors.New("page_cursor: signature mismatch")
	}
	rawPayload, err := base64.RawURLEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return cursorPayload{}, errors.New("page_cursor: invalid payload encoding")
	}
	var payload cursorPayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return cursorPayload{}, errors.New("page_cursor: invalid payload")
	}
	if payload.Version != cursorVersion {
		return cursorPayload{}, fmt.Errorf("page_cursor: unsupported version %d", payload.Version)
	}
	if payload.Fingerprint != fingerprint {
		return cursorPayload{}, errors.New("page_cursor: request fingerprint mismatch")
	}
	if payload.Sort != sort {
		return cursorPayload{}, errors.New("page_cursor: sort mismatch")
	}
	if payload.OccurredAt.IsZero() || payload.DocumentID == "" {
		return cursorPayload{}, errors.New("page_cursor: incomplete sort tuple")
	}
	return payload, nil
}

func (c cursorCodec) sign(payload string) string {
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
