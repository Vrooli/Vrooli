package optimization

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServiceSubmitRejectsUnknownAndTrailingJSON(t *testing.T) {
	store := newMemoryStore()
	engine := &Engine{Store: store, Adapters: map[string]Adapter{"ollama-role": &fakeAdapter{rights: "allow", disk: "allow", winner: "m2"}}}
	manager, err := NewManager(engine)
	require.NoError(t, err)
	require.NoError(t, manager.Start(context.Background()))
	defer manager.Stop()
	service, err := NewService(manager)
	require.NoError(t, err)
	raw, _ := json.Marshal(map[string]any{"run_id": "r", "target": "ollama-role", "suite_id": "s", "minimum_cases": 20, "case_ids": make([]string, 20), "candidates": []any{}, "unexpected": true})
	_, err = service.Submit(context.Background(), string(raw))
	require.ErrorIs(t, err, ErrInvalidRequest)
	_, err = service.Submit(context.Background(), `{} {}`)
	require.ErrorIs(t, err, ErrInvalidRequest)
	_, err = service.Submit(context.Background(), "")
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestServiceSubmitRejectsDuplicateJSONKeysAtAnyDepth(t *testing.T) {
	store := newMemoryStore()
	engine := &Engine{Store: store, Adapters: map[string]Adapter{"ollama-role": &fakeAdapter{rights: "allow", disk: "allow", winner: "m2"}}}
	manager, err := NewManager(engine)
	require.NoError(t, err)
	require.NoError(t, manager.Start(context.Background()))
	defer manager.Stop()
	service, err := NewService(manager)
	require.NoError(t, err)

	for _, raw := range []string{
		`{"run_id":"first","run_id":"second"}`,
		`{"subject":{"role":"chat.default","role":"other"}}`,
		`{"subject":{"role":"chat.default","r\u006fle":"other"}}`,
	} {
		_, err := service.Submit(context.Background(), raw)
		require.ErrorContains(t, err, "duplicate object key")
		require.ErrorIs(t, err, ErrInvalidRequest)
	}
}

func TestServiceWaitValidatesRunIdentity(t *testing.T) {
	store := newMemoryStore()
	engine := &Engine{Store: store, Adapters: map[string]Adapter{"ollama-role": &fakeAdapter{rights: "allow", disk: "allow", winner: "m2"}}}
	manager, err := NewManager(engine)
	require.NoError(t, err)
	require.NoError(t, manager.Start(context.Background()))
	defer manager.Stop()
	service, err := NewService(manager)
	require.NoError(t, err)
	_, _, err = service.Wait(context.Background(), "", time.Second)
	require.Error(t, err)
	_, _, err = service.Wait(context.Background(), "missing", time.Second)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceWaitRejectsNonCanonicalIdentityAndUnboundedDuration(t *testing.T) {
	store := newMemoryStore()
	engine := &Engine{Store: store, Adapters: map[string]Adapter{}}
	manager, err := NewManager(engine)
	require.NoError(t, err)
	require.NoError(t, manager.Start(context.Background()))
	defer manager.Stop()
	service, err := NewService(manager)
	require.NoError(t, err)

	for _, testCase := range []struct {
		name  string
		runID string
		wait  time.Duration
	}{
		{name: "leading whitespace", runID: " missing", wait: time.Second},
		{name: "trailing whitespace", runID: "missing ", wait: time.Second},
		{name: "zero wait", runID: "missing", wait: 0},
		{name: "sub-millisecond wait", runID: "missing", wait: time.Microsecond},
		{name: "negative wait", runID: "missing", wait: -time.Second},
		{name: "over maximum wait", runID: "missing", wait: maxServiceWait + time.Millisecond},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, _, err := service.Wait(context.Background(), testCase.runID, testCase.wait)
			require.ErrorIs(t, err, ErrInvalidRequest)
		})
	}
}
