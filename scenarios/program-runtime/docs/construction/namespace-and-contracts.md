# Namespace and contracts

```python
contract = describe("search-hub/query/query")
rows = search_hub.query.query(query="program runtime")
print(rows.head(5))
```

Do not prefix scenario bindings with `vrooli.`. That name is reserved for the
project CLI. If a scenario name is shadowed by a local variable, the binding
remains available through `__vrooli__.scenario_name`.
