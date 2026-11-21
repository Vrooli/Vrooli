# Git Control Tower maturity scope assessment

Date: 2026-09-05. Status: discussion input, not an implementation plan or an approved replacement PRD.

This assessment combines current source inspection, Swarm Manager goal/backlog reads, the scenario skill/program standards, and official host webhook documentation. It does not establish a readiness score. No scenario test suites, agent runs, repository mutations, or remote writes were executed. Existing tests contain real Git mutations, so running the suite requires a prior safe-test-boundary change.

## 1. Product direction and authority

Proposed direction: GCT owns repository/change identity, attributable change evidence, grounded summaries and drafts, advisory review, repository-host collaboration surfaces, and operator-facing repository controls. Skills carry judgment; programs compose bounded operations; the scenario owns durable facts and invariants.

The operator's constraint is authoritative: potentially destructive Git operations are human-only, including testing. Agents do not commit, push, merge, revert, reset, checkout/switch, stash, manipulate worktrees, stage/unstage, or arrange fixtures using those operations. A temporary repository or sandbox is not an exception. Existing human controls can remain a product capability; their implementation tests use fakes, supplied immutable data, and read-only inspection. Real actuation evidence must be human-produced and separately identified.

Repository reads must also avoid hidden effects: no implicit fetch, submodule update, hooks, external diff/textconv execution, or executing code from an untrusted PR. A local snapshot export can create a new artifact without modifying source Git state. Publishing comments, PRs, releases, or mirror updates is a separate external-write authority, not permission conferred by a request to draft text. No such publication was authorized in this assessment.

## 2. Reading map

Paths below are repository-relative. Registry skills are read through `prompt-manager skill read <id>`; their source paths can move.

### Core concepts

| Read | Why |
|---|---|
| `docs/concepts/RECURSIVE_SELF_IMPROVEMENT.md`, especially section 4 | Three-speed stack and the requirement to simplify skills/programs after scenario promotion. |
| `docs/agent-system/SKILL_AUTHORING.md`, especially Scenario skill sets and Universal quality bars | Usage/improve/feature roles, step rungs, learning spine, programs as steps, conditioning defects and contract prompts. |
| `docs/agent-system/LAYERS.md` and `PRIMITIVES.md` | Separate authoritative truth, judgment, execution, and captured learning. |
| `docs/agent-system/PROMOTION_LADDER.md` | What earns promotion; what prose should be retired or retained. |
| `docs/agent-system/TARGET_MODEL.md` and `FRAMEWORK_HEALTH.md` | Setpoints, sensors, ownership, deadbands, and control-loop boundaries. |
| `docs/concepts/ECOSYSTEM.md` | Placement of integration infrastructure, interface enablers, products, and reusable capabilities. |

### Authoring and quality skills

| Skill | Use |
|---|---|
| `skill-set-authoring` | Derive owed roles, inventory real sensors/programs, declare the set and honest waivers. |
| `skill-authoring-tools` | Usage decisions and operational feature skills. |
| `skill-authoring-practice`, `skill-authoring-search` | Review/curation practices and provenance retrieval where their category applies. |
| `improve-skill-authoring` | Measured setpoint, sensor board, routes, anti-gaming, evidence, stop rules. |
| `skill-validation` | Executability, real references, capability map, failure behavior, and the divergence probe. |
| `skill-improvement-suggestions` | Conditioning cost, redundant instructions, compression and promotion candidates. |
| `program-runtime` | Governed composition, binding selection, typed inference/delegation, budgets and result interpretation. |
| `vrooli-memory` | Scoped recall, outcome-linked attempt capture, advice supersession, comparable learning measurements. |
| `measures-adoption`, `improvement-do-and-dont` | Real instruments and resistance to improving scores by weakening evidence. |
| `writing-standards` | Placement of controlled language, requirements and behavioral acceptance conventions. |

### Executable contracts

- `scenarios/program-runtime/docs/guides/program-construction.md`
- `scenarios/program-runtime/docs/guides/program-contracts.md`
- `scenarios/program-runtime/schemas/program-contract.schema.json`
- `.vrooli/schemas/service.schema.json` (skill declaration)
- `packages/proto/schemas/vrooli-memory/v1/learning/learning.proto`
- `scenarios/program-runtime/api/handlers/programs-validation/module.go`
- `docs/TESTING.md`

