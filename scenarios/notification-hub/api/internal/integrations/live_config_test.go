package integrations

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"notification-hub/internal/modules"
)

func TestLiveConfigSetterChangesWithoutRestart(t *testing.T) {
	store := NewLiveConfigStore(LiveConfig{Pattern: "incident.*"})
	server := httptest.NewServer(Handler(store, nil))
	defer server.Close()
	request := httptest.NewRequest("PUT", "/", nil)
	_ = request
	store.Set(LiveConfig{Pattern: "incident.opened.v1"})
	if got := store.Get().Pattern; got != "incident.opened.v1" {
		t.Fatalf("pattern = %q", got)
	}
}

func TestLiveConfigCopiesTemplates(t *testing.T) {
	store := NewLiveConfigStore(LiveConfig{Pattern: "incident.*"})
	store.Set(LiveConfig{Templates: map[string]EventTemplate{
		"incident.opened.v1": {Title: "Incident: {{check_id}}", Body: "{{message}}"},
	}})
	config := store.Get()
	if config.Templates["incident.opened.v1"].Title != "Incident: {{check_id}}" {
		t.Fatalf("templates = %#v", config.Templates)
	}
}

func TestPersistentLiveConfigSurvivesRestart(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	routed := database.NewFromPrimary(primary)
	initial := LiveConfig{Pattern: "incident.*", Templates: map[string]EventTemplate{"incident.opened.v1": {Title: "Incident {{check_id}}", Body: "{{message}}"}}, SensitivityBySeverity: map[string]string{"critical": "restricted"}}
	store, err := NewPersistentLiveConfigStore(context.Background(), routed, initial)
	require.NoError(t, err)
	require.NoError(t, store.Apply(context.Background(), LiveConfig{EventsAPIBase: "http://events", WebhookURL: "http://hub/events", Pattern: "incident.opened.v1", Templates: initial.Templates, SensitivityBySeverity: initial.SensitivityBySeverity}))

	reloaded, err := NewPersistentLiveConfigStore(context.Background(), routed, LiveConfig{Pattern: "fallback"})
	require.NoError(t, err)
	require.Equal(t, "http://events", reloaded.Get().EventsAPIBase)
	require.Equal(t, "http://hub/events", reloaded.Get().WebhookURL)
	require.Equal(t, "incident.opened.v1", reloaded.Get().Pattern)
	require.Equal(t, initial.Templates, reloaded.Get().Templates)
	require.Equal(t, "restricted", reloaded.Get().SensitivityBySeverity["critical"])
}
