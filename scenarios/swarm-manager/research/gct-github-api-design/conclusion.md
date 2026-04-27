# Research Conclusion: GitHub API Integration for Git Control Tower

## Research Question
How should Git Control Tower integrate with the GitHub API so that (a) every GitHub call is mockable behind a single Go interface, (b) credentials reuse the existing encrypted credential store, (c) tests make zero real GitHub API calls, and (d) review-panel results can be transformed into PR descriptions? Specifically: which Go library to use, which auth modes (PAT, GitHub App, SSH for transport vs. token for API), what PR metadata to surface, and how to wire all of this into the existing handler/service/runner pattern.

## Summary
Round 1 locked the four core design decisions and round 2 has narrowed the remaining open questions to four operational choices.

**Locked (round 1, all selected = A):**
- **Library:** `github.com/google/go-github/v66`, wrapped behind a narrow `GitHubClient` interface.
- **Auth (V1):** GitHub PAT only; defer GitHub App (installation tokens) to a follow-up.
- **PR description:** deterministic Markdown template fed by `ReviewDimensions`; LLM polish deferred.
- **Scope:** this single research item locks the shared client / auth / mock design for both `execute/gct-github-pr-api` and `execute/gct-github-release-api`; each execute child owns its endpoint shape.

**Round 2 open questions (await user input):**
- Where to put rate-limit / retry / pagination logic (above go-github, in transport, or not at all in V1).
- Whether GitHub Enterprise URL configurability ships in V1 (one optional `api_base_url` credential field) or defers.
- How the PR description template renders missing/stale review dimensions (omit, placeholder, or refuse).
- Whether to design `GitHubClient` as concrete-now or behind a `VCSProviderClient` super-interface for future GitLab/Bitbucket.

GCT already has the foundations needed: `CredentialTypeHTTPS` (token-based), an encrypted `~/.config/git-control-tower/credentials.enc` store, a hand-rolled-fake mock convention, structured `ReviewDimensions` data with a configurable readiness calculator, and an in-repo retry/backoff template (`agent_manager_client.go`). SSH stays the transport for git push/pull; the new token-based path is needed only for REST/GraphQL calls. The credential model needs one small additive field (`Service`) to let a GitHub API token live alongside remote-bound git HTTPS credentials.

## Methodology
- Read item spec, initiative neighborhood (`gct-github-integration` — 5 members, upstream `git-control-tower-ai-provenance`, downstreams `gct-merge-and-conflicts` and `gct-release-pipeline`), orchestration summary, and PRD at `scenarios/git-control-tower/PRD.md`.
- Inventoried existing primitives in `scenarios/git-control-tower/api/`:
  - Credential storage: `credentials_store.go`, `credentials_model.go`, `credentials_service.go`, `credential_resolve.go`.
  - Git execution seam: `git_runner.go` interface + `git_runner_fake_test.go` hand-rolled fake (~600 lines, error-injection fields).
  - Review data shape: `review_model.go` (`ReviewDimensions`, `CodeQualityDimension`, `TestsDimension`, `StandardsDimension`, `VisualDimension`, `ProvenanceDimension`).
  - Readiness calculator: `review_readiness.go` (`DefaultReadinessThresholds`, `CalculateReadiness`).
  - Routing & handler pattern: `routes.go`, `http_handler.go` (`RepoRead`, `RepoWrite`, `HandlerContext`).
  - Push/pull auth: `push_pull_service.go` + `git_runner.go` `gitCredentialEnv()`.
  - Retry / backoff precedent: `agent_manager_client.go` (`retryDelay`, `doWithRetry`, `waitForRetry`).
- Confirmed `scenarios/git-control-tower/api/go.mod` has no `go-github`/`go-gh` today (only `google/uuid`, `gorilla/mux`, `gorilla/handlers`, `vrooli/api-core`, `vrooli/repo-contract-go`, `golang.org/x/sync`, `modernc.org/sqlite`); a new dependency is required.

## Findings

### Finding 1: The credential store already supports GitHub tokens, with one additive schema delta
`CredentialTypeHTTPS` (`api/credentials_model.go:10`) carries `Username + Token`, persisted AES-256-GCM-encrypted at `~/.config/git-control-tower/credentials.enc` (`api/credentials_store.go:53,170-189`). A GitHub PAT is exactly an HTTPS token. **However**, today every credential is keyed by a git remote (`Remote`, `URL`); a single PAT is not bound to a single remote. **Implication:** add an optional `Service string` field to `Credential` and `StoredCredential` (e.g., `"github.com"`, `"github-enterprise.example.com"`); credentials with `Type=https && Service != ""` are treated as service-scoped API tokens, not remote-bound git HTTPS credentials. This is purely additive and does not affect the encrypted on-disk format beyond a new optional JSON field. SSH continues to handle git transport via `GIT_SSH_COMMAND` (`api/git_runner.go:185-243`); the GitHub API path stays independent and token-only.

