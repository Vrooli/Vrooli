---
name: "local-experimentation"
description: "Run bounded local experiments against a baseline, admit resource and acquisition costs before trials, verify recovery and cleanup, and decide adopt, investigate, or reject from retained evidence."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["experiment", "benchmark", "capacity", "reproducibility", "local"]
  icon: "flask-conical"
  status: "active"
  revision: 4
  createdAt: "2026-10-20T00:00:00Z"
  updatedAt: "2026-10-21T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "scenario-dependency-analyzer"]
    commands: ["vrooli capacity", "scenario-dependency-analyzer deps", "search-hub query"]
  origin:
    kind: "authored"
---
## Practice focus: Local experimentation

Resolve one bounded hypothesis by comparing a candidate with the current baseline under declared resource and recovery limits. Retain enough evidence to reproduce the decision, including failures and routes that were not tested.

### Scope

Use this method for uncertain local tools, models, or production techniques. It owns experimental judgment and evidence. Existing capacity, acquisition, lifecycle, and media owners retain execution and state. It does not authorize purchases, license agreements, credential changes, publication, service eviction, or security changes. Use `scientific-debugging` for a known defect and `scenario-work-ladder` for a scenario repair.

Keep one record with the existing effort; use [the record guide](references/record.md) for applicable evidence. Read `path:docs/configuration/host/capacity.md` for capacity-governed work, `path:docs/package-governance.md` for acquisitions, and `program-runtime` only when reusing a governed program. Owner contracts define mechanics; this skill does not create another scheduler or admission policy.

For adaptive optimization or copied production code, read [adaptive and external experiments](references/adaptive.md). Use `adaptive-research` when the next question must be discovered from evidence. A retained report or reproducible prototype can complete research without maintained integration.

### Phase 1: Bound the question

**Entry:** A user-authorized outcome has an owner and destination.

