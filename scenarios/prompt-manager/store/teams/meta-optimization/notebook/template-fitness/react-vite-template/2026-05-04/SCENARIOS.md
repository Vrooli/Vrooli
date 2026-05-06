# Measurement Scenarios — frozen at iteration 1 (2026-05-04)

Six workflows. Each is a deterministic recipe an agent can execute against any version of `templates/scenarios/react-vite/` and produce comparable numbers.

**Frozen.** A future iteration may revise a scenario only by appending a "Scenario revision" entry below with date, scenario number, change, reason. Without that, every iteration runs the same recipe.

## Common preliminaries (apply to every scenario)

1. **Tree under measurement** is `templates/scenarios/react-vite/` at the iteration's HEAD. Record the SHA in the BASELINE/RESULTS row. The Vrooli repo's working tree may be dirty in unrelated areas (other scenarios, in-progress work) — that's fine; the measurement is scoped to the throwaway scenario tree only.
2. **No git mutations**: the recipe is **commit-free**. Per project policy, the agent never commits, reverts, resets, or stashes. Snapshot-copy is the diff baseline. (The workflow may set a working branch name for context, but no `git checkout -b` is required.)
3. **Generate a throwaway scenario from the template** to do the work in. The template is copied, not edited in place — measurements include the work an actual scenario-author would do, including any per-scenario rename steps.
   ```bash
   vrooli scenario generate react-vite \
     --id harness-<iteration>-scen<N>-<workflow-slug> \
     --display-name "Harness <iteration> scen<N>" \
     --description "Reference-pattern-fitness measurement scenario"
   ```