Some prose still describes programs validation as planned; the current provider module implements the `programs` phase and advertises execution support. Verify the actual provider contract during planning instead of carrying forward dated statements. Presence, static conformance, executed fixtures, and real-use efficacy are distinct claims.

### Maturity and future planning

Read `scenario-work-ladder`, `prd-authoring`, `requirements-traceability-steer`, `scenario-maturity-ladder`, `storage-steer`, `ecosystem-fit`, and `scenario-readiness-review`. The eventual planning workflow uses `implementation-plan-authoring`; execution uses `implementation-plan-execution`. This assessment does not run the maturity gates or claim to pass them.

Examples worth borrowing selectively: BAS and Device Control usage/improve skills and `author-flow` programs for exact identity and evidence-preserving repair; Deployment Manager for exact candidate evidence; Command Center for bounded briefings and operator feedback; Plan Manager for routing friction across all three layers; Web Search for structured advisory results and retained uncertainty. They are examples to validate, not blanket certifications.

## 3. Historical work inventory

Authoritative retrieval: `swarm-manager goals get --name <name>` and `swarm-manager backlog get --kind <kind> --name <name>`. Archived goals and dropped items are historical input, not automatic authorization to restore their original scope.

| Family | Records and desired outcomes | Reconciliation needed |
|---|---|---|
| Trailers | Archived goal `gct-commit-initiative-linking`; research `gct-trailer-system-design`; execute `gct-trailer-support`, `gct-swarm-manager-integration` | Parse/validate namespaced links, propose operator-editable trailers, preserve unresolved references. Reconcile initiative terminology with current work identities. |
| PR hosting | Active goal `gct-github-integration`; research `gct-github-api-design`; execute `gct-github-pr-api`, `gct-github-pr-ui` | Generalize host connection and PR identity; separate read/draft/publication/human merge. SSH credentials do not establish API authorization. |
| Release communication | Archived goal `gct-release-pipeline`; execute `gct-pr-description-generator`, `gct-release-notes-generator`, `gct-github-release-api`, `gct-release-authoring-ui` | Local drafting need not wait for hosting APIs. Generated PR prose is not sufficient release evidence by itself. Deployment Manager retains release/promotion authority. |
| Merge/conflict controls | Goal `gct-merge-and-conflicts`; research `gct-merge-architecture`; execute `gct-merge-api`, `gct-conflict-resolution-ui`, `gct-merge-time-checks` | Human-only operations. Shared readiness policy rather than a second gate configuration. Whether all new human controls belong in the first comprehensive scope remains an operator choice. |
| AI provenance | Archived goal `git-control-tower-ai-provenance`; execute `gct-pending-ai-provenance-hardening`, `gct-committed-ai-provenance-history` | Preserve pending/committed standing, run outcome, per-file state, origin and uncertainty; reuse Workspace Sandbox/Agent Manager authority. |
| Provenance search | idea `build-git-control-tower-git-provenance-search` | A real indexed provenance corpus, filters, freshness and retrieval evaluations, federated through Search Hub. Distinct from regex file-content search. |
| Undo | research `run-level-undo-and-revert-design`; execute `run-level-undo-and-revert` | Human-only, with partial/mixed provenance explicit. Do not infer reversibility from pathname overlap. |
| Known repair work | execute `restore-git-control-tower-api-test-harness-compatibility`; fix `gct-review-summary-tests-missing-on-failed-check`; fix `readiness-git-control-tower` | Reproduce only through a safe validation path; readiness is an outcome, not a substitute for defined obligations. |
| Superseded proposals | Dropped `gct-security-review-tab`, `gct-commit-level-gating`, `gct-commit-level-agent`, `qa-git-control-tower-tests-playbook-schema-20260515`, and `ecosystem-loop-health-snapshot-contract` | Read rationale before any revival. The old merge goal references gating machinery whose historical owner item may no longer represent the current design. |

The full scenario-filtered backlog read returned 30 records, including cross-owner sandbox research/rollout and completed code-quality work. Those dependencies must be reconciled rather than copied into a new plan as unfinished GCT implementation.

## 4. Current code findings and maturity work

These are source observations and design consequences, not a complete security audit or runtime verification.

### Contract and authority

