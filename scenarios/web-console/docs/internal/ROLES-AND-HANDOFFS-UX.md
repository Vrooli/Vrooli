# Roles and handoffs — UX design reference

Durable record of the design investigation and the operator decisions behind
web-console's group-first sessions, waiting roles, and the generic handoff verb.
The plan `web-console-roles-and-handoffs-group-first-sessions-waiting`
implemented this design; this document holds the rationale, the measured
starting facts, the vocabulary, the generalization rules, and the decisions.

Visual companion: [`ROLES-AND-HANDOFFS-UX.html`](./ROLES-AND-HANDOFFS-UX.html) —
open in a browser for rendered mockups of every new and changed surface, in
web-console's real palette, across ten sections. **The HTML file is the authority on visual
arrangement, control placement, and copy.** This document is the authority on
mechanism. Where the two disagree on how something works, this document wins;
where they disagree on how something looks, the HTML wins.

---

## 1. The friction

The operator described a daily workflow and asked why it feels convoluted:
start a session with one coding agent for design and planning, talk until a
plan exists, then start a second session with a different coding agent to
implement that plan. Both sessions belong together, so they get grouped. The
group is thrown away afterwards.

That workflow cost nine manual steps: open the dialog, pick an agent, plan,
copy the plan path, open the dialog again, pick the other agent, create a
group, name and colour it, assign both sessions, paste the path plus an
instruction, and later clean up the dead group.

## 2. Measured findings

Every finding below was read out of the code on branch `agi`, not inferred.

| # | Friction | Evidence |
|---|---|---|
| 1 | The new-session dialog never states which group the session will join | `Workspace.tsx` held the pending group in `pendingLauncherGroupRef` and opened the dialog. `TerminalLauncherProps` had no group field at all, so the dialog *could not* render the destination. |
| 2 | Session styling is applied silently | The session-reconcile effect writes `header_color`, `theme_id`, and `font_size` from `workspace.default*` after the session exists. The launcher never showed or offered these values. |
| 3 | One task costs two trips through the dialog | `handleLaunch` called `launchSession` once. No multi-session creation path existed in the UI, the API, or the proto. |
| 4 | Group creation is disconnected from the work | `useGroupActions.createGroup()` always created the literal name `New Group`, then the drawer started an inline rename. |
| 5 | Assigning a session opens the whole manager | `ManageGroupsDrawer` was both the assign picker and the CRUD surface. |
| 6 | The plan path is copied and pasted by hand | No handoff concept existed. `submitToActiveTerminal(data, intent, targetId)` already accepted an explicit target pane, so the missing part was the UI and the queueing, not the transport. |
| 7 | Dead groups accumulate with no sweep | Nothing reaped empty groups. The drawer deleted one group per confirm dialog. |
| 8 | The launcher spends its space on the wrong decision | The target block rendered one full-width card per machine plus a readiness grid. The agent list rendered eight full-width rows, four of them `(attributed)` duplicates. |
| 9 | The Codex sign-in card is a built-in with no state behind it | `codexDeviceAuthCommand` was a hardcoded constant rendered unconditionally. A repository-wide search for signed-in detection returned only this constant. The card could not know whether the operator was signed in. |

Finding 1 is not a missing label. The dialog was structurally incapable of
showing its destination because the group id lived in a ref that was never
passed down. That is a symptom of the model: the console had one first-class
noun, the session, and treated a group as decoration applied afterwards.

## 3. Vocabulary

| Term | Meaning |
|---|---|
| **Session** | One terminal and one process. Unchanged by this work. |
| **Group** | One piece of work, named for the task. The primary object. |
| **Role** | A named position inside a group. **Running** when it holds a session id, **waiting** when it holds a command and no session id. |
| **Handoff** | The act of sending a message to one or more roles in the same group. A handoff starts a waiting role before it sends. |
| **Payload** | The optional text a handoff carries — a file path, a selected passage, or empty. The console does not classify payloads. |
| **Prompt template** | Text stored on a *receiving* role, containing at most one `{{payload}}` placeholder. |
| **Group template** | A saved list of role definitions. Creating a group from one creates the group and its roles in a single action. |
| **Capture rule** | A named pattern that decides when the console *offers* a handoff. It never sends anything. |

