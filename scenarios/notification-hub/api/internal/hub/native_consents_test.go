package hub_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"notification-hub/internal/hub"
	"sync"
	"testing"
	"time"
)

type consentReader struct {
	mu       sync.Mutex
	statuses map[string]hub.NativeConsentStatus
	calls    []string
	fail     bool
}

func (r *consentReader) ReadConsent(_ context.Context, id string) (hub.NativeConsentStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, id)
	if r.fail {
		return hub.NativeConsentStatus{}, errors.New("offline")
	}
	return r.statuses[id], nil
}
func TestNativeConsent_GivenRecipientEvidence_WhenSourceIsRead_ThenOnlyCurrentPendingOrUnknownRemain(t *testing.T) {
	ctx := context.Background()
	s, db := testService(t)
	reader := &consentReader{statuses: map[string]hub.NativeConsentStatus{}}
	s.SetNativeConsentReader(reader)
	ids := map[string]string{}
	for _, state := range []string{"pending", "approved", "revoked", "superseded", "expired", "approving", "future-state"} {
		id := uuid.NewString()
		ids[state] = id
		_, err := s.Send(ctx, hub.SendInput{RequestedBy: "alice", Title: state, Body: "Permission scope", SensitivityLabel: "private", IdempotencyKey: id, DedupeKey: id, SourceScenario: "agent-manager", Category: "authorization-requested", ContextURL: "https://agent-manager.example.test/api/v1/authorizations/" + id + "/approve"})
		require.NoError(t, err)
		reader.statuses[id] = hub.NativeConsentStatus{AuthorizationID: id, State: state, ReviewExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
	}
	for _, owner := range []string{"bob", ""} {
		page, err := s.ListNativeConsents(ctx, owner, "", "", 25)
		if owner == "" {
			require.ErrorIs(t, err, hub.ErrNotFound)
		} else {
			require.NoError(t, err)
			require.Empty(t, page.Consents)
		}
	}
	require.Empty(t, reader.calls)
	page, err := s.ListNativeConsents(ctx, "alice", "", "", 25)
	require.NoError(t, err)
	require.Len(t, page.Consents, 3)
	for _, c := range page.Consents {
		if c.AuthorizationID == ids["pending"] {
			require.Equal(t, "pending", c.State)
			require.NotEmpty(t, c.ObservedAt)
		} else {
			require.Equal(t, "unavailable", c.State)
			require.Empty(t, c.ObservedAt)
		}
	}
	var asks, answers int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM asks").Scan(&asks))
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM answers").Scan(&answers))
	require.Zero(t, asks)
	require.Zero(t, answers)
	reader.fail = true
	page, err = s.ListNativeConsents(ctx, "alice", "pending", "", 25)
	require.NoError(t, err)
	require.Len(t, page.Consents, 1)
	require.Equal(t, "unavailable", page.Consents[0].State)
}
func TestNativeConsent_GivenOlderOrMalformedEvidence_WhenPaging_ThenNoHiddenCapOrUnsafeFetch(t *testing.T) {
	ctx := context.Background()
	s, _ := testService(t)
	reader := &consentReader{statuses: map[string]hub.NativeConsentStatus{}}
	s.SetNativeConsentReader(reader)
	for i := 0; i < 4; i++ {
		id := uuid.NewString()
		source, url := "agent-manager", "https://agent-manager.example.test/api/v1/authorizations/"+id+"/approve"
		if i == 1 {
			source = "other"
		}
		if i == 2 {
			url = "javascript:alert(1)"
		}
		_, err := s.Send(ctx, hub.SendInput{RequestedBy: "alice", Title: fmt.Sprint(i), Body: "native", SensitivityLabel: "private", IdempotencyKey: id, DedupeKey: id, SourceScenario: source, Category: "authorization-requested", ContextURL: url})
		require.NoError(t, err)
		reader.statuses[id] = hub.NativeConsentStatus{AuthorizationID: id, State: "pending", ReviewExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
	}
	cursor := ""
	count := 0
	for {
		page, err := s.ListNativeConsents(ctx, "alice", "", cursor, 1)
		require.NoError(t, err)
		for _, c := range page.Consents {
			count++
			if c.State == "unavailable" {
				require.Empty(t, c.ReviewURL)
			}
		}
		if page.NextCursor == "" {
			break
		}
		_, err = s.ListNativeConsents(ctx, "bob", "", page.NextCursor, 1)
		require.ErrorIs(t, err, hub.ErrInvalidArgument)
		cursor = page.NextCursor
	}
	require.Equal(t, 3, count)
	require.Len(t, reader.calls, 2)
}