- `PRD.md` still calls for agents to perform version-control workflows and includes agent-facing stage/commit/branch/stash capabilities. Its single-repo direction also lags the implemented repo registry. Rewrite the contract around the accepted human/agent boundary before linking new requirements.
- `api/routes.go` exposes legacy REST mutations; `api/connect_wiring.go` mounts newer Connect services. `api/internal/policygate/interceptor.go` maintains a separate mutation map containing Worktree procedures; unlisted procedures pass through that interceptor.
- `api/internal/policygate/policygate.go` treats unknown callers as non-agents; caller and override headers are signals, not independently verified human identity. Require authenticated authority at the owning operation, complete transport coverage, and a narrow agent capability surface. Do not label the existing coarse gate a sufficient public-webhook boundary.
- `.vrooli/agent-manager/reviewer.json` uses `code.default`, a 500-turn ceiling, and skipped permission prompts. `api/agent_manager_handler.go` requests sandboxed execution. Reuse the launcher only after defining and proving a constrained advisory profile whose setup cannot mutate Git.
- Tests in `api/repo_status_service_test.go`, `staging_service_test.go`, and `diff_service_test.go` invoke real checkout/commit operations. Replace fixture arrangement with safe supplied data and fake seams before automated execution; do not simply remove expected behavioral coverage.

### Shared evidence and durable reviews

- `api/diff_model.go` already supports staged/unstaged/untracked, base, commit and annotated content. Extend this foundation into an explicit change subject and bounded evidence bundle. Define merge-commit parent/base semantics, selected paths/hunks, renames, binary/large files, content digests, omissions and freshness.
- Current changes must be scoped explicitly in a shared workspace. No ambient active-repository switch should retarget an in-flight review. Host PR identity includes host instance, repository ID, PR/MR ID, base/head SHA and reviewed revision.
- `api/review_store.go` is an in-memory map; cleanup deletes entries by age, including entries still running. A mature advisory job needs durable state, idempotent creation, attachment/wait, restart recovery, terminal classification and retention independent of active work.
- `api/review_handler.go` rebuilds summaries after running checks. `fetchTestsDimension` in `review_handler_dimensions.go` reads the latest execution and returns nil when the provider is unavailable. Preserve each requested check's exact result and failure evidence; do not substitute whatever execution is latest. This is also a named backlog defect.
- Reuse Test Genie's receipts, descriptors, comparison semantics and artifact IDs. Keep test outcome, AI finding, draft completeness, operator acceptance and release authorization separate.

### Ownership and deduplication

- Consolidate legacy REST/handwritten wire models into typed domain services and generated clients as the relevant domains mature. `ui/src/lib/api-types-operations.ts` and `api-scenarios.ts` illustrate handwritten contracts alongside newer generated services. Retire the replaced routes/types with their callers.
- `api/connect_wiring.go` explicitly documents a second Git runner adapter. Share low-level safe execution mechanics while retaining narrow read and human-write interfaces; avoid one unrestricted runner for advisory code.
- `api/internal/dbschema/schema.sql` holds audit, repository and precommit domains together. Apply per-domain schema ownership and repository interfaces for new durable review/draft/provider state, migrating existing ownership deliberately under `storage-steer`.
- `approved_changes_model.go` lacks the run outcome, per-file provenance state and originating-surface fields called out in old work. Verify source-to-UI propagation, not only the schema of the producer. Path overlap alone is not exact authorship or proof a change entered a commit.
- Existing `ui/src/features/baselines/` already has shared primitives. Preserve that consolidation. Inspect legacy visual/workflow capture routes against the newer descriptor-based evidence surface before retiring any route: existence alone does not prove duplication, but two competing baseline/artifact authorities must not survive.
- The security posture and coherence docs contain placeholders; PROBLEMS includes historical scaffold statements alongside later shipped work. Update these to evidence-backed current contracts and retain dated findings as history.
- Check pagination, cancellation, bounded output, cache identity, accessibility, mobile layouts, safe diff rendering, and large repositories with meaningful representative fixtures. Recorded tidiness budgets are not a current measured score.

## 5. Proposed skill and program set

Names are illustrative until contracts are authored. Reuse one evidence bundle across all interfaces.

