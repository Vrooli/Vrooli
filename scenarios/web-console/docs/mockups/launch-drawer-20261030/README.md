# Launch drawer target (2026-10-30)

Current design target for the New Session drawer and Manage agents.
`mockup.html` holds the clickable phone and desktop mockups and the review findings.
Hosted copy: https://claude.ai/artifact/TisBM6JjoGCc7SrPvv5FQB

The operator approved this target and its recommendations on 2026-10-30.

## Decisions

| ID | Decision | Rule |
|---|---|---|
| LD-1 | Choose the session type first | The drawer order is machine, then Agent, Shell or Group, then settings for that type, then launch. Agent controls never appear before the type is chosen. |
| LD-2 | A stale model list warns | A stale catalog does not hide model choice. Metered choices stay blocked while the catalog is stale. |
| LD-3 | Updates take one tap and keep rollback | A supported update installs without a qualification step. The previous version stays available for rollback. Manage says when the running version is newer than the version its permission policy was reviewed against. |
| LD-4 | Provider data decides usage windows | A window appears only while the provider reports it. Declared windows only reserve card height and show loading placeholders. |

## Layout

- The header holds the title and the machine picker. The title is "New session" everywhere.
- A segmented control offers Agent, Shell and Group.
- Agent shows a horizontal rail of cards, the settings panel for the selected harness, and a footer with a one-line summary and the launch button. A harness is always selected, so "Clear selection" is removed.
- Shell shows the command field and shell shortcuts. It shows no agent settings.
- Group holds the existing group picker.
- Appearance stays a collapsed row. "Compact idle sessions" leaves the launch flow, because it acts on running sessions. It moves to the drawer header's overflow menu.
- Desktop puts settings beside the choice, not below it. The content and data hook are the same.

## Harness cards

- Card height is identical while loading, loaded, and without usage. Height comes from the largest declared window count among visible harnesses.
- Each window row reads `Week  87% · 4d`, with a thin bar under it. The accessible name is the full sentence, such as "Weekly 87% used, resets in 4 days".
- Bars turn amber at 80% and red at 95%.
- A refresh keeps the current values until new values arrive.
- During one open drawer visit, descriptor resolution or hiding a harness never shrinks the reserved usage space. A provider reporting additional windows may increase it; those windows remain visible.
- A small dot marks a harness with something to do. The Manage button carries a count of available actions.

## Settings panel

Rows appear only when the selected harness supports them.

- Model, plus Provider where the descriptor declares provider selection.
- Compact at, with options for the selected model. Options never exceed that model's window. The row shows the window in tokens and a gauge. An unknown window is stated, not guessed.
- Save usage, as an on/off row that states its end time and what it lowers. The model row shows the lowered model.
- A notice row for a problem that affects this launch, with a link to its fix.

## Manage agents

- The list has a Custom or Recent order control, an "Update N ready" button, and one row per harness with status and a Shown switch.
- A harness detail page has, in order: available actions, usage with absolute reset times, version, and update checks.
- One sentence states the usage source, its age, and that the launch may use a different account.
- "Pinned" is renamed "Permissions reviewed for".

## Harness states

Each harness has one state and at most one primary action.

| State | Primary action |
|---|---|
| Missing | Install |
| Older copy runs first | Use Vrooli's copy |
| Not linked to Vrooli's copy | Link Vrooli's copy |
| Needs setup (root-owned) | Show `sudo vrooli setup --include-optional` |
| Update ready | Update |
| Up to date | None |
| Check failed | Retry, with no update indicator |

Web Console never elevates. Full deployments use the control plane through `cliinvoke`; standalone desktop uses the shared lifecycle package for verified native downloads into user-owned paths (decision 8 remains provisional for acceptance). Manage includes a read-only installation check and a cleanup preview whose identity must match before removal; independently found copies are preserved.

## Data owners

| Data | Owner |
|---|---|
| Harness identity and features | The `harness` block in each `resources/<name>/resource.json`, read through `packages/cli-core/agentcatalog` |
| Usage readings | Agent Manager live quota |
| Model context windows | Claude's status line feed and Codex's model cache, read through the shared package |
| Install state, eligibility, update, repair, rollback | `vrooli resource update`, in `internal/` |
| Launch command composition | The Web Console API, from a structured request |

## Implementation refinements

- **E33-stable-usage-space** (agent-authored, 2026-11-01): keep a per-machine, per-visit high-water mark for card rows. Source inspection and desktop/phone component regressions reproduced a two-row opening budget shrinking to one when descriptor data arrived. This preserves LD-4 and REQ-L3 without capping provider windows. J7 verifies the served production build through real Chrome at phone and desktop sizes, including measured card heights throughout delayed data loading.
- **E33-effective-launch-model** (agent-authored correction, 2026-11-01): preserve a saved stale metered choice visibly but block agent launch by button and Enter until refreshed or changed (LD-2). Full-power context rows use the usual model's own catalog data, state an unknown window if absent, and do not claim the Save usage context limit still applies. Shell remains independently launchable, including Enter when the last agent's billing account is unavailable. These correct implementation inconsistencies, preserving the approved design and capabilities.
- **E33-action-availability** (agent-authored correction, 2026-11-01): an update state with no server-offered update action earns no update dot or bulk update button. The reason remains visible in Manage. Control-plane action eligibility remains authoritative.
- **E33-usage-read-recovery** (agent-authored correction, 2026-11-02): retry the usage-mode read twice after a brief connection failure so supported settings return without reopening the drawer or waiting a minute. Mutations are never retried by this recovery. The unit regression verifies recovery; the real Chrome J7 journey verifies the operated context/settings controls. The visual composition is unchanged.