## 4. The generalization rules

The operator stated the constraint directly: the design must not encode one
workflow. It must not assume groups of two, must not assume the payload is a
plan, and must let other people build flows the author never imagined — for
example handing a plan to a critic rather than an implementer.

That makes **"nothing in the data model may know what a plan is"** a hard
design test. These ten prohibitions are review tests a reader can check
mechanically, not preferences.

1. **No field named for a workflow.** `plan_path`, `planPath`, `plan_file`,
   `implementer_id`, `planner_id`, `critic_id` appear nowhere in the schema,
   the proto, the Go types, or the UI types. The payload field is `payload`.
2. **No pair-shaped model.** No `primary`/`secondary`, no `left`/`right`, no
   singular `source_role`/`target_role` pair on a group. A group holds an
   ordered list of roles of any length from one upward.
3. **No fixed target count.** The handoff composer accepts a multi-selection.
4. **No detection in the send path.** `renderHandoffPrompt` and the send code
   do not import, call, or reference the capture-rule matcher. Deleting every
   rule changes no line of the send path's behaviour.
5. **No privileged built-ins.** Seeded content is written through the same
   public write path the UI uses. No `is_builtin` column, no code path that
   treats seeded rows differently, no guard preventing their deletion.
6. **No hardcoded agent identity in the role model.** A role stores a command
   string, never an enum of known agents.
7. **No workflow assumption in prompt substitution.** `renderHandoffPrompt`
   never inspects the payload's file extension, path, or content.
8. **No terminal-view detection UI.** See decision D11.
9. **No requirement that a group have a template.** Creating a group with no
   template and adding roles by hand works.
10. **No requirement that a session be a role.** Dragging a session into a
    group keeps working and creates no role row.

