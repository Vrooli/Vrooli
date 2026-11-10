# Flows — Client Workflow Rehearsal

## First rehearsal: import → simulate → repair → review → handoff

1. The operator submits a synthetic workflow document and title. The importer normalizes supported n8n graph nodes and connections, classifies trigger entry points, strips credentials and secret-shaped values, validates topology and rejects unsupported executable nodes.
2. The service passes only a normalized credential-free graph to its typed engine adapter. Multiple entry points require an explicit trigger selection.
3. The local engine walks connected nodes, follows If/Switch outputs, invokes description-driven synthetic HTTP operations and generic native-node resource/operation bindings, and executes included subworkflow or error-workflow graphs with recursion/depth guards. No workflow-provided URL is opened and native service code is never invoked.
4. The engine returns an ordered trace. The service evaluates the seeded fault rule, stores failed assertions, and records repaired results as separate evidence events.
5. The record enters `needs_review`. The operator can inspect provenance, credential replacement counts, simulated calls, fault/repair evidence, and a handoff preview.
6. A human explicitly approves or requests changes. The choice, reviewer note, and timestamp are durable. Successful execution never implies approval.

The graph engine, synthetic OpenAPI/recorded-exchange catalog, and HTTP handler are local contract implementations. They do not claim native n8n import or execution. In particular, no task-runtime routing, TLS/host-alias behavior, external DNS denial, or built-in Gmail/Sheets/CRM virtualization has been verified. See `docs/internal/SEAMS.md` for the constrained bridge hypothesis and gates.

## State model

`imported → running → needs_review → approved | changes_requested`

A run can instead enter `failed` with a typed reason and retained trace. `approved` and `changes_requested` are operator decisions, never inferred from a successful run. Re-import creates a new immutable workflow revision; it does not overwrite evidence for an earlier run.

## Remaining charter work

The current synthetic fixtures exercise graph branching, multiple trigger selection, subworkflow/error paths and generic description-driven HTTP matching. They do not count as imports or execution of public workflows. The charter remains 30 unlike license-checked community exports, ten seeded defects repaired, all trigger families proven in n8n, and one Gmail/Sheets/CRM client-shaped export through review and handoff. Large payload coverage also requires a separately designed bounded storage/replay path.
