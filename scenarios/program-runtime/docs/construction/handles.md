# Handle shaping

A `Handle` keeps rows in the kernel and keeps response metadata separate, so the
cost of an answer is the size of the answer rather than the size of the data it
came from.

```python
rows = search_hub.query.query(query="runtime", rows="ranked")
summary = rows.select("id", "score").sort("score", reverse=True).head(10)
print(summary)
print(rows.meta())
```

`rows="ranked"` is required here: this response carries four repeated fields, so
the projection is ambiguous and fails closed without it. Read the candidates
from the error, or from `describe("search-hub/query/query")`.

Shape before you materialize. Use `filter`, `map`, `select`, `sort`, `unique`,
`group_by`, `join`, and `agg` on the handle; call `materialize(limit)` only when
the rows themselves are the answer, and always pass a limit.

Joining two handles keeps both sides in the kernel:

```python
runs = test_genie.runs.list(scenario="program-runtime", limit=20)
print(runs.group_by("status"))
```

`meta()` returns the non-row response fields — latency, routing, totals.
`raw()` returns the decoded response for diagnostics. Neither escapes the
program's output bound.
