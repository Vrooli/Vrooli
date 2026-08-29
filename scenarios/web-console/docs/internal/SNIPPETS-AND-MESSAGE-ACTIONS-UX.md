# Snippets and message actions — UX design reference

Durable record of the design investigation and decisions behind sender-owned
message snippets, handoffs beyond a group, and the bounded message-action row.
It extends [`ROLES-AND-HANDOFFS-UX.md`](./ROLES-AND-HANDOFFS-UX.md); its D3
receiver-owned role prompt and all ten prohibitions remain in force.

Visual authority:

- `/home/matthalloran8/Vrooli/scenarios/web-console/docs/internal/SNIPPETS-AND-MESSAGE-ACTIONS-UX.html`
- `/home/matthalloran8/Vrooli/scenarios/web-console/docs/internal/snippets-mockups/`

The HTML memo and six artboards own visual arrangement, control placement, and
copy. This Markdown record owns mechanism. If they disagree about behaviour,
this record wins; if they disagree about arrangement, the visual artifacts win.

## 1. The friction

Reusable text currently belongs only to a receiver role and is reachable only
through a handoff inside a group. Operators therefore keep repeatable prompts
outside Web Console, ungrouped sessions cannot hand work anywhere, and each new
message action adds another tiny prop-fed icon to a row already too narrow for
a phone. The missing object is sender-owned reusable text with no destination;
the missing interaction model is a bounded, declared action set.

## 2. Measured findings

Every finding was read from the `agi` worktree on 2026-08-28.

| # | Finding | Evidence |
|---|---|---|
| 1 | Ungrouped sessions have no handoff targets because the resolver returns early. | `ui/src/hooks/useHandoff.ts:151-155` |
| 2 | No schema, proto, Go type, or UI type stores sender-owned reusable text. | `api/internal/sessions/schema.sql:160`; `api/internal/grouptemplates/types.go:36-44` |
| 3 | The seeded implementer prompt is intentionally one workflow-neutral line. | `api/internal/grouptemplates/seed.go` |
| 4 | The handoff template language supports only `{{payload}}` and has no parser. | `ui/src/lib/handoff.ts:31-37`; `docs/internal/ROLES-AND-HANDOFFS-UX.md` D4 |
| 5 | An empty handoff payload blanks the placeholder and can leave a silent hole. | `ui/src/lib/handoff.test.ts` |
| 6 | The message header renders a flat run of as many as seven actions with no overflow. | `ui/src/components/MessagesPane.tsx:277-455` |
| 7 | Existing message action hit areas are about 18 px, below the console's 44 px mobile pattern. | `ui/src/components/MessagesPane.tsx:284-292,336-352` |
| 8 | `MessageRowProps` carries 32 props, 18 used only to feed actions. | `ui/src/components/MessagesPane.tsx:150-186` |
| 9 | Send-to-composer is not wired in the live messages surface. | `ui/src/components/WorkspacePaneShell.tsx:215-236`; `ui/src/components/ArchiveDrawer.tsx:473` |
| 10 | A message can be handed off only when a capture rule happens to match it. | `ui/src/components/MessagesPane.tsx:1321-1326` |
| 11 | A new-session handoff target requires a group id. | `ui/src/hooks/useHandoff.ts:26` |
| 12 | SQL millisecond timestamps and Go RFC3339Nano timestamps sort inconsistently as text. | `api/internal/sessions/schema.sql`; `api/internal/grouptemplates/types.go` |

## 3. Vocabulary

| Term | Meaning |
|---|---|
| **Snippet** | Named, colour-tagged reusable message text owned by the sender, with no receiver, group, or role. |
| **Variable** | A `{{name}}` token whose name matches `[a-z][a-z0-9_]*`; an unfilled variable stays visible. |
| **Message action** | A declared action with identity, label, icon, placement, applicability, and execution behaviour. |

Session, Group, Role, Handoff, Payload, and Prompt template retain the meanings
defined in the predecessor record.

## 4. Generalization rules

These are mechanical review tests.

1. **No workflow-named field.** `plan_path`, `planPath`, `plan_file`,
   `implementer_id`, `planner_id`, and `critic_id` do not enter schema, proto,
   Go types, or UI types. Reusable text is `body`.
