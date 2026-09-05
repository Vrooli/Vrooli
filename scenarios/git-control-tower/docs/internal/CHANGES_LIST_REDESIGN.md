# Changes List Redesign — Investigation Record

Status: **design record, pre-implementation**
Date: 2026-08-30
Branch at time of reading: `agi`
Mockups: [`assets/changes-list-redesign-mockups.html`](assets/changes-list-redesign-mockups.html)
(absolute path: `/home/matthalloran8/Vrooli/scenarios/git-control-tower/docs/internal/assets/changes-list-redesign-mockups.html`)
Published mockup artifact: <https://claude.ai/code/artifact/85413e70-2efb-4427-b359-19bbf9d6e5d0>

This document is the durable record of the investigation that produced the
"Changes list hierarchy, control adoption, and honest agent attribution" plan.
It records **what was measured**, not what was proposed. The plan owns the work;
this document owns the evidence.

---

## 1. Why this exists

Git Control Tower is a monetized scenario: it must work on any repository, and
it must look like a product. The mobile Changes panel is the surface an operator
spends the most time in. An operator review on 2026-08-30 raised six distinct
complaints against it. Each one was traced to a specific line of source. Two of
them turned out to have working fixes already present elsewhere in this
repository that Git Control Tower never adopted.

Repo-contract-derived grouping is the mode under review. The scenario detects
`.vrooli/repo-contract.json` in the target repository and groups changed files by
resolved contract target. That mode already works; its **presentation** is the
subject here. Repositories with no contract keep the manual-rule and flat modes
unchanged.

---

## 2. The six findings

### F1 — Contract groups sort by label only; `Kind` is computed then discarded

`ResolveChangeGroups` writes the resolved target kind onto every contract group
and then never uses it for ordering.

```go
// scenarios/git-control-tower/api/grouping_groups.go:64-72
addResolvedGroup(groups, resolvedChangeGroup{
    group: ChangeGroup{
        Key:    "contract:" + string(target.Kind) + ":" + target.ID,
        Kind:   string(target.Kind),   // <- resolved
        ID:     target.ID,
        Label:  target.ID,
        Root:   target.Root,
        Source: groupSourceContract,
    },
    order:      0,                     // <- constant for every contract group
    sourceRank: 1,
}, path)
```

```go
// scenarios/git-control-tower/api/grouping_groups.go:87-95
sort.SliceStable(resolved, func(i, j int) bool {
    if resolved[i].sourceRank != resolved[j].sourceRank {
        return resolved[i].sourceRank < resolved[j].sourceRank
    }
    if resolved[i].order != resolved[j].order {   // always equal for contract groups
        return resolved[i].order < resolved[j].order
    }
    return resolved[i].group.Label < resolved[j].group.Label
})
```

Because `order` is a constant for the entire contract tier, the tie-break is
always the label. The whole contract tier collapses into one alphabetical run.

**Why it looks like "scenarios first."** The alphabetically earliest contract
targets in this repository happen to be scenarios (`agent-inbox`,
`agent-manager`, `agent-metareasoning-manager`, `ai-chatbot-manager`,
`ai-gateway`, `algorithm-library`, `api-health`). Resources, packages and
control-plane groups interleave further down the list where they are not visible
in a first screenful.

The repo contract already defines the axis this needs:

```go
// packages/repo-contract-go/contract.go:87-95
TargetKindScenario     TargetKind = "scenario"
TargetKindResource     TargetKind = "resource"
TargetKindTool         TargetKind = "tool"
TargetKindSafeguard    TargetKind = "safeguard"
TargetKindTeam         TargetKind = "team"
TargetKindPackage      TargetKind = "package"
TargetKindControlPlane TargetKind = "control-plane"
TargetKindDocs         TargetKind = "docs"
TargetKindProject      TargetKind = "project"
```

The UI already receives `kind` on every group and uses it for exactly one thing —
deciding whether the "open scenario review" button renders
(`ui/src/components/FileList.tsx:768`).

### F2 — Each group is a bordered card with a 16 px gutter and a redundant path line

```tsx
// scenarios/git-control-tower/ui/src/components/FileList.tsx:739
className="mb-4 rounded-lg border border-slate-800/80 bg-slate-950/40"
```

