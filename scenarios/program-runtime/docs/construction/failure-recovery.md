# Failure recovery

Submission performs static name preflight. An unresolved root returns a
diagnostic with the offending name and nearest match before the kernel runs:

```python
explanation = test_geni.runs.list()  # correct spelling is test_genie
```

Binding failures are retained with a closed `failure_cause` vocabulary such as
`unknown_field`, `ambiguous_response`, `unreachable_scenario`,
`refused_no_grant`, `deadline_exceeded`, and `bridge_transport`. Fix the named
cause, then submit a new program; do not retry a refused destructive call
without the required grant.
