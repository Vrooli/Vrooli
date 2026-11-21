// Responsibility: owner-bound child issuance and credential custody for coding backup.
package cliutil

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const CodingBackupIdentity = "vrooli/coding-backup"

// CodingBackupRecord belongs in the host credential authority. ParentKey is
// service custody, never child environment, a command argument or a UI reply.
// The same record lets an unattended launcher work without Web Console running.
type CodingBackupRecord struct {
	RevocationPending bool                `json:"revocation_pending,omitempty"`
	Consent           CodingBackupConsent `json:"consent"`
	Owner             string              `json:"owner"`
	BaseURL           string              `json:"base_url"`
	ParentID          string              `json:"parent_id"`
	ParentKey         string              `json:"parent_key"`
}

func CodingBackupField(owner, harness string) string {
	hash := sha256.Sum256([]byte(owner))
	return harness + "-" + hex.EncodeToString(hash[:16])
}

type CodingBackupSession struct {
	ID          string
	ExpiresAt   int64
	Source      ModelSource
	Environment map[string]string
	Revoke      func(context.Context) error
}

// IssueCodingBackupSession performs one issuance, with no retry on an unknown
// outcome. LPBS enforces the shared parent cap and current harness eligibility.
func IssueCodingBackupSession(ctx context.Context, record CodingBackupRecord, owner string, client *http.Client) (CodingBackupSession, error) {
	c := record.Consent
	if record.RevocationPending || record.Owner != owner || owner == "" || !c.Valid(time.Now()) || record.ParentID == "" || record.ParentKey == "" {
		return CodingBackupSession{}, errors.New("coding backup consent unavailable or expired")
	}
	source := ModelSource{ID: "Vrooli credits", Kind: "vrooli_credits", Model: c.Model, BaseURL: strings.TrimRight(record.BaseURL, "/") + "/api/v1/ai", SecretEnv: "VROOLI_CODING_SESSION_KEY"}
	// Validate the authority URL and adapter before handing it a credential.
	if _, _, err := ResolveModelSource(c.Harness, source, nil, []string{source.SecretEnv + "=validation-only"}); err != nil {
		return CodingBackupSession{}, err
	}
	ttl := min(int64(3600), int64(time.Until(c.ExpiresAt).Seconds()))
	if ttl <= 0 {
		return CodingBackupSession{}, errors.New("coding backup consent expired")
	}
	payload, _ := json.Marshal(map[string]any{"harness": c.Harness, "model": c.Model, "max_credits": c.MaxCredits, "ttl_seconds": ttl})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, source.BaseURL+"/keys/session", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+record.ParentKey)
	req.Header.Set("Content-Type", "application/json")
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return CodingBackupSession{}, errors.New("coding backup authority unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return CodingBackupSession{}, fmt.Errorf("coding backup refused (HTTP %d)", resp.StatusCode)
	}
	var key struct {
		ID        string `json:"id"`
		Key       string `json:"key"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 16384)).Decode(&key) != nil || key.ID == "" || key.Key == "" || key.ExpiresAt <= time.Now().Unix() || key.ExpiresAt > c.ExpiresAt.Unix() {
		return CodingBackupSession{}, errors.New("invalid backup session credential")
	}
	encoded, _ := json.Marshal(source)
	return CodingBackupSession{ID: key.ID, ExpiresAt: key.ExpiresAt, Source: source, Environment: map[string]string{ModelSourceEnvironment: string(encoded), source.SecretEnv: key.Key}, Revoke: func(ctx context.Context) error {
		req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, source.BaseURL+"/keys/"+key.ID, nil)
		req.Header.Set("Authorization", "Bearer "+record.ParentKey)
		response, err := client.Do(req)
		if err != nil {
			return errors.New("backup revocation unavailable")
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return fmt.Errorf("backup revocation refused (HTTP %d)", response.StatusCode)
		}
		return nil
	}}, nil
}

// ResolveModelSourceEnvironment is the common path for host/sandbox launchers
// that already own their native process lifecycle and attribution.
func ResolveModelSourceEnvironment(harness string, args, environment []string) ([]string, []string, error) {
	source, err := modelSourceFromEnvironment(environment)
	if err != nil {
		return nil, nil, err
	}
	if source == nil {
		return args, environment, nil
	}
	return ResolveModelSource(harness, *source, args, environment)
}