```tsx
// scenarios/git-control-tower/ui/src/components/FileList.tsx:757-765
<div className={`font-semibold uppercase tracking-wider text-slate-300 ...`}>
  {group.label}
</div>
{group.displayPrefix && (
  <div className={`text-slate-500 ...`}>{group.displayPrefix}</div>
)}
```

Measured from the classes: 14 px label line + 12 px path line + `py-2`
(16 px) + 2 px border + `mb-4` (16 px) ≈ **70 px of vertical pitch per collapsed
group**, carrying one integer.

The path line is the redundant one. For a contract group,
`scenarios/agent-inbox/` is fully reconstructible from the kind plus the group
name. Introducing the kind band is therefore what makes the path line
removable — the hierarchy pays for its own row and then some.

### F3 — "Reveal in file tree" is one call away and unwired

`ProjectTreeView` already expands every ancestor directory of a target path and
scrolls the row into view:

```tsx
// scenarios/git-control-tower/ui/src/components/ProjectTreeView.tsx:485-546
useEffect(() => {
  if (!scrollToFile) return;
  const segments = scrollToFile.split("/");
  // ... expands each ancestor directory ...
  const element = document.querySelector(`[data-file-path="${CSS.escape(scrollToFile)}"]`);
  // ... scrolls it into view, then calls onScrollComplete()
}, [scrollToFile]);
```

`App.tsx:186` already owns `const [scrollToFile, setScrollToFile] = useState<string | undefined>()`
and `App.tsx:1517-1525` already owns the completion handler. `fileViewMode` is
plain local state in the same component (`App.tsx:142`), persisted to
`localStorage` at `App.tsx:1771`.

Nothing routes a user from a file row to the tree. The only missing piece is a
menu action that sets both values in one tick.

### F4 — Toolbar controls are hand-rolled; React Component Library ships the control

```tsx
// scenarios/git-control-tower/ui/src/components/ViewModeCycleButton.tsx:56
className={`${compact ? "h-11 w-11" : "h-9 w-9"} inline-flex items-center
justify-center rounded-full border transition-colors ${color}`}
```

```tsx
// scenarios/git-control-tower/ui/src/components/IconButton.tsx (whole file, 21 lines)
className={cn("p-2 rounded-md hover:bg-slate-800 transition-colors disabled:opacity-50", className)}
```

`packages/react-component-library/dist/components/IconButton/versions/3.1.2/IconButton.d.ts`
documents a control that covers every one of these needs: an `xs` rung at 32 px,
four surfaces of which `ghost` needs no border, `selected` that sets
`aria-pressed`, `morph` that animates a glyph change with no call-site work, and
`swapIdentity` for a control rendered from more than one parent. Its release
notes name web-console's view toggle — the same widget, one scenario over — as
the adoption that found the remount defect `swapIdentity` fixes.

Git Control Tower's `ui/package.json` declares **no** `@vrooli/react-component-library`
dependency. web-console and experience-manager both declare it as
`"file:../../../packages/react-component-library"`.

### F5 — Four surfaces claim the top of the screen and no two agree

| Surface | Value | Source |
| --- | --- | --- |
| `meta[name=theme-color]` | `#0f172a` | `ui/index.html:6` — hard-coded literal |
| App shell | `bg-slate-950` = `#020617` | `ui/src/App.tsx:2812` |
| Header **and notch strip** | `bg-slate-900/95` composited over slate-950 | `ui/src/components/MobileHeader.tsx:89` |
| Blur backdrop | varies with scrolled content | `backdrop-blur-sm` on the same element |

`MobileHeader` carries `pt-safe`, so the header element also paints the iOS
top safe area — at 95 % opacity, over a different colour than the meta tag
declares.

React Component Library already owns this problem:

```
packages/react-component-library/dist/services/ChromeTheme/versions/1.0.0/ChromeTheme.d.ts
```

> "The colour of the status bar" is two different mechanisms, and a surface that
> sets only one is correct on one platform and stale on the other […] This
> service owns both, resolves them from one value, and writes them together — so
> the two can never drift.

It ships `chromeTheme.setBase()`, `chromeTheme.contribute(key, chrome, priority)`,
`useChromeContribution()`, and the `StatusBarFill` strip component. web-console
adopted it — see `scenarios/web-console/ui/src/lib/chromeTheme.ts` and
`scenarios/web-console/ui/src/components/TopSafeArea.tsx`. Git Control Tower did
not.

### F6 — The banner calls sandbox output "approved" and hides per-run provenance the API already serves

