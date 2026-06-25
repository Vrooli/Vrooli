package plans

// builtinTemplates are the per-surface starter plans. Each pre-scaffolds the
// phase skeleton so a plan starts with the right shape and a small model only
// fills prose (OT-P1-002, PM-STORE-002). Templates are in-memory and
// deterministic; CreateFromTemplate clones a template's phases into a new plan.
type planTemplate struct {
	PlanTemplate
	Purpose     string
	Scope       string
	Constraints string
	Phases      []Phase
}

var builtinTemplates = map[string]planTemplate{
	"generic": {
		PlanTemplate: PlanTemplate{
			ID:          "generic",
			Name:        "Generic implementation plan",
			Description: "A neutral phase skeleton: anchor → implement → validate.",
			Surface:     "generic",
		},
		Purpose: "",
		Scope:   "",
		Phases: []Phase{
			{Title: "Anchor & baseline", Intent: "Capture the regression anchor and confirm a green starting point.", Acceptance: "Anchor recorded; baseline green.", Status: PhaseStatusTodo},
			{Title: "Implement", Intent: "Build the change as a vertical slice.", Acceptance: "Change builds and unit tests pass.", Status: PhaseStatusTodo},
			{Title: "Validate", Intent: "Run the derived baseline/check set and verify the Definition of Done.", Acceptance: "DoD verified against the regression anchor.", Status: PhaseStatusTodo},
		},
	},
	"cli": {
		PlanTemplate: PlanTemplate{
			ID:          "cli",
			Name:        "CLI feature plan",
			Description: "Thin-CLI-over-API skeleton: proto → handler → CLI group → tests.",
			Surface:     "cli",
		},
		Phases: []Phase{
			{Title: "Contract", Intent: "Define the proto/Connect contract for the new command.", Acceptance: "Proto lints and generates clean.", Status: PhaseStatusTodo},
			{Title: "API edge", Intent: "Implement the handler over the domain service behind seams.", Acceptance: "Handler tests pass.", Status: PhaseStatusTodo},
			{Title: "CLI group", Intent: "Wire the manifest-driven command group and bindings.", Acceptance: "Manifest coverage test passes.", Status: PhaseStatusTodo},
			{Title: "Validate", Intent: "Run the baseline/check set and verify the DoD.", Acceptance: "DoD verified.", Status: PhaseStatusTodo},
		},
	},
	"proto": {
		PlanTemplate: PlanTemplate{
			ID:          "proto",
			Name:        "Proto contract plan",
			Description: "Wire-contract-first skeleton: design → generate → consume.",
			Surface:     "proto",
		},
		Phases: []Phase{
			{Title: "Design", Intent: "Author the proto messages/services; lock field numbers.", Acceptance: "buf lint clean.", Status: PhaseStatusTodo},
			{Title: "Generate", Intent: "Regenerate Go/TS/Connect; refresh consumers.", Acceptance: "Generated code compiles; regression-scoped.", Status: PhaseStatusTodo},
			{Title: "Consume", Intent: "Wire the generated types into handlers/clients.", Acceptance: "Consumers build and test green.", Status: PhaseStatusTodo},
		},
	},
	"ui": {
		PlanTemplate: PlanTemplate{
			ID:          "ui",
			Name:        "UI feature plan",
			Description: "Operator-console skeleton: client → board → polish/a11y.",
			Surface:     "ui",
		},
		Phases: []Phase{
			{Title: "Client", Intent: "Add the Connect-Web client for the new surface.", Acceptance: "Client typechecks.", Status: PhaseStatusTodo},
			{Title: "Board", Intent: "Build the page/board with loading/error/empty states + i18n.", Acceptance: "vitest + axe clean.", Status: PhaseStatusTodo},
			{Title: "Polish", Intent: "Design-system coherence, keyboard nav, responsive, WCAG AA.", Acceptance: "ui-health green.", Status: PhaseStatusTodo},
		},
	},
}
