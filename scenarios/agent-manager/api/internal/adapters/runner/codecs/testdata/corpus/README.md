# Replay corpus

These files are redacted copies of the sanitized provider-native captures in the
parent directory. They are the deterministic replay corpus: every file is
parsed without process launch or network access by `corpus_test.go`.

The refresh command is:

```sh
go run ./cmd/harvest-replay --root ~/.vrooli/data/vrooli/agent-manager/runs \
  --out internal/adapters/runner/codecs/testdata/corpus --max-per-runner 25
```

The committed corpus intentionally retains only representative traces from each
available provider dialect. The harvester redacts credential values and absolute
home paths before writing replacements.

