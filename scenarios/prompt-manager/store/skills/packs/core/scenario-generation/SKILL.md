# Scenario Generation

## Purpose

Turn an approved scenario idea into a well-planned, lifecycle-valid permanent
capability through normal swarm-manager phased execution. Establish the
scenario-wide invariants and route each implementation slice to focused skills;
do not act as a one-shot generator or a substitute for those skills.

## Scope

**In scope:** template selection, the initial scenario plan, scaffold and
onboarding gates, product-contract setup, and the transition to evidence-driven
maintenance.

**Out of scope:** choosing every implementation detail, writing a complete
scenario in one pass, or maintaining a hard-coded map from technologies to
skills. Use prompt-manager discovery for the focused work in each slice.

## Required Reading

- `prompt-manager skill read implementation-plan-authoring`
- `prompt-manager skill read ecosystem-fit`
- The selected template's `README.md` and generated `docs/START-HERE.md`

## Plan Before Scaffold

The canonical plan-manager plan is the execution contract. Before a scenario
directory is generated, it must state:

- the capability, intended users, non-goals, and acceptance boundary;
- the selected template and why it fits the needed surfaces;
- the first real vertical slice and how it will be proven;
- the initial PRD/requirements, domain, API/CLI/proto, UI/UX, and validation
  work that applies to this scenario; and
- which decisions are fixed versus deliberately deferred.

Use the template registry as the authority. Do not default to a template merely
because it is familiar:

```bash
template-manager registry list --kind scenario
template-manager registry show "<template-id>"
```

| Decision | Plan requirement |
|---|---|
| Existing scenario or resource can satisfy the need | Stop and revise the idea around composition or extension. |
| A new scenario is justified | Name the selected template, rationale, and acceptance scope before scaffolding. |
| Template has a UI | Define user journeys, states, and design direction before treating generated pages as product UI. |
| API, CLI, proto, or persistence is needed | Put the applicable contract and domain slices in the plan; mark genuinely inapplicable surfaces explicitly. |

## Establish the Local Continuation Contract

Generate only after the plan selects a template:

```bash
template-manager generate "<template-id>" --id "<name>" \
  --display-name "<title>" --description "<one-line purpose>"
```

Immediately read `scenarios/<name>/docs/START-HERE.md`. It is the generated
scenario's local onboarding and continuation contract. The plan must require
every agent working an initial-development slice to read it before changing
that scenario, then follow its truthful gates rather than treating generated
examples or placeholders as product work.

The first post-scaffold slice must make the onboarding guide and its referenced
documents truthful for the chosen capability. Preserve durable template seams;
replace illustrative domains, pages, copy, and starter requirements only as
the real scenario decisions supersede them.

## Initial Development Slices

Use normal phased-plan drain. Keep each slice coherent and validated; do not
ask one agent to finish every surface.

| Slice | Required outcome | Skill routing |
|---|---|---|
| Foundation | Scaffold is lifecycle-valid; PRD and requirements are scenario-specific; START-HERE and the domain map are truthful. | This skill, then `prompt-manager discover` for the selected template and business-contract work. |
| Contracts and domain | Real bounded contexts own their data, flows, API/CLI/proto contracts, and integration decisions. | Discover architecture, API, CLI, proto, storage, and dependency guidance that the plan needs. |
| Vertical slice | One user-valuable flow works end-to-end with intentional UX states and design language. | Discover the relevant domain and UX skills; read `DESIGN.md` and experience docs when present. |
| Evidence | Tests, requirements links, and applicable Test Genie evidence prove the slice; gaps are recorded as bounded follow-up work. | Use the test and provider-recommended skills relevant to the evidence. |

For every slice:

1. Recall prior work with `search-hub query "<slice intent>" --type record,skill,doc`.
2. Run `prompt-manager discover` for the slice's domain, technology, and
   surface; read only skills that materially improve that slice.
3. Record the true frontier in the plan-manager handoff so the next agent can
   continue without rediscovering the scenario's foundations.

## Product Contract and Documentation

The business contract is owned by business-health. Preserve a refined baseline
when one exists; otherwise use the wizard, validate the result, and replace
template starter requirements before product implementation:

```bash
business-health wizard start "<name>" --interactive
business-health wizard preview "<name>"
business-health wizard apply "<name>"
vrooli scenario requirements validate "<name>" --json
```

Make the plan explicitly cover the applicable local authorities:

- `docs/START-HERE.md` — initial-development gates and agent continuation;
- `PRD.md` and `requirements/` — product intent and verifiable obligations;
- `docs/concepts/DOMAINS.md`, `FLOWS.md`, `DATA.md`, and `INTEGRATIONS.md` —
  owned boundaries and intentional omissions;
- `DESIGN.md` and experience documents — user journeys and UI quality where a
  UI exists; and
- API, CLI, proto, configuration, operations, seam, and testing references
  for every surface the plan introduces.

## Transition to Maintenance

Initial development ends when a real vertical slice is lifecycle-valid and has
the applicable evidence—not when every aspirational feature is built. Subsequent
maintenance is ordinary swarm-managed work: Test Genie supplies structured
evidence and descriptor-presented skill recommendations; swarm-manager applies
its configured policy, creates or executes bounded work, then remeasures.

Do not create a second task queue or rely on a separate steering profile. The
plan and swarm-manager execution history are the durable handoff.

## Anti-Patterns

- Do not scaffold before the plan names and justifies its template.
- Do not treat starter pages, example domains, or starter requirements as the
  product.
- Do not replace focused skill discovery with a giant all-purpose generation
  prompt.
- Do not declare a scenario complete without a working vertical slice and
  applicable evidence.
