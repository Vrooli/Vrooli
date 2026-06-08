# Problems — CLI Health

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### Pre-existing STANDARDS + security campaign blocks `vrooli scenario test`

**Symptom:** `vrooli scenario test cli-health` fails the `standards` phase
(`fail_on=high`, highest=critical) and reports security findings. As of
2026-06-03: 236 findings (4 blocker/error, 111 warn, 121 info), 36 standards
violations (2 critical, 33 medium, 1 low).

**Cause:** Template/scaffolding and repo-wide debt, **not** the
aisearch-adoption-hardening work:
- **critical** `api/internal/httpc/doer.go:34` "HTTP Client Without Timeout" —
  false-positive: the flagged line is a *comment* (`// callers pass
  &http.Client{...} directly`), not code.
- **critical** `Makefile` "Scenario Required Structure / lifecycle wrapper
  targets" — template structural gap.
- **medium** "Unstructured Logging in API" across `aisearch/service.go`,
  `command_index.go`, `main.go` — the whole package uses `log.Printf` by
  convention; converting to `slog` is a package-wide refactor (separate
  campaign). New log lines added by this work follow the existing convention.
- **security** `osv.GO-2026-5038 / GO-2026-5039` stdlib@1.25.0 (api + cli
  `go.mod`) — bump toolchain to ≥1.25.11; repo-wide, toolchain-level.

**Workaround:** The actual code gates are green — `gofumpt`, `go build`,
`golangci-lint`, `go test` (api + cli + packages/aisearch-go), and the UI
`eslint` / `tsc` / `vitest` all pass. Track the standards debt as a campaign.

**Real fix:** `architecture-cartographer campaign create cli-health
--from-audit <audit.json>`; fix the httpc false-positive in scenario-auditor's
`resource-management-v1` rule; add the Makefile lifecycle targets; bump the Go
toolchain repo-wide.

**Owner:** unassigned (campaign).

**Refs:** `docs/plans/aisearch-adoption-hardening-plan.md`; auditor run
`scenario-auditor standards scan cli-health`.

### Two unsynchronized cli-health command-search corpora — RESOLVED (Phase 1)

> **RESOLVED 2026-06-07 (Search Self-Tuning System, Phase 1).** Both corpora were
> collapsed into one `tests` block (`cases` + `negatives`) inside the new
> scenario-owned SSOT `scenarios/cli-health/.vrooli/search.json` (provider
> `cli-health.commands`). Both old files were **deleted**
> (`testdata/search_queries.json` and `search-hub/.../eval/seeds/cli-health.commands.primary.json`);
> `loadCommandCorpus` (recall_test.go) now reads gradeable cases from the SSOT and
> the recall gate builds its engine from the SSOT tuning (gate ↔ prod can no longer
> drift). The union was preserved: full-path labels (`expected_paths`) + leaf-id
> labels (`expect_ids`) + per-case difficulty signals (`tags`,
> `expect_within_top_k`, `expect_min_score`) + the gibberish negative. Soft
> eval-only cases (validate-manifest, remove-bookmark, embeddings) live in `cases`
> WITHOUT `expected_paths`, so the recall gate skips them. search-hub's eval suite
> is re-derived from the same file via self-registration in **Phase 2** (its
> embedded eval seed is gone; `seeds_test.go` relaxed accordingly). The historical
> divergence write-up is retained below for context.

**Symptom (historical):** The same provider (`cli-health.commands`) was described
by **two hand-curated, drifted test corpora** that nothing kept in sync:

| File | Count | Schema | Read by |
|---|---|---|---|
| `scenarios/cli-health/testdata/search_queries.json` | 20 | `{version, _note, cases:[{id, query, expected_paths:[<full command path>]}], scoring:{recall_at:5, recall_target:0.8}}` | `loadCommandCorpus` in `internal/aisearch/recall_test.go` → the REQ-P0-004 gate `TestCommandRecall` + the recall-experiment harnesses |
| `scenarios/search-hub/api/internal/eval/seeds/cli-health.commands.primary.json` | 9 | `{name, description, provider_id, suite_id, state, cases:[{case_id, query, tags:[…], expect_ids:[<short id>], expect_within_top_k:3, expect_min_score:0.5, note}]}` | search-hub eval seed loader (`internal/eval/seeds`) |

