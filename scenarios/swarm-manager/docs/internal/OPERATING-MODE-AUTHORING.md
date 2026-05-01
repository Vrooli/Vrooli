# Operating Mode Authoring Guide

> Current State (2026-05-01): Swarm Manager operating modes are static Go definitions. A new mode should be authorable through one focused mode definition file, prompt-manager skills, targeted tests, and docs. Shared framework control flow should not need mode-specific branches.

This guide is the source material for future agents and for a future `mode-authoring` skill. It explains where operating-mode behavior belongs, what files should usually change, and which framework files should stay untouched when adding a new methodology.

## Authoring Rule

Adding a new static operating mode should mostly require:

1. One new `api/internal/operatingmode/mode_<name>.go` definition file.
2. A new registry entry in `api/internal/operatingmode/registry.go`.
3. Prompt-manager skills matching the mode phases.
4. Focused tests proving registry, prompt catalog, capabilities, transitions, artifacts, metrics, and lifecycle behavior.
5. User/internal docs for the new operator workflow.

Do not add mode-specific branches to handlers, stats aggregation, UI components, CLI commands, prompt catalog lookup, artifact appliers, phase runners, or lock/activity shared packages. If a new mode appears to require that, first ask whether the missing behavior is a registry policy that should be generalized.

## Files To Touch

Typical files for a new initiative-scoped mode:

| File | Expected change |
|---|---|
| `api/internal/operatingmode/mode_<name>.go` | New mode definition built with `buildInitiativeMode` |
| `api/internal/operatingmode/registry.go` | Add the mode constant and one registry map entry |
| `api/internal/operatingmode/registry_test.go` | Validate the new mode definition and capability contract |
| `api/internal/operatingmode/service_test.go` or a focused test file | Exercise lifecycle paths that are unique to the mode |
| Prompt-manager skill source | Add one skill per phase, matching `SkillID`/`CatalogID` |
| `docs/concepts/EXECUTION-MODES.md` or `docs/guides/<mode>.md` | Explain when operators should use the mode |
| `docs/manifest.json` | Register new operator docs when they should appear in the docs UI |

Files that should not need mode-specific edits:

| File | Why it should stay generic |
|---|---|
| `api/internal/operatingmode/state.go` | Phase availability is derived from `PhaseGraph.Transitions` and `TransitionRules` |
| `api/internal/operatingmode/artifact_applier.go` | Derived artifacts are declared through `ResultBindings` |
| `api/internal/promptcatalog/catalog.go` | Operating-mode prompt entries are generated from the registry |
| `api/internal/stats/engine.go` | Replan and acceptance semantics come from `MetricsPolicy` |
| `api/internal/agentactivity/types.go` | Initiative-owned activity purposes accept registry-authored purpose tokens |
| `api/internal/initiativelock/lock.go` | Lock purposes are registry-authored strings |
| `ui/src/components/initiative/operating-mode/` | Controls should render backend-declared capabilities and phase actions |
| `cli/cmd_initiatives_operating_mode.go` | CLI remains a thin API wrapper over catalog/workspace responses |

## Definition Shape

Initiative-scoped modes should use `buildInitiativeMode` from `api/internal/operatingmode/definition_builder.go`. The builder keeps repeated policy in one place:

- initiative scope
- mode artifact root and round root
- prompt catalog ID and skill ID derivation
- phase profile map generation
- default backlog reconciliation capabilities
- metrics event source
- initiative-exclusive locking
- operating-mode UI tab
- structured output contract defaults
- required artifact extraction
- result-binding artifact merging
- default lock purpose from activity purpose

Minimal structure:

```go
package operatingmode

const ModeExample Mode = "example-mode"

func exampleModeDefinition() Definition {
	return buildInitiativeMode(initiativeModeSpec{
		Mode:                ModeExample,
		Label:               "Example Mode",
		RunStrategy:         RunStrategyOperatorGatedLoop,
		ArtifactRoot:        "modes/example-mode",
		PromptCatalogPrefix: "swarm-manager-example-mode",
		DefaultProfileKey:   ProfileDeepWork,
		StartPhase:          "assess",
		Terminal:            []Phase{"review"},
		Transitions: map[Phase][]Phase{
			"assess": {"execute"},
			"execute": {"review", "assess"},
			"review": {"assess"},
		},
		TransitionRules: map[Phase][]TransitionRule{
			"execute": {
				{
					When: TransitionCondition{
						Kind:       TransitionConditionPayloadBool,
						PayloadKey: payloadReplanNeeded,
						BoolValue:  true,
					},
					Next: []Phase{"assess"},
				},
				{
					When: TransitionCondition{Kind: TransitionConditionAlways},
					Next: []Phase{"review"},
				},
			},
		},
		Phases: []initiativePhaseSpec{
			{
				Phase:           "assess",
				Purpose:         "example_mode_assess",
				PromptPurpose:   "Assess the initiative state for example-mode execution.",
				ProfileKey:      ProfileDeepWork,
				OutputArtifacts: []ArtifactDefinition{requiredOutputArtifact("modes/example-mode/assessment.md", "text/markdown")},
			},
			{
				Phase:      "execute",
				Purpose:    "example_mode_execute",
				ProfileKey: ProfileDeepWork,
				WritesRepo: true,
				Metrics:    PhaseMetricsSpec{CountsReplanSample: true},
			},
			{
				Phase:            "review",
				Purpose:          "example_mode_review",
				ProfileKey:       ProfileAnalysis,
				RequiresVerdict:  true,
				RequiresCriteria: true,
				Metrics:          PhaseMetricsSpec{CountsAcceptanceSample: true},
			},
		},
	})
}
```

