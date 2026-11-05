# Systems and infrastructure design coverage

Use when the requested design concerns a platform, integration, data flow, infrastructure or operating system boundary. Cover the system coherently at the requested scope; do not force app screens, prescribe a new platform or mistake a diagram for deployed behavior. Add product/service coverage only where it resolves a real human or delivery obligation.

## Establish purpose and authority

State the outcome, quality goals, constraints, non-goals, current baseline and proposed change. Compare the smallest no-new-infrastructure/manual alternative with the recommendation when relevant. Keep observed current behavior, source-stated contracts, accepted choices and proposed architecture visually and verbally distinct.

Name owners, actors, consumers, authoritative sources, derived projections and external dependencies. State who may read/write which object, which decisions remain project-local, and how conflicts are resolved. A central index or convenience view must not silently become authority over independently owned sources.

## Select explanatory views

Choose the views that answer consequential questions; avoid a compulsory diagram set:

- Context/ownership: actors, dependencies, responsibilities, trust and permission boundaries
- Logical/data: components, identities, schemas, provenance, authoritative versus derived state and lifecycle
- Sequence/state: transactions, asynchronous work, concurrency, reconciliation, interrupted or uncertain outcomes
- Deployment/operations: environments, topology, isolation, health, cost/performance and operating responsibilities, only when relevant

Use consistent names and direction/meaning for edges across diagrams and prose. Label current versus proposed elements. Retain editable diagram sources and check label size at the actual delivered page width. A diagram should explain a decision or failure path rather than merely decorate an architecture claim.

## Specify behavior at risky boundaries

For consequential interfaces define inputs/outputs, identity/revision binding, preconditions, valid transitions, invariants, errors, retry/reconciliation and ownership. Discuss consistency, ordering, duplicate delivery, concurrency, idempotency and stale/unknown data where they can change an outcome. Separate content revisions from status/current pointers. Explain what happens when evidence, pins or providers are missing rather than silently marking data current.

Specify permission/threat boundaries, minimum access and sensitive-data handling at design level. Keep proposed credentials, access expansion and security settings behind their actual approval requirements. Do not put secrets in the dossier or perform access changes to validate a diagram.

Define observable signals and limits: what was attempted, committed, acknowledged, failed, uncertain or stale; who notices; who may act; and how the consumer avoids mistaking absence of evidence for success. Describe relevant performance/cost assumptions with units and workload bounds, not unmeasured guarantees.

## Walk operation, recovery and change

Trace at least one representative end-to-end case and one material interruption/recovery case through the stated contracts, preferably a single connected fixture. Show detection, affected scope, safe hold/fallback, owner action, reconciliation, user/consumer consequence and acceptance evidence. Define what must not be retried while a prior side effect is uncertain. Keep lost acknowledgment distinct from failure. Apply the common concrete-example gate: show specific initiating state/inputs, expected output or receipt and the permitted next action when that receipt is missing or admission remains unknown; do not assume recovery evidence always exists.

Where change or migration is proposed, specify compatibility, prerequisites, cutover conditions, rollback limits, source preservation and recheck evidence. Identify irreversible steps explicitly. A minimum first deployment needs a bounded route through components, sources/permissions, operating responsibilities, evidence, blocking decisions and exclusions; later automation is separate from present readiness.

## Acceptance evidence

Bound each proposed experiment using the common feasibility gate, including prerequisite-discovery effort, permitted effects, a stop/continue decision and recurring operating/upkeep burden. Unknown runtime availability narrows the first discovery step rather than authorizing unlimited setup.

Map each quality goal or critical invariant to a design inspection, planned test or dated observed runtime result. Include owner, target revision/environment and pass/fail or unresolved result when known. Demonstrate that authority is preserved during a normal flow, stale input and interrupted update, with no false success or unauthorized retry. A rendered sequence is design evidence, not a concurrency/recovery test. Separate evidence integrity from the intended operational outcome: a correctly labeled inconclusive accounting or capture report does not settle the cost question or establish a successful round trip.

Index every consequential unresolved architecture/operating choice with alternatives, recommendation and dependencies. Record unresolved permission, capacity, recovery and migration evidence as blockers where appropriate. Dossier completion, approval and source application never stand in for deployment or verified operation.
