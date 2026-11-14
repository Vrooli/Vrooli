package hub

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// NativeConsentReader reads only the source's effective state. No approve,
// dismiss, grant, delivery, or retry operation is part of this seam.
type NativeConsentReader interface {
	ReadConsent(context.Context, string) (NativeConsentStatus, error)
}
type NativeConsentStatus struct{ AuthorizationID, State, ReviewExpiresAt string }
type NativeConsent struct {
	AuthorizationID, NotificationID, Source, Title, Body, ReviewURL string
	State, ReviewExpiresAt, ObservedAt, CreatedAt                   string
}
type NativeConsentPage struct {
	Consents   []NativeConsent
	NextCursor string
}
type consentCursor struct{ Recipient, Query, CreatedAt, ID string }

func (s *Service) SetNativeConsentReader(reader NativeConsentReader) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nativeConsentReader = reader
}

// ListNativeConsents projects existing recipient-owned source notifications.
// It is not a shadow approval store. Closed records disappear only after an
// authoritative read; unread/unknown records remain visibly unverified.
func (s *Service) ListNativeConsents(ctx context.Context, recipient, query, cursor string, limit int) (NativeConsentPage, error) {
	out := NativeConsentPage{Consents: []NativeConsent{}}
	recipient, query = strings.TrimSpace(recipient), strings.TrimSpace(query)
	if recipient == "" {
		return out, ErrNotFound
	}
	if len(query) > 500 || len(cursor) > 2048 {
		return out, ErrInvalidArgument
	}
	if limit <= 0 || limit > 50 {
		limit = 25
	}
	var after consentCursor
	if cursor != "" {
		b, e := base64.RawURLEncoding.DecodeString(cursor)
		if e != nil || json.Unmarshal(b, &after) != nil || after.Recipient != recipient || after.Query != query || after.ID == "" || after.CreatedAt == "" {
			return out, ErrInvalidArgument
		}
	}
	sql := `SELECT id, title, body, context_url, dedupe_key, created_at FROM notifications WHERE requested_by = ? AND source_scenario = 'agent-manager' AND category = 'authorization-requested' AND instr(lower(title || ' ' || body), lower(?)) > 0`
	args := []any{recipient, query}
	if cursor != "" {
		sql += ` AND (created_at < ? OR (created_at = ? AND id < ?))`
		args = append(args, after.CreatedAt, after.CreatedAt, after.ID)
	}
	sql += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit+1)
	rows, err := s.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return out, err
	}
	candidates := []NativeConsent{}
	for rows.Next() {
		var c NativeConsent
		if err = rows.Scan(&c.NotificationID, &c.Title, &c.Body, &c.ReviewURL, &c.AuthorizationID, &c.CreatedAt); err != nil {
			rows.Close()
			return out, err
		}
		c.Source = "agent-manager"
		c.State = "unavailable"
		candidates = append(candidates, c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	if len(candidates) > limit {
		candidates = candidates[:limit]
		last := candidates[len(candidates)-1]
		b, _ := json.Marshal(consentCursor{Recipient: recipient, Query: query, CreatedAt: last.CreatedAt, ID: last.NotificationID})
		out.NextCursor = base64.RawURLEncoding.EncodeToString(b)
	}
	s.mu.RLock()
	reader := s.nativeConsentReader
	s.mu.RUnlock()
	// Bound the owner page, even if the source is slow. Failure is visible, never
	// interpreted as an empty queue. At most four read-only requests run at once.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	jobs := make(chan int)
	closed := make([]bool, len(candidates))
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				c := &candidates[i]
				if !validNativeConsentIdentity(*c) {
					c.ReviewURL = ""
					continue
				}
				if reader == nil {
					continue
				}
				status, e := reader.ReadConsent(ctx, c.AuthorizationID)
				if e != nil || status.AuthorizationID != c.AuthorizationID {
					continue
				}
				switch status.State {
				case "pending":
					expiry, e := time.Parse(time.RFC3339Nano, status.ReviewExpiresAt)
					if e != nil || !expiry.After(s.clock.Now()) {
						continue
					}
					c.State = "pending"
					c.ReviewExpiresAt = status.ReviewExpiresAt
					c.ObservedAt = s.clock.Now().UTC().Format(time.RFC3339Nano)
				case "approved", "revoked", "expired", "superseded":
					closed[i] = true
				}
			}
		}()
	}
	for i := range candidates {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	for i, c := range candidates {
		if !closed[i] {
			out.Consents = append(out.Consents, c)
		}
	}
	return out, nil
}

func validNativeConsentIdentity(c NativeConsent) bool {
	id, e := uuid.Parse(c.AuthorizationID)
	if e != nil || id.String() != c.AuthorizationID {
		return false
	}
	u, e := url.Parse(c.ReviewURL)
	if e != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Hostname() == "" {
		return false
	}
	if u.Scheme != "https" {
		ip := net.ParseIP(u.Hostname())
		if u.Scheme != "http" || (u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback())) {
			return false
		}
	}
	return u.Path == fmt.Sprintf("/api/v1/authorizations/%s/approve", c.AuthorizationID)
}
