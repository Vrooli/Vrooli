# Problems — Image Tools

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

### 2026-06-16 — Documentation foundation complete; implementation not started — RESOLVED 2026-06-17

**Symptom (original):** Orientation reported 5/8 gates: `scaffold-health`,
`dependency-decisions`, and `example-domain-removed` were red because only the
planning foundation existed.

**Resolution (2026-06-17):** Phase 0 + Phase 1 landed. The `notes` worked
example is **removed** end-to-end (Gate 7); the `models`/`jobs` spine domains
replace it across api/cli/proto/ui; `.vrooli/service.json` now declares the
optional ComfyUI resource + optional ffmpeg hostTool (`dependency-decisions`).
Host hardware facts come from the root `vrooli host inventory --json` CLI behind
the `capabilities` seam (not system-monitor). API/CLI/proto green + e2e boot
test green; UI tests green (162/162). See [`PROGRESS.md`](PROGRESS.md).

**Remaining for `scaffold-health` / full `make test`:** run the complete
`vrooli scenario test image-tools` suite (standards/unit/coverage/measures/
smoke) — measures phase has nothing to grade until ops declare measures
(Phase 2+).

**Refs:** `PRD.md`, `requirements/`, [`DECISIONS.md`](DECISIONS.md), [`PROGRESS.md`](PROGRESS.md).

### 2026-06-17 — react-vite template lint + standards debt (the lone `vrooli scenario test` failure)

**Symptom:** Two surfaces, both confined to **pristine template files** (verified
present at HEAD, unchanged by feature work):
1. `phase-standards` fails — OWASP "Security Headers" HIGH ×4 on
   `api/internal/httpx/errors.go` (template error writer) + three template
   test files (`internal/middleware/logging_test.go`,
   `internal/module/module_test.go`, `internal/server/server_test.go`). This is
   the only failing phase in the full suite (17/18 green).
2. `pnpm lint` (`eslint .`) reports a handful of errors (plus react-refresh
   warnings): `theme.choice.{light,dark,system}` "no callsite" (dynamic-access
   false positive — they ARE used via `strings.theme.choice[c]`),
   `AppShell.tsx` aria-label string-literal, and `ThemeProvider.tsx`
   "unnecessary conditional (always falsy)" SSR guards.

**Root cause:** All are present on the **pristine react-vite scaffold at HEAD** —
not introduced by the jobs/models migration. (Standards) the OWASP
"Security Headers" rule flags any HTTP-response write lacking security headers,
including template error-rendering (`httpx/errors.go`) and HTTP-constructing
test files — a broad heuristic that over-fires on tests; this is the documented
fleet-wide standards campaign. (Lint-1) `theme.choice.*` ARE used, via dynamic
`t(strings.theme.choice[c])` in `TopBar`/`SettingsPage`; the
`strings/no-unused-keys` rule only follows static member access, so dynamic keys
read as unused (false positive). (Lint-2) The `AppShell` `aria-label="Main
content"` literal and the `ThemeProvider` `typeof window === "undefined"` SSR
guards live in files unchanged by this work.

**Workaround:** None needed for correctness — the app and API behave correctly,
and every other feature-introduced finding the suite raised (security G109, docs
manifest, go-mod-tidy, pnpm release-age, structure/proto `proto_payloads`) was
fixed. This matches the codebase's existing treatment of react-vite
template/scaffolding standards+lint debt as a tracked **upstream-template**
campaign (cf. the ecosystem-manager UI and the fleet standards campaign), not a
per-scenario fix — fixing it per scenario diverges from the template and breaks
on the next template sync.

**Real fix:** Upstream in `templates/scenarios/react-vite` + api-core: add a
security-headers middleware to the api-core server chain (so `httpx/errors.go`
responses inherit them) and exempt `*_test.go` from the OWASP Security-Headers
rule; teach `strings/no-unused-keys` to recognize dynamic `strings.x.y[expr]`
namespace access; route the `AppShell` main-content label through i18n; and drop
the dead SSR guards in `ThemeProvider` (the SPA is client-only). Then every
generated scenario inherits the fix.

**Owner:** unassigned (react-vite template + api-core maintainers).

**Refs:** `api/internal/httpx/errors.go`, `api/internal/{middleware,module,server}/*_test.go`,
`ui/src/consts/strings.generated.ts`, `ui/src/layout/AppShell.tsx`,
`ui/src/theme/ThemeProvider.tsx`, `ui/src/layout/TopBar.tsx`,
`ui/src/pages/SettingsPage.tsx`.

### 2026-06-16 — prd-control-tower generation gotchas (tooling, not scenario)

**Symptom:** `prd-control-tower prd generate --publish` failed with
`ORPHANED_CRITICAL_TARGETS`; `prd-control-tower requirements generate`
exceeded the client HTTP timeout for a large (28-target) registry.

**Root cause:** `--publish` enforces that every P0 target already has a
linked requirement (stricter than the START-HERE flow implies), and the
requirements generator builds all modules in one synchronous LLM call
that can outrun the CLI's HTTP client timeout.

**Workaround:** Generate the PRD draft without `--publish`, read the
markdown from `.generation.generated_text` in the `--json` output, and
write `PRD.md` directly; author the requirements modules directly from the
PRD's OT ids (`prd_ref: "OT-Pn-NNN"`). Then validate with
`prd-control-tower prd validate` and `... requirements validate` (both
report healthy).

**Real fix:** N/A for this scenario — upstream prd-control-tower
ergonomics. Captured so the next scenario author doesn't re-discover it.

**Owner:** unassigned.

**Refs:** prd-control-tower CLI; `requirements/index.json`.

### 2026-06-17 — metadata `strip-gps` removes ALL metadata in v1 (privacy-safe superset)

**Symptom:** `image-tools ops metadata --strip-gps` and `--strip-all` behave
identically: both produce an output with no EXIF/IPTC/XMP at all, not just GPS
removed.

**Root cause:** Selective GPS-IFD removal that preserves other EXIF requires
rebuilding the EXIF block (dsoprea/go-exif IFD builder). v1 implements strip by
re-encoding through Go's encoders, which write no metadata — so GPS is
definitely gone, but so is everything else.

**Workaround:** None needed for the privacy goal (location IS removed). Callers
who must preserve non-GPS tags should not strip in v1.

**Real fix:** Surgically rebuild EXIF minus the GPS IFD via the go-exif builder
and re-embed it; flip this entry to resolved. Tracked for a later enhancement.

**Owner:** unassigned.

**Refs:** `api/internal/ops/metadata.go` (`stripMetadata`); DECISIONS.md.

### 2026-06-17 — Runtime op-latency measures substrate not yet mounted

**Symptom:** The `/measures` substrate (op latency p50/p95, throughput,
queue-wait, fallback-tier usage) named in the implementation plan is not yet
served; deterministic ops record durable jobs but emit no runtime metrics.

**Root cause:** Deliberately deferred (Phase 1 decision): mounting an empty
measures substrate is dead scaffolding. Deterministic ops are instant, so
latency percentiles are far more meaningful once heavy AI ops land (Phase 3).
The manifest `measure` (semantic-query) blocks are a separate concern; the ops
domain is stateless and needs none, so the test-genie measures phase stays
green.

**Workaround:** Per-op timing is observable via recorded job timestamps.

**Real fix:** Mount the measures-go serve registry when the first AI op declares
runtime measures (Phase 3/4).

**Owner:** unassigned.

**Refs:** `api/main.go` (measures-substrate comment); plan §6/§8.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
