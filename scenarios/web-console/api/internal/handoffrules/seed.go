package handoffrules

import "context"

// SeedRuleID is the stable id of the shipped example. It exists so the seeder
// is idempotent, NOT so any code path can treat the row as special: there is
// no built-in marker column, no delete guard, and no branch anywhere that reads
// this constant at runtime.
const SeedRuleID = "example-plan-file"

// SeedExamples writes the shipped example rule through the SAME Upsert call
// the UI uses, and only into an EMPTY store, so deleting it is permanent.
//
// The example ships ENABLED. Suggestion is the part most likely to annoy
// daily, so an enabled default has to earn itself: this one does because the
// glob is narrow (one directory, one extension) and because a wrong match
// costs a dismissed chip and nothing else. A rule that fired on every path
// would ship disabled instead.
func SeedExamples(ctx context.Context, store Store) error {
	existing, err := store.List(ctx)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}
	_, err = store.Upsert(ctx, UpsertRequest{
		ID:       SeedRuleID,
		Name:     "Plan file",
		Enabled:  true,
		Source:   SourceFilePath,
		Pattern:  "**/.vrooli/plans/*.md",
		Surfaces: []string{"messages"},
	})
	return err
}