### Finding 2: A `GitRunner`-style seam is the right model for `GitHubClient`
`GitRunner` (`api/git_runner.go:19-161`) is explicitly designated the test seam (file header comment) and is mocked by a hand-rolled `FakeGitRunner` (`api/git_runner_fake_test.go`, ~600 lines). Every fake_test.go in the repo follows this pattern (`audit_logger_fake_test.go`, `db_checker_fake_test.go`, `fileio_fake_test.go`, `workspace_sandbox_fake_test.go`) — error-injection fields plus in-memory state, no codegen. **Implication:** introduce `GitHubClient` interface with the minimum surface needed (PR create/get/list/merge, repo info, branch protection read, release create/list/publish, tag create/list, rate-limit ping) and a `FakeGitHubClient` in a `_test.go` file with the same shape (e.g., `CreatePRError`, `MergePRError`, `RateLimitNextReset` for forcing 403 paths).

### Finding 3: Review data is already PR-description-ready, and the readiness badge is already centralized
`ReviewDimensions` (`api/review_model.go:29-45`) bundles code-quality scores, test pass/fail counts, standards violations with file/line/severity, visual capture metadata, and provenance. `review_readiness.go:34-51` produces a green/yellow/red status via `CalculateReadiness(dims, thresholds)`, and `DefaultReadinessThresholds()` (lines 14-26) is the canonical configuration. **Implication:** PR description generation in V1 is a deterministic template that consumes `ReviewDimensions` + commit range + thresholds and emits a structured Markdown body (Summary, Readiness traffic-light, Tests N passed/M failed, Code Quality score, Top Standards Violations, Visual artifacts, Provenance). Critically, the template should call `CalculateReadiness` with the same `ReadinessThresholds` the UI uses so the PR-body badge matches what users see in GCT — no new readiness logic. LLM-assisted prose can layer on top later (mirrors how OT-P1-002 left commit-message AI as an optional layer).

### Finding 4: Routing and concurrency model fit new endpoints cleanly — but use RepoRead, not RepoWrite, for GitHub writes
`routes.go` registers grouped `gorilla/mux` routes. New endpoints (`/api/v1/repo/pr/create|list|view|merge`, `/api/v1/repo/release/...`) plug in alongside existing handlers. DI uses per-domain `Deps` structs (`CredentialsDeps`, `PushPullDeps`). `http_handler.go:36-107` provides `RepoRead` (no lock, uses `--no-optional-locks` for concurrent reads) and `RepoWrite` (per-repo `.git/index.lock` mutex). **Implication:** GitHub *API* calls do NOT touch `.git/index.lock` — they mutate server-side state, not the local working tree. PR/release reads should use `RepoRead`; PR/release writes should use a NEW lightweight `GitHubWrite` `HandlerContext` that resolves the repo + GitHub credential WITHOUT acquiring the per-repo mutex. Forcing GitHub writes through `RepoWrite` would serialize unrelated calls behind the index lock for no benefit. Add `GitHubDeps { Client GitHubClient; Creds CredentialsService; Now func() time.Time }` and pass it through service constructors — no architectural surgery beyond that.

### Finding 5: Library shortlist (resolved → go-github/v66)
- **`github.com/google/go-github/v66`** *(selected, round 1 d1)*: complete REST coverage (PRs, releases, tags, branches), `WithAuthToken()` and `WithEnterpriseURLs()` for GHE, `httpcache`-friendly transport, well-known mock approach (interface wrap; `httptest` server in tests).
- `github.com/cli/go-gh/v2`: thin client used by the `gh` CLI. Lower surface, less type-rich; assumes an external `gh auth` session, which conflicts with reusing the encrypted credential store.
- Hand-rolled HTTP: maximal control, but pays a per-endpoint integration tax. Not justified.

### Finding 6: Initiative-graph implications (resolved → bundled scope)
The initiative has two execute children that depend on this research: `execute/gct-github-pr-api` and `execute/gct-github-release-api`. Both consume the same `GitHubClient` interface and credential model. Round-1 d4 = A locks this research as the single source of truth for the shared client / auth / mock design; PR-vs-release-specific endpoint shape is owned by each execute item. The orchestration summary's deferred "GitLab/Bitbucket future" question is addressed by round-2 d4 (recommend concrete-now, separate research only if a real second-provider requirement appears).

### Finding 7: Retry / backoff template already exists in-repo
`agent_manager_client.go` implements an in-repo retry pattern: `retryBaseDelay = 200ms` (line 30), `retryDelay()` exponential capped at 2s (lines 74-80), `doWithRetry()` and `waitForRetry()` honour `context.Done()` (lines 288-349). **Implication:** whichever layer wins round-2 d1, the GitHubClient retry wrapper should reuse this shape (constants, context discipline) rather than introducing a new backoff vocabulary. This also simplifies code review — engineers reading the GCT API see one retry idiom, not two.