4. **Snapshot the generated tree** before any workflow implementation begins. This is the diff baseline; `git diff` is not used (the repo may be dirty in unrelated areas, and we don't commit).
   ```bash
   SCEN_DIR="/home/matthalloran8/Vrooli/scenarios/harness-<iteration>-scen<N>-<workflow-slug>"
   SNAPSHOT="/tmp/harness-<iteration>-scen<N>-snapshot"
   rm -rf "$SNAPSHOT"
   cp -a "$SCEN_DIR" "$SNAPSHOT"
   # Also snapshot the proto outputs we may regenerate:
   cp -a /home/matthalloran8/Vrooli/packages/proto/schemas/harness-<iteration>-scen<N>-<workflow-slug> \
         "$SNAPSHOT/_proto-schemas"
   cp -a /home/matthalloran8/Vrooli/packages/proto/gen/go/harness-<iteration>-scen<N>-<workflow-slug> \
         "$SNAPSHOT/_proto-gen-go" 2>/dev/null || true
   cp -a /home/matthalloran8/Vrooli/packages/proto/gen/typescript/harness-<iteration>-scen<N>-<workflow-slug> \
         "$SNAPSHOT/_proto-gen-ts" 2>/dev/null || true
   ```
5. **Formatters locked**: `gofumpt` from the cli-core toolchain (`cd packages/cli-core && go env GOTOOLCHAIN`); `prettier` from `templates/scenarios/react-vite/ui/package.json` `devDependencies`. Run formatters before measuring lines.
6. **Cleanup after measurement**. Proto codegen output uses three different naming conventions for the same scenario id — clean all of them explicitly. **The hyphen→underscore substitution for python is the gotcha that left ~24 directories of residue after iteration 1.**

   ```bash
   SCEN_ID="harness-<iteration>-scen<N>-<workflow-slug>"        # hyphens
   SCEN_ID_UNDER="${SCEN_ID//-/_}"                              # underscores (python)

   vrooli scenario stop "$SCEN_ID" 2>/dev/null || true
   rm -rf "scenarios/$SCEN_ID" \
          "packages/proto/schemas/$SCEN_ID" \
          "packages/proto/gen/go/$SCEN_ID" \
          "packages/proto/gen/typescript/$SCEN_ID" \
          "packages/proto/gen/typescript/js/$SCEN_ID" \
          "packages/proto/gen/python/$SCEN_ID_UNDER" \
          "$SNAPSHOT"
   ( cd packages/proto && make generate )

   # Verify zero residue (both lines should print nothing):
   ls -d packages/proto/gen/go/*"$SCEN_ID"* packages/proto/gen/typescript/js/*"$SCEN_ID"* 2>/dev/null
   ls -d packages/proto/gen/python/*"$SCEN_ID_UNDER"* 2>/dev/null
   ```

   Why three name forms:
   - **Go gen** (`gen/go/<scenario-id>`) — module path is the scenario id verbatim, hyphens.
   - **TypeScript gen** (`gen/typescript/<scenario-id>` and `gen/typescript/js/<scenario-id>`) — buf emits both a TS-source tree and a compiled-JS tree. In iteration 1 only `js/` was populated, but clean both to stay future-proof.
   - **Python gen** (`gen/python/<scenario_id_underscored>`) — Python module names disallow hyphens, so the protoc-gen-python plugin rewrites `harness-iter1-scen1-add-endpoint` to `harness_iter1_scen1_add_endpoint`. **A `rm -rf` against the hyphenated form is a silent no-op against the python path.** This is the bug iteration 1 hit.

## Central-registry-edit definition

A "central-registry edit" is any modification to a file outside the scenario's domain folder for the workflow's primary domain. The exact list of "central" files counted:

- `api/handlers/<root>.go` (or wherever `api/main.go` mounts modules)
- `api/internal/endpoints/endpoints.go` (descriptor table)
- `cli/domains.go` (or `cli/main.go`'s domain-registration block)
- `docs/manifest.json`
- `docs/internal/REPLACING-NOTES.md` (for delete walkthroughs)
- `cli_commands_seed.json`
- `ui/src/App.tsx` (router additions for new domains)
- `packages/proto/schemas/<scenario>/<service>.proto`
- `module_test.go` (route-vs-endpoint parity, if added in iteration 2+)
- The scenario's `.vrooli/endpoints.json`

If a scenario's recipe touches a file not on this list and the agent thinks it should count, log a "definition gap" note in the row and propose adding the file in the next scenario revision.

## Measurement command suite

All commands are **commit-free** and operate on the snapshot diff. `$SNAPSHOT` is the snapshot path created in step 4 above; `$SCEN_DIR` is the working scenario tree.

Per-replica cost (lines added, non-test, post-formatter):

```bash
total_added=0
for dir in cli api ui/src; do
  added=$(diff -rN \
    --exclude='*_test.go' \
    --exclude='*.test.ts' \
    --exclude='*.test.tsx' \
    "$SNAPSHOT/$dir" "$SCEN_DIR/$dir" \
    | grep -E '^>' | wc -l)
  total_added=$((total_added + added))
done
echo "$total_added"
```

Lines removed (Scenario 5 only — delete workflow):

```bash
total_removed=0
for dir in cli api ui/src; do
  removed=$(diff -rN \
    --exclude='*_test.go' \
    --exclude='*.test.ts' \
    --exclude='*.test.tsx' \
    "$SNAPSHOT/$dir" "$SCEN_DIR/$dir" \
    | grep -E '^<' | wc -l)
  total_removed=$((total_removed + removed))
done
echo "$total_removed"
```

Drift surface count:
```
# Manual count by walking the workflow's changed files and applying the
# four-sub-lens checklist from REFERENCE_PATTERN_FITNESS.md §"Drift Surface
# Map". Record each as (location-A, location-B, enforcement: type/CI/hope).
# Only "hope" entries count toward the metric.
```

Contract location:
```
# Manual classification per non-trivial precondition introduced. Record as
# (precondition, location: type-signature/CI-check/code-comment/nowhere).
```

Central-registry edits (files outside the workflow's primary domain folder that changed or were newly added):

```bash
DOM="<workflow-domain>"   # e.g. "notes" for scenarios 1, 3, 4, 5, 6; "tasks" for scenario 2
diff -rqN "$SNAPSHOT" "$SCEN_DIR" \
  | grep -E '^(Only in|Files .* differ)' \
  | sed -E 's|^Only in ([^:]+): (.+)$|\1/\2|; s|^Files ([^ ]+) and .*|\1|' \
  | sed -E "s|^$SNAPSHOT/||; s|^$SCEN_DIR/||" \
  | grep -v -E "(^|/)cli/domains/$DOM/" \
  | grep -v -E "(^|/)api/internal/$DOM/" \
  | grep -v -E "(^|/)api/handlers/.*$DOM" \
  | grep -v -E "(^|/)ui/src/features/$DOM/" \
  | grep -v -E "(^|/)ui/src/lib/$DOM\." \
  | grep -v -E '^_proto-' \
  | sort -u | tee /tmp/central-edits.txt | wc -l
# Cross-reference /tmp/central-edits.txt against the central-files list above;
# if a file shows there that's not on that list, log a definition gap in
# BASELINE.md / RESULTS.md.
```

Note that the proto generated artifacts (`packages/proto/{schemas,gen}/<scenario>/...`) live outside `$SCEN_DIR` but still count as central edits when modified — those are tracked separately:

```bash
# Proto artifact edits — count modified files under the scenario's proto tree.
PROTO_SCHEMAS="/home/matthalloran8/Vrooli/packages/proto/schemas/<scenario-id>"
diff -rqN "$SNAPSHOT/_proto-schemas" "$PROTO_SCHEMAS" \
  | grep -cE '^(Only in|Files .* differ)'
```

Meta-metric:
```
Yes/No: could a junior, given only docs/internal/REPLACING-NOTES.md and the
template tree (no Slack, no senior shoulder-tap), do this workflow?
+ one-sentence reason (what blocks them, or what makes it learnable from the
docs alone).
```

## Scenario 1 — Add a new endpoint to an existing domain

**Goal**: Implement `notes update` (HTTP PATCH `/api/v1/notes/{id}`) on the throwaway scenario. Title/body editable; updated_at refreshed.

**Recipe**:
1. Generate scenario per common preliminaries with workflow-slug `add-endpoint`.
2. Implement the endpoint:
   - Add `UpdateNoteRequest` and `UpdateNoteResponse` to `packages/proto/schemas/<scenario>/notes/v1/notes.proto`.
   - `( cd packages/proto && make generate )`
   - Service method: `Update(ctx, id, input) (*Note, error)` in `api/internal/notes/service.go`.
   - Repository method: `Update(ctx, id, input) (*Note, error)` in `api/internal/notes/repository.go`.
   - Handler: `Update` in `api/internal/notes/handler.go`; route registered.
   - Endpoint descriptor row in `api/internal/endpoints/endpoints.go`.
   - CLI handler `update` in `cli/domains/notes/handlers.go`; registered in `register.go`; `cli_commands_seed.json` row.
   - UI client `updateNote(id, input)` in `ui/src/lib/notes.ts`; integration into `ui/src/features/notes/` (the simplest existing edit affordance — add a "Save" button on an existing note row, or a new edit form).
3. Run scenario CI gates (`go test`, `pnpm test`, etc.) until green.
4. Run the four measurement commands; record numbers in this iteration's BASELINE.md (Phase B) or RESULTS.md (Phase E).

**Success criterion**: PATCH `/api/v1/notes/{id}` round-trips through CLI and UI. Tests green.

## Scenario 2 — Add a new domain end-to-end

**Goal**: Add a `tasks` CRUD domain (list, create, get) to the throwaway scenario.

**Recipe**:
1. Generate scenario per common preliminaries with workflow-slug `add-domain`.
2. Implement the domain following the structure of `notes`:
   - `packages/proto/schemas/<scenario>/tasks/v1/tasks.proto` with `Task`, `ListTasksResponse`, `CreateTaskRequest/Response`, `GetTaskRequest/Response`.
   - `( cd packages/proto && make generate )`
   - `api/internal/tasks/{service.go, repository.go, handler.go, *_test.go}`.
   - Migration for `tasks` table (matches `notes` migration shape).
   - Module wiring in the central modules registry.
   - Endpoint rows for all three methods.
   - `cli/domains/tasks/{handlers.go, register.go, handlers_test.go}` registered in `cli/domains.go`.
   - `cli_commands_seed.json` rows.
   - `ui/src/lib/tasks.ts` (mirroring `notes.ts`); `ui/src/features/tasks/`; route in `App.tsx`.
3. Run scenario CI gates until green.
4. Measure.

**Success criterion**: `<scenario> tasks list/create/get` work end-to-end. UI route renders. Tests green.

## Scenario 3 — Add an optional field to a request shape

**Goal**: Add `tags []string` to `CreateNoteRequest`; persist; return on `Note`.

**Recipe**:
1. Generate scenario per common preliminaries with workflow-slug `add-field`.
2. Add `tags` to `Note` and `CreateNoteRequest` in the proto. Regenerate.
3. Add column + migration. Service + repository carry the field through. Handler accepts it.
4. CLI `create` accepts `--tags` (repeatable or comma-separated; pick one and document).
5. UI form gains a tags input; `lib/notes.ts::createNote` passes it through.
6. Tests updated to cover the new field on a happy path; backwards-compat path (no tags supplied) still passes.
7. Measure.

**Success criterion**: `notes create --title hi --tags a,b` persists tags; `notes get` returns them; UI displays them.

## Scenario 4 — Add a new error-code path

**Goal**: Server returns `409 conflict` with envelope code `notes/duplicate_title` when creating a note whose title duplicates an existing one (case-insensitive). UI surfaces a user-readable error.

**Recipe**:
1. Generate scenario per common preliminaries with workflow-slug `add-error`.
2. Service method detects duplicate; returns a typed sentinel error.
3. Handler maps the sentinel to envelope `{status: 409, code: "notes/duplicate_title", message: ...}`.
4. CLI error message reads the envelope code and prints "A note with that title already exists."
5. UI form catches the `ApiError`, branches on `.code`, displays the friendly string inline.
6. Tests cover: handler 409 path, CLI error path, UI error path.
7. Measure.

**Success criterion**: Two consecutive `notes create --title hi` calls; second exits non-zero with the friendly message; UI form shows the inline error on duplicate submit.

## Scenario 5 — Delete a domain

**Goal**: Remove the `notes` domain from the throwaway scenario per `docs/internal/REPLACING-NOTES.md`.

**Recipe**:
1. Generate scenario per common preliminaries with workflow-slug `delete-domain`.
2. Follow `docs/internal/REPLACING-NOTES.md` start to finish exactly. Record each `sed`/`rm`/edit step.
3. Run scenario CI gates until green.
4. Measure. Per-replica cost is **negative** (lines removed); for this scenario, record the absolute count of lines removed across `cli/`, `api/`, `ui/src/`. Drift-surface and contract-location are usually 0 (deletions don't add new contracts). Central-registry edits remain meaningful.

**Success criterion**: Scenario builds and tests pass with no notes references.

## Scenario 6 — Rename a CLI flag

**Goal**: Rename `--title` to `--name` on `notes create`.

**Recipe**:
1. Generate scenario per common preliminaries with workflow-slug `rename-flag`.
2. Update the CLI flag declaration. Update help text. Update CLI tests (the captured-request still goes to `Title:` in the proto if proto stays; if proto also changes, this becomes a breaking change requiring proto + handler update — either way is a valid test of the workflow as long as scope is recorded).
3. Update `cli_commands_seed.json` if it indexes flag names.
4. Measure.

**Success criterion**: `notes create --name hi --body bye` works; `--title` is gone (no compatibility shim — greenfield rule).

## Scenario revisions

(Append entries here when iteration N+ revises a scenario. Each entry: date, scenario number, what changed, why.)

(none yet)
