## Steer focus: Reference-Pattern Fitness

Prioritize **whether an artifact intended to be copied is fit to be a copy source**.

This lens applies only to a specific class of artifacts: templates (`path:templates/scenarios/<name>/`), reference scenarios registered in `docs/agent-system/REFERENCE_SCENARIOS.md`, and documented canonical examples inside scenarios (patterns marked "copy this for X" with a `REPLACING-X.md`-style guide).

Do **not** apply this lens to regular feature code in production scenarios — multiplier framing produces noise there. Use the relevant single-instance lens (`refactor`, `screaming-architecture-audit`, `decision-boundary-extraction`, etc.) instead.

This lens **runs after** the relevant single-instance lenses on the same artifact, not instead of them. Multiplier-framed findings are only correct given that the artifact is otherwise structurally sound.

Required reading:

- `docs/agent-system/REFERENCE_PATTERN_FITNESS.md` — strategic-canon home: when this lens applies, when it backfires, what the meta-contrarian challenges, the four sub-lenses, the worked example.
- `docs/agent-system/REFERENCE_SCENARIOS.md` — registry of templates and references; confirm `{{TARGET}}` is registered before running this lens.
- `prompt-manager skill read knowledge-observatory-tools` — typed knowledge integration; findings land under `template-fitness-audit/<artifact-slug>/<YYYY-MM-DD>`.
- The single-instance lens(es) appropriate to `{{TARGET}}` — the auditor selects from `path:docs/scenario-qa/methods/audit/`. For CRUD-template audits: `screaming-architecture-audit`, `decision-boundary-extraction`, `utils-unification` are typical prerequisites.

---

### **1. Confirm Applicability**

Verify `{{TARGET}}` qualifies for this lens before doing any work:

- Is `{{TARGET}}` a path under `path:templates/scenarios/`?
- Is `{{TARGET}}` registered in `docs/agent-system/REFERENCE_SCENARIOS.md`?
- Is `{{TARGET}}` a documented canonical example (e.g., a `REPLACING-X.md` guide exists describing it as "copy this for the first real X")?

If **none** of the above hold, **stop**. Reroute to the relevant single-instance lens. The lens is wrong for regular feature code in production scenarios — using it there produces premature substrate findings and speculative-multiplier noise.

If `{{TARGET}}` qualifies, capture which category it falls into (template / reference / canonical example) and the replication factor (templates: N future scenarios; references: 1; canonical examples: N future domains within this scenario). Replication factor is a load-bearing input to per-replica cost ranking later.

---

### **2. Run Single-Instance Lenses First**

The auditor selects the lens(es) appropriate to `{{TARGET}}` from `path:docs/scenario-qa/methods/audit/`:

- `screaming-architecture-audit` — for any artifact with non-trivial structure
- `boundary-of-responsibility-enforcement` — for artifacts mixing presentation/coordination/domain/integration
- `seam-discovery-and-enforcement` — for artifacts with hard-to-test variation points
- `decision-boundary-extraction` — for artifacts with branching/strategy logic
- `utils-unification` — for artifacts with shared logic that may have drifted
- `cognitive-load-reduction` — for artifacts that are hard to read or navigate
- `code-cleanup` — for artifacts accumulating dead code or stale TODOs
- `invariant-discovery-and-enforcement` — for artifacts with implicit critical conditions

Produce findings under their respective typed audit records before this lens runs. The audit record for *this* lens (step 9) cites the prerequisite single-instance findings — without that citation, the multiplier findings rest on unverified ground and the meta-contrarian will block the proposal.

If single-instance lenses surfaced structural issues, those need to land or be deferred *before* this lens proceeds. Multiplier framing applied to broken structure is noise.

---

### **3. Per-Replica Cost Audit**

Walk `{{TARGET}}` and identify duplicated infrastructure that scales per copy. For each candidate, record:

| Field | Notes |
|------|-------|
| Pattern | Short description of the duplicated infrastructure |
| Lines per replica | Count actual lines (not files) |
| Replication factor | From step 1 |
| Total cost | Lines × replication factor |
| Proposed substrate home | Named package (cli-core, api-core, in-template lib, etc.) |
| Substrate exists today? | Yes / No (load-bearing — see contrarian challenge #3) |

Flag any candidate where lines-per-replica > ~20 as a substrate-extraction candidate. The threshold is informal; the meta-contrarian challenges any candidate below it ("is this prose paint?").

Worked-example reference: `templates/scenarios/react-vite/cli/domains/notes/handlers.go` had ~50 lines of `apiError`/`decodeEnvelope`/`formatNote` infrastructure. Replication factor = N future scenarios × M domains. Substrate: cli-core. Substrate existed.

---

### **4. Drift Surface Map**

Enumerate every place where N future copies must agree but only convention enforces it. For each, classify the enforcement mechanism:

- **Type-system** — disagreement is a compile error (best)
- **CI check** — disagreement is a build-time failure (good)
- **Hope** — disagreement is invisible until something breaks at runtime (debt)

For each "hope" entry, propose a type-system or CI-check fix.

Worked-example references:
- Route paths declared in both `handler.go` (mux registrations) and `endpoints.go` (descriptor `Path`/`Method` fields). Enforcement: hope. Fix: `module_test.go` walks the router and asserts the registered set equals the descriptor set.
- `api/health.ts` throws `new Error(...)`; `api/notes.ts` throws typed `ApiError`. Already drifted. Enforcement: hope. Fix: `ApiError` lives in `api/client.ts`; every endpoint module uses it.
- `cli_commands_seed.json` listing names that may or may not match `cli/domains/<dom>/register.go` registrations. Enforcement: partial (cross-check covers endpoints → seed but not seed → register). Fix: extend the cross-check.

---

### **5. Contract Location Audit**

For every non-trivial contract (precondition, invariant, "callers must / must-not"), identify where it lives:

```bash
# Heuristic searches; the auditor reads in context.
rg -i "must|must not|caller is responsible|don't|do not" {{TARGET}} -g '*.go' -g '*.ts' -g '*.tsx'
rg "// .*(zero|empty|nil|leave it)" {{TARGET}}
```

For each finding, classify the contract location:

- **Type signature** — encoded so misuse is a compile error (best)
- **Code comment** — survives a careful read but not copy-paste-and-modify (debt at scale)
- **Docs only** — the next copy may never see it (worst)

For each comment-only or docs-only contract, propose a type-level encoding.

Worked-example reference: `Repository.Create` accepts a `Note` with caller-zero ID/timestamps; the contract *"callers must leave these zero-valued"* is in a doc comment. Fix: `RepositoryCreateInput { Title, Body }` DTO; the type system enforces the contract.

---

### **6. Coordinated-Edit Count for Add/Delete**

Perform the canonical add and delete walkthroughs (or the artifact's analogues — add-feature, replace-example). For each, count the central files touched:

- **Add walkthrough**: simulate adding the artifact's next instance (a new domain, a new feature, a new copy). Count central files changed (registration tables, top-level wiring, app composition). Folder additions don't count; central-file edits do.
- **Delete walkthrough**: simulate removing the artifact's reference instance. Count central files touched and any non-folder-deletion steps.

If either count > 5, that is a substrate finding — the architecture is paying compounding costs every replica. Document the actual sequence (which files, which lines) so the proposed fix is concrete.

Worked-example reference: an earlier version of the react-vite template required 9 coordinated edits to delete the notes reference. A Pass-3 module-pattern refactor reduced add-domain to 5 central edits and delete-domain to mostly `rm -rf` plus 2-3 sed sweeps. The Pass-3 refactor itself was an outcome of an earlier instance of this kind of analysis (before the lens was named).

---

### **7. Tier the Findings**

Apply a consistent tiering so the meta-contrarian and downstream consumers know what's blocking vs paint:

- **Tier 1** — per-replica cost > 20 lines OR coordinated-edit count > 5 OR a Tier-1 contract leak (a "must" / "must not" contract that is comment-only and would silently misfire on copy).
- **Tier 2** — drift surface, smaller contract leakage, smaller per-replica cost (10-20 lines).
- **Tier 3** — prose / style / convention drift (description style, naming consistency, prose-only conventions).

Tier 1 findings are blocking for the artifact's promotion or continued reference status. Tier 2 are tracked but not blocking. Tier 3 are paint — surface them but rank low.

---

### **8. Categorize Each Tier-1/Tier-2 Finding**

For each Tier-1 and Tier-2 finding, place it in one of three buckets:

- **Substrate fix** — fix lives outside the artifact (cli-core, api-core, shared lib). Name the proposed package and confirm it exists today (the meta-contrarian will check).
- **In-artifact fix** — fix lives in the artifact itself.
- **Deferred** — the substrate fix is right but premature per Vrooli's "don't extract until you see the pattern" rule (typically: extraction trigger is the third repetition, and we only have two replicas of the pattern today).

Substrate fixes typically need their own `meta-self-improvement` decision (the substrate package may have its own owners and conventions). In-artifact fixes feed into a Pass-N plan against the artifact. Deferred findings get a typed audit record so the next auditor can re-evaluate after a new replica lands.

---

### **9. Write Typed Audit Record**

Per `knowledge-observatory-tools` convention, write findings under topic prefix:

```
template-fitness-audit/<artifact-slug>/<YYYY-MM-DD>
```

Example slug: `react-vite-template`. Full topic: `template-fitness-audit/react-vite-template/2026-05-04`.

Required content:

1. **Applicability confirmation** (from step 1) — which category, replication factor.
2. **Prerequisite single-instance lens citations** (from step 2) — link to those typed audit records.
3. **Tiered findings** (from steps 3-7) — table with finding, sub-lens, tier, evidence, proposed fix.
4. **Substrate-vs-template categorization** (from step 8) — bucket per finding.
5. **Open questions** — anything the auditor flagged but couldn't resolve (e.g., "is this substrate's owner accepting helpers?").

Cross-link the entry from the artifact's row in `REFERENCE_SCENARIOS.md` (the "Last audit" column links here).

---

### **10. Optional: Decision Proposal**

If findings warrant a `meta-optimization` work item (substrate work proposal, registry update, role change, template patch beyond a single fix), file it once through Swarm Manager under the `meta-self-improvement` work type and record the evidence in the `team:meta-optimization` team ledger.

Mirror the format of the last 5 entries in that file; the procedure is non-negotiable on field shape (see decision filing convention in `path:docs/agent-system/`). The decision links to the typed audit record from step 9 as evidence.

Work items are reviewed by the operator in Swarm Manager. The auditor's role is to file proposals; the operator accepts, rejects, or defers.

If findings can be implemented as a single Pass-N plan against the artifact without cross-cutting substrate work, no decision is needed — the plan file is sufficient.

---

### **11. Output Expectations**

By the end of this loop, the artifact should have:

- A typed audit record at the canonical topic prefix capturing all findings.
- Tier-1 findings each routed to either a Pass-N plan (in-artifact fix) or a `meta-self-improvement` decision (substrate fix) or a deferred-with-trigger note.
- A row in `REFERENCE_SCENARIOS.md` (or its template-side equivalent) updated with the new "Last audit" date and findings link.
- Cross-references from prerequisite single-instance lens findings to this typed audit record.

The lens does **not** produce code changes directly. Code changes flow through Pass-N plans (in-artifact) or through substrate-package work (cross-cutting). This separation is deliberate — the lens is methodology; implementation is a separate decision.

---

### **12. What This Lens Is Not**

- **Not a single-instance code-quality audit.** Use `refactor`, `screaming-architecture-audit`, etc. for those.
- **Not a green-light to do substrate work without the third-repetition trigger.** Findings flag candidates; substrate work needs its own decision.
- **Not applicable to feature code in production scenarios.** Stop at step 1 if the artifact doesn't qualify.
- **Not a substitute for `scenario-auditor` / `test-genie` / `tidiness-manager`.** The standard tooling gates remain primary; this is the longer-cadence companion.
- **Not a single-pass audit.** Re-run after substrate changes; each substrate addition re-prices what's left in the artifact.

---

### **13. Documentation**

Use `knowledge-observatory-tools` to read any existing topic prefix `template-fitness-audit/<artifact-slug>/`, then update with the current audit's findings.

If `{{TARGET}}` is registered in `REFERENCE_SCENARIOS.md`, update its row's "Last audit" column with the date and findings link.