After adding the definition, register it in `registry.go`:

```go
const (
	ModeExample Mode = "example-mode"
)

var registry = map[Mode]Definition{
	ModeItemLevel:       itemLevelDefinition(),
	ModeHolisticLoop:    holisticLoopDefinition(),
	ModePhasedPlanDrain: phasedPlanDrainDefinition(),
	ModeExample:         exampleModeDefinition(),
}
```

## Phase Fields

Use `initiativePhaseSpec` for each phase:

| Field | Use |
|---|---|
| `Phase` | Stable phase token used in routes, rounds, stats, and UI |
| `Purpose` | Lowercase snake-case audit token for agent activity |
| `LockPurpose` | Optional lowercase snake-case token; defaults to `Purpose` |
| `PromptSuffix` | Optional suffix when prompt IDs should not use the raw phase token |
| `PromptTitle` | Prompt Center display title; defaults to a humanized phase |
| `PromptPurpose` | Prompt Center purpose text |
| `PromptTrigger` | Prompt Center trigger text |
| `ProfileKey` | Scenario-owned AgentManager profile key such as `swarm-manager/deep-work` |
| `WritesRepo` | True when the phase is expected to modify code/files |
| `OutputArtifacts` | Durable mode artifacts the phase may write |
| `ResultBindings` | Derived writes from structured result fields |
| `Metrics` | Per-phase opt-in for replan and acceptance metric samples |
| `RequiresProgress` | Completed result must include a valid progress decision |
| `RequiresVerdict` | Completed result must include a verdict |
| `RequiresHandoff` | Completed result must include a handoff |
| `RequiresCriteria` | Phase start requires initiative acceptance criteria |

Purpose tokens must be lowercase snake-case. Do not add a shared constant for each new phase in `agentactivity` or `initiativelock`; the registry owns those purpose strings.

## Transitions

Every phase-to-phase edge must be declared in `Transitions`. Conditional routing belongs in `TransitionRules`.

Supported conditions:

| Condition | Use |
|---|---|
| `TransitionConditionAlways` | Default route after more specific rules |
| `TransitionConditionPayloadBool` | Route based on a boolean in the round payload, such as `replan_needed` |
| `TransitionConditionProgressDecision` | Route based on `operating_mode_result.progress.decision` |

Rules may return no next phase. This is how a classifier can intentionally block progression, such as a `blocked` progress decision.

Do not encode transition behavior in `state.go`, UI components, or CLI commands.

## Output Contracts

All initiative phase completions require a structured `operating_mode_result` envelope. The builder derives `OutputContract.RequiredArtifacts` from required `OutputArtifacts`, and it merges `ResultBindings` into the declared output list.

Use these flags to make phase outputs explicit:

- `OutputArtifacts`: agent-provided or derived files that belong under the mode artifact root.
- `RequiresProgress`: requires a valid `ProgressState`.
- `RequiresVerdict`: requires a review verdict.
- `RequiresHandoff`: requires a handoff summary for sequential continuation.
- `RequiresCriteria`: requires acceptance criteria before the phase can start.

The refresher must reject incomplete successful runs instead of advancing the graph with malformed output.

## Result Bindings

Result bindings convert structured result fields into durable artifacts without adding mode-specific code to the artifact applier.

Current binding:

```go
ResultBindings: []ResultBinding{
	progressResultArtifact("modes/example-mode/progress.json"),
},
RequiresProgress: true,
```

The bound artifact path must also be valid under the mode artifact root. The builder adds the bound artifact to `OutputArtifacts`, and validation fails if a binding writes an undeclared artifact.

## Prompt Catalog

Operating-mode prompt catalog entries are generated by `api/internal/operatingmode/prompt_catalog_entries.go` from phase definitions. `api/internal/promptcatalog/catalog.go` consumes those entries.

