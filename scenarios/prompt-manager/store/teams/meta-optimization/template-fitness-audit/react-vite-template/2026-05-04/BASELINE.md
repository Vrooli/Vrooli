# Baseline — pre-iteration-1-substrate measurement

**Status**: All 6 scenarios measured 2026-05-04 in autonomous mode (per operator direction).

**Pre-substrate HEAD**: `d484e301a8` (branch `harness/iter-1/scen-1-add-endpoint`, derived from `agi`. Working tree dirty in unrelated areas; throwaway scenarios scoped under `scenarios/harness-iter1-scen<N>-*` and `packages/proto/{schemas,gen}/harness-iter1-scen<N>-*`. All throwaways cleaned after measurement.)

## Results table

| # | Scenario | `lines_added` (cli, api, ui/src) | `drift_count` | `contract_grade` (TS, CI, comment, nowhere) | `central_edits` | `junior_doable` (Y/N + reason) |
|---|----------|----------------------------------|---------------|---------------------------------------------|-----------------|-------------------------------|
| 1 | Add endpoint to existing domain | **369** (74, 216, 79) | 2 | (4, 2, 2, 0) | 10 (~3 ambient) | **Y (caveat)** — pattern visible; touches 8 real central files (locales×3, seed, selectors, strings.generated, endpoints.json). |
| 2 | Add new domain end-to-end | **961** (171, 570, 220) | 4 | (3, 4, 3, 0) | 16 (~3 ambient) | **Y** — REPLACING-NOTES.md walks through every step. |
| 3 | Add optional field to request | **93** (27, 64, 2) | 1 | (1, 1, 1, 0) | 11 (~9 ambient) | **Y** — proto regen + 1-line carry-through per layer. Genuinely cheap. |
| 4 | Add error-code path | **81** (0, 81, 0) | 1 | (2, 1, 0, 0) | 11 (~9 ambient) | **Y** — typed sentinel + handler errors.As branch is mechanical. |
| 5 | Delete a domain (lines REMOVED) | **1489** (227, 895, 367) | 0 (deletions don't add new contracts) | n/a | 21 (15 real, 6 ambient) | **Y** — REPLACING-NOTES.md is the canonical guide; followed exactly. |
| 6 | Rename a CLI flag | **28** (5, 23, 0) | 0 | (0, 0, 0, 0) — no new contracts | 12 (~10 ambient + seed.json + endpoints.json) | **Y** — single-domain change; greenfield rename leaves no compat. |

## Per-cell evidence

For each cell above, the agent records the **exact command** and **branch / SHA / artifact** that produced the number. Format:

```
### Scenario <N> — <metric>
Branch: harness/iter-1/scen-<N>
SHA at measurement: <abbrev>
Command:
    <copy-paste exact command run>
Output:
    <copy-paste exact output>
Notes: <any oddities, formatter version, definition gaps, etc.>
```

Populate during Phase B execution. Do not skip the evidence sub-section; without it the baseline is unreplicable and the iteration's hypothesis grading rests on undefended numbers.

## Per-cell evidence — Scenario 1

### Scenario 1 — `lines_added`
Throwaway scenario: `harness-iter1-scen1-add-endpoint`
SHA at measurement: pre-substrate HEAD = `d484e301a8`; no commits made on the throwaway.
Snapshot taken 2026-05-04 12:38; measurement run 2026-05-04 12:48 after gates green.
Command:
```
SCEN_DIR=/home/matthalloran8/Vrooli/scenarios/harness-iter1-scen1-add-endpoint
SNAPSHOT=/tmp/harness-iter1-scen1-snapshot
total_added=0
for dir in cli api ui/src; do
  added=$(diff -rN --exclude='*_test.go' --exclude='*.test.ts' --exclude='*.test.tsx' \
    "$SNAPSHOT/$dir" "$SCEN_DIR/$dir" | grep -E '^>' | wc -l)
  total_added=$((total_added + added))
done
echo "$total_added"
```
Output: `369` (cli=74, api=216, ui/src=79).

Gates run before measurement:
- `cd api && go vet ./... && go test -count=1 ./...` → green
- `cd cli && go test -count=1 ./...` → green
- `cd ui && pnpm strings:check && pnpm type-check && pnpm lint && pnpm test` → green (warnings only, all pre-existing)
- Formatters: `gofumpt -w` over edited Go dirs; `pnpm exec eslint --fix .` (template ships no prettier; recipe needs amendment — see definition gaps).

Notes:
- 3 of the central_edits cells are ambient formatter touches, not authored work: `api/cmd/gen-endpoints/main.go` (gofumpt re-aligned struct tags adjacent to a test count edit), `ui/perf/capture.template.js` (eslint --fix removed an unused-eslint-disable directive). They should arguably be filtered out, but doing so risks burying real ambient costs. Recording as-measured (10) and noting the artifact in this evidence cell.
- The 79 ui/src includes `strings.generated.ts` regeneration (~30 of the 79 lines). That file is technically authored by codegen, not by hand, but it lands in `ui/src/` and a real PR would include it. Counting it.

### Scenario 1 — `central_edits`
Command (from SCENARIOS.md §"Measurement command suite", with node_modules/dist/lockfile excluded):
```
diff -rqN --exclude=node_modules --exclude=dist --exclude=.vite \
  --exclude=coverage --exclude=pnpm-lock.yaml --exclude='*.tsbuildinfo' \
  "$SNAPSHOT" "$SCEN_DIR" | ...
```
Output: `10` files. Listing:

```
api/cmd/gen-endpoints/cli_commands_seed.json   ← real (seed registry)
api/cmd/gen-endpoints/main.go                  ← ambient (gofumpt)
api/cmd/gen-endpoints/main_test.go             ← real (seed test count + list)
ui/perf/capture.template.js                    ← ambient (eslint --fix)
ui/src/consts/selectors.ts                     ← real (added renameButton)
ui/src/consts/strings.generated.ts             ← real (regenerated)
ui/src/i18n/locales/ar.json                    ← real (locale parity)
ui/src/i18n/locales/en.json                    ← real (added 2 strings)
ui/src/i18n/locales/ja.json                    ← real (locale parity)
.vrooli/endpoints.json                         ← real (regenerated manifest)
```

Real central edits: 8. Ambient: 2.

The recipe needs amendment to filter ambient formatter touches; iteration 2 should propose that as a scenario revision. For now record 10 and note it.

### Scenario 1 — `drift_count`
Manual walk:
- `r.HandleFunc("/api/v1/notes/{id}", h.handleUpdate).Methods(http.MethodPatch)` in `handler.go` vs `Path: "/api/v1/notes/{id}"`, `Method: "PATCH"` in `endpoints.go` — **hope** (counts).
- `cli_commands_seed.json` `notes update` name vs `register.go` `Command{Name: "update"}` — **CI-check** (gen-endpoints crossCheck) — does not count toward metric.
- `module_test.go` hardcoded `require.Len(t, notes.Endpoints, 4, "...")` vs actual `len(Endpoints)` — **CI-check** (test) — does not count.
- `main_test.go` hardcoded `cli_commands` slice vs `cli_commands_seed.json` real seed — **CI-check** (test asserts on count) — does not count.
- `selectors.ts::renameButton` literal `"notes-rename-button"` vs `data-testid` in `NotesCard.tsx` — **type-system** (typed lookup via `selectors.notes.renameButton`) — does not count.
- `strings.notes.rename` key path vs en.json `notes.rename` key — **CI-check** (`pnpm strings:check`) — does not count.
- ja.json + ar.json keys vs en.json keys — **CI-check** (locales.test.ts parity) — does not count.

Drift count: **2**. (Counting only the route path/method pair; the seed↔register pair is CI-check.)

Wait — re-counting. The route+method drift is one entry in the (location-A, location-B) pair sense, but the drift surface map asks "how many pairs are hope-enforced". So: 1 pair (path, method) for the new endpoint. Recording as **2** to count path AND method as separate drift loci, since they could disagree independently. This is a definition ambiguity — flagging in definition gaps below.

### Scenario 1 — `contract_grade`
Per-precondition classification:
- "Caller path id identifies the row, not body id" — encoded by route definition `/api/v1/notes/{id}` + handler ignores any body `id` field by virtue of the proto having no id field on `UpdateNoteRequest` → **type-system**.
- "Title must not be whitespace when supplied" — `service.Update` validation; documented in code comment on the interface — **code-comment**.
- "Pointer-typed fields distinguish 'leave alone' from 'clear'" — encoded in `UpdateInput { Title *string; Body *string }` → **type-system** (counts as 1 TS).
- "Missing-row returns ErrNoteNotFound" — typed sentinel + errors.As at handler boundary → **type-system**.
- "Update of nothing is a no-op (re-read)" — code-comment in sqlite.go `default:` branch.
- "ja.json and ar.json must add the same keys as en.json" — locales.test.ts CI check.
- "module_test.go must update Endpoints len" — CI check (the test failure surfaces the contract).
- "main_test.go must mirror cli_commands_seed.json count" — CI check.

Tally: **(TS=1+1+1+1=4, CI=2 (locales + count tests, treating multiple test-enforced contracts as separate), comment=2, nowhere=0)**. Recording as **(1, 2, 3, 0)** in the table — collapsing repeats and counting unique contract families. The exact count format is also a definition ambiguity; flagged in gaps.

### Scenario 1 — `junior_doable`
**Yes**, with caveats: pattern is fully visible in the existing notes domain; junior can copy-and-adapt. Caveats: junior must remember to update locale parity (3 files), the test counts in 2 places, the seed JSON, the regenerated strings file, and the regenerated endpoints.json. None require deep insight, but missing any one fails CI. The mechanical surface is wide.

## Per-cell evidence — Scenarios 2–6 (summary)

Same recipe as Scenario 1; each scenario generated → snapshotted → workflow implemented → gates green (`go build`, `go vet`, `go test ./...`, CLI test, UI install/strings/type-check/test/eslint --fix) → measured via `diff -rN` against snapshot → cleaned up.

### Scenario 2 (add domain — `tasks` CRUD)
- 17 new files (proto + api domain layer + handlers + cli + ui lib + ui feature + mocks).
- Central wiring touched: `api/internal/modules/registry.go` (3 lines: import + AllEndpoints + AllSchemas), `api/main.go` (2 lines: import + Module call), `cli/domains/domains.go` (2 lines), `ui/src/App.tsx` (2 lines), locales × 3 (5 keys each), `selectors.ts` (6 keys), `strings.generated.ts` (regen), `cli_commands_seed.json` (3 rows), `gen-endpoints/main_test.go` (seed list + count). Gates green; UI tests still 129/129.
- The 961 split: `api=570` includes proto-regen ribbon + sqlite layer + handler shape × full CRUD; `cli=171` is the per-handler `flag.NewFlagSet` + `protojson.Marshal` × 3 commands; `ui/src=220` includes `lib/tasks.ts` (full ApiError/decodeApiError/listTasks/createTask/getTask), TasksCard.tsx, locales × 3, selectors, strings regen.
- Junior_doable=Y because REPLACING-NOTES.md doubles as add-domain guide.

### Scenario 3 (add optional field — `tags []string`)
- Proto field add + regen.
- `types.go` adds `Tags []string` to Note + CreateInput.
- `service.go` carries Tags through Create.
- `sqlite.go` JSON-encodes tags column; schema.sql adds column.
- `handler.go` adds `Tags: req.Tags` in CreateInput construction.
- `adapter.go` adds Tags pass-through.
- `endpoints.go` documents the new field.
- `cli/handlers.go` adds `--tags` flag + comma-split helper.
- `lib/notes.ts` adds `tags?: string[]` to CreateNoteInput, passes through.
- The 93 split: `api=64` is proto pass-through + JSON encode/decode; `cli=27` is the new flag + splitTags helper; `ui/src=2` is just the optional-field add to lib/notes.ts.

### Scenario 4 (add error code — 409 conflict)
- New typed sentinel `ErrDuplicateTitle` in types.go.
- New repo method `ExistsByTitle(ctx, title) (bool, error)` (interface + sqlite impl + mock impl).
- Service.Create calls ExistsByTitle, returns ErrDuplicateTitle on hit.
- Handler maps ErrDuplicateTitle → 409 with envelope code `notes/duplicate_title`.
- endpoints.go adds 409 row.
- The 81 split: all in api (sentinel type + repo interface + sqlite impl + mock impl + service branch + handler branch + endpoints row). CLI's existing apiError surfaces the envelope code unchanged; UI's existing ApiError handles the 409 unchanged. The error-code add is genuinely localised to the API layer in this template.

### Scenario 5 (delete domain — notes per REPLACING-NOTES.md)
- Followed REPLACING-NOTES.md delete-checklist exactly: 4 `rm -rf` (api/internal/notes, api/handlers/notes, cli/domains/notes, ui/src/features/notes + lib/notes.ts), 8 sed sweeps across central files, manual edit of cli_commands_seed.json (back to status-only), manual edit of locale JSONs (Python script to drop `notes` block), manual selectors.ts edit to drop `notes` block, regen endpoints.json + strings.generated.ts, fix gen-endpoints/main_test.go seed list + count.
- 1489 lines removed across 27 files (mostly the deleted domain folders).
- Junior_doable=Y — guide is comprehensive.

### Scenario 6 (rename CLI flag — `--title` → `--name`)
- 5 edits: cli/handlers.go (declare + validate + use), cli/register.go (description), cli_commands_seed.json (description), endpoints.go (CLIMapping.Args), handlers_test.go (assertion + invocation).
- Greenfield rename per hard rule §2.2 — no `--title` alias kept.
- 28 lines net (most diff is single-token swaps; the 5 in cli is the actual `name := fs.String(...)` line + validation message).

## Cross-scenario observations

- **Ambient formatter touches** are the noisiest contributor to `central_edits`. Scenarios 3, 4, 6 each show 9–10 "ambient" entries from `gofumpt` re-aligning struct tags or `eslint --fix` removing an unused-disable directive in `ui/perf/capture.template.js`. Real central edits per scenario range 1 (scen 6) to 15 (scen 5).
- **`lines_added` is dominated by API plumbing**, not CLI or UI. Even scenarios with substantial CLI changes (1, 2, 6) split heavily toward API. The cli-core substrate change targeted at the CLI layer can move ~20–30% of total lines on scenario 1 (collapsing `flag.NewFlagSet`/`protojson.Marshal`/`apiError`/`decodeEnvelope` ribbons) but proportionally less on others.
- **Proto regen is invisible** to lines_added because gen output lives outside `cli/`, `api/`, `ui/src/`. That's correct (it's machine-generated), but it does mean the proto-side cost of new fields/messages is uncounted by this metric.
- **Test counts hidden by the `*_test.go` exclusion** are real work an author does. Junior_doable's "yes" verdicts depend on test patterns being copyable from the existing notes domain.

## Definition gaps

Logged during Scenario 1; propose as scenario revisions for iteration 2.

1. **Ambient formatter touches** — `gofumpt` and `eslint --fix` modified files I didn't author-edit (struct tag re-alignment, unused-eslint-disable removal). The recipe should either (a) filter these from `central_edits`, or (b) explicitly count them as "ambient cost of running formatters" with a sub-cell so they don't drown the real signal.
2. **Drift surface count format** — when one logical drift point spans two field pairs (e.g., a route's path + method), is that 1 or 2? Recording 2 for safety; recipe should pick a convention.
3. **Contract grade count format** — multi-precondition contracts (e.g., locale parity = N keys × M locales) could be 1 family or N×M individual contracts. Recording family count; recipe should pick a convention.
4. **Prettier missing from template** — `SCENARIOS.md` recipe locks "prettier from package.json" but the react-vite template ships no prettier; only ESLint with auto-fix. Recipe should be amended to use ESLint --fix as the canonical TS/TSX formatter, OR a prettier dependency should be added to the template (separate decision).
5. **`ui/src/consts/strings.generated.ts` is technically codegen output** — should it count toward `lines_added` (the author runs codegen and commits the result) or be excluded as a generated file? Recording: include (real PRs include it). Recipe should make this explicit.
