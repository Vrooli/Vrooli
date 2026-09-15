---
name: tech-tree-designer
description: "Use Tech Tree Designer to inspect observed interfaces, planned proto contracts and capability coverage; distinguish existing operations from repository-wide proposal targets."
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [tools]
  tags: [ecosystem, graph, planning, proposals, ontology]
  status: active
  revision: 1
  requires:
    scenarios: [tech-tree-designer, program-runtime]
    commands: [tech-tree-designer, program-runtime library run, prompt-manager skill read]
  origin:
    kind: authored
---
## Tools focus: Tech Tree Designer

Choose an operation from the implemented graph, proto-planning and ontology surfaces.
Keep observed facts, authored intent and proposed changes distinct.

In scope: use and interpretation of existing operations. Product development uses
`prompt-manager skill read tech-tree-designer-improve`. Repository-wide bundles,
revision-bound approval and recoverable application are targets, not commands.
Read `path:scenarios/tech-tree-designer/docs/internal/PROBLEMS.md` before investigation.

### Choose the operation

Read the selected command's `--help` for arguments. Use the first matching row.

| Task / observable condition | Next step |
| --- | --- |
| Need to know which design operations exist | Read `path:scenarios/tech-tree-designer/docs/reference/cli-commands.md` [S0]. |
| Inspect observed interfaces for a bounded neighborhood | Run `tech-tree-designer graph neighbors` with the selected node [S1]. |
| Find a relationship between two known nodes | Run `tech-tree-designer graph path` [S1]. |
| Inspect an existing planned proto workspace | Run `tech-tree-designer plan tree` [S1]. |
| Check a planned proto contract | Run `tech-tree-designer plan validate` [S1]. Validation does not approve publication. |
| Inspect capability coverage | Run `tech-tree-designer ontology coverage` [S1]. Mappings are authored claims, not verified fulfillment. |
| Read development targets and diagnostic availability | Run `tech-tree-designer.setpoint-read` with `program-runtime library run tech-tree-designer.setpoint-read` [S3]. Interpret the envelope using the contract beside its source. |
| Need repository-wide draft, review or apply | Read `path:scenarios/tech-tree-designer/docs/START-HERE.md` for the current capability boundary [S0]. Do not substitute proto materialization. |
| Explicitly authorized to create or change a proto plan or ontology mapping | Read the selected operation in `tech-tree-designer plan help` or `ontology help` before mutation [S1]. |
| Explicitly authorized to publish a validated proto plan | Read `tech-tree-designer plan materialize --help` [S1]. This writes canonical proto sources and runs generation; it is not preview or generic apply. |

The board composes repeated owner reads. Individual operations remain CLI leaves;
their output contracts already own syntax, so no wrapper Action is added merely
to repeat help. Preserve source errors, freshness and partial coverage. Return the
selected operation, observed outcome, evidence references and unsupported claims.

### In-use settings

| Symptom | Setting move within the caller's authority | Record |
| --- | --- | --- |
| Whole graph is too large | Select a neighborhood with `graph neighbors` instead of exporting the fleet | Node, requested bounds and reported truncation |
| Need a different runtime | Use the explicit global `--instance` selector | Selected instance; do not assume isolation from its name |
| Runtime diagnostics are not needed | Run the board with `--input collect_diagnostics=false` | Inventory-only mode; no live measurement occurred |
| Source unavailable | Inspect `vrooli scenario status tech-tree-designer` | Failure and source context; no private host repair or automatic restart |

### Evidence and learning

Use the existing engagement's evidence record and shared Memory work-record path.
Follow `prompt-manager skill read vrooli-memory` for capture mechanics. This role
does not create a private learning scope. Reuse automatic capture when present.
Never copy draft contents, secrets or raw source payloads into general learning.

### Troubleshooting & Edge Cases

| Observation | First check and response |
| --- | --- |
| Board `ok` with unknown outcomes | Read required rows, not wrapper status. Inventory and diagnostics do not certify a product. |
| Board `partial` | Inspect affected diagnostic reasons and child status. Preserve healthy readings; do not infer an empty ecosystem. |
| Board `failed` | For `invalid_input`, correct the input; for `kernel_runtime`, inspect the recorded program and route its defect within authority. |
| Runtime ends without a JSON envelope | Retain the runtime failure and required outcome inventory as unresolved; an empty result is not an empty obligation set. |
| Owner refuses a write | Retain the refusal and request the missing grant. A sensor or valid proto does not grant it. |
| Materialize is offered for a documentation proposal | Stop that branch. The implemented operation is proto-only publication. |
| Apply acknowledgement is lost | Resolve owner receipts before retrying. Generic apply recovery is still a target; do not simulate it with direct Git writes. |

Promote recurring missing draft operations to the owning scenario; do not grow
this skill into an alternative workspace or recovery implementation.
