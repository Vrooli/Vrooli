# Precision corpus

Labelled windows of real agent-manager run events, used by `precision_test.go`
to compute per-pattern precision and recall (SPEC 11.2, "Precision gate").

## Files

- `corpus.json`: 269 windows. Each has `pattern`, `label` (`true`, `false`,
  `ambiguous`, `clean`), a one-line `reason`, `runId`, the old episode's
  `startEventId`/`endEventId`, `goalRun` (the run had a goal_id) and the reduced
  `events`, and optionally `episodes`: further labelled spans in the same window, each
  `{pattern, start_event, end_event, label (true|false|ambiguous), reason, labeler}`.
- `results.json`: golden per-pattern results. Regenerate with
  `go test ./internal/runsignal -run Precision -update` after reviewing the logged table.

## Provenance

- Source: the live `scenarios/agent-manager/data/agent-manager.db`, opened read-only,
  taken 2026-10-29. Episodes are rows of `invocation_read_model_episodes` written by
  the pre-pass-3 detectors (classifier `friction-episode.v13`).
- Selection: for each pattern, a seeded random sample (up to 14 episodes, one per run;
  up to eight per run for edit-revert and oscillation, which occur in few runs), skipping
  episodes longer than 100 events (700 for edit-revert/oscillation). The audit's known
  true and false examples were forced in. 18 `clean` windows are random 30-event stretches
  of runs where no old episode lies within 10 events.
- Window: from 5 events before the episode start to 5 events after its end, counting only
  message, tool_call, tool_result, status, goal_status_changed and error events, capped at
  120. Two very long read-then-reread episodes keep 50 events from each end.
- Targeted sampling (second pass): for patterns with no true labels, a heuristic scan of about
  1,500 random runs proposed plausible true cases (operator messages containing "no,", "instead",
  "wrong"; the same file read twice within 5 calls with no edit; Edit calls whose old and new strings
  are swapped; three or more differing failing calls of one command path; identical calls seconds
  apart; a vrooli CLI call followed by curl/sqlite3/python against the same service). Each candidate
  was read and labelled; most were false. False and ambiguous windows were capped per pattern
  (12 and 3) to stay under 3 MB; true windows and the audit's known examples were always kept.
- Reduction: exit evidence is preserved. A tool result keeps `exitCode`, and any "Process exited
  with code N", "Exit code: N", `exit_code` or `is_error` text found beyond the 300-character head is
  appended to `output` as a bracketed line.
- Reduction: message text 400 characters; tool input strings 500; tool output, error and
  output blocks 300; at most three output blocks. Emails, API keys, bearer tokens, JWTs,
  Fernet tokens, `password=`/`token=` values and opaque runs of 44 or more characters are
  replaced with `<redacted>` or `<redacted:HASH6>` (same input, same placeholder, so equality
  between calls survives). Total size must stay under 3 MiB.
- Built with scratch scripts (sample, read each window, label, reduce) that are not kept in
  source; the steps above reproduce them.

## Label rules

- `command-failure` is scenario friction: `true` when a Vrooli project-owned command (scenario CLI,
  `vrooli ...`) fails with an error, whatever the cause (agent misuse, or the scenario or its
  infrastructure broken or unavailable: capacity broker down, read-only filesystem, unreachable API);
  `false` for a non-project command (`ls`, `rg`, `du`, `python`, `git`), a no-match, an interrupt or a
  documented informational verdict. Labels changed under this rule carry `relabel:
  "scenario-friction-2026-10-30"` and `oldLabel`. The blind `command-failure` labels were re-judged
  the same way, so they no longer follow the "Rules used" sentence below for command-failure.

- `true`: a real, avoidable agent friction of that pattern.
- `false`: not that friction. Examples: handoff-continuation in a goal run; a fallback whose
  start and end are one call; guidance-repair on a task-notification, teammate, goal or wake
  message; a stall that is an idle gap between turns; a reread after a script SyntaxError.
- `ambiguous`: excluded from precision. Use it when the window cannot decide.
- `clean`: ordinary work. Any detection in it counts as a false positive.
- Detections are scored against every label of their window (the window label and its `episodes`):
  - overlapping a true or false span of the same pattern (and not over 3x wider): found, counted as a
    detected-true or detected-false;
  - overlapping an ambiguous span: ignored;
  - in a `clean` window, or overlapping a span labelled false: a false positive (`falsePositives`);
  - anything else: `unlabelled`. Not known false, just not judged yet.