```tsx
// scenarios/git-control-tower/ui/src/components/FileList.tsx:664-694
<ShieldCheck className="h-3.5 w-3.5 text-emerald-300" />
<span>Approved changes ready</span>
<span className="text-emerald-300">{approvedChanges?.committableFiles ?? 0} files</span>
<Button ... data-testid="legacy-bulk-action">Legacy bulk action</Button>
```

Auto-apply is a configuration choice made once, in Agent Manager, possibly days
earlier. It is not review. The banner presents it as review and attaches a
bulk-stage action to it.

The data needed for an honest presentation is already served **per file**:

```go
// scenarios/git-control-tower/api/approved_changes_model.go:12-20
type ApprovedChangeFile struct {
    RelativePath      string `json:"relativePath"`
    Status            string `json:"status"`
    SandboxID         string `json:"sandboxId,omitempty"`
    SandboxOwner      string `json:"sandboxOwner,omitempty"`
    ChangeType        string `json:"changeType,omitempty"`
    AgentManagerRunID string `json:"agentManagerRunId,omitempty"`
}
```

and grouped by run at a second endpoint (`api/approved_changes_handler.go:66`,
`handleProvenance`), rendered by `ui/src/components/AIProvenanceTab.tsx`. That tab
is mounted only inside the Review panel (`ui/src/components/ScenarioReviewPanel.tsx:218`).
There is no path to it from a file in the Changes list.

Today the UI collapses all of that to one aggregate count plus a per-row
`approved` pill (`ui/src/components/FileRow.tsx:154-162`) whose label repeats the
same untrue claim.

---

## 3. Why nothing caught F5

This is the part worth keeping. The claim exists, is correctly worded, and is
guarded by an assertion that cannot fail for the reason it was written.

`experience/fixtures/mobile-regression-oracle.json`, case **GCT-MOBILE-002**:

```json
{
  "id": "GCT-MOBILE-002",
  "name": "chrome-declaration-and-render-alignment",
  "owner": "brand-manager + ui-health",
  "claim": "Declared theme/status chrome metadata matches the rendered top safe-area surface.",
  "evidence": ["source_snapshot", "screenshot_png", "layout_json"],
  "expected_before_repair": "fail",
  "remediation": "Declare theme-color, viewport-fit, and mobile status metadata that matches the shell's top surface."
}
```

The fixture's status is `repaired_and_guarded`, and
`docs/internal/MOBILE_REGRESSION_ORACLE.md` states that "changing a case requires
updating its evidence and owner rather than suppressing a finding."

It degraded at three hand-offs:

| Layer | What it should assert | What it actually asserts |
| --- | --- | --- |
| **The guard** — `ui/src/MobileShellContract.test.ts:10` | The declared colour equals the rendered top surface | `expect(indexHTML).toContain('name="theme-color"')` — presence of the tag. Green for any value, including a wrong one. It never reads the header's colour. |
| **The spec** — `experience/pages/workspace.json` | A machine-tier claim about chrome colour agreement | Cases 002 and 006 were merged into one claim, `gct-mobile-002-006-chrome-pinned`, of `"type": "chrome-pinned"`, whose statement is purely geometric: *"Mobile header and bottom navigation stay inside the viewport as structural chrome without overlapping the active panel."* The colour half has no representative in the spec. |
| **The validator** — experience-manager | Flag an oracle case with no corresponding spec claim | `api/internal/checks/registry.go` holds four checks: `BASReferenceCheck`, `StateCoverageCheck`, `attestation.Check`, `reconcile.Check`. `spec.ParseScenario` (`api/internal/spec/spec.go:376-421`) never reads `experience/fixtures/`. The oracle is an unreferenced document. |

Git Control Tower is presently the **only** scenario in the repository with an
`experience/fixtures/` directory (1 file), out of 42 scenarios carrying an
`experience/` contract. The check that does not exist has a blast radius of one
scenario today and generalizes to every scenario that later ships an oracle.

### What the platform can already do

The gap is a missing check, not a missing capability:

- `reconcile.AXNode` carries `ComputedStyle map[string]string`
  (`scenarios/experience-manager/api/internal/reconcile/snapshot.go:37-39`), and
  the BAS capturer requests it (`bas_capturer.go:65-72`, `InlineComputedStyle: true`).
