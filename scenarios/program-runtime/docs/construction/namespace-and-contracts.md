# Namespace and contracts

Read the contract before calling an unfamiliar binding, then call it by its flat
top-level name:

```python
contract = describe("search-hub/query/query")
print(contract.head(10))

rows = search_hub.query.query(query="program runtime", rows="ranked")
print(rows.count())
```

`describe` resolves through the live registry, so it is the same contract that
will accept or refuse the call.

Do not prefix a scenario binding with `vrooli.` — that name is the project
control plane, permanently:

```python
vrooli.scenario.status(name="program-runtime")
```

If a local variable shadows a scenario name, submission warns and the binding
stays reachable through the stable root:

```python
search_hub = ["a local list that shadows the binding"]
rows = __vrooli__.search_hub.query.query(query="retention", rows="ranked")
print(rows.count())
```

Runtime-owned names cannot be shadowed at all. Assigning `vrooli`, `discover`,
`validate`, or any other verb is refused before the program runs.
