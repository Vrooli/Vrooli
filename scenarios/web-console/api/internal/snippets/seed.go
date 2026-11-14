package snippets

import "context"

// Stable ids make seeding idempotent; no runtime branch reads them and neither
// row is privileged after it is written through Store.Upsert.
const (
	SeedDirectID   = "example-demand-evidence"
	SeedVariableID = "example-check-scenario"
)

func SeedExamples(ctx context.Context, store Store) error {
	existing, err := store.List(ctx)
	if err != nil {
		return err
	}
	if len(existing) != 0 {
		return nil
	}
	for _, req := range []UpsertRequest{
		{ID: SeedDirectID, Name: "Demand real evidence", Body: "Show the exact command, output, and file that prove this claim.", Color: "#22d3ee"},
		{ID: SeedVariableID, Name: "Check a scenario first", Body: "Before changing {{scenario}}, inspect {{evidence}} and name the seam that owns the behavior.", Color: "#a78bfa"},
	} {
		if _, err := store.Upsert(ctx, req); err != nil {
			return err
		}
	}
	return nil
}
