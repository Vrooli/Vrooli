# P2-W2 fixture deletion inventory (read-only, nothing deleted)

Generated 2026-10-29 from `agent-manager.db?mode=ro` and the live web-console state dir.

## Summary

| Item | Count | Detail |
|---|---|---|
| Fixture transcript files indexed by AM | 4,700 | all under `~/.vrooli/state/vrooli/web-console/sessions/codex/<wc-session>/sessions/...` |
| AM runs to delete | 4,700 | one run per file, none shares a file with a non-fixture transcript; all `status=complete`, `execution_mode=imported`, `tag=agent-manager-imported`, started 2026-10-29 07:05-08:37 |
| Fixture session dirs to delete | 4,700 | 118,824 KB (about 116 MB); each holds exactly one regular file (the fixture rollout); `skills`/`auth.json`/etc. inside are symlinks into `~/.codex`, which `rm -rf` does not follow |
| Real web-console codex dirs (kept) | 896 of 5,596 | their rollouts are `rollout-<timestamp>-<uuid>.jsonl` |

The earlier "3,633" figure was a title subset taken earlier; the full set grew to 4,700 (the same fixtures re-imported per test run).

### Criteria

A transcript is a fixture when it lies under the web-console codex base and its basename is one of
`rollout-test123.jsonl` (903), `rollout-e2e.jsonl` (905), `rollout-stale.jsonl` (903), `rollout-resume.jsonl` (902),
`rollout-backfill.jsonl` (902), `rollout-mid-session-test.jsonl` (185). Codex always writes
`rollout-<timestamp>-<thread uuid>.jsonl`; every one of the 4,700 lacks a uuid, and the equivalent query "web-console path whose basename has no uuid" returns the same 4,700.

Verification that no real session is included:
- every listed file exists and none contains `session_meta`, `agent_message`, `user_message` or `task_started` (real Codex rollouts always start with `session_meta`); contents are one to three hand-written `response_item` lines ("Hello from the tailer", "late reply", "Old/New assistant message", "Backfilled assistant message", "Hello from mid-session codex");
- no listed run has any transcript file outside the set (`runs_with_other_files = 0`); all 4,700 runs exist;
- every dir has `files=1` (the fixture), `real_rollouts=0`.

All 4,700 runs hang off one shared imported task, so `maintenance purge` (matches run **tag**, which is `agent-manager-imported` for every imported run, real ones included) is unsafe here. Do not use it.

## Lists

- `p2-fixture-runs.txt`: `run_id|transcript_path`, 4,700 lines.
- `p2-fixture-dirs.txt`: `session_dir<TAB>size_kb`, 4,700 lines.

## Governed deletion (needs operator grant)

Order: (1) orchestrator rebuilds and restarts agent-manager so the new indexer rule (below) is live; (2) delete runs; (3) delete dirs; (4) run an import sweep and confirm no fixtures return.

```
cd scenarios/agent-manager/docs/efforts/usage-intelligence   # from the repo root
# 2. runs: governed per-run delete (domain.CanDeleteRun allows complete runs; notifies conversation-search)
./p2-delete-fixture-runs.sh --delete   # checked; dry run without --delete
# 3. dirs: only the listed paths, never a glob
./p2-delete-fixture-dirs.sh --delete   # checked; dry run without --delete
# 4. verify
agent-manager run import-sweep
sqlite3 "file:scenarios/agent-manager/data/agent-manager.db?mode=ro" \
  "select count(*) from transcript_files where path like '%/web-console/sessions/codex/%' and (path like '%/rollout-test123.jsonl' or path like '%/rollout-e2e.jsonl' or path like '%/rollout-stale.jsonl' or path like '%/rollout-resume.jsonl' or path like '%/rollout-backfill.jsonl' or path like '%/rollout-mid-session-test.jsonl')"   # expect 0
```

If the grant is for `run delete` only, step 3 is a separate host-file deletion and needs its own grant.

## Prevention shipped in this task

- web-console tests: `newCodexTailerTestServer` (`scenarios/web-console/api/codex_tailer_test.go`) and `TestCodexTailer_AttributesMidSessionRolloutToCorrectSession` (`mid_session_conversation_test.go`) now call `useIsolatedSessionState(t)`; `t.Setenv("HOME", ...)` alone was ineffective because the state root resolves through `WC_SESSION_STATE_ROOT` / the storage resolver, not HOME.
- AM indexer: `isTestRollout` in `transcriptindex/discover.go` makes `Walk` skip web-console rollouts whose name carries no thread uuid; test `TestIndexerSkipsWebConsoleTestRollouts`.