- The gate fails a pattern whose unlabelled detections exceed 20% of its detections, which forces
  labelling. It also requires all-label recall >= 0.8 once a pattern has 3 true labels. A shown pattern also needs at least 5 matched true detections and a 95% Wilson lower bound of all-label precision of at least 0.7 (`wilsonLower` in results.json), so a small sample cannot pass on a perfect ratio.

## Blind labels

`episodes` entries with `labeler: "blind-2026-10-30"` were written after the detector rules were
fixed and without reading detector source (`*_detectors.go`, `detector_helpers.go`, `patterns.go`):
the labeller had only the pattern definitions (SPEC 11.1 P3-F4, USAGE-INTELLIGENCE.md) and the
window events. They cover detections that earlier had no label, so they are out-of-sample: no rule
was tuned on them. `results.json` reports them as the `blind` split per pattern, and the gate's
precision includes them.

- Scope: the shown-candidate patterns command-failure, retry-after-failure, abandoned-command-family,
  guidance-repair, repeated-work and read-then-reread. wait-misuse and edit-revert had no unlabelled
  detections.
- Sampling: retry-after-failure (63 unlabelled), abandoned-command-family (59) and repeated-work (46) had
  more than 40; a deterministic sample of 40 (ordered by FNV-32a of `pattern|window|start|end`) was
  labelled. The rest stay `unlabelled`. Regenerate the dump with
  `PRECISION_DUMP=<file> PRECISION_DUMP_PATTERNS=<comma list> go test ./internal/runsignal -run DumpUnlabelled`.
- Rules used: command-failure was true for avoidable CLI misuse and false for environment faults (superseded by
  the scenario-friction rule above); retry-after-failure is true for repeated retries of the same misuse and false for a
  single fix-and-rebuild cycle, informational non-zero exits (`effort lint` exit 4) or an outage;
  abandoned-command-family is true when the agent gave up the command family after avoidable
  failures; repeated-work and read-then-reread are false when the two calls are one call recorded
  twice (identical, consecutive, no result between) or the content changed between reads.

- Second blind pass (`labeler: "blind-2026-10-30b"`, command-failure only): the 53 detections still unlabelled
  under the scenario-friction rule were labelled without reading detector source: 32 true (a project CLI
  failed: usage or flag errors, 404/403/409 API errors, capacity-broker or read-only-filesystem build
  admission failures), 21 false (20 `agent-manager effort lint` exit 4 is a documented informational verdict;
  one jq failure; 0 ambiguous). `precision_test.go` matches the blind split on the exact string `blind-2026-10-30`, so the `b`
  labels count in the all/tune/hold-out columns but not in the `blind` column. After the pass command-failure
  has 0 unlabelled, all-label precision 0.74 and recall 0.97 (hold-out 0.72/0.93), still below the 0.8
  floor, so it stays quarantined. The false labels are one detector behaviour (exit 4 of `effort lint`).

## Tune and hold-out splits

Detector rules (regexes, exit-code handling, quarantine reasons) were developed by reading
this corpus, so every window is "tune" data in the sense that the rules were fitted to it.
To keep the gate honest the test also splits windows deterministically: a window whose id
hashes (FNV-32a) into the lowest 30% is **hold-out**, the rest is **tune**. Rule development
must look only at tune windows; do not read or label-fit against hold-out windows. Precision
and recall are reported for all, tune, hold-out and blind labels in the logged table and in
`results.json`. A pattern with at least 5 hold-out labels is gated on its hold-out precision
and recall (and its all-label precision must also clear the floor, so a small hold-out cannot
rescue a pattern); a pattern with fewer is gated on all labels and marked `in-sample`. The
existing rules predate this split, so their first hold-out numbers are an after-the-fact
check, not a clean out-of-sample estimate; new corpus windows are the real hold-out.

## How the test scores

A detection counts for a labelled episode when it has the same pattern and overlaps the
labelled episode's events. Per pattern: precision = labelled-true episodes detected / (those
+ labelled-false episodes detected + false positives). Recall = labelled-true
windows detected / labelled-true windows. A pattern with fewer than 10 true+false labels,
precision under 0.8, or (with at least 3 true labels) recall under 0.8 must be listed in
`precisionQuarantined` in the test.

## Limits

- Detectors see only events. `goalRun` is corpus metadata the production path does not give
  them, so "never fires in goal runs" can only pass if the detector derives goal mode from
  events.
- The sample is small and old-detector-biased: true labels exist only where the old
  detectors fired, so recall measures regressions, not unseen misses.

## Extending

Add windows by sampling more episodes (or new-detector detections) the same way, label them,
append to `corpus.json`, keep ids unique, rerun with `-update`, and move a pattern out of
`precisionQuarantined` only when the logged table shows it passes.
