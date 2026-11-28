package onboard

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apiDB "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"

	internalmachines "vrooli-bridge/internal/machines"
	internalonboard "vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/registry"
)

type generationAttemptLookup struct {
	internalonboard.AttemptStore
	attempt internalonboard.EnrollmentAttempt
}

func (s generationAttemptLookup) GetAttemptByCorrelation(context.Context, string) (internalonboard.EnrollmentAttempt, error) {
	return s.attempt, nil
}

func TestMachineLinkerAssignsAndRepairsNodeEnrollmentGeneration(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	clock := scheduletest.New(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))
	require.NoError(t, apiDB.EnsureSchemas(ctx, d,
		apiDB.SchemaProviderFunc(internalmachines.Schema),
		apiDB.SchemaProviderFunc(registry.Schema),
	))

	machineRepo := internalmachines.NewSQLiteRepository(d, clock)
	machine, err := machineRepo.Create(ctx, internalmachines.CreateInput{ID: "machine-generation", Locators: []internalmachines.Locator{{Kind: "hostname", Value: "node.local"}}})
	require.NoError(t, err)
	registrySvc := registry.NewService(registry.NewSQLiteRepository(d, clock))
	attempt := internalonboard.EnrollmentAttempt{MachineID: machine.ID, CorrelationID: "enroll-1"}
	node1, err := registrySvc.Register(ctx, registry.RegisterInput{Name: "node-one", OS: "linux", Arch: "amd64", PairingCorrelationID: attempt.CorrelationID})
	require.NoError(t, err)
	linker := machineLinkerAdapter{attempts: generationAttemptLookup{attempt: attempt}, machines: internalmachines.NewService(machineRepo), registry: registrySvc}
	require.NoError(t, linker.LinkCorrelatedNode(ctx, attempt.CorrelationID, node1.ID))
	require.NoError(t, linker.LinkCorrelatedNode(ctx, attempt.CorrelationID, node1.ID), "reconciling the same enrollment must be idempotent")

	boundMachine, err := machineRepo.Get(ctx, machine.ID)
	require.NoError(t, err)
	boundNode, err := registrySvc.Get(ctx, node1.ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, boundMachine.EnrollmentGeneration)
	require.EqualValues(t, 1, boundMachine.Lineage[0].EnrollmentGeneration)
	require.EqualValues(t, 1, boundNode.EnrollmentGeneration)

	attempt.CorrelationID = "enroll-2"
	node2, err := registrySvc.Register(ctx, registry.RegisterInput{Name: "node-two", OS: "linux", Arch: "amd64", PairingCorrelationID: attempt.CorrelationID})
	require.NoError(t, err)
	linker.attempts = generationAttemptLookup{attempt: attempt}
	require.NoError(t, linker.LinkCorrelatedNode(ctx, attempt.CorrelationID, node2.ID))
	boundMachine, err = machineRepo.Get(ctx, machine.ID)
	require.NoError(t, err)
	boundNode, err = registrySvc.Get(ctx, node2.ID)
	require.NoError(t, err)
	require.EqualValues(t, 2, boundMachine.EnrollmentGeneration)
	var currentGeneration int64
	for _, lineage := range boundMachine.Lineage {
		if lineage.NodeID == node2.ID && lineage.Current {
			currentGeneration = lineage.EnrollmentGeneration
		}
	}
	require.EqualValues(t, 2, currentGeneration)
	require.EqualValues(t, 2, boundNode.EnrollmentGeneration)
}
