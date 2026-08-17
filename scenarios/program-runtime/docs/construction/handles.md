# Handle shaping

Handles keep row data bounded and keep response metadata separate:

```python
rows = search_hub.query.query(query="runtime")
summary = rows.select("id", "score").sort("score", reverse=True).head(10)
print(summary)
print(rows.meta())
```

Use `filter`, `map`, `unique`, `group_by`, `join`, and `agg` before calling
`materialize`. Materialize only the small slice the agent needs.
