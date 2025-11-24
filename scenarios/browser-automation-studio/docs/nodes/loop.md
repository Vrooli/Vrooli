# Loop Node

`[REQ:BAS-NODE-LOOP]` enables data-driven automation by executing a sub-graph multiple times. Loop bodies are defined by connecting nodes to the Loop node’s `loop_body` handle; special `loop_continue` and `loop_break` handles allow advanced control without extra conditionals.

## Loop Types & Fields

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Loop type** | `forEach`, `repeat`, or `while` | Yes | Determines which additional fields are used. |
| **Array source** | Workflow variable containing an array | For `forEach` | e.g., `dataset.users`. Each iteration exposes `{{loop.item}}`. |
| **Repeat count** | Positive integer | For `repeat` | Number of iterations; also sets the max-iteration clamp when larger than the default. |
| **Condition type** | `variable` or `expression` | For `while` | Defaults to `variable` if omitted. |
| **Condition variable** | Workflow variable to evaluate | For `while` + `variable` | Supports the same operators as Conditional nodes. |
| **Condition expression** | JavaScript returning truthy/falsey | For `while` + `expression` | Runs inside the current frame. |
| **Condition operator/value** | Comparison metadata | Optional | Used for `while` + `variable`. |
| **Item variable** | Variable name that holds the current array entry | Optional | Defaults to `loop.item`. |
| **Index variable** | Variable name for the zero-based index | Optional | Defaults to `loop.index`. |
| **Max iterations** | Safety guard against infinite loops | Optional | Clamped between 1 and 5 000 (default 100). |
| **Iteration timeout (ms)** | Per-iteration wall-clock limit | Optional | Clamped between 250 ms and 600 000 ms (default 45 000 ms). |
| **Total timeout (ms)** | Whole-loop limit | Optional | Clamped between 1 000 ms and 1 800 000 ms (default 300 000 ms). |

## Runtime Behavior

1. `api/automation/executor/plan_builder.go` compiles the loop body into its own `PlanGraph` and wires the special break/continue edges.
2. `api/browserless/runtime/instructions.go` validates the configuration, clamps limits, and records metadata (`loopArraySource`, `loopCount`, timeouts, iteration variables) in the instruction payload.
3. `api/automation/executor/flow_executor.go` drives the loop:
   - Maintains executor-scoped variables per iteration so `loop.item`/`loop.index` stay consistent.
   - Executes the nested graph sequentially, honoring `loop_continue` and `loop_break` handles without leaving orphaned nodes.
   - Enforces per-iteration and total timeouts, aborting with descriptive errors when the workflow risks hanging.
4. Loop telemetry (iteration counters, break reasons) is stored in execution artifacts via `DBRecorder` so you can debug dataset-driven runs later.

## Examples

### For-each over extracted rows

```json
{
  "type": "loop",
  "data": {
    "loopType": "forEach",
    "arraySource": "table.rows",
    "itemVariable": "row",
    "indexVariable": "rowIndex",
    "maxIterations": 250,
    "iterationTimeoutMs": 20000
  }
}
```

Populate `table.rows` with a Set Variable node (e.g., via Extract → storeIn) before this loop. Inside the body you can reference `{{row.name}}` and `{{rowIndex}}`.

### Repeat N times with early exit

```json
{
  "type": "loop",
  "data": {
    "loopType": "repeat",
    "count": 5,
    "totalTimeoutMs": 120000
  }
}
```

Attach the body to `loop_body` and drive `loop_break` from an Assert or Conditional node whenever the repeat no longer needs to continue.

### While variable matches condition

```json
{
  "type": "loop",
  "data": {
    "loopType": "while",
    "conditionType": "variable",
    "conditionVariable": "pagination.hasNext",
    "conditionOperator": "equals",
    "conditionValue": true,
    "maxIterations": 50
  }
}
```

Use upstream nodes to update `pagination.hasNext` at the end of each iteration (e.g., Extract button disabled state) so the loop exits cleanly.

## Tips & Limitations

- Always set realistic `maxIterations` when scraping unknown datasets; the runtime hard-caps at 5 000 iterations.
- Large datasets benefit from reducing `iterationTimeoutMs` so stuck elements fail fast.
- Nested loops are supported but keep overall timeouts in mind to avoid cascading cancellations.
- Combine with Conditional nodes inside the body to branch per record instead of duplicating workflows.

## Related Nodes

- **Conditional** – Use inside loops for per-item branching.
- **Set/Use Variable** – Prepare the arrays/flags consumed by Loop nodes.
- **Workflow Call** – Loop through datasets and call smaller workflows for each record.
