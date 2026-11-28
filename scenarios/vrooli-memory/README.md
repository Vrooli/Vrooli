# Vrooli Memory

Capture observations, retrieve related context, and inspect the original evidence.
Source Ledger owns the append-only journal and derived recall indexes. Vrooli
Memory provides the shared agent CLI and API. Search Hub queries the same
Source Ledger recall provider.

## Shared observations

```sh
vrooli-memory remember --evidence "run-239" "Browser capture times out on remote sessions"
vrooli-memory remember "Voice cloning is missing from the editor" "Local browser capture works after the fix"
vrooli-memory recall "browser capture setup"
vrooli-memory recall --category "bug reports" "remote browser capture" --limit 5
vrooli-memory recall --category "capability gaps"
vrooli-memory recall zoom --scope agent-memory <node-id>
```

Give each observation its own argument. Options apply to the whole batch, so use
separate calls when observations have different evidence. Record corrections as
new observations and name the conditions you verified. Exact retries can reuse
a request key and the same payload.

### Read the results

- Start with a specific topic, scenario, symptom, or environment. Use
  `--category` when you need a category filter. A category-only query lists
  results from that category. The default scope is `agent-memory`.
- Recall ranks summaries and observations. In JSON output, `score` helps explain
  result order within that query. Use the printed `recall zoom` command to inspect
  the underlying observations and any recorded evidence before acting. Follow
  constituent summaries as needed. For a JSON result, pass the hit's `node_id`
  and `scope`.
- A lexical fallback still returns text matches when semantic ranking is
  unavailable. A pending category index can leave related observations out of
  filtered results. In either case, broaden or rephrase the query before
  concluding that evidence is missing. Check `degradations` in JSON output.
- Search Hub can query a specific memory scope with `--group source-ledger
  --scope "scenario:<memory-scope>"`. Its memory results include the scope and
  node ID for zoom. Use the capability registry to check what an agent can use
  now.

Scenario QA uses recall and source evidence to investigate defects. Its case
accounts record current findings by environment or condition. A verified local
fix can coexist with an open remote failure. See the [Scenario QA operating
guide](../../docs/scenario-qa/README.md).

## Learning capture contract

Program Runtime records an immutable attempt tree for a learning task.
The root owns task identity. Child steps retain inputs, outcomes, and parent
order. Delivery retries reuse the frozen finish intent.

Structured notes use the vocabulary in
[`note-kinds.json`](.vrooli/program-runtime/note-kinds.json). See the
[Program Runtime learning contract](../program-runtime/docs/guides/program-contracts.md)
for attempts, recall, choice, notes, and outcome comparisons.

## Running the scenario

Use `make setup` and `make start` in this directory. Run `make test` for the
scenario suite.

| Need | Start with |
|---|---|
| Install and configure | [Quick start](docs/QUICKSTART.md) |
| Understand the service boundary | [Architecture](docs/concepts/ARCHITECTURE.md) |
| Work with scopes, facets, and learning | [Vrooli Memory skill](skills/vrooli-memory/SKILL.md) |
| Look up interfaces | [CLI commands](docs/reference/cli-commands.md) and [API endpoints](docs/reference/api-endpoints.md) |
| Run and troubleshoot | [Runbook](docs/operations/RUNBOOK.md) and [Observability](docs/operations/OBSERVABILITY.md) |
