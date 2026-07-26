# Replay corpus

These files are redacted copies of the sanitized provider-native captures in the
parent directory. They are the deterministic replay corpus: every file is
parsed without process launch or network access by `corpus_test.go`.

Each committed provider dialect is an expectation fixture. The replay test
asserts its recovered session id, message count, tool-event count, cost metric
count, and successful terminal state. It also replays the recorded session
identity through a fresh parser (the continuation boundary) and checks the
provider-native failure terminal shape for every supported runner. Those tests
are deliberately parser-only: they neither launch a CLI nor open a network
connection.

The refresh command is:

```sh
go run ./cmd/harvest-replay --root ~/.vrooli/data/vrooli/agent-manager/runs \
  --out internal/adapters/runner/codecs/testdata/corpus --max-per-runner 25
```

The committed corpus intentionally retains only representative traces from each
available provider dialect. The harvester redacts credential values and absolute
home paths before writing replacements.

After harvesting, inspect every candidate before committing it, retain at least
one representative success/resume dialect per supported runner, and update the
exact expectations in `corpus_test.go`. Do not use a live paid-agent run to
debug a parser when a corpus fixture can reproduce the issue.