For each phase, create a prompt-manager skill whose ID matches the derived `SkillID`:

```text
<PromptCatalogPrefix>-<PromptSuffix or Phase>
```

Examples:

- `swarm-manager-holistic-loop-investigate`
- `swarm-manager-phased-plan-execute-next`

`ValidatePromptCatalog` checks catalog ID, skill ID, mode, phase, and output paths. Phase starts fail closed if the prompt catalog is missing or the rendered prompt is empty.

## Metrics

Metrics semantics are registry-driven through `MetricsPolicy`.

Use `PhaseMetricsSpec` in the phase definition:

```go
Metrics: PhaseMetricsSpec{CountsReplanSample: true}
Metrics: PhaseMetricsSpec{CountsAcceptanceSample: true}
```

The builder derives:

- `Metrics.EventSource` from the mode ID.
- `Metrics.ReplanSamplePhases` from `CountsReplanSample`.
- `Metrics.AcceptanceSamplePhases` from `CountsAcceptanceSample`.
- `Metrics.AcceptedVerdicts` when at least one acceptance phase exists.

Do not add phase-name checks to `api/internal/stats/engine.go`.

## Capabilities

The API declares mode capabilities in both catalog and workspace responses. UI and CLI code should consume these fields instead of inferring mode behavior locally:

- `supports_phases`
- `can_start_phases`
- `can_complete_items`
- `can_apply_backlog_sync_proposals`
- `requires_acceptance_criteria`
- `supports_artifacts`
- `supports_handoffs`
- `uses_item_execution_flow`

Capabilities are derived in `api/internal/operatingmode/workspace.go` from registry policies and phase contracts. New mode behavior that affects rendering should be exposed through capability derivation or phase actions, not through frontend mode-name checks.

## Backlog Audit

Initiative modes can execute initiative-wide work, but backlog items remain the audit and scope trail. Agents must not edit backlog item `spec.json` files directly to mark work complete or create follow-ups.

Use the operating-mode reconciliation endpoints:

- `complete-items` for run-id-validated item completion.
- `apply-backlog-sync` for proposal-backed item create/update/follow-up reconciliation.

The reconciler attaches source metadata: entrypoint, initiative, mode, phase, round, run ID, and requester.

## Tests

At minimum, add or update tests for:

- Registry validation for the new mode and malformed variants.
- Prompt catalog generation and `ValidatePromptCatalog`.
- Catalog/workspace capabilities.
- Phase action transitions, including conditional routes.
- Output contract enforcement for required artifacts/progress/verdict/handoff.
- Result binding writes, if used.
- Metrics policy semantics, if the mode opts into replan or acceptance samples.
- Lifecycle behavior that is unique to the new mode.

Use `api/internal/operatingmode/synthetic_mode_test.go` as the reference harness for proving that a non-production mode can exercise the framework without leaking into the production registry.

Targeted validation:

```bash
cd scenarios/swarm-manager/api
GOWORK=off go test ./internal/operatingmode ./internal/promptcatalog ./internal/stats ./internal/agentactivity ./internal/initiativelock
```

When UI or CLI behavior changes:

```bash
cd scenarios/swarm-manager/cli && GOWORK=off go test ./...
cd scenarios/swarm-manager/ui && pnpm test -- --run
```

Scenario validation when available:

```bash
vrooli scenario start swarm-manager
test-genie execute swarm-manager --preset comprehensive
```

## Authoring Checklist

- [ ] The mode has one focused definition file.
- [ ] The registry map contains the new mode and `ValidateRegistry` passes.
- [ ] All phase purpose tokens are lowercase snake-case and owned by the mode definition.
- [ ] Every phase has a matching prompt-manager skill.
- [ ] Prompt catalog validation catches missing or mismatched phase metadata.
- [ ] Transition behavior is declared through `Transitions` and `TransitionRules`.
- [ ] Derived artifact writes use `ResultBindings`.
- [ ] Metrics semantics are expressed through phase metrics policy.
- [ ] UI/CLI behavior uses backend capabilities and phase actions.
- [ ] Backlog mutations flow through operating-mode reconciliation endpoints.
- [ ] Tests prove the new mode does not require shared framework branches.

## Prohibited Patterns

- Do not add `if mode == ...` behavior branches to shared framework code.
- Do not make UI components decide phase sequencing or backlog sync availability.
- Do not duplicate generated operating-mode prompt catalog facts by hand.
- Do not weaken output contracts to accept incomplete agent results.
- Do not use fallback prompts for repo-writing phases.
- Do not add per-phase constants to shared activity or lock packages.
- Do not create runtime-editable modes as part of static authoring work.