### Finding 8: Mock convention is hand-rolled fakes — confirmed for GitHub
The repo's mock convention is hand-rolled `*_fake_test.go` with error-injection fields and in-memory state (see `git_runner_fake_test.go`, `audit_logger_fake_test.go`). `FakeGitHubClient` should follow the same shape — no `mockgen`, no `gomock` — and live in `github_client_fake_test.go`. Concretely: in-memory PR/release/tag stores keyed by (owner, repo, number), and per-method `Error` fields (`CreatePRError`, `ListPRsError`, `MergePRError`, etc.) plus a `RateLimitedUntil time.Time` for simulating 403 secondary-rate-limit responses.

### Finding 9: Minimum PAT scopes for the V1 surface
Classic PAT needs `repo` for private repos (covers PRs, contents/tags, releases) or `public_repo` for public-only. Fine-grained PATs need `Pull requests: Read and write`, `Contents: Read and write` (for tags), and `Metadata: Read`. The credentials UI should link to GitHub's PAT-creation page with the right scopes pre-selected via query string so users do not have to remember which boxes to tick.

## Limitations
- **Rate-limit / pagination strategy** (round-2 d1) is not yet locked. The recommended approach (wrap retry+pagination inside `GitHubClient`) is a strong default but the user may have a preference.
- **GitHub Enterprise URL** (round-2 d2) — including this in V1 is cheap but adds a credential-schema field; user has not yet confirmed the deployment posture.
- **Partial / stale review data** (round-2 d3) — three rendering policies are on the table; choice affects how the template handles common partial-coverage PRs.
- **Multi-VCS provider abstraction** (round-2 d4) — recommendation is YAGNI now, but if multi-provider is on the near-term roadmap this is the cheapest moment to introduce a `VCSProviderClient` super-interface.
- **GitHub App vs PAT** — locked to PAT for V1 (round-1 d2). If GCT eventually runs as a shared multi-user/multi-org service, App-based auth and per-installation tokens become required; that is a follow-up research item, not a V1 limitation.
- **Token rotation / revocation flow** inside the encrypted store has not been examined for GitHub-specific concerns (fine-grained PAT expiry dates, revocation event handling). Likely a small UI/UX item once the credential schema delta lands.
- **PR description template prototype** has not yet been rendered against real `ReviewDimensions` payloads; field availability when reviews are partial/stale is what round-2 d3 directly addresses.
- **Confidence:** high on the *shape* of the design (interface, fake, deps wiring, credential delta, concurrency model, library choice, auth choice, scope). Medium on rate-limit/pagination layering until d1 resolves. Medium on partial-data template policy until d3 resolves.

## Actions

<!-- Final list will be confirmed after round-2 decisions resolve. The shape below assumes the recommended option (A) for d1-d4. -->

### Pending direction (assumes round-2 d1=A, d2=A, d3=B, d4=A)

- **Update backlog item** `execute/gct-github-pr-api`: append a "Locked design contract from research/gct-github-api-design" section to its plan/spec context citing (a) `GitHubClient` interface seam wrapping `go-github/v66`, (b) PAT-only auth via the new `Service` field on `Credential`, (c) `FakeGitHubClient` hand-rolled in `github_client_fake_test.go`, (d) retry/backoff layer above go-github (reusing `agent_manager_client.go` shape), (e) `GitHubWrite` HandlerContext (no per-repo mutex), (f) deterministic Markdown PR template using `CalculateReadiness` + `DefaultReadinessThresholds`, with omit-on-unavailable / STALE-badge policy.
- **Update backlog item** `execute/gct-github-release-api`: same locked contract, plus the GitHub Releases + Tags subset of the `GitHubClient` interface; reuse the same `FakeGitHubClient` for tests.
- **Update document** `scenarios/git-control-tower/PRD.md`: add a "GitHub Integration" sub-section under Tech Direction Snapshot noting non-goals (no GitHub App in V1, no LLM-generated PR prose in V1, no multi-provider in V1) and the canonical credential field delta (`Service`).
- **Create backlog item (chore)** "Document GitHub PAT scopes in credentials UI": link to GitHub's `/settings/tokens/new?scopes=repo` (classic) and the fine-grained PAT page with the right resource permissions pre-selected.
- **(Conditional on d2=A)** the `Service` field plus `api_base_url` field land together in the same execute item — no separate migration for GHE.
- **(Conditional on d4=B only)** **Create backlog item (research)** "Multi-VCS provider abstraction (GitLab/Bitbucket)" — only if the user opts to design the super-interface now instead of when a real second-provider requirement appears.
- **Initiative-level** — no `depends_on` changes are needed; the upstream (`git-control-tower-ai-provenance`) and downstream (`gct-merge-and-conflicts`, `gct-release-pipeline`) initiatives are unaffected by these decisions.
