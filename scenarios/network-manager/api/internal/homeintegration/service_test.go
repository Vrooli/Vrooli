package homeintegration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServiceListsStableHomeAutomationActions(t *testing.T) {
	// [REQ:NM-P0-007] Home Automation receives a stable action contract without owning network state.
	service := NewService(Config{})

	actions := service.ListActions(context.Background())

	require.ElementsMatch(t, []string{
		"network.health.run",
		"network.adblock.pause_device",
		"network.policy.apply_profile",
	}, actionNames(actions))
	require.False(t, actions[0].ApprovalRequired)
	require.Equal(t, "read", actions[0].Effect)
}

func TestServiceApprovalGatesWriteAction(t *testing.T) {
	// [REQ:NM-P0-007] Write actions return approval_required until explicitly acknowledged.
	repo := newMemoryRepo()
	service := NewService(Config{Repo: repo, Now: fixedNow})

	invocation, event, err := service.InvokeAction(context.Background(), "network.adblock.pause_device", map[string]string{"device": "kid-tablet"}, false)

	require.NoError(t, err)
	require.Equal(t, "approval_required", invocation.Status)
	require.Equal(t, "network.quality.degraded", event.Type)
	require.NotContains(t, event.Summary, "kid-tablet")
	require.Contains(t, event.Summary, "approval_required")
	require.Len(t, repo.invocations, 1)
	require.Equal(t, "[redacted]", repo.invocations[0].Params["device"])
}

func TestServiceApprovedWriteActionIsManualRequiredWithoutCapability(t *testing.T) {
	// [REQ:NM-P0-007] Approval does not fake unsupported automation success.
	repo := newMemoryRepo()
	service := NewService(Config{Repo: repo, Now: fixedNow})

	invocation, event, err := service.InvokeAction(context.Background(), "network.policy.apply_profile", map[string]string{"profile": "school-night"}, true)

	require.NoError(t, err)
	require.Equal(t, "manual_required", invocation.Status)
	require.Equal(t, "network.quality.degraded", event.Type)
	require.Contains(t, invocation.Message, "No supported Home Automation write adapter")
	require.NotContains(t, event.Summary, "school-night")
}

func TestServiceRecordsPublisherFailureWithoutFailingInvocation(t *testing.T) {
	// [REQ:NM-P0-007] Publisher failures are recorded but do not break core Network Manager workflows.
	repo := newMemoryRepo()
	publisher := &CapturingPublisher{Err: errors.New("home automation unavailable")}
	service := NewService(Config{Repo: repo, Publisher: publisher, Now: fixedNow})

	invocation, event, err := service.InvokeAction(context.Background(), "network.health.run", nil, false)

	require.NoError(t, err)
	require.Equal(t, "accepted", invocation.Status)
	require.Equal(t, "publish_failed", event.PublishStatus)
	require.Equal(t, "home automation unavailable", event.PublishError)
	require.Len(t, publisher.Events, 1)
}

func TestServiceRejectsUnknownAction(t *testing.T) {
	service := NewService(Config{Repo: newMemoryRepo()})

	_, _, err := service.InvokeAction(context.Background(), "network.router.reboot", nil, true)

	require.ErrorIs(t, err, ErrUnknownAction)
}

func actionNames(actions []Action) []string {
	out := make([]string, 0, len(actions))
	for _, action := range actions {
		out = append(out, action.Name)
	}
	return out
}

func fixedNow() time.Time {
	return time.Date(2026, 6, 23, 17, 30, 0, 0, time.UTC)
}

type memoryRepo struct {
	events      []Event
	invocations []Invocation
}

func newMemoryRepo() *memoryRepo { return &memoryRepo{} }

func (r *memoryRepo) SaveEvent(_ context.Context, event Event) (Event, error) {
	r.events = append(r.events, event)
	return event, nil
}

func (r *memoryRepo) UpdateEventPublish(_ context.Context, id, status, publishError string) (Event, error) {
	for i := range r.events {
		if r.events[i].ID == id {
			r.events[i].PublishStatus = status
			r.events[i].PublishError = publishError
			return r.events[i], nil
		}
	}
	return Event{}, errors.New("event not found")
}

func (r *memoryRepo) ListEvents(_ context.Context, limit int) ([]Event, error) {
	if limit > 0 && len(r.events) > limit {
		return r.events[:limit], nil
	}
	return append([]Event(nil), r.events...), nil
}

func (r *memoryRepo) SaveInvocation(_ context.Context, invocation Invocation) (Invocation, error) {
	for key, value := range invocation.Params {
		if strings.TrimSpace(key) == "" || value != "[redacted]" {
			return Invocation{}, errors.New("params must be redacted")
		}
	}
	r.invocations = append(r.invocations, invocation)
	return invocation, nil
}
