package machines_test

import (
	"context"
	"testing"

	"vrooli-bridge/internal/machines"

	"github.com/stretchr/testify/require"
)

// [REQ:BRG-MEC-001] Profile resolution is reproducible, while suggested scopes
// remain policy input rather than Registry-approved authorization.
func TestBuiltInProfileResolvesToImmutableSnapshot(t *testing.T) {
	snapshot, err := machines.ResolveProfile("machine-1", "development-runner", "v1", nil)
	require.NoError(t, err)
	require.Contains(t, snapshot.JSON, "runs.execute")
	d, clk := newDB(t)
	repo := machines.NewSQLiteRepository(d, clk)
	_, err = repo.SavePolicySnapshot(context.Background(), snapshot)
	require.NoError(t, err)
	_, err = repo.SavePolicySnapshot(context.Background(), snapshot)
	require.Error(t, err, "policy history cannot be silently overwritten")
}

func TestProfileRegistryIncludesOnlyVersionedBuiltInsAndCopiesSlices(t *testing.T) {
	expectedScenarios := map[string][]string{
		"managed-connection": {"vrooli-bridge"},
		"presence":           {"vrooli-bridge"},
		"deployment-target":  {"deployment-manager", "vrooli-bridge"},
		"production-runtime": {"system-monitor", "vrooli-bridge"},
		"development-runner": {"test-genie", "vrooli-bridge"},
		"custom":             {"vrooli-bridge"},
	}
	for id, expected := range expectedScenarios {
		snapshot, err := machines.ResolveProfile("machine-1", id, "v1", nil)
		require.NoError(t, err, id)
		require.NotEmpty(t, snapshot.JSON)
		require.Equal(t, expected, snapshot.Scenarios, id)
	}
	first, err := machines.ResolveProfile("machine-1", "presence", "v1", nil)
	require.NoError(t, err)
	first.SuggestedScopes[0] = "forged.authorization"
	second, err := machines.ResolveProfile("machine-2", "presence", "v1", nil)
	require.NoError(t, err)
	require.Equal(t, "presence.read", second.SuggestedScopes[0])
	_, err = machines.ResolveProfile("machine-1", "presence", "v1", map[string]string{"preset": "unsafe"})
	require.Error(t, err)
}

func TestApplyPolicyAppendsImmutableHistoryAndUsesVersion(t *testing.T) {
	d, clk := newDB(t)
	repo := machines.NewSQLiteRepository(d, clk)
	ctx := context.Background()
	machine, err := repo.Create(ctx, machines.CreateInput{ID: "m1", Locators: []machines.Locator{{Kind: "hostname", Value: "host.example"}}})
	require.NoError(t, err)
	updated, snapshot, err := repo.ApplyPolicy(ctx, machines.PolicyChangeInput{MachineID: "m1", ExpectedVersion: machine.Version, ProfileID: "presence", ProfileVersion: "v1", Actor: "owner", Reason: "presence only"})
	require.NoError(t, err)
	require.Equal(t, int64(2), updated.Version)
	require.Equal(t, "presence", snapshot.ProfileID)
	_, _, err = repo.ApplyPolicy(ctx, machines.PolicyChangeInput{MachineID: "m1", ExpectedVersion: machine.Version, ProfileID: "development-runner", ProfileVersion: "v1"})
	var conflict machines.ErrConflict
	require.ErrorAs(t, err, &conflict)
	var entries int
	require.NoError(t, d.QueryRowContext(ctx, "SELECT COUNT(*) FROM machine_policy_snapshot_history WHERE machine_id='m1'").Scan(&entries))
	require.Equal(t, 1, entries)
}

func TestLatestPolicySnapshotPreservesOverrides(t *testing.T) {
	d, clk := newDB(t)
	repo := machines.NewSQLiteRepository(d, clk)
	ctx := context.Background()
	machine, err := repo.Create(ctx, machines.CreateInput{Locators: []machines.Locator{{Kind: "hostname", Value: "profile-host.example"}}})
	require.NoError(t, err)
	updated, snapshot, err := repo.ApplyPolicy(ctx, machines.PolicyChangeInput{MachineID: machine.ID, ExpectedVersion: machine.Version, ProfileID: "custom", ProfileVersion: "v1", Overrides: map[string]string{"scenarios": "alpha,beta", "optional_resources": "redis"}})
	require.NoError(t, err)
	require.Equal(t, int64(2), updated.Version)
	got, err := repo.(machines.PolicyReader).LatestPolicySnapshot(ctx, machine.ID)
	require.NoError(t, err)
	require.Equal(t, snapshot.JSON, got.JSON)
	require.Equal(t, []string{"alpha", "beta"}, got.Scenarios)
	require.Equal(t, []string{"redis"}, got.OptionalResources)
}

func TestApplyPolicyRequiresConfirmationForDowngrade(t *testing.T) {
	d, clk := newDB(t)
	repo := machines.NewSQLiteRepository(d, clk)
	ctx := context.Background()
	machine, err := repo.Create(ctx, machines.CreateInput{ID: "m-downgrade", Locators: []machines.Locator{{Kind: "hostname", Value: "host.example"}}})
	require.NoError(t, err)
	present, _, err := repo.ApplyPolicy(ctx, machines.PolicyChangeInput{MachineID: machine.ID, ExpectedVersion: machine.Version, ProfileID: "presence", ProfileVersion: "v1"})
	require.NoError(t, err)
	_, _, err = repo.ApplyPolicy(ctx, machines.PolicyChangeInput{MachineID: machine.ID, ExpectedVersion: present.Version, ProfileID: "managed-connection", ProfileVersion: "v1"})
	var invalid machines.ErrInvalid
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "confirm_removal", invalid.Field)
	_, _, err = repo.ApplyPolicy(ctx, machines.PolicyChangeInput{MachineID: machine.ID, ExpectedVersion: present.Version, ProfileID: "managed-connection", ProfileVersion: "v1", ConfirmRemoval: true})
	require.NoError(t, err)
}