Three greps enforce rules 1, 4, and 5; they are listed in
[`SEAMS.md`](./SEAMS.md#the-handoff-seam) and must return no match.

## 5. Decisions

| # | Decision | Rejected alternative | Tradeoff accepted | Revisit trigger |
|---|---|---|---|---|
| D1 | The new noun is **role**, not step, stage, slot, or seat | `step` / `stage` | "Role" carries no ordering, so a log watcher running beside an implementer is expressible. Slightly less obvious for a linear pipeline. | Operators consistently describe roles as an ordered pipeline. |
| D2 | Roles live in a new `workspace_roles` table | Make `workspace_panes.session_id` nullable with a synthetic PK | The sidebar merges two ordered lists. Avoids touching `ReassignPane` and session recovery. | A second feature needs pane rows without sessions. |
| D3 | The prompt lives on the **receiving** role | Store it on the sender, per source/target pair | Any session can hand off to a role and get that role's framing, so handoffs compose. Per-source wording uses the composer's edit field. | Operators repeatedly retype the same override for one source. |
| D4 | Template language is one `{{payload}}` placeholder | A general expression or variable language | No parser, no escaping rules, no error states. A template cannot compute. | Two distinct payload kinds must appear in one prompt. |
| D5 | Handoff delivery goes through the existing pending-input queue | Poll for a shell prompt and type when ready | The operator sees queued text and can discard or flush it. Delivery is not instant when the agent is slow to boot. | Queued handoffs routinely sit undelivered. |
| D6 | Capture rules only produce suggestions | Rules that send automatically | A wrong rule costs a dismissed chip, never a message sent to an agent. | Never — this is the safety property that makes rules shippable. |
| D7 | Auto-close a group when it has no panes **and** no waiting roles | Keep every group until deleted by hand | Empty groups stop accumulating. The waiting-role exemption covers the templated case. | Operators report losing groups they meant to keep. |
| D8 | Undo is client-side, with no tombstone column | An `archived_at` column and a server-side restore | No schema surface for a ten-second window. A reload during the window loses the undo. | Undo must survive a reload or work across devices. |
| D9 | Role RPCs extend `WorkspaceService` | A new `RolesService` | Roles are workspace layout; `GetLayout` stays the single call that returns the whole workspace. | `WorkspaceService` exceeds a size the team wants in one service. |
| D10 | Templates and rules use a JSON child column | Child tables with foreign keys | Matches `shortcut_profiles.shortcuts`, so there is one storage idiom. Child rows are not independently queryable. | A query must filter templates by a property of one role. |
| D11 | No handoff UI inside the terminal view | Overlay suggestion chips on terminal output | No hit-testing against reflowing text, no alternate-screen-buffer breakage. The terminal keeps a header button. | Never for detection UI; a header control is always acceptable. |
| D12 | The Codex sign-in card is deleted, not made conditional | Detect signed-in state and hide the card | There is no signed-in detection anywhere in the codebase, so "conditional" means building an agent-auth probe for a card whose need disappears once the agent tells the operator. | A general agent-auth capability exists for another reason. |
| D13 | Capture rules ship last | Ship rules with the handoff verb | Suggestion is the only part that can be wrong in a way that annoys daily. | None; this is sequencing, not architecture. |
| D14 | Attributed shortcut variants fold into their parent agent card | Keep eight separate cards | Halves the agent list and states the real choice: an agent plus an attribution setting. | Attribution stops being a per-launch toggle. |
| D15 | Roles are optional inside a group | Every session in a group becomes a role | Drag-to-group and the assign picker keep working with no role rows. Two code paths render group members. | Roles cover every grouping case in practice. |

### Decisions confirmed during implementation

- **The picker absorbed the manager's two jobs that a picker can do.** It
  listed every group in one flat run and offered no way to maintain one, so
  an operator who wanted to rename or sweep a group had to know a separate
  drawer existed. It now carries the Manage Groups mockup's split — groups
  holding at least one session, then groups holding none, with "Close all
  empty" on the second section's heading — and an Edit toggle that turns the
  rows into maintenance: recolour from the swatch, rename in place, close from
  the row.

  A group of waiting roles counts as EMPTY, matching the manager: a group of
  placeholders is still one you might sweep, and its row states the waiting
  count so the decision is informed rather than blind.

  Closing routes through `closeGroup`, which ungroups every member before it
  removes the group and captures an undo snapshot first. So closing a group
  releases its sessions and never ends them, and it is reversible — which is
  what makes it safe to offer behind a single tap with no per-row confirm. The
  footer says so in edit mode rather than leaving it to be discovered.

  The listbox role is dropped while editing: the rows are forms, not options,
  and claiming option semantics over a set of text fields would lie to a
  screen reader.

- **Closing a group had no entry point on the group.** The header menu
  carried collapse, new-session and a deep link into the manager; closing was
  reachable only by opening the manager and finding the row. So the way it
  actually got done was closing every session by hand — which left the group
  behind anyway, since a group outlives its members and then shows up as an
  empty row in the picker.

  The header menu now ends with a destructive **Close group…**, on both the
  sidebar and the tab strip. The ellipsis is a promise: it opens one
  confirmation, rendered once in `Workspace` off `closeGroupTarget` in the
  store, because two surfaces opening two dialogs would be two implementations
  of the same consequences.

  The dialog's shape follows from what is and is not reversible. Closing the
  group alone is cheap — the sessions survive as ungrouped panes and the undo
  banner covers the group — so that is the default and the dialog is mostly a
  statement of consequences. Closing the sessions too is what the operator
  usually wants and is a real destruction, so it is an opt-in checkbox, off on
  every open (a remembered "yes" would turn one deliberate choice into a
  standing one), and it routes through the SAME handler the tab menu's Close
  uses — the sessions are archived, not destroyed, and the dialog says so.
  The group is closed first so its undo snapshot records every member while
  they are still members; archiving first would let auto-close fire and leave
  the handler closing a group that no longer exists.

- **Waiting roles were persisted but never read back.** `GetLayout` returns
  roles alongside panes and groups, and `api/workspace.ts` decoded them — but
  nothing in the client ever called `setRoles` with the result. The store's
  `roles` array was only ever written by live actions, so every waiting role
  vanished on reload while still sitting in `workspace_roles`. It read as a
  feature that had never been persisted at all; in fact only the last step
  was missing.

  Hydration now writes them, deliberately OUTSIDE the `store.panes.length === 0`
  guard: that guard exists so a second hydration cannot clobber live pane
  state, and roles have no competing source. Two rules ride along. A role
  whose session is no longer in the live list comes back WAITING rather than
  pointing at a dead id — a handoff aimed at that id would go nowhere — and
  the correction is local only, because rewriting it on every hydrate would
  turn one transient empty session list into a write storm. And any role
  created locally while the request was in flight is merged in rather than
  dropped.

- **The launcher remembers the template you used last.** A persisted
  preference (`lastGroupTemplateId`), validated against the live list on every
  open so a deleted template cannot preselect a name that resolves to nothing.
  Restoring is one-shot per open: an operator who then picks "No template" has
  made a choice, and re-applying the remembered id would undo it — which is
  also why "No template" is stored as `null` rather than left unrecorded.

- **A waiting role rendered as a detached box under its group.** Two causes,
  both in the sidebar. `groupPosition` was computed from panes alone, so a
  group's last *pane* was always marked `last` — which closes the block's
  border and adds its bottom margin — even when waiting roles still had to be
  emitted after it. And `RoleRow` drew a fully dashed border of its own plus a
  `mb-1`, so the roles then formed a second box below the first. The result
  was that a group with one running session and one waiting role read as two
  unrelated things stacked on top of each other, which is the opposite of what
  the mockup's "group at rest" figure shows.

  The block is now one container: header, running sessions, then waiting
  roles, with only the last member closing it. "Not running" moved entirely
  into the row's interior — a hollow accent bar where a running session has a
  solid one, a muted ground, faint text and a WAITING pill — because the
  outline belongs to the group, not to the member's state.

- **The launcher was rebuilt against the mockup a second time.** The first
  pass kept the old page furniture around the new controls, and the result did
  not look like the design record at any width. Four things were wrong and are
  now fixed: the mode switch was a hand-rolled `role="tablist"` rather than the
  library's `Tabs`; machine and destination sat in a `sm:grid-cols-2` under a
  "Location" heading with a refresh button beside it, which on a phone stacked
  into two full-width cards and a header; the catalog's state ("Remote node
  readiness is shown for every registered node") was a full-width amber card
  explaining a list the operator had not opened; and the agent grid collapsed
  to one column, so the thing you came to choose was a list you read rather
  than a grid you scan.

  Now: the library's `Tabs` owns the mode strip. Machine and destination are
  compact triggers on one wrapping row with no heading. The catalog state, the
  refresh control, "Link a machine…" and "Manage machines" all live *inside*
  the machine menu, beside the list they describe — nothing about the fleet
  costs dialog space until you open it. An unreachable machine is one amber
  line, not a card. The agent grid is two columns at every width and gains a
  dashed card pointing at the shortcut editor that owns its contents. Appearance
  and Session Options are two inline disclosures on one row.

- **Group mode's role rows were a form, not a list.** Three boxed inputs
  squeezed onto one line — a 7rem label field, a command field, a chip and a
  bin — left every control too small to hit and gave a role no readable
  identity. The row is now a card with the group's colour on its leading edge:
  the name is a full-width field that reads as text until focused, the command
  sits under it behind a `$`, and a waiting role carries its incoming message
  as a third line, which the launcher previously could not set at all. The
  status chip states the cost (`Starts now` spends a process; `Waiting` does
  not) and toggles. The template picker is a trigger and a menu matching the
  machine picker rather than a native `<select>`, and it shares the opening row
  with the machine trigger exactly as the mockup has it. The footer carries
  "Edit template", Cancel and Create group.

- **Manage Groups gained the header facts and the sort control** from its
  mockup: the count of groups and how many are empty (counted across the
  workspace, not the filtered view, so filtering cannot hide the number that
  says whether the surface is worth opening), and a Recent/Name toggle. The
  colour swatch became a labelled 28px control instead of a 16px dot the
  operator had to guess was a button.

- **The group picker is one overlay, not a combobox in two dresses.** The
  first build used `Combobox.allowCreate` twice: inline in the launcher as the
  destination, and inside an anchored popover for the session menu's "Add to
  group". Two problems surfaced in use. The anchored popover was rendered
  inside the sidebar's *archive* branch, so "Add to group" from the session
  list — the only place an operator actually reaches it — opened nothing at
  all; a control mounted inside one view branch is invisible from the other,
  and no test covered the branch it was missing from. And a text option in a
  dropdown showed a group's name and nothing else, which made choosing a
  group feel like filling in a form field rather than picking a thing that has
  a colour and a size.

  Both entry points now open the same `GroupPickerOverlay`, mounted at the top
  level of its host. Groups render as cards carrying colour, name and size
  ("3 sessions · 1 waiting", or "Empty"), creating a group is a named row
  Both entry points now open the same `GroupPickerOverlay`, mounted at the top
  level of its host and laid out as the mockup has it: one field that both
  filters the list and names a new group, tight rows carrying a colour swatch
  and a size ("3 sessions · 1 waiting", or "Empty"), an ungroup row muted at
  the end, and a pinned create action that never scrolls away. Typing a name
  that matches nothing and pressing it is the same gesture as picking one that
  does, so you never create an empty group first and assign to it afterwards.
  The launcher's destination is a compact trigger wearing the group's own
  colour — "Into <group> · 2 sessions" — and pressing it opens that overlay.
  `Combobox` is no longer involved in group choice.
- **`UndoBanner/1` could not be used as the undo surface.** It takes no props
  beyond `className`/`style` and reads its records from an `UndoManagerProvider`
  context that web-console does not mount. Wiring that provider would put a
  library-owned service in charge of a state machine this scenario needs to own
  (the snapshot must survive a failed replay). The undo control is therefore a
  small local component driven by `closedGroupUndo` in the workspace store.
  The plan's reuse table named `UndoBanner`; that entry did not survive contact.
- **Queued input had no desktop surface.** `MobileToolbar` rendered the pending
  chips; no desktop pane did. Since a handoff to a not-yet-mounted terminal
  reports `queued`, a silent queue would be as bad as a dropped message, so a
  desktop pending-input strip was added alongside the handoff verb.
- **A recovered session keeps its role.** `ReassignPane` re-keys a pane row
  during session recovery. `ReassignRoleSession` performs the matching move for
  `workspace_roles`, so a role never points at a session id that no longer
  exists. The alternative — sending the role back to waiting — would have
  silently discarded the operator's running work on every recovery.

## 6. Where a handoff is offered

| Surface | Entry point | Why |
|---|---|---|
| File viewer header | A **Hand off** button beside Copy path | The operator's existing habit. `useFilePreviewController(sessionId)` already holds both the resolved path and the source session. |
| Messages view | A dismissible suggestion chip, when a rule matches | The structured surface, where a chip can attach to a message without fighting the renderer. |
| Pane header | A **Hand off** control beside the pane menu | The always-available manual path, with an empty payload. |
| Sidebar role row | A **Hand off** glyph when the group holds another role | Reaches a waiting role without first focusing its pane. |
| Terminal view | **Not built** | The terminal is a byte stream with cursor addressing and an alternate screen buffer. Overlaying detection UI means hit-testing reflowing text and breaking every time an agent redraws. See D11. |

## 7. The end state

The operator opens the launcher once, picks **Plan → Implement**, types the task
name, and gets a named group holding one running session and one waiting role.
The operator plans. The operator opens the plan file from the Messages view, as
they already do, and presses **Hand off** in the file viewer header. The
implementer starts and receives `Implement the plan at <path>`. When the
operator closes both sessions, the group closes itself and offers an undo.

Nine manual steps become three. No copy-paste. No leftover group.