- `computedStyleValue(node, "background-color")` and `parseColor` already exist
  and are used by the `state-contrast` evaluator
  (`claim_evaluators.go:241-252`, `:728-745`, `:816-821`).
- `claimEvaluators` (`claim_evaluators.go:97+`) and
  `claimtypes.implemented` (`api/internal/claimtypes/claimtypes.go:7-42`) are the
  two lists a new claim type registers in.
- Git Control Tower's BAS case already runs arbitrary `evaluate` nodes at the
  390×844 mobile profile (`bas/cases/mobile-regression/workspace-mobile-stateful.json`),
  which is the one channel that can read `meta[name=theme-color]` — the AX tree
  cannot see head elements.

---

## 4. Design decisions

### D1 — The second grouping level is a band, not a container

A nested container per kind would add a wrapper, an indent, and a border around
every group. A **band** is a single sticky 26 px rule with a label and a count,
emitted once per kind. It groups without indenting.

Rejected: nested collapsible kind sections. They read well on desktop and cost a
full extra tap depth on mobile, where the operator's most common action is
"scan to my scenario."

### D2 — Kind order is fixed, not alphabetical

`scenario → resource → package → control-plane → tool → safeguard → team →
docs → project → (manual rules) → other`. Within a band, alphabetical by label,
as today. Manual rules keep their existing precedence above the contract tier and
render in their own band.

Rationale: the operator's mental model of this repository is "my scenarios, then
the shared things underneath." Alphabetical ordering of kinds would put
`control-plane` first and `scenario` seventh, which is the opposite of use
frequency.

### D3 — The path line is dropped for contract groups only

Band + name reconstructs a contract group's root exactly. Manual-rule groups and
the builtin `Other` group keep the path, because for those the root is not
derivable from the label.

### D4 — Attribution is a property of the file, not a banner

Per-file `agentManagerRunId` drives a 3 px coloured spine in the row gutter plus a
compact run chip. The hue is derived deterministically from the run id so two
runs in the same list are separable at a glance and stable across refreshes. A
collapsed group row shows one dot per distinct run it contains.

### D5 — The word "approved" is retired from the agent-attribution surface

Replaced by "from a sandbox run" and, in the run sheet, an explicit
`Auto-applied by the sandbox. Nothing here has been reviewed.` line. The banner's
only unique datum — the count — becomes an `Agent N` filter chip. The chip
filters; it has no staging action attached.

### D6 — Every staging path routes through the existing selection flow

The run sheet's primary action is **Select all N**, which feeds the selection
toolbar that already exists. The sheet stages nothing itself. Agent-written files
are then staged with exactly the same gesture and the same deliberateness as
every other file.

Rejected: keeping the legacy bulk action behind a confirmation. A confirmation dialog on
a bulk action still presents the aggregate as reviewable, and it adds a modal to
the most common screen.

### D7 — Chrome colour is proven at three rungs, not one

| Rung | Mechanism | What it proves |
| --- | --- | --- |
| A | jsdom unit test in `ui/src/` | The `theme-color` meta content and `--rcl-status-fill` resolve to the same value after the shell mounts. Value equality, not tag presence. |
| B | New `chrome-color-agreement` claim evaluator in experience-manager | In a real browser at the mobile profile, the computed `background-color` of the status-bar fill equals that of the header surface. Generalizes to every scenario. |
| C | New `oracle.claim_coverage` check in experience-manager | Every oracle fixture case id is named by at least one spec claim, and a case declaring `screenshot_png`/`layout_json` evidence is named by at least one `tier: machine` claim. Catches the class of degradation, not this instance. |

Rung C is authored and run **before** the F5 repair, so its first result is a
real failure on GCT-MOBILE-002. A check whose first observed state is green
proves nothing.

---

## 5. Density arithmetic

| Measure | Now | Proposed |
| --- | --- | --- |
| Collapsed group pitch | ~70 px (52 px card + 16 px gutter + 2 px border) | 44 px row, 0 gutter, 1 px divider |
| Kind band cost | n/a | 26 px, once per kind present |
| Groups visible in a 640 px list | ~9 | ~14 (with 3 bands) |
| Lines of text per collapsed group | 2 (label + path) | 1 (label) |
| Grouping levels legible | 1 | 2 |

---

## 6. Closing implementation record

