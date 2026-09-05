# Delegation

Bounded typed inference and a delegated agent run are different cost tiers. Use
`ai.*` for short, schema-constrained work. Use `agent.*` only when the work is
genuinely agentic and unbounded.

Delegation is explicit and returns a Handle. Start and collect are separate, so
several runs proceed concurrently instead of serializing on one wait:

```python
request = {
    "owner": "development-toolchain-validator",
    "workflow_key": "development-toolchain-validator/skill-experiment-audit",
    "input": {
        "experiment": {
            "name": "delegation-pattern",
            "objective": "Return structured evidence from a bounded audit.",
        },
        "assignments": [{"id": "sample", "token": "delegated runtime smoke"}],
    },
}

started = agent.start(**request)
result = agent.collect(started, wait_seconds=30)
print(result.head(1))
```

`workflow_key` must name a workflow the owning scenario actually declares; an
unknown key returns a typed not-found failure rather than a silent no-op.

The Go bridge validates the session, applies the session's delegated-run spend
ceiling, and owns waiting. Do not implement a polling loop inside a program.
`agent.run(**request)` is the convenience form that starts and collects in one
call when concurrency does not matter.
