package plans

// builtinTemplates are the per-surface starter plans. Each pre-scaffolds the
// phase skeleton so a plan starts with the right shape and a small model only
// fills prose (OT-P1-002, PM-STORE-002). Templates are in-memory and
// deterministic; CreateFromTemplate clones a template's phases into a new plan.
// Phases scaffold the professional fields (affected areas, ordered steps,
// expected outputs, validation, acceptance) so a starter plan is already a
// credible review artifact, not a thin skeleton.
type planTemplate struct {
	PlanTemplate
	Purpose            string
	Scope              string
	Constraints        string
	ProblemStatement   string
	TargetOutcome      string
	TechnicalApproach  string
	ValidationStrategy string
	Phases             []Phase
}

var builtinTemplates = map[string]planTemplate{
	"generic": {
		PlanTemplate: PlanTemplate{
			ID:          "generic",
			Name:        "Generic implementation plan",
			Description: "A neutral phase skeleton: anchor → implement → validate.",
			Surface:     "generic",
		},
		TechnicalApproach:  "Inspect available diagnostic context, implement the intended outcome, and assess it with concrete evidence.",
		ValidationStrategy: "Use focused checks, observations, or code review; retain advisory findings and uncertainty without requiring a green broad suite.",
		Phases: []Phase{
			{
				Title:           "Inspect context",
				Intent:          "Identify the intended change and available diagnostic observations.",
				Steps:           []string{"Inspect relevant code and existing results", "Record known findings and missing prior state"},
				ExpectedOutputs: []string{"Relevant code references", "Diagnostic context with limitations"},
				Validation:      "Review references and observation provenance without replacing shared-worktree files.",
				Acceptance:      "The intended change and available context are understood.",
				Status:          PhaseStatusTodo,
			},
			{
				Title:           "Implement",
				Intent:          "Build the change as a vertical slice.",
				Steps:           []string{"Make the change behind the relevant seam", "Add or update unit tests"},
				ExpectedOutputs: []string{"The change compiles", "Unit tests cover the change"},
				Validation:      "Build the package and run its unit tests.",
				Acceptance:      "Change builds and unit tests pass.",
				Status:          PhaseStatusTodo,
			},
			{
				Title:           "Validate",
				Intent:          "Assess delivered outcomes under the declared completion policy.",
				Steps:           []string{"Review outcome evidence", "Retain finding dispositions and limitations"},
				ExpectedOutputs: []string{"Outcome assessment with concrete references"},
				Validation:      "Inspect focused-check, observation, or code-review evidence.",
				Acceptance:      "Promised outcomes are supported and material unmet outcomes are resolved.",
				Status:          PhaseStatusTodo,
			},
		},
	},
	"cli": {
		PlanTemplate: PlanTemplate{
			ID:          "cli",
			Name:        "CLI feature plan",
			Description: "Thin-CLI-over-API skeleton: proto → handler → CLI group → tests.",
			Surface:     "cli",
		},
		TechnicalApproach:  "Thin CLI over the API: define the proto contract, implement the handler behind seams, then wire the manifest-driven CLI group.",
		ValidationStrategy: "Lint/generate proto, run handler + manifest-coverage tests, then the scenario test.",
		Phases: []Phase{
			{Title: "Contract", Intent: "Define the proto/Connect contract for the new command.", Steps: []string{"Author the proto messages/service", "Run buf lint and generate"}, ExpectedOutputs: []string{"Generated Go/TS/Connect code"}, Validation: "buf lint and make generate are clean.", Acceptance: "Proto lints and generates clean.", Status: PhaseStatusTodo},
			{Title: "API edge", Intent: "Implement the handler over the domain service behind seams.", Steps: []string{"Add the domain service method", "Wire the connect handler + convert"}, ExpectedOutputs: []string{"A passing handler test"}, Validation: "go test the handler package.", Acceptance: "Handler tests pass.", Status: PhaseStatusTodo},
			{Title: "CLI group", Intent: "Wire the manifest-driven command group and bindings.", Steps: []string{"Add the command to the manifest", "Wire the handler binding"}, ExpectedOutputs: []string{"A working CLI command"}, Validation: "Run the manifest coverage test.", Acceptance: "Manifest coverage test passes.", Status: PhaseStatusTodo},
			{Title: "Validate", Intent: "Run the baseline/check set and verify the DoD.", Steps: []string{"Run the scenario test", "Diff against the anchor"}, ExpectedOutputs: []string{"A passing scenario run"}, Validation: "Run the scenario test and baseline diff.", Acceptance: "DoD verified.", Status: PhaseStatusTodo},
		},
	},
	"proto": {
		PlanTemplate: PlanTemplate{
			ID:          "proto",
			Name:        "Proto contract plan",
			Description: "Wire-contract-first skeleton: design → generate → consume.",
			Surface:     "proto",
		},
		TechnicalApproach:  "Wire-contract-first: design the messages/services, regenerate code, then wire the generated types into consumers.",
		ValidationStrategy: "buf lint, make generate, then build + test every consumer that imports the contract.",
		Phases: []Phase{
			{Title: "Design", Intent: "Author the proto messages/services; lock field numbers.", Steps: []string{"Add messages/services with stable field numbers", "Run buf lint"}, ExpectedOutputs: []string{"A lint-clean proto file"}, Validation: "buf lint is clean.", Acceptance: "buf lint clean.", Status: PhaseStatusTodo},
			{Title: "Generate", Intent: "Regenerate Go/TS/Connect; refresh consumers.", Steps: []string{"Run make generate", "Confirm generated code compiles"}, ExpectedOutputs: []string{"Refreshed generated code"}, Validation: "make generate then go build the generated packages.", Acceptance: "Generated code compiles; regression-scoped.", Status: PhaseStatusTodo},
			{Title: "Consume", Intent: "Wire the generated types into handlers/clients.", Steps: []string{"Update converters/handlers", "Build and test consumers"}, ExpectedOutputs: []string{"Green consumer tests"}, Validation: "go test the consumer packages.", Acceptance: "Consumers build and test green.", Status: PhaseStatusTodo},
		},
	},
	"ui": {
		PlanTemplate: PlanTemplate{
			ID:          "ui",
			Name:        "UI feature plan",
			Description: "Operator-console skeleton: client → board → polish/a11y.",
			Surface:     "ui",
		},
		TechnicalApproach:  "Operator-console slice: add the Connect-Web client, build the board with full states + i18n, then polish for a11y/design coherence.",
		ValidationStrategy: "tsc + vitest + axe per phase; finish with ui-health.",
		Phases: []Phase{
			{Title: "Client", Intent: "Add the Connect-Web client for the new surface.", Steps: []string{"Add the client wrapper", "Type the request/response"}, ExpectedOutputs: []string{"A typed client"}, Validation: "Run tsc.", Acceptance: "Client typechecks.", Status: PhaseStatusTodo},
			{Title: "Board", Intent: "Build the page/board with loading/error/empty states + i18n.", Steps: []string{"Build the board component", "Add loading/error/empty states + i18n"}, ExpectedOutputs: []string{"A working board with full states"}, Validation: "Run vitest and axe.", Acceptance: "vitest + axe clean.", Status: PhaseStatusTodo},
			{Title: "Polish", Intent: "Design-system coherence, keyboard nav, responsive, WCAG AA.", Steps: []string{"Apply design-system tokens", "Add keyboard nav + responsive behavior"}, ExpectedOutputs: []string{"A polished, accessible board"}, Validation: "Run ui-health.", Acceptance: "ui-health green.", Status: PhaseStatusTodo},
		},
	},
}