| Skill | Program composition candidates | Evidence for success |
|---|---|---|
| `git-control-tower` usage | Select existing safe operations; evidence and baseline reading | Correct explicit subject, next action and complete attempt record. |
| Change summary | `change-evidence`, `summarize-changes` | Claims grounded in selected change evidence; omissions and unknown intent visible. |
| Commit-message drafting | `draft-commit-message`, shared `trailer-context` | Editable message and supported links for the exact selected changes; no commit. |
| PR drafting | `draft-pull-request` | Reviewer-oriented problem/behavior/validation narrative tied to base/head, separate from publication. |
| Change review | `review-changes` | Actionable findings with locations, reasoning, evidence and coverage limits; no code modification. |
| Release-note drafting | `draft-release-notes` | Audience-appropriate change communication traced through source/PR/release evidence; no invented compatibility claim. |
| Provenance investigation | `provenance-context` | Resolved references and uncertainty, with source-backed results from the provenance owner. |
| Improve | `setpoint-read`, `learning-read`, bounded corpus evaluation where needed | Comparable before/after evidence and an attributable repair that simplifies the appropriate layer. |

A deterministic one-command operation remains a CLI leaf. Do not create every illustrative program unless composition earns it. Typed inference uses the shared AI infrastructure; delegated review uses a declared, constrained workflow. All programs have bounded inputs/output, classified failures, declared bindings/effects, budgets, fixtures and one interpretable result on each path.

Learning records retain failed/unresolved attempts, actual advice use, comparison context, evidence identities, caller provenance, observed effort and capture coverage. Test attempts remain separate. Improve metrics include scope accuracy, unsupported claims, stale-evidence rate, trailer correctness, review precision/known-defect recall, operator corrections, completion effort, binding condition and external friction. Human acceptance alone is not quality; fewer findings alone is not better review. Baselines remain unearned until representative comparable observations exist.

## 6. Integration Hub and host neutrality

The existing scenario is `integration-hub`, singular. Read its `README.md`, `.vrooli/service.json`, `docs/internal/SEAMS.md`, `api/hub.go`, `api/main.go`, and `api/credential_cli.go`.

Observed: a small programmatic pilot, generated ConnectionService contract, OpenRouter-specific dispatch, credential-authority references, JSON metadata persistence, thin CLI, and a Web Console connection (`scenarios/web-console/api/main.go`). No PRD, requirements tree, testing config or scenario skills were present in the inspected file inventory. There is insufficient evidence to assert how it was originally generated. Its current shape lacks much of the standard scenario maturity contract; an API-only scenario does not require a gratuitous UI.

Important maturation work before public/multi-user use:

- Verified identity: `identity()` accepts a caller-supplied identity header or hashes a bearer value; this code does not itself authenticate the asserted identity.
- Honest connection verification: `checkConnection()` checks credential-authority configuration. That does not prove the OpenRouter account or requested provider capability works. Credential presence, successful remote authentication and effective permissions need distinct standing.
- Real connector interface, OAuth/App installation lifecycle where applicable, token expiry/rotation/revocation, scope changes, installation/repository access changes, provider-instance identity and audit.
- Durable idempotency and failure recovery across credential side effects and metadata persistence, bounded retained state, concurrency, health semantics, HTTP timeouts/shutdown, tests and measured operations.
- Template alignment, PRD/requirements/domain ownership, generated CLI contracts, skills/programs, and discoverable measures.

Recommended ownership:

| Owner | Responsibility |
|---|---|
| Integration Hub | Connections, installations, credential references, consent and provider authentication lifecycle; shared connector lifecycle mechanics. |
| Credential authority | Secret material and secure credential operations. |
| GCT | Repository-host capability contracts, PR/MR/change/review/release semantics, exact revision mapping and presentation. |
| Switchboard | Conversational ingress, sender/thread trust, capability attenuation, budgets and reply routing when treating host comments as a conversational channel. |
| Agent Manager / Program Runtime | Governed agent execution and bounded compositions. |

Do not put every external service's business API into Integration Hub. A host adapter must expose capabilities explicitly: comments, review threads, checks/statuses, releases, diffs, pagination and permissions are not identical across GitHub, GitLab and Bitbucket. Support a finite tested adapter set plus an extension contract, not an unsupported promise that any host works.

Switchboard has actual channel/trust/dispatch code despite README/progress text claiming no implementation. Its maturity needs current validation; neither that prose nor file presence proves a working end-to-end Git-host channel. Reuse the owner boundary without assuming the integration is complete.

