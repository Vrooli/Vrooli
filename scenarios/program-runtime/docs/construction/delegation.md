# Delegation

Delegated work is explicit and returns a Handle:

```python
started = agent.start(owner="agent-manager", workflow_key="workflow-key", input={"task": "..."})
result = agent.collect(started, wait_seconds=30)
print(result.head(1))
```

The Go bridge validates the session, applies spend ceilings, and owns waiting.
Do not implement polling loops inside a program.