The redesign shipped with server-side contract-kind ordering, dense kind bands,
React Component Library controls, one ChromeTheme authority, reveal-in-tree
navigation, and per-run attribution with select-only run sheets. The oracle
fixture is now schema-linked and enforced by experience-manager; its chrome
claim is separate from the pinned-chrome claim. During execution, the safe-area
strip required an accessible wrapper because the library's paint element is
intentionally `aria-hidden`; the wrapper preserves the shared color while making
the browser contract observable. Full Test Genie validation still reports
pre-existing CLI, branding, proto, and test-helper debt; targeted checks for the
changed surfaces pass. The live approved-changes response populated
`agentManagerRunId`, so that is the preferred run key; the live provenance
response supplied application timestamps but had null `runId` values, so the
per-file key is retained when enriching those records. A real mobile chrome
capture is recorded at
`/home/matthalloran8/.vrooli/plan-artifacts/gct-changes-list/after-chrome.png`.
Interactive captures of the grouped list, an expanded group, the run sheet, and
file actions are recorded at:

- `/home/matthalloran8/.vrooli/plan-artifacts/gct-changes-list/after-grouped-list.png`
- `/home/matthalloran8/.vrooli/plan-artifacts/gct-changes-list/after-expanded-group.png`
- `/home/matthalloran8/.vrooli/plan-artifacts/gct-changes-list/after-run-sheet.png`
- `/home/matthalloran8/.vrooli/plan-artifacts/gct-changes-list/after-file-actions.png`

The Plan Manager execution ledger records the before-state findings and these
capture paths. The producer-owned behavioral baseline was started for both
members; its wait remains in progress while the targeted validation evidence
above is already green.

The final UI gate subsequently passed in full: lint, type-check, and all 43 UI
test files (398 tests). The full experience-manager API suite also passed. The
Git Control Tower API suite remains unable to compile because the current
`api-core/servertest` dependency does not expose helpers used by existing tests;
Scenario QA bug `knw-1788117350937726395` records that unrelated debt.

## 7. Files this record cites

```
scenarios/git-control-tower/api/grouping_groups.go
scenarios/git-control-tower/api/grouping_model.go
scenarios/git-control-tower/api/approved_changes_model.go
scenarios/git-control-tower/api/approved_changes_handler.go
scenarios/git-control-tower/ui/index.html
scenarios/git-control-tower/ui/package.json
scenarios/git-control-tower/ui/src/App.tsx
scenarios/git-control-tower/ui/src/MobileShellContract.test.ts
scenarios/git-control-tower/ui/src/components/FileList.tsx
scenarios/git-control-tower/ui/src/components/FileListTypes.tsx
scenarios/git-control-tower/ui/src/components/FileRow.tsx
scenarios/git-control-tower/ui/src/components/FileListMobileActions.tsx
scenarios/git-control-tower/ui/src/components/IconButton.tsx
scenarios/git-control-tower/ui/src/components/ViewModeCycleButton.tsx
scenarios/git-control-tower/ui/src/components/MobileHeader.tsx
scenarios/git-control-tower/ui/src/components/ProjectTreeView.tsx
scenarios/git-control-tower/ui/src/components/AIProvenanceTab.tsx
scenarios/git-control-tower/ui/src/components/ScenarioReviewPanel.tsx
scenarios/git-control-tower/experience/index.json
scenarios/git-control-tower/experience/pages/workspace.json
scenarios/git-control-tower/experience/fixtures/mobile-regression-oracle.json
scenarios/git-control-tower/bas/cases/mobile-regression/workspace-mobile-stateful.json
scenarios/git-control-tower/docs/internal/MOBILE_REGRESSION_ORACLE.md
scenarios/experience-manager/api/internal/checks/registry.go
scenarios/experience-manager/api/internal/checks/bas_references.go
scenarios/experience-manager/api/internal/spec/spec.go
scenarios/experience-manager/api/internal/claimtypes/claimtypes.go
scenarios/experience-manager/api/internal/reconcile/claim_evaluators.go
scenarios/experience-manager/api/internal/reconcile/snapshot.go
scenarios/experience-manager/api/internal/reconcile/bas_capturer.go
packages/repo-contract-go/contract.go
packages/react-component-library/dist/components/IconButton/versions/3.1.2/IconButton.d.ts
packages/react-component-library/dist/services/ChromeTheme/versions/1.0.0/ChromeTheme.d.ts
scenarios/web-console/ui/src/lib/chromeTheme.ts
scenarios/web-console/ui/src/components/TopSafeArea.tsx
```