Divergences beyond the count: (1) different label vocab — `expected_paths`
(full `"<origin> <group> <name>"`) vs `expect_ids` (short leaf ids like
`restart`); (2) the search-hub corpus carries per-case difficulty signals the
cli-health one lacks (`tags:["strong"]`, `expect_within_top_k`,
`expect_min_score`) **and a gibberish negative** (`gibberish-1`), while
`search_queries.json` has **zero negatives**; (3) different query phrasings for
the same intent (`"restart a scenario"` vs `"how do I restart a scenario"`).
Optimizing search against either in isolation is overfittable, and the two can
silently disagree about what "good" means for the same provider.

**Root cause:** Pre-`search.json` history — the cli-health corpus was authored
for the REQ-P0-004 recall gate; the search-hub corpus was authored separately
"from the review's queries" for the federated eval tracker. There was never a
single scenario-owned source of truth for descriptor + tuning + tests.

**Fix delivered (Search Self-Tuning System, Phase 1 — greenfield, no bridge):**
collapsed **both** files into one `tests` block (`cases` + `negatives`) inside
the scenario-owned `scenarios/cli-health/.vrooli/search.json` SSOT; **deleted**
both old files; repointed `loadCommandCorpus` (recall_test.go) to read cases from
`search.json`. search-hub will derive the cli-health eval suite from the same
file via self-registration (Phase 2) rather than its embedded seed (deleted in
Phase 1). The union was preserved: full-path labels, leaf-id labels, the per-case
difficulty signals, and the negatives. See
`/home/matthalloran8/.vrooli/plans/search-self-tuning-system-search-json-ssot-secured-override-reindex-contract-overfit-safe-sweep-corpus-generation.md`
§Phase 1.

**Owner:** Search Self-Tuning System plan (Phase 1 complete; Phase 2 wires the push).

**Refs:** Search Self-Tuning System plan §0/§3.6/§7-Phase-1; `recall_test.go`
`loadCommandCorpus`; `requirements/02-semantic-and-text-mode-command-search/module.json`
(REQ-P0-004 gate).

### Phase 0 greenfield cleanup — measured-and-rejected search levers deleted

**Symptom (resolved):** cli-health carried two embedding/scoring "levers" that
prior measurement had shown to HURT recall (enriched embedding text 0.70→0.40;
canonical-origin authority boost 0.70→0.65) but that remained as live,
production-reachable code.

**Resolution (2026-06-07, Search Self-Tuning System Phase 0):** deleted from
`internal/aisearch` — `composeCommandEmbeddingTextEnriched` and its helpers
(`humanizePath`/`splitIdentifier`/`splitCamel`/`cleanDescription`/`indexFold`),
plus `newAuthorityDecorator`/`CanonicalOriginBoost`/`canonicalOrigin` and the
now-producerless `Options.Decorate` passthrough. `enrichment_test.go` (tested
only the deleted code) was removed; the recall-experiment harnesses kept only
their live-lever arms (task-prefix, hybrid, rerank, floor). The generic
`Compose`/`Decorate` seams remain in `packages/aisearch-go`; the hybrid
sparse leg (`Options.Sparse`) is retained as a genuine live lever. Findings
preserved in `packages/aisearch-go/docs/graduation-retrospective.md`. Surface
shrank (one fewer test file, ~150 fewer LOC); `go build`/`go vet`/`gofumpt`/unit
tests green.

