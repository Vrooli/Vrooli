package grouptemplates

import "context"

// SeedTemplateID is the stable id of the shipped example. It exists so the
// seeder is idempotent, NOT so any code path can treat the row as special:
// there is no built-in marker column, no delete guard, and no branch anywhere that
// reads this constant at runtime. Deleting the row is an ordinary delete.
const SeedTemplateID = "example-plan-then-implement"

// SeedExamples writes the shipped example template through the SAME Upsert
// call the UI uses (prohibition 5: shipped content is data, not behaviour).
//
// It seeds only into an EMPTY store. That is what makes the example deletable
// in the way the operator expects: once they remove it, no later boot brings
// it back. The cost is that an operator who deletes every template they own
// gets the example again on the next start, which is the friendlier failure of
// the two.
//
// The example is deliberately a TWO-role template while the test fixtures use
// three and five, so the general case is exercised by default rather than the
// shape that happens to match one operator's workflow.
func SeedExamples(ctx context.Context, store Store) error {
	existing, err := store.List(ctx)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}
	_, err = store.Upsert(ctx, UpsertRequest{
		ID:    SeedTemplateID,
		Name:  "Plan → Implement",
		Color: "#22d3ee",
		Roles: []TemplateRole{
			{
				Label:     "Planner",
				Command:   "vrooli agent launch --runner claude --arg=--dangerously-skip-permissions",
				StartMode: StartModeEager,
			},
			{
				Label:          "Implementer",
				Command:        "codex --yolo",
				IncomingPrompt: "Implement the plan at {{payload}}",
				StartMode:      StartModeWaiting,
			},
		},
	})
	return err
}
