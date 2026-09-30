# Git Control Tower effect and authority matrix

This is the current transport inventory for the advisory maturity work. A
request header is never evidence of identity or human intent.

| Surface | Examples | Effect | Current boundary |
|---|---|---|---|
| Read REST | fallback repository resolution and remaining migration reads | pure read | allowed only during migration; typed RepoService owns status, diff, history, approved changes, and provenance |
| Advisory REST | mention webhook admission only | durable advisory queue write | `webhook_receiver` exception; typed drafts use `AdvisoryService.Draft` and have no repository writer authority |
| Auditor fix Connect | `AuditorService.PreviewFix` / `ApplyFix` | advisory preview / repository mutation | preview is advisory; apply requires verified human authority and an exact single-use intent bound to the selected scenarios and rules |
| Remaining repository REST | review/audit-adjacent writes and other non-typed domains | repository mutation/external write | each independent writer is migration residue pending a typed contract; typed Connect-RPC is the required surface for repository registry, credentials, remote URL, SSH keys, file operations, discard, ignore, push, pull, precommit, stage, unstage, commit, and branch |
| Repository registry Connect | list, active, open, clone, remove | scenario-owned registry and possible repository mutation | typed handler boundary requires a verified human principal; read resolution cannot create or activate a record |
| Connect read | RepoService reads, worktree list/get | pure read | policy interceptor bypasses mutation gate |
| Connect worktree mutation | create/remove/lock/unlock/move/prune | repository mutation | policy interceptor denies absent verified principal; caller headers ignored |
| Test-isolation routing | leased database and file-root selection | test-only persistence/file writes | `database.RoutedDB`, `filerouting.RoutedRoots`, and `apihttp.TestModeMiddleware` select the lease; production requests remain on primary roots |
| SSH/configuration services | typed key generation/deletion and precommit settings | host/filesystem mutation | RepoService writers require a verified human principal and exact operation intent; remaining configuration writers are inventoried separately |
| CLI | local adapters and Connect bindings | depends on command | commit obtains authority status, exact preview, and a single-use intent; client headers are attribution only |

## Principal contract

The authentication owner is `scenario-authenticator`. GCT's relying-party
middleware verifies its RS256 JWT locally against discovered JWKS, checks
issuer `scenario-authenticator`, realm audience (default
`scenario-authenticator:default`), expiry, algorithm, key id, and subject, and
attaches `policygate.Principal`. `X-Vrooli-Caller`, `X-Vrooli-Authorized`, and
similar caller headers never create that principal. An optional
`GCT_AUTH_VALIDATE_URL` enables the authenticator's live `Validate` RPC for
immediate logout/password-change revocation; without it, expiry is the
revocation boundary.

For a human writer, `Verified`, `Subject`, and repository binding are required.
Commit additionally requires a server-issued `HumanIntent` matching the exact
repository, operation, expected revision, subject digest, expiry, and unused
intent ID. A missing, expired, replayed, or mismatched intent is refused before
the Git writer. Agents may inspect and prepare evidence but cannot issue human
intents or invoke repository writers.

The canonical human flow is the generated Connect-RPC
`HumanControlService.GetAuthorityStatus`,
`HumanControlService.PrepareMutation`, exact operator confirmation,
`HumanControlService.ConfirmMutation`, then the typed mutation request. The UI
and CLI both use these generated clients; there is no parallel REST authority
surface. Read-only routes remain usable when authentication is missing or
unavailable.

## Writer inventory and proof obligations

| Writer family | Owner/effect | Principal requirement | Intent requirement | Focused proof |
|---|---|---|---|---|
| Stage / unstage / discard / ignore | GCT typed RepoService with exact intent | verified human at Connect interceptor and service seam | single-use intent bound to the repository subject | policygate interceptor, staging/discard/service tests |
| Commit | `RepoService.CreateCommit` Connect-RPC service; durable Git history mutation | verified human | exact single-use `repo.commit` intent consumed before `CreateCommit` | `intent*_test.go`, Connect commit handler tests |
| Push / pull / upstream | GCT typed RepoService; external or ref mutation | verified human at Connect interceptor and service seam | exact single-use intent | push service tests and Connect authority boundary |
| Repository file writers | `RepoService.DeletePath` / `SaveFileContent` Connect-RPC; filesystem mutation | verified human at interceptor and service seam | exact single-use intent bound to repository revision and subject digest | file service tests plus Connect handler/policy tests |
| Repository settings/remediation writers | `RepoService.SaveGroupingRules`, `MoveGitignoreEntry`, `UntrackBinary` Connect-RPC; config/index/filesystem mutation | verified human at interceptor and service seam | exact single-use intent bound to repository revision and subject digest | grouping, gitignore, tracked-binary service tests plus Connect policy tests |
| Config/credential/SSH writers | GCT RepoService typed domain adapters; filesystem/host mutation | verified human at Connect interceptor and domain service seam | exact single-use intent for credential, remote URL, and SSH key writers; no agent path | direct `requireHumanMutation` tests and typed refusal/live probes |
| Worktree Connect mutations | GCT WorktreeService; repository mutation | verified principal in Connect context | `DecideAuthenticated` requires exact intent; absent/mismatched intent denies | policygate interceptor tests |
| HumanControl Connect/REST | GCT human-control adapter; authorization preparation | verified human for confirmation | issues durable intent only; it never writes Git | human-control handler tests |

All repository mutations use the same verified-human, exact-preview, short-lived,
single-use intent contract. A future provider-backed reauthentication flow can
be added as a separate policy layer if the deployment develops that requirement;
the current contract does not pretend that a caller-supplied boolean is stronger
authentication.
