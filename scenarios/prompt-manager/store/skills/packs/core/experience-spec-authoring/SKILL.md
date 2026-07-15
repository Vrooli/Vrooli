# Experience Spec Authoring

## Purpose

Use this skill when authoring or updating a scenario's `experience/` folder:
the design-first UX contract validated by `experience-manager`.
`requirements/` says what the scenario does; `experience/` says what the UI
must communicate and how evidence should prove it.

## Required Reading

- `path:docs/reference/experience-alignment.md` — doctrine, finding codes, and
  maturity model.
- `path:.vrooli/schemas/scenario-experience-spec.schema.json` — canonical JSON
  schema.
- `path:scenarios/experience-manager/docs/reference/cli-commands.md` — validate,
  render, authoring, fleet, attestation, and fix commands.
- `path:scenarios/prompt-manager/store/skills/packs/core/prd-authoring/SKILL.md`
  — PRD targets are the roof that experience specs link back to.

## Authoring Flow

Start from route truth, not prose:

1. List user-facing routes from `ui/src/app/routes.tsx` or the scenario's route
   registry.
2. Ensure `experience/index.json` lists one page spec per real route that users
   should experience.
3. Keep each `page.prd_refs[]` linked to real PRD operational targets.
4. Validate early: `experience-manager spec validate <scenario> --json`.

Then deepen each page only as evidence becomes real:

- **L0 identity** — page id, title, route, purpose, PRD refs.
- **L1 priorities** — the ranked communication priorities, capped and written
  as outcomes.
- **L2 grounded claims** — elements, claims, and bindings that can be tied to
  accessible roles/names and stable selectors.
- **L3 states** — loading, empty, partial, stale, errors, and scenario-specific
  modes users can perceive.
- **L4 journeys** — cross-page flows whose steps name existing pages/states.

## Claims And Tiers

Author claims as perceivable outcomes, not implementation inventory.

- `machine` claims must be deterministic today. They should reference elements
  with stable role/name expectations and selector bindings.
- `manual` claims need human attestations with author, rationale, and expiry.
- `aspirational` claims record intent the validator cannot check yet; they are
  useful, but never gate.
- `custom` claims are open-world and may not be machine tier.

Bindings are the volatile seam. Refactors should usually update
`bindings.elements.<id>.testid`, not rewrite the claim.

## Affordance Heuristics

Declare the affordances users need to operate a component, not just the
component's existence. A table, list, form, or chooser that is present but
missing the expected controls is experience drift.

Use these defaults unless the PRD or page purpose clearly says otherwise:

- Tables with more than 10 expected rows declare sort on the primary decision
  columns, filter on status/category columns, and search when row names or free
  text are part of the workflow.
- Lists, queues, galleries, and catalogs declare search when users may need to
  locate a known item, and filter when the list has durable states, owners,
  types, or severities.
- Forms declare validation states for required fields, invalid values,
  submission failure, and successful save or create feedback.
- Destructive or irreversible actions declare confirmation, cancellation, and
  post-action feedback affordances.
- Long-running actions declare progress, stale/refresh, retry, and failure
  affordances.
- Multi-step flows declare the current step, completed steps, next action, and
  a safe way to leave without losing context.

For now, encode these as grounded claims using the closest deterministic claim
type available, or as `custom` manual/aspirational claims when no checker exists
yet. Keep the referenced controls in `elements` and `bindings` so the future
affordance checker can promote them without rewriting the page intent.

## Validation And Fix Loop

Use the deterministic loop before and after edits:

```bash
experience-manager spec validate "<scenario>" --json
experience-manager spec render "<scenario>" "<page>" --json
experience-manager fix preview "<scenario>" --json
experience-manager fix apply "<scenario>" --json
```

For manual claims:

```bash
experience-manager spec attest "<scenario>" "<page>" "<claim>" \
  --author "<name>" \
  --rationale "<what was checked>" \
  --expires-at "<RFC3339 timestamp>"
```

## Anti-Gaming Rules

Never:

- invent machine-tier claims for UI that is not implemented;
- mark pages `active` before the route reconciles green;
- add fake `data-testid` bindings that no element uses;
- delete claims or states just to silence findings;
- attach expired or unverifiable manual attestations;
- leave stale route specs after a route is removed.

Passing validation is the floor. The real goal is a true UX contract that lets
future agents detect drift instead of rediscovering it by inspection.
