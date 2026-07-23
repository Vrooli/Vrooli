# Initialize: Research Backlog Item

Research is a backlog kind, not a separate lifecycle. Investigate the question,
preserve useful evidence in the item's folder, then author an actionable
implementation plan through the normal `plan.author` path.

## Required reading

Read `swarm-manager-backlog-tools` and `swarm-manager-processing-guidance`
before working. Use the item's existing files, archive materials, and goal or
milestone context as evidence; never overwrite user-provided files.

## Procedure

1. Read the item, its files, scope/acceptance fields, and any owning milestone.
2. Store concise research evidence in a descriptive item artifact (for example
   `research-notes.md` or `evidence/<source>.md`). `conclusion.md` may be read
   as historical evidence but is never created or treated as canonical.
3. If evidence supports an implementation, run or request `plan.author` and
   produce a Plan Manager-backed implementation plan with concrete steps,
   validation, and acceptance criteria. Research items bind that plan through
   `plan_ref` and require ordinary plan acceptance before queueing.
4. If evidence supports no action, emit an `archive_item` mutation proposal
   with a clear rationale; do not invent a conclusion workflow or archive the
   item directly.
5. Put unresolved operator choices in the shared Decide surface as questions
   or typed proposals. Do not create workshop rounds as an alternate lifecycle.

## Boundaries

Do not edit `archive/`, queue work, or mutate backlog state directly. Do not
write `plan.md` or `conclusion.md`; Plan Manager owns the canonical plan and
historical conclusion files remain inert evidence in the Files tab.
