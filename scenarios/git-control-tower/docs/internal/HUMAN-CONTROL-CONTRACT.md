# Human repository control contract

Read and draft operations may be advisory. Stage, commit, branch, worktree,
discard, merge, undo, push, pull, and publication operations require a
verified human principal, exact repository identity, reviewed subject digest,
expected revision, expiry, and single-use intent.

Merge preparation is a pure preview containing source and destination refs,
expected base, source head, conflicts, and subject digest. It does not merge
or checkout. Undo preparation requires a per-file current digest, exact
preimage digest, and `verified` attribution. Mixed, missing, or path-only
provenance is unsafe and is refused.

Recording fakes are the acceptance mechanism. A fake must observe that no
writer call occurs before verification and that replayed, expired, consumed,
or stale intents are rejected. Human completion is recorded separately from
the system's preparation receipt.

## Live operator evidence

For a separately authorized human run, capture the following without exposing
the access token: sign in through scenario-authenticator, open GCT, record the
authority status as human, review the exact preview, confirm once, and retain
the resulting commit hash plus the authority/intent audit receipt. Repeat the
same request with the consumed intent and record the refusal. A second run with
an agent identity or forged caller headers must show read-only access and no
writer receipt. Automated validation proves the refusal and durability rules;
it does not substitute for this human-produced actuation evidence.
