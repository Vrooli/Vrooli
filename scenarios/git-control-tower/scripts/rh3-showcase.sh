#!/usr/bin/env bash
set -euo pipefail

scenario_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# API cases create and commit only inside t.TempDir repositories. The UI case
# exercises the same start/attach contract with controlled responses, so this
# showcase never targets the invoking checkout's Git index.
(
  cd "$scenario_dir/api"
  go test ./... -run 'Test(CandidateCheckJoinsAndReusesDurableReceipt|ConcurrentExplicitRetriesJoinOneNewAttempt|FinalAdmissionRerunsWhenStageChangesDuringCheck|CandidateReceiptQualifiesRealTemporaryGitCommit)$' -count=1
)

(
  cd "$scenario_dir/ui"
  pnpm exec vitest run src/features/commit-message/CommitMessageComposer.test.tsx \
    -t 'drafts while an identity-bound staged check runs|retries a terminal check once|requests a fresh candidate check after the staged selection changes'
)