1. Recall relevant experiments with `search-hub query "<experiment intent>" --type library,record,skill,doc`. Follow the repository fallback if discovery fails.
2. Inspect ongoing work, source state, and the supported execution surface. Record actual permissions; requested access is not observed access.
3. State one falsifiable hypothesis and the baseline. Freeze common inputs and acceptance criteria before viewing candidate outputs.
4. Bound candidates, repetitions, elapsed time, tokens, and peak storage/resource demand for the selected route. Include warmups, failures, verification, and recovery costs. For unattended work, qualify the existing execution owner under [the unattended guidance](references/adaptive.md#bound-unattended-work).
5. Define stopping criteria: failed admission, new terms or authority requirements, OOM, resource interference, repeated failure limit, or exhausted budget. Stop only task-owned work.

**Exit/artifact:** A dated experiment record fixes scope, owners, comparable inputs, success/failure criteria, budgets, and stop conditions. If these cannot be fixed, end with `investigate`; do not start a heavy trial.

### Phase 2: Admit the exact route

**Entry:** The record is fixed and no conflicting writer owns the same effect. Apply acquisition and capacity checks when that route needs them; a lightweight check does not require a new lease or installation.

1. Pin the inputs and dependency identities needed for reproducibility. Record unknown identities and local/cloud behavior.
2. Route acquisitions and dependency previews through Scenario Dependency Analyzer or the existing resource/model owner. Record terms, expanded bytes, install/cache ownership, and simultaneous storage demand before acquiring.
3. Use the capacity owner's current observations and admission contract for governed execution. Record effective enforcement mode and result. An advisory warning, recorded claim, enforced refusal, and caller safety hold have different meanings; do not reinterpret them in a private policy.
4. Compare peak demand plus headroom with fresh available capacity. Obtain and retain any required owner lease across the heavy lifetime. A passive observation is not a reservation. Unknown demand or missing required admission means `not_tested`.

**Exit/artifact:** Evidence identifies an executable route, marks inapplicable acquisition/admission gates, or names the failed required gate. Do not acquire weights for a route that already fails memory, disk, terms, or authority admission.

### Phase 3: Run sequential trials

**Entry:** Inputs, identities, budgets, and required owner gates pass for this trial.

1. Use the supported owner for submission, status, cancellation, and output retrieval. Record reproducible inputs and the owner operation identity or synchronous request scope before waiting. A synchronous path does not require a new job registry.
2. Count pre-execution admission refusals separately from executed trials, while retaining both. Resubmit a refused slot only after fresh evidence shows its failed gate may have changed and the preset retry policy permits it. Record any authorized protocol amendment before dependent work; never overwrite the earlier refusal or change acceptance after seeing output.
3. Run heavy trials sequentially without overlapping heavy acquisition/extraction. Reuse existing owner operations and suitable governed programs; retain their input/contract identities and receipts.
4. Measure the declared outcome and resource costs. Separate setup from trial and selection costs. Label sampled peaks and unavailable metrics; preserve metric definitions across candidates.
5. For long-lived effects, reuse applicable owner cancellation evidence or qualify the selected path once while active. Verify final owned processes/leases and partial outputs. Reconcile the original operation before retrying; retain qualification costs and failures.
6. Verify actual outputs against hashes and technical acceptance, then review the artifact for the declared usefulness criteria. A successful command or decodable file is not acceptance.

**Exit/artifact:** Every preset trial has a terminal receipt or an explicit unresolved state. Any unresolved effects stop duplicate/dependent trials. A trial that was refused before execution is an admission failure, not a model-quality failure or a cancellation test.

### Phase 4: Decide and close

**Entry:** The trial budget is exhausted, a stopping criterion fired, or the hypothesis has enough evidence.

Use the first matching row for each route:

| Evidence | Decision | Next action |
|---|---|---|
| A required effect, acceptance, permission, identity, admission or cleanup check remains unverified; or route is `not_tested` | `investigate` | Name the exact gap, owner and bounded next check. Preserve the baseline. |
| Completed comparable trials fail a preset quality/cost criterion or the hypothesis | `reject` | Retain the failure and baseline. Do not expand the budget to chase a win. |
| Comparable trials pass quality/cost criteria and recovery/cleanup, and improve the declared outcome over baseline | `adopt` | Retain the result and name the smallest maintained integration. Implement integration only when it belongs to the authorized outcome. |

1. Retain accepted outputs, reproducible inputs, source/configuration hashes, receipts, and the decision. Inventory each retained path, bytes, purpose, owner and license.
2. Preview cleanup against the task ownership ledger. Prefer the owner's reversible removal or quarantine. Remove only verified task-owned expendable assets; never clear global caches, unrelated processes, or shared model versions.
3. Verify cleanup through owner status/claims and filesystem inventory. Report remaining leases, processes, caches, outputs, and uncertain effects explicitly. Keep recovery evidence until its owner accepts closure.
4. Record the result with `vrooli-memory journal note --kind work-record`. Reuse existing owner issues for gaps.

**Exit/artifact:** The record supports `adopt`, `investigate`, or `reject`; retained assets and remaining effects have owners. Adoption names the maintained integration, version/admission/status/cancellation/provenance behavior, focused checks, and a useful acceptance artifact. Do not claim integrated production support from a scratch success.

### Anti-patterns

| Mistake | Consequence | Replacement |
|---|---|---|
| Admit against nominal GPU size or yesterday's free memory | Competes with current work | Fresh observations plus owner admission |
| Install first and measure disk afterward | Extraction/copies exhaust storage | Peak simultaneous storage budget before acquisition |
| Change prompts, seeds or acceptance after seeing one result | Invalid comparison | Frozen shared inputs and explicit new experiment |
| Retry an uncertain job or kill all matching processes | Duplicate effects or unrelated loss | Reconcile the exact job and cancel only its owned scope |
| Hide failures or call an untested route rejected | Misleading selection | Preserve all preset trial dispositions and unknowns |

### Troubleshooting & Edge Cases

| Observation | Action |
|---|---|
| Executor cannot read GPU or local sockets | Report the executor restriction. Use supported reviewed diagnostics; do not infer a host outage or bypass containment. |
| Exact version, terms or memory requirement is unknown | Mark the route `not_tested` and decision `investigate`. Do not replace it silently with another model. |
| Owner has no suitable admission, acquisition or cancellation operation | Record the missing operation against that owner. Continue independent baseline work only if its required gates pass. |
| Cancellation returned but children/leases remain | Hold retry and cleanup. Preserve receipt and recovery ownership. |
| Candidate is text-only but comparison uses reference images | Label conditioning as different. Do not rank it as an equivalent image-conditioned trial. |
| Baseline already meets acceptance and no candidate can run | Retain baseline evidence. Report the comparison as incomplete; do not invent a winner. |