**Refs:** `packages/aisearch-go/docs/graduation-retrospective.md` ("Things that
sounded right and measured WRONG"); Search Self-Tuning System plan §0/§Phase 0.

### 2026-06-08 — Self-registers `cli-health.commands` with search-hub at boot

**Symptom:** (by design) cli-health now pushes its search provider descriptor to
search-hub at startup instead of relying on an operator to register it.

**Root cause / change:** Search Self-Tuning System plan Phase 2. `main.go` launches
`searchregister.Register` (from `packages/searchregister-go`) in a background
goroutine, reading the `cli-health.commands` descriptor from the `.vrooli/search.json`
SSOT and upserting it via `RegistryService.RegisterProvider`. search-hub is declared
an **optional** scenario dependency (`.vrooli/service.json` →
`dependencies.scenarios.search-hub`, `required:false`, `try_start`): if the hub is
down at boot, registration retries briefly then degrades, and cli-health serves
command search normally. The registry upsert is idempotent, so re-registering every
boot is safe.

**Deferred:** the registration carries only the descriptor today; the `tuning`/`tests`
blocks and the control token ride in once `registry.proto` gains those fields (plan
Phase 3). The override/reindex/config-write verbs that consume the token are Phases 4–5.

**Refs:** `api/main.go` (self-register goroutine), `.vrooli/service.json`,
`packages/searchregister-go/`; Search Self-Tuning System plan §7 Phase 2.

### 2026-06-08 — Shared SearchControlService replaces private reindex proto (Phase 5)

**Change:** the cli-health-private `cli-health/v1/reindex` proto + `handlers/reindex`
are **deleted**. cli-health now implements the shared, token-gated
`search-hub.v1.control.SearchControlService` (`handlers/searchcontrol`) — the same
reindex + config-write contract any search provider speaks, so search-hub's sweep
drives index-time experiments uniformly. `search.json` declares `reindex_endpoint`
+ `config_endpoint`; the CLI `reindex` group is repointed onto the shared service
and takes a `--control-token` flag (or `CLI_HEALTH_SEARCH_CONTROL_TOKEN`). The whole
plane is gated by `CLI_HEALTH_SEARCH_CONTROL_ENABLED` (default OFF) + the minted
control token.

**Known boundary (deferred to Phase 6 — NOT a bug):** `WriteConfig` validates and
atomically rewrites `search.json`, and on an INDEX-TIME factor change (engine /
embed_model / embed_task_prefix) it triggers a reindex and returns
`reindex_triggered=true`. But the live reconciler embeds with the **boot-time
recipe**, so an index-time recipe change only fully applies after the process
restarts and re-reads `search.json` (the boot path rebuilds the engine and the
recipe-aware drift hash re-embeds). The triggered reindex still reconciles corpus
membership and gives search-hub a job to poll. Making an index-time change apply
**in-process** (live engine rebuild + reconciler swap, so no restart is needed)
is the Phase-6 sweep's responsibility. Query-time-only changes need no reindex and
take effect on next boot. Until then, an operator applying an index-time tuning
change should restart cli-health.

**Manifest-validation note:** the cli-health manifest validator
(`internal/services/manifestvalidation`) now loads the shared `search-hub/v1/control`
proto alongside each scenario's own protos and classifies its services as **Shared**
— bindable from a CLI but NOT coverage-checked — so a provider that adopts the shared
control plane (binds e.g. `SearchControlService.Reindex`) is not flagged
`binding.unknown_service`, and an unbound shared RPC like `WriteConfig` is not an
orphan.

**Refs:** `handlers/searchcontrol/`, `api/main.go`, `cli/domains/reindex/`,
`cli/manifest.json`, `.vrooli/search.json`, `packages/aisearch-go/searchjson_write.go`,
`internal/services/manifestvalidation/{seams,protoloader}.go`; plan §7 Phase 5.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| UI reindex button (`ui/src/features/status/StatusPanel.tsx`, `ui/src/api/clients.ts`) | Calls the **deleted** `cli-health/v1/reindex` `ReindexService` (Phase 5). The UI still *builds* (its `@vrooli/proto-types` node_modules copy is a stale pre-control build that still ships the reindex TS types), but the button now hits an unmounted endpoint → 404 at runtime. | Medium — a visible operator action is runtime-dead. | Repoint onto the shared `SearchControlService` (TS gen already exists at `packages/proto/gen/typescript/search-hub/v1/control/control_pb.ts`; the node_modules proto-types package must be rebuilt first). BUT the control plane is token-gated + flag-gated, and a browser has no control token — so the correct fix is most likely to **remove** the unauthenticated UI reindex trigger (keep the status display) and drive reindex from the token-gated CLI (`cli-health reindex run --control-token …`) / the search-hub sweep. Deferred per plan §4 (UI out of scope) + §7 Phase 8 (adoption parity). Touches: `StatusPanel.tsx`, `clients.ts`, `StatusPanel.test.tsx`, `SearchPanel.test.tsx` + `ValidatePanel.test.tsx` (reindexClient mocks), and the `status.reindex*` strings/selectors. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
