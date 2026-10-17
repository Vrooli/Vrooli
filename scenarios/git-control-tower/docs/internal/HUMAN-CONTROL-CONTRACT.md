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

Commit proposals split drafting from actuation. Agents may anchor, create,
edit, combine, refresh and withdraw proposals; those writes reach GCT's database only.
`ProposalService.ApplyProposal` is the human writer. Its intent operation is
`repo.apply_proposal` with subject context `proposal:<id>@<revision>`, plus
`;prior-work=acknowledged` when uncertain prior work is explicitly accepted.
The digest binds destination HEAD/branch, proposal base, exact blobs/modes,
rendered message and index. Strict mode also binds live contents. Opt-in
`keep_later_changes` binds `;working-files=preserved` after any acknowledgment
suffix; it commits saved bytes/modes and leaves newer working edits untouched.
The UI uses this saved-version mode. Caller agent headers remove authority
even when local authentication maps to the operator. Under the repository lock,
apply consumes the intent and refuses strict-mode working drift, changed branch, affected base moves,
foreign staged files, unavailable captures or missing acknowledgment. Saved-version
mode also refuses different staged proposal versions before changing the index. It stages
only reviewed paths, checks and commits a private candidate, verifies exact tree
and message before publication, then revalidates source/revision under Git and
proposal-store leases. Pre-publication failure restores the index. A published
commit is never concealed by a later recording error or reset of the index.
Closing/excluding/grouping drafts never removes operator code or changes index.

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