Operator direction after this assessment: GitHub, GitLab and other host connectors belong in Integration Hub. The operator will discuss Integration Hub's overall scope separately; do not prepare its maturity plan here. GCT depends on precise connection and host-operation contracts rather than taking over connector implementation.

## 7. Mention-driven assistance

GitHub exposes comment/review webhook events; an app can receive a comment event and inspect an explicit invocation such as `@vrooli summarize` or `@vrooli review`. GitLab and Bitbucket expose analogous comment events, with different payloads and permissions. This is event handling plus command interpretation, not a universal mention primitive.

Recommended behavior: verify delivery, deduplicate, resolve exact host/repo/thread/revision and requester authority, apply a bounded advisory command, produce a grounded result, and publish only through a separately authorized reply operation. Ignore self-generated events, bound spend/rate/output, cancel or mark results stale when head changes, and retain an operation ID for delivery recovery. A public mention does not grant access to internal plans, logs, credentials or unpublished changes.

Never execute PR code, hooks or user-supplied commands to answer a mention. Review content is untrusted data. Start with public-safe summary/review/explanation commands; allow broader internal context only through explicit data-access authority. External replies can be an install-time operator-authorized capability while repository mutation remains human-only; this decision is still open.

Official references:

- https://docs.github.com/en/webhooks/webhook-events-and-payloads
- https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
- https://docs.gitlab.com/user/project/integrations/webhook_events/
- https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/

## 8. Focused scenario repositories and promotion

Distinguish three products: a showcase with source/release links; a buildable source distribution; a bidirectionally synchronized development repository. Recommended starting point for source sharing: a reproducible, one-way source distribution with the monorepo authoritative. Do not imply independent buildability for a showcase-only projection.

A source distribution needs more than copying `scenarios/<name>`:

- Explicit source commit or approved content digest; source-to-export path mapping and export recipe version.
- Recursive shared package/proto/schema/assets/build-tool closure, local Go replacements, workspace package links and generated artifacts.
- Explicit runtime dependency requirements and remote/local modes; build closure and runtime closure are different.
- Allowlisted publishable files, secret/private-data exclusion, licenses/notices, safe symlink handling, and no inherited private Git history.
- Honest standalone build/run instructions, known limitations, reproducible checksums and provenance manifest.
- Clean export validation using prepared data without Git mutation; export failure when required files/dependencies cannot be resolved.
- Drift detection, operator-reviewed update preview, refusal to overwrite downstream edits, contribution routing back to canonical paths, and lifecycle for renamed/retired scenarios.

SDA should resolve dependency graphs and file/package obligations; its current DAG and bundle-manifest surfaces are useful inputs, not evidence that standalone source export is implemented. Deployment Manager owns target/release approval and artifact identity. A source-distribution ramp can own packaging if that becomes a recurring multi-scenario product. GCT owns repository mapping, host presentation, source provenance and operator publication handoff. Decide the ramp's placement through ecosystem-fit before naming a new scenario.

History-preserving extraction and bidirectional sync are substantially larger products, with privacy, attribution, conflict and publication consequences. Neither should be silently included in a promotional mirror. Creating or updating Git history and pushing mirror contents remain human actions under the current constraint.

Useful promotional outputs from this same foundation: scenario-specific README drafts, source links, release highlights, changelog entries, evidence-linked screenshots/demos supplied by their owners, install links and contribution guidance. GCT supplies verifiable change facts; product/marketing owners choose positioning and claims. Generated copy should never imply deployment readiness merely because an export exists.

## 9. Scope boundary for the eventual plan

One coherent GCT plan can cover the accepted product contract, safe authority/test boundary, shared change evidence, durable advisory operations, skills/programs/learning, trailers/provenance/search, host-neutral repository collaboration, mention integration, and selected operator-only controls. It must retire replaced contracts/workarounds and reconcile old backlog states and dependencies.

Integration Hub implementation is externally owned and excluded from this planning discussion. The operator supports investigating source distribution as a new deployment ramp; GCT retains its interface and publication handoff. This separation prevents the GCT plan from acquiring credential authority, general messaging, test execution, or all deployment packaging.

