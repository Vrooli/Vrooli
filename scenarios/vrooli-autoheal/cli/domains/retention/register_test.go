package retention

import (
	"testing"

	"github.com/vrooli/api-core/retention"
	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterIsOfflineOperatorCommand(t *testing.T) {
	group := Register((*cliapp.ScenarioApp)(nil))
	if group.Name != "retention" {
		t.Fatalf("group name = %q, want retention", group.Name)
	}
	if group.NeedsAPI {
		t.Fatal("retention commands must remain usable while the scenario is stopped")
	}
	if len(group.Subcommands) != 2 {
		t.Fatalf("subcommand count = %d, want status and enforce", len(group.Subcommands))
	}
}

func TestSQLiteTableSpecsIgnoreFrameworkFilesystemBudgets(t *testing.T) {
	specs := []retention.Spec{
		{Budget: retention.Budget{Name: "logs"}, Target: retention.Target{Kind: retention.TargetDirectory}},
		{Budget: retention.Budget{Name: "health_results"}, Target: retention.Target{Kind: retention.TargetSQLiteTable}},
		{Budget: retention.Budget{Name: "receipt"}, Target: retention.Target{Kind: retention.TargetFile}},
		{Budget: retention.Budget{Name: "action_logs"}, Target: retention.Target{Kind: retention.TargetSQLiteTable}},
	}

	got := sqliteTableSpecs(specs)
	if len(got) != 2 || got[0].Budget.Name != "health_results" || got[1].Budget.Name != "action_logs" {
		t.Fatalf("sqlite table specs = %#v", got)
	}
}

func TestRetentionScenarioNamespaceRejectsAmbientCrossScenarioHijack(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    string
		wantErr bool
	}{
		{name: "foreign live falls back to autoheal", env: map[string]string{"VROOLI_STORAGE_NAMESPACE": "web-console", "VROOLI_VARIANT": "live"}, want: "vrooli-autoheal"},
		{name: "own live is honored", env: map[string]string{"VROOLI_STORAGE_NAMESPACE": "vrooli-autoheal", "VROOLI_VARIANT": "live"}, want: "vrooli-autoheal"},
		{name: "own shadow is honored", env: map[string]string{"VROOLI_STORAGE_NAMESPACE": "vrooli-autoheal_shadow", "VROOLI_VARIANT": "shadow"}, want: "vrooli-autoheal_shadow"},
		{name: "foreign shadow fails loud", env: map[string]string{"VROOLI_STORAGE_NAMESPACE": "web-console_shadow", "VROOLI_VARIANT": "shadow"}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := retentionScenarioNamespace(func(key string) string { return tc.env[key] })
			if tc.wantErr {
				if err == nil {
					t.Fatalf("namespace = %q, want error", got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("namespace = %q, err = %v, want %q", got, err, tc.want)
			}
		})
	}
}
