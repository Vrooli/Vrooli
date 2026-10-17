# Inference and discovery

Inference, discovery, and recall are governed runtime surfaces, not arbitrary
network access. Each fails closed and names its unavailable dependency rather
than falling back to a shell call or a direct provider call.

```python
verdict = discover("the capability that lists live sessions")
row = verdict.head(1)[0]
if row.get("null_verdict"):
    print({"stop": True, "reason": row["reason"]})
else:
    print(row["binding_id"])
```

A null verdict is an answer: stop and clarify the intent, or read the contract
with `describe(...)`. Do not guess a binding path from natural language.

Bounded typed inference is schema-validated locally, whatever the provider
returns:

```python
schema = {"type": "object", "properties": {"label": {"type": "string"}}, "required": ["label"]}
result = ai.classify("The build failed after a proto field was renamed.", schema,
                     instruction="Return one label: build, docs, or infra.")
print(result.head(1))
```

`ai.classify`, `ai.extract`, and `ai.judge` map to the `classify.fast`,
`extract.structured`, and `judge.default` roles and are deterministic: each
refuses a caller-supplied `temperature`. `ai.write` maps to the overridable
`write.default` role and accepts `temperature=` and `max_output_tokens=`.
`ai.batch(sources, schema)` classifies a small corpus in one governed call and
preserves input order.

For convenience, `ai.classify(source=["one", "two"], labels=["bug", "feature"])`
uses that same governed batch route and returns one bounded row per input,
including the input text. `recall` accepts an intent and optional depth only;
use `describe` or `discover` when you already have a binding id.

`recall("intent")` returns governed records and docs through search-hub;
`recall("intent", depth="deep")` widens the result set. Inference and delegation
both draw on the session's spend ceilings, so a long fan-out can be stopped by
`inference_spend_exceeded` rather than running unbounded.

## Read a program before you run one

Discovery, `recall`, and `search-hub` can surface an existing *program* — not just
a binding — that looks like a fit for your intent. Before you run a program you did
not author, read it first: the contract is the caller boundary. `program-runtime
library get <scenario>.<program> --json` returns the program's contract and source.
Confirm its `purpose` matches what you actually want, its `bindings`/`verbs` and
effects are what you expect (a read-only `gather` program never mutates; a program
that writes says so), its `inputs` are the ones you mean to pass, and its
`outputs`/`output_schema` give a signal shape you can consume. A program whose
purpose or effects do not match your intent is the wrong tool even if its name looks
right — stop and pick another, the same way a null discovery verdict means stop
rather than guess. This is the program-level form of the binding rule in
[`namespace-and-contracts.md`](namespace-and-contracts.md): read the contract before
you call something unfamiliar.