Before authoring: settle mirror mode; human-only merge/undo feature inclusion; host adapter support scope; publication/reply authority; source-export ownership; and the accepted definition of mature. Completion must mean all selected behavioral, authority, durability, evidence, performance and interface obligations pass, with unknown/unavailable evidence represented honestly. R3 or R4 is selected through the maturity owner, not asserted by this report. No full implementation sequence, phase estimates, or execution plan is created here.

## 10. Follow-up investigation: source ramp and UX

These are design proposals based on source and experience-contract inspection, not a live visual audit. No UI or production code was changed.

### Ramp fit and implementation implications

`packages/delivery-ramp-go/doc.go` and `adapters.go` already define shared probing/building/driving/distribution boundaries, artifact references, checksums, and target evidence. `scenarios/deployment-manager/docs/DEPLOYMENT-GUIDE.md` explicitly gives the ramp packaging, evidence production, and publication responsibilities while Deployment Manager owns the release decision. The ramp requests that decision. Reuse this direction.

There are two gaps to resolve rather than copying another ramp verbatim:

1. Deployment Manager's `ui/src/lib/tiers.ts` and `features/deployments/GuidedFlow.tsx` still use local/desktop/mobile/cloud/appliance choices. A source repository is a delivery format, not another numbered runtime tier. Introduce a capability-derived delivery-format choice and keep runtime requirements separate. Existing numeric aliases can remain compatible.
2. The shared ramp evidence contract includes protocol, visual, and release-visual profiles with visual launch/capture states. A source distribution needs a truthful nonvisual validation profile. Reuse artifact identity, bounded execution, persistence and governance, but prove source closure, archive integrity, reproducibility, documentation and promised clean build/run behavior. Do not manufacture video evidence or pretend a skipped visual check proves source-package quality. Real visual journeys remain appropriate if the exported application claims a working UI.

Working name: `scenario-to-repository`, displayed as **Source repository**. This names the user outcome without binding the ramp to GitHub or promising bidirectional mirroring. The name is a proposal, not a created scenario.

The ramp should own export recipes, deterministic assembly, source manifests, verification, destination mapping and publication receipts. Integration Hub owns host connectors. GCT supplies change/provenance evidence and review/presentation affordances. Deployment Manager owns exact-artifact approval. Distribution must preserve the human-only Git policy: preparing an archive/directory is distinct from creating commits, tags or pushes, including mutations through a host API. Publication cannot be reported complete until the human action has verifiable destination evidence.

A reusable source-closure primitive could also help plugin distribution, which asks for a standalone-installable scenario. Verify that seam against actual plugin contracts; do not make every ramp consume a new source packager by default.

### GCT workspace UX

Preserve the diff-first desktop layout and mobile panel navigation in `experience/pages/workspace.json`. Existing `AppPanels.tsx`, `CommitPanel.tsx`, `ScenarioReviewPanel.tsx`, `GitHistory.tsx`, and `SettingsTabIntegrations.tsx` provide the natural insertion points.

Add a persistent **subject bar** above evidence and advisory content:

- Repository and branch or remote PR/MR identity.
- Subject: selected current changes, staged changes, commit, range, or pull request.
- Scope: selected paths/targets and inclusion count, with an expandable exclusions list.
- Snapshot time/revision and a clear changed-since-review indicator.

Selecting a subject only changes the inspected data. Selecting a PR or historical commit never checks out a branch, fetches implicitly, or stages files. Keep the subject stable when the operator opens a different view; switching it explicitly creates/selects the corresponding draft/review context.

Suggested desktop shape: file list on the left, diff in the center, optional inspector on the right with **Summary / Findings / Drafts / Evidence**. On narrow displays these become full-width views; do not shrink three columns into a phone screen. Existing scenario review remains available for broad validation evidence, with the subject bar making the distinction from an exact change review clear.

### Concrete interactions

**Summarize selected changes.** Select files or a target group, choose Summarize, and receive a short explanation of behavior changes plus links to supporting hunks. Expand to rationale, risks and unknowns. Scope selection is independent of Git staging. Untracked, binary and omitted files must remain visible in coverage. A changed subject produces “Files changed since this summary” with an explicit refresh action; it does not silently overwrite text.