2. **The send path cannot reach matching.** `lib/handoff.ts`,
   `lib/snippetVars.ts`, and `hooks/useHandoff.ts` import neither
   `lib/captureRules` nor `api/handoffrules`.
3. **Substitution imports nothing.** `lib/snippetVars.ts` contains no import.
4. **No privileged built-ins.** There is no `is_builtin`, delete guard, or
   runtime branch on a seeded snippet id.
5. **No snippet id on receiver-owned objects.** Roles and group templates store
   no snippet identifier or foreign key.
6. **Existing message-action selectors survive.** Every established
   `data-testid` continues to resolve after the action refactor.

## 5. Decisions

| # | Decision | Rejected alternative | Tradeoff accepted | Revisit trigger |
|---|---|---|---|---|
| D16 | A snippet is sender-owned and has no destination. | Extend receiver-owned `incoming_prompt`. | It cannot carry receiver framing; role prompts continue to do that. | Operators consistently want automatic receiver framing on the same reusable body. |
| D17 | Snippets live in Web Console's database. | Make capture depend on prompt-manager. | No governance, indexing, or cross-machine sync. | Agents, rather than operators, must discover snippets. |
| D18 | Variables are flat named `{{name}}` lookups. | Keep only `{{payload}}` or add expressions. | No computation, branching, escaping, or nesting. | A value must be derived from another value. |
| D19 | An absent variable remains verbatim. | Blank it like handoff composition. | Two similar-looking renderers intentionally differ. | One renderer is proven to satisfy both contracts. |
| D20 | Auto-resolved variables are the closed set `payload`, `cwd`, `session`, `selection`. | Resolve arbitrary context keys. | A fifth name requires a reviewed change. | Two independent flows need the same fifth name. |
| D21 | Using a snippet in a role copies text, never an id. | Reference the snippet from a role. | Copies may drift. | Drift repeatedly costs operators. |
| D22 | Handoff targets are ordered `group`, `other`, `new` sections. | One flat list. | The target surface is taller. | Headings prove to be noise in use. |
| D23 | Usage is recorded by atomic `TouchSnippet`. | Re-send the body through upsert. | One extra RPC. | Sibling domains adopt the same atomic touch seam. |
| D24 | Promotion to a skill is explicit and one-way. | Synchronize snippet and skill. | The two may drift and store no mutual identifiers. | Skills become safely editable from the console. |
| D25 | Message actions come from one declared list with bounded primary placement. | Continue adding row props and buttons. | An indirection sits between control and handler. | Per-action state cannot be expressed cleanly. |
| D26 | Existing message-action `data-testid` values are preserved. | Rename them with the new action ids. | The old and new names are not perfectly uniform. | A separately validated selector migration. |
| D27 | One picker is reachable from all five composing/capture surfaces. | Ship one surface first. | More integration surface in one delivery. | Never; partial reach recreates the defect. |
| D28 | Snippets are never suggested while typing. | Match draft text continuously. | Operators must open the picker. | Operators request suggestions after using the picker. |
| D29 | Snippet colour reuses `HEADER_COLORS`. | Add a snippet-only palette. | The palette remains limited to eight. | The shared palette changes everywhere. |
| D30 | Ordering is pinned, recent use, use count, then id. | Order by count or name alone. | A recent one-off can outrank an older staple. | Operators repeatedly lose staple snippets below recent noise. |

## 6. Mechanism

Snippets are an independent domain with a repository interface, domain-owned
SQLite schema, Connect-RPC service, CLI group, typed UI client, and shared hook.
`renderSnippet` performs substitution only. `renderHandoffPrompt` continues to
compose receiver framing and payload without modification.

Handoff text resolves per target in this order: explicit target edit, selected
snippet, receiver role prompt, payload. Targets remain sequentially processed
and retain distinct sent, queued, and failed outcomes.

Message actions are filtered from one ordered registry. At most three primary
controls render inline; all remaining applicable actions use the established
`ContextMenu/1` overflow. Every interactive action affords a 44-by-44-pixel
target while retaining the existing 14-pixel icon scale.

## 7. Deliberate omissions

There is no typing-time suggestion, cross-machine synchronization, tag/folder
taxonomy, automatic send, snippet-to-skill synchronization, role-held snippet
identifier, or change to capture-rule matching. `renderHandoffPrompt` and its
existing empty-payload behaviour remain untouched.
