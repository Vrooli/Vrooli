# Example doc

The marketing team writes scans under `audience-scan/<date>/<slug>`. This
prefix is declared by the marketing-crew researcher member in its
topics.json output, so the docs-domain join (any-team-anywhere union) is
satisfied.

A pedagogical CLI invocation inside a fenced code block is excluded from
the scan because this file lives under the agent-system docs domain:

```bash
# Inside the agent-system docs domain, fenced blocks are pedagogical examples.
prompt-manager team knowledge-add example --topic="not-declared-anywhere/example"
```