**Draft a commit message.** The existing commit panel gains a Draft message action. It first confirms the subject in plain language, then shows editable title/body and suggested work-link chips. Links distinguish verified references from unresolved suggestions. Applying the text to the editor is distinct from the human Commit action. Regeneration shows a proposed revision and preserves manual edits. A draft made from selected unstaged changes must not be silently used for a different staged set.

**Review a change.** Findings carry severity, a concrete failure explanation, file/hunk location, evidence and suggested next investigation. Show coverage separately from finding count: “No findings in 8 reviewed files; 2 files not reviewed” is truthful. Resolution or dismissal records a reason and the revision; neither implies the code was fixed. Reruns preserve previous findings and mark outdated locations. Existing test evidence and AI conclusions remain separate sections.

**Read or draft a PR.** A Pull Requests workspace lists host, repo, base/head, author, updated time and review standing. Selecting one reuses the same diff/inspector. Draft PR opens a text editor with summary, behavior, validation and limitations plus evidence references. Connection absence still permits local drafting from an explicit range. A separate operator-controlled publication action shows the exact destination and content; no automatic branch creation or push is hidden in that action.

**Trace changes.** AI Changes and historical provenance share a timeline of runs, applied changes, linked work and commits, preserving mixed/unknown attribution. Link chips drill into evidence instead of auto-selecting or mutating files. A public view shows only publishable references; private work IDs cannot leak through a generated description.

**Use a host mention.** The host comment is an entry point into the same review operation. Reply with a compact summary, exact reviewed revision, coverage, and a link to the result where access permits. GCT can show these requests in the PR's activity stream with queued/running/partial/completed standing and delivery failures. Do not create an unrelated chatbot product or bury failures in an agent transcript.

**Connection settings.** Keep repository Git transport credentials distinct from host API permissions. Show the Integration Hub connection, bound host/repository, last verification and capabilities such as reading PRs or posting replies. Manage connection links to its owner. A configured SSH credential must not make the UI claim PR API access.

### Source-repository UX

Entry points: **Share source** from a scenario in GCT, or **Source repository** as a delivery format in Deployment Manager. Both open the same ramp-owned distribution record with return links; avoid duplicate wizards and state.

The first-time flow is:

1. **Choose source:** scenario, exact source revision and distribution name. Working changes can produce a labeled preview, but are not silently assigned a commit identity.
2. **Review contents:** an export tree separates app code, required shared code, generated files, docs/assets, exclusions and unresolved dependencies. Explain why each shared component is included. Separate required runtime services from bundled source files.
3. **Preview the public repository:** render README, screenshots/demo links, installation instructions, limitations, contribution routing and canonical-source link. Suggest description/topics from approved product material. Private/internal references are excluded.
4. **Verify:** individual results for closure, build, promised runtime behavior, reproducibility and publication eligibility. Each failed row has a concrete next action. A successful archive alone does not earn “standalone app.”
5. **Review and hand off:** choose a connected destination, inspect visibility/name and exact update diff, obtain Deployment Manager's artifact decision, then perform the human publication step. Offer a prepared archive independently. Record publication only from destination evidence.

Afterward show a durable card: canonical revision, exported revision, destination URL, verification time, last publication and update standing. Suggested states: **Up to date**, **Source changes available**, **Export needs repair**, **Destination changed independently**, **Publication unverified**. For downstream edits, show the divergence and stop rather than overwrite. “Prepare update” reuses the same recipe and previews exactly what would change.

Promotion is helped by an honest, polished destination: clear product introduction, quick start, demo/install links, source provenance, release highlights and a contribution path. License choices and marketing positioning remain owner decisions; neither is inferred from package contents.

### UX implementation quality

GCT has a detailed workspace experience contract but no root `DESIGN.md` in the inspected tree. Establish its design contract while preserving the recently implemented mobile layout. Adopt shared RCL components through their supported asset workflow. New panels need keyboard/focus behavior, deep links, durable drafts, resumable operations, explicit empty/error/partial/stale states, accessible status wording, and safe rendering of source/Markdown. Do not show internal skill/program identifiers in routine product flows.

UI evaluation should include: changing selection after drafting, two concurrent operators, stale PR heads, missing host access, absent provenance, very large diffs, a disconnected learning store, mobile navigation, export closure failure and independently edited destinations. Use fake host adapters and safe supplied repository evidence; no Git mutation to set up or clean up tests.
