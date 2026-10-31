# Coverage and traceability

Use this as a set of questions, not a mandatory document schema. The project chooses names, storage, and depth.

## Establish the source basis

Inventory the requested outcomes, actors/customers/operators, constraints, prior approvals, specifications, research, specimens, implementation or operating evidence, and known issues. For consequential facts retain a stable locator, observation date, authority and confidence. Mark stale implementation evidence explicitly. Search for existing material before regenerating it.

Resolve contradictory sources visibly. Preserve both the original and the proposed resolution when authority is unclear. Maintain the distinction between proposed design and verified current implementation or operation throughout every chapter.

## Select coverage deliberately

Declare the primary profile and any hybrid additions, with the reader/outcome that needs each. Treat profiles as coverage questions, not compulsory chapter titles. Retain relevant hard cases and justify material exclusions. For a scoped chapter, assess its stated boundary and cross-boundary contracts; do not demand an unrelated whole-product expansion.

- Product/UX: use the experience questions below
- Service/business: use service-and-business-design.md for the offer, delivery system, commercial logic and validation
- Systems/infrastructure: use systems-and-infrastructure-design.md for authority, architecture, operating behavior and acceptance evidence
- Hybrid: connect shared identities, states, promises and dependencies across profiles; do not stack three disconnected checklists

## Product/UX experience coverage

Cover where relevant:
- Product purpose, user outcomes, non-goals, release scope and success measures
- Actors, contexts, constraints and accessibility needs
- Information architecture, navigation and entry/exit points
- End-to-end journeys, alternate paths, interruption/resume and failure recovery
- Screen inventory with purpose, hierarchy, primary action, content, controls and responsive behavior
- Shared components and interaction/state contracts
- Domain concepts, relationships, lifecycle rules and data ownership
- Calculations, units, missing values, validation and worked examples
- Persistence, offline/concurrency behavior, permissions, privacy and external boundaries
- Import/export, onboarding, settings, help, empty/loading/error/destructive states
- Visual language, asset provenance, layout and motion/accessibility rules
- Acceptance scenarios, decisions, risks, dependencies and unresolved evidence gaps

Include deeper technical contracts only where they resolve a real design ambiguity. Do not invent backend architecture simply to fill a chapter.

## Trace critical paths

For each important outcome, connect the initiating need or promise to its flow, responsible actors, relevant interface/service/system elements, state transitions, data/invariants, specimens, and acceptance examples. Trace the handoffs between profiles in hybrid designs. Use project-local identifiers or clear links. A matrix is useful only if it remains readable and helps find omissions.

Test round trips: creation → modification → persistence → reopening → dependent outputs. Test cancellation, undo, stale data, empty inputs, invalid units, conflicting choices, failure midway and retry. Use realistic scenarios alongside adverse cases. Record expected outcomes precisely enough to expose contradictions across chapters.

Make at least one critical path concrete: initiating inputs and state → expected outputs/records → adverse or unknown result → permitted next action. Use specific illustrative values, identities or payloads where they expose an ambiguity, labeled synthetic when applicable. Include the material missing-evidence branch: a recovery story that assumes a receipt exists must also explain absent receipts or uncertain admission when those are possible. Do not turn the example into a compulsory schema or claim it was executed.

## Bound feasibility and acceptance evidence

For a proposed validation experiment, specify the smallest useful observation, permitted effects, prerequisite-discovery and work/cost boundary, and stop/continue decision. Include operator attention and ongoing upkeep when recommending new process or infrastructure. Use supported estimates or explicit unknowns; do not invent savings, capacity or precise budgets. If prerequisites or cost are unknown, bound the first evidence-gathering step before committing to the larger experiment.

Report evidence/reporting integrity separately from success on the intended outcome. An honest partial or inconclusive result can pass the integrity check while leaving the underlying question unresolved. State what was established, what remains open and which evidence or decision would change that status; do not mark the intended outcome complete merely because uncertainty was reported correctly.

## Prepare owner decisions

For each consequential open choice include the question, why it matters, alternatives, recommendation, tradeoffs, relevant visual/source, dependencies, and what remains provisional. Never call the recommendation approved without evidence. Group low-risk independent defaults for efficient acceptance; keep dependent or consequential choices separate.

Provide a short review route through the dossier and a resumable next question. Do not force the owner to approve the whole document at once. Preserve settled decisions when new evidence only affects a subset.

At design stage, walk scenarios through the stated rules, relevant specimens and models and record contradictions or missing behavior. Execute tests only when an authorized prototype or implementation exists. Mark planned checks, design-inspection results and observed execution results distinctly; never infer implemented persistence or accessibility from mockups.

## Complete decision-index gate

Before delivery, scan every chapter, diagram, specimen, appendix and companion boundary for consequential unresolved choices. Reconcile them against one complete, navigable document review index. Give each choice a stable local label or anchor, a working section link, a recommendation, alternatives/tradeoffs, dependencies and the consequence of leaving it open. Keep figure and decision namespaces distinct; detect duplicate labels, missing entries and broken/backward references after export. Preserve project identifiers; do not rename live records just to match this method.

The document index is an inventory of reviewable design choices, not a second actionable queue. Only choices currently actionable under the project's own rules belong in that queue. Mark blocked, deferred and paused choices in the document without activating them. If there are no consequential open choices, state the scope and evidence for that conclusion rather than inventing decisions.

## First outcome versus future breadth gate

When the requested design includes a bounded first sale, release, service or deployment plus wider future capability, name the first outcome and provide a linked minimum route. Connect promise/need → service or system step → product/manual/operating capability → required evidence → blocking owner choice. Identify the minimum responsibilities, dependencies and exclusions; distinguish essential first-outcome requirements from later important capability. Link companion dossiers where they supply an obligation or design dependency.

Check that a reader can identify which unresolved choices actually block the next intended outcome, which can be deferred, and what evidence would permit progression. A manual bridge needs an explicit operator, capacity and failure path; it is not a way to hide absent capability. If no narrower first outcome is in scope, record that and do not invent a release plan.
