# Program Runtime kernel

The kernel is a supervised, one-process-per-session sidecar. `host/engine.py`
implements the stable JSON-lines protocol and bounded result handles using only
the Python standard library.

## Runtime environment contract

The Go supervisor provisions the kernel environment through the shared
`packages/pyenv-go` package. On first use it asks `uv` for CPython 3.12,
creates or repairs a private virtualenv under `SCENARIO_DATA_DIR`, and syncs the
committed `internal/pydeps/requirements.lock`. The supervisor then starts the
kernel with that interpreter's absolute path.

The lock is intentionally standard-library-only. Domain data access and
shaping are provided by governed bindings and `Handle` operations; the kernel
does not silently acquire a third-party package surface. The lock remains a
materialized, hashed input even when it contains no package requirements.

The process receives only the explicitly allowlisted baseline environment
(`PATH`, `HOME`, and `LANG`) plus supervisor-owned binding variables. It is
started with Python isolated mode (`-I`), so host `PYTHONPATH` and user-site
configuration cannot alter imports. If `uv`, CPython 3.12, the lock, or the
virtualenv cannot be provisioned, the kernel fails closed with an actionable
`vrooli host install uv` remediation; it never falls back to host `python3`.

This is a local supervisor boundary, not adversarial isolation. Programs are
trusted local agent workloads for this scenario; a future untrusted or
multi-tenant deployment must add a stronger workspace/container/VM boundary.

Run the unit tests with `python3 -m pytest kernel/tests` when pytest is present.

The public namespace is `vrooli.<scenario>.<group>.<command>` with hyphens
normalized to underscores. `vrooli.ai.classify`, `extract`, and `judge` are
typed facades over the governed ai-gateway inference binding. Top-level
`await` is supported, but every binding call executes eagerly and returns an
awaitable `Handle`, so bare and awaited calls use one convention. Use
`vrooli.gather` with zero-argument callables for independent parallel fan-out.
Default output is capped; `Handle.materialize(limit)` is the explicit bounded
escape hatch for row data. Keep shaping inside the kernel with `filter`,
`map`, `select`, `sort`, `unique`, `agg`, `join`, `group_by`, and slicing;
these return bounded `Handle` objects except for the bounded scalar returned
by `agg`. Missing keys identify the operation, requested key, and available
fields. Joins are capped at 100 million row comparisons to prevent accidental
unbounded memory work.
