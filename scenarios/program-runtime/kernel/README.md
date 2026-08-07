# Program Runtime kernel

The kernel is a supervised, one-process-per-session sidecar. `host/engine.py`
implements the stable JSON-lines protocol and bounded result handles using only
the Python standard library. The optional IPython adapter must preserve that
protocol and is intentionally not enabled when IPython is unavailable.

Run the unit tests with `python3 -m pytest kernel/tests` when pytest is present.

The public namespace is `vrooli.<scenario>.<group>.<command>` with hyphens
normalized to underscores. `vrooli.ai.classify`, `extract`, and `judge` are
typed facades over the governed ai-gateway inference binding. Top-level
`await` is supported, and independent calls may be combined with
`asyncio.gather`. Default output is capped; `Handle.materialize(limit)` is the
explicit bounded escape hatch for row data.
