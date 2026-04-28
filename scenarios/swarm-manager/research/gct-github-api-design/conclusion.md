# Research Conclusion: GitHub API Integration for Git Control Tower

## Research Question
How should Git Control Tower integrate with the GitHub API so that (a) every GitHub call is mockable behind a single Go interface, (b) credentials reuse the existing encrypted credential store, (c) tests make zero real GitHub API calls, and (d) review-panel results can be transformed into PR descriptions? Specifically: which Go library to use, which auth modes (PAT, GitHub App, SSH for transport vs. token for API), what PR metadata to surface, and how to wire all of this into the existing handler/service/runner pattern.

## Summary
All eight design decisions are now locked across rounds 1 and 2. The shared client / auth / mock design is fully specified for both downstream execute children (`execute/gct-github-pr-api`, `execute/gct-github-release-api`).

**Locked design contract:**
- **Library:** `github.com/google/go-github/v66`, wrapped behind a narrow `GitHubClient` interface (round 1 d1).
- **Auth (V1):** GitHub PAT only, stored in the existing encrypted credential store with one additive `Service` field; GitHub App / installation tokens deferred (round 1 d2).
- **PR description:** deterministic Markdown template fed by `ReviewDimensions` and `CalculateReadiness`; LLM polish deferred (round 1 d3).
- **Scope:** this single research item locks the shared client / auth / mock design for both `execute/gct-github-pr-api` and `execute/gct-github-release-api`; each execute child owns its endpoint shape (round 1 d4).
- **Rate limit / retry / pagination:** wrapped *inside* the `GitHubClient` interface (one layer above go-github), reusing the `agent_manager_client.go` exponential-backoff shape; pagination auto-handled by the wrapper (round 2 d1 = A).
- **GitHub Enterprise:** ship in V1. Add an optional `api_base_url` field to the GitHub credential record; default to `api.github.com`, route through `WithEnterpriseURLs()` when set (round 2 d2 = A).
- **Partial / stale review dimensions in PR template:** omit sections for unavailable dimensions; render stale dimensions with an explicit `STALE` badge (round 2 d3 = B).
- **Multi-provider abstraction:** YAGNI. Design `GitHubClient` as a concrete interface now; treat GitLab/Bitbucket as a separate research item if a real requirement appears later (round 2 d4 = A).

GCT already has the foundations needed: `CredentialTypeHTTPS` (token-based), an encrypted `~/.config/git-control-tower/credentials.enc` store, a hand-rolled-fake mock convention, structured `ReviewDimensions` data with a configurable readiness calculator, and an in-repo retry/backoff template (`agent_manager_client.go`). SSH stays the transport for git push/pull; the new token-based path is needed only for REST/GraphQL calls. The credential model gains two small additive fields (`Service`, `APIBaseURL`) so a GitHub API token can live alongside remote-bound git HTTPS credentials and target either github.com or a GHE host.

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

### Finding 1: The credential store already supports GitHub tokens, with two additive schema deltas
`CredentialTypeHTTPS` (`api/credentials_model.go:10`) carries `Username + Token`, persisted AES-256-GCM-encrypted at `~/.config/git-control-tower/credentials.enc` (`api/credentials_store.go:53,170-189`). A GitHub PAT is exactly an HTTPS token. **However**, today every credential is keyed by a git remote (`Remote`, `URL`); a single PAT is not bound to a single remote. **Implication:** add two optional fields to `Credential` and `StoredCredential`:
- `Service string` (e.g., `"github.com"`, `"github-enterprise.example.com"`) — credentials with `Type=https && Service != ""` are treated as service-scoped API tokens, not remote-bound git HTTPS credentials.
- `APIBaseURL string` (round 2 d2) — optional override; when set on a GitHub-service credential, the `GitHubClient` factory routes through `go-github`'s `WithEnterpriseURLs()`. Default is `api.github.com`.

Both deltas are purely additive to the encrypted on-disk JSON format. SSH continues to handle git transport via `GIT_SSH_COMMAND` (`api/git_runner.go:185-243`); the GitHub API path stays independent and token-only.

### Finding 2: A `GitRunner`-style seam is the right model for `GitHubClient`
`GitRunner` (`api/git_runner.go:19-161`) is explicitly designated the test seam (file header comment) and is mocked by a hand-rolled `FakeGitRunner` (`api/git_runner_fake_test.go`, ~600 lines). Every fake_test.go in the repo follows this pattern (`audit_logger_fake_test.go`, `db_checker_fake_test.go`, `fileio_fake_test.go`, `workspace_sandbox_fake_test.go`) — error-injection fields plus in-memory state, no codegen. **Implication:** introduce `GitHubClient` interface with the minimum surface needed (PR create/get/list/merge, repo info, branch protection read, release create/list/publish, tag create/list, rate-limit ping) and a `FakeGitHubClient` in a `_test.go` file with the same shape (e.g., `CreatePRError`, `MergePRError`, `RateLimitNextReset` for forcing 403 paths). Per round 2 d1, retry and pagination are handled *inside* the `GitHubClient` wrapper, so `FakeGitHubClient` exposes the same retry/pagination behavior to tests through error-injection fields rather than an HTTP-layer interceptor.

### Finding 3: Review data is already PR-description-ready, and the readiness badge is already centralized
`ReviewDimensions` (`api/review_model.go:29-45`) bundles code-quality scores, test pass/fail counts, standards violations with file/line/severity, visual capture metadata, and provenance. `review_readiness.go:34-51` produces a green/yellow/red status via `CalculateReadiness(dims, thresholds)`, and `DefaultReadinessThresholds()` (lines 14-26) is the canonical configuration. **Implication:** PR description generation in V1 is a deterministic template that consumes `ReviewDimensions` + commit range + thresholds and emits a structured Markdown body (Summary, Readiness traffic-light, Tests N passed/M failed, Code Quality score, Top Standards Violations, Visual artifacts, Provenance). Critically, the template calls `CalculateReadiness` with the same `ReadinessThresholds` the UI uses so the PR-body badge matches what users see in GCT — no new readiness logic. **Partial-data policy (round 2 d3 = B):** sections for unavailable dimensions are *omitted* entirely; sections for available-but-stale dimensions render with an explicit `STALE` badge inline next to the section header. This keeps PR bodies clean for narrow-scope changes (e.g., docs-only, backend-only) while still flagging stale data. LLM-assisted prose can layer on top later.

### Finding 4: Routing and concurrency model fit new endpoints cleanly — but use RepoRead, not RepoWrite, for GitHub writes
`routes.go` registers grouped `gorilla/mux` routes. New endpoints (`/api/v1/repo/pr/create|list|view|merge`, `/api/v1/repo/release/...`) plug in alongside existing handlers. DI uses per-domain `Deps` structs (`CredentialsDeps`, `PushPullDeps`). `http_handler.go:36-107` provides `RepoRead` (no lock, uses `--no-optional-locks` for concurrent reads) and `RepoWrite` (per-repo `.git/index.lock` mutex). **Implication:** GitHub *API* calls do NOT touch `.git/index.lock` — they mutate server-side state, not the local working tree. PR/release reads should use `RepoRead`; PR/release writes should use a NEW lightweight `GitHubWrite` `HandlerContext` that resolves the repo + GitHub credential WITHOUT acquiring the per-repo mutex. Forcing GitHub writes through `RepoWrite` would serialize unrelated calls behind the index lock for no benefit. Add `GitHubDeps { Client GitHubClient; Creds CredentialsService; Now func() time.Time }` and pass it through service constructors — no architectural surgery beyond that.

### Finding 5: Library shortlist (resolved → go-github/v66)
- **`github.com/google/go-github/v66`** *(selected, round 1 d1)*: complete REST coverage (PRs, releases, tags, branches), `WithAuthToken()` and `WithEnterpriseURLs()` for GHE, `httpcache`-friendly transport, well-known mock approach (interface wrap; `httptest` server in tests).
- `github.com/cli/go-gh/v2`: thin client used by the `gh` CLI. Lower surface, less type-rich; assumes an external `gh auth` session, which conflicts with reusing the encrypted credential store.
- Hand-rolled HTTP: maximal control, but pays a per-endpoint integration tax. Not justified.

### Finding 6: Initiative-graph implications (resolved → bundled scope, no multi-provider research item)
The initiative has two execute children that depend on this research: `execute/gct-github-pr-api` and `execute/gct-github-release-api`. Both consume the same `GitHubClient` interface and credential model. Round-1 d4 = A locks this research as the single source of truth for the shared client / auth / mock design; PR-vs-release-specific endpoint shape is owned by each execute item. The orchestration summary's deferred "GitLab/Bitbucket future" question is now closed (round 2 d4 = A): no `VCSProviderClient` super-interface in V1, no new research item created. If a real second-provider requirement surfaces later, the right action is a new research item then — the `GitHubClient` seam is the natural extension point. No initiative `depends_on` changes are needed.

### Finding 7: Retry / backoff layer (resolved → wrapped inside GitHubClient)
`agent_manager_client.go` implements an in-repo retry pattern: `retryBaseDelay = 200ms` (line 30), `retryDelay()` exponential capped at 2s (lines 74-80), `doWithRetry()` and `waitForRetry()` honour `context.Done()` (lines 288-349). **Resolution (round 2 d1 = A):** the `GitHubClient` wrapper layer above `go-github` owns retry, backoff, and auto-pagination. It reuses the `agent_manager_client.go` shape (constants and context discipline) so engineers see one retry idiom across the GCT API, and `FakeGitHubClient` exposes the same behavior via error-injection fields (e.g., `RateLimitedUntil time.Time`, `TransientErrorOnCallN int`) — mirroring the `FakeGitRunner.PushError` pattern. Pagination is handled inside the wrapper: methods return fully-walked slices for V1 (cursor-streaming can be added if a slow path appears).

### Finding 8: Mock convention is hand-rolled fakes — confirmed for GitHub
The repo's mock convention is hand-rolled `*_fake_test.go` with error-injection fields and in-memory state (see `git_runner_fake_test.go`, `audit_logger_fake_test.go`). `FakeGitHubClient` should follow the same shape — no `mockgen`, no `gomock` — and live in `github_client_fake_test.go`. Concretely: in-memory PR/release/tag stores keyed by (owner, repo, number), and per-method `Error` fields (`CreatePRError`, `ListPRsError`, `MergePRError`, etc.) plus a `RateLimitedUntil time.Time` for simulating 403 secondary-rate-limit responses.

### Finding 9: Minimum PAT scopes for the V1 surface
Classic PAT needs `repo` for private repos (covers PRs, contents/tags, releases) or `public_repo` for public-only. Fine-grained PATs need `Pull requests: Read and write`, `Contents: Read and write` (for tags), and `Metadata: Read`. The credentials UI should link to GitHub's PAT-creation page with the right scopes pre-selected via query string so users do not have to remember which boxes to tick.

### Finding 10: GHE configurability lands together with PAT support (round 2 d2 = A)
Adding `WithEnterpriseURLs()` is one factory branch and one optional credential field (`APIBaseURL`). Doing it now avoids a credential-schema migration later, costs essentially nothing, and matches GCT's stated multi-tier deployment vision. The factory: if `APIBaseURL == ""`, call `github.NewClient(httpClient).WithAuthToken(token)`; otherwise call `github.NewClient(httpClient).WithAuthToken(token).WithEnterpriseURLs(apiBaseURL, apiBaseURL)` (upload URL = api URL is the standard GHE setup). Validation rejects malformed URLs at credential-save time, not at request time.

## Limitations
- **Token rotation / revocation flow** inside the encrypted store has not been examined for GitHub-specific concerns (fine-grained PAT expiry dates, revocation event handling). Likely a small UI/UX item once the credential schema delta lands.
- **PR description template prototype** has not yet been rendered against real `ReviewDimensions` payloads; the omit-on-unavailable + STALE-badge policy (d3 = B) needs at least one execute-time render against a partial-coverage payload to confirm the Markdown layout reads cleanly.
- **GitHub App vs PAT** — locked to PAT for V1. If GCT eventually runs as a shared multi-user/multi-org service, App-based auth and per-installation tokens become required; that is a follow-up research item, not a V1 limitation.
- **Multi-VCS abstraction** — locked to YAGNI. If a real GitLab/Bitbucket requirement surfaces, a separate research item ("Multi-VCS provider abstraction") is the right entry point; the `GitHubClient` interface is the natural extension seam.
- **Pagination behavior under very large result sets** — the wrapper auto-walks all pages and returns a full slice. For a hypothetical repo with 10,000+ open PRs, this is suboptimal; if such a case appears in practice, add a streaming variant in a follow-up.
- **Confidence:** high on all eight locked decisions and the resulting design contract (interface, fake, deps wiring, credential delta, concurrency model, library, auth, retry layer, GHE, template policy, scope). The two execute children can be picked up immediately against this contract.

## Actions

- **Update backlog item** `execute/gct-github-pr-api`: append a "Locked design contract from research/gct-github-api-design" section to its plan/spec context citing (a) `GitHubClient` interface seam wrapping `go-github/v66`, (b) PAT-only auth via the new `Service` and `APIBaseURL` fields on `Credential`, (c) `FakeGitHubClient` hand-rolled in `github_client_fake_test.go`, (d) retry/backoff/auto-pagination wrapped *inside* the `GitHubClient` (reusing `agent_manager_client.go` shape), (e) `GitHubWrite` HandlerContext (no per-repo mutex), (f) deterministic Markdown PR template using `CalculateReadiness` + `DefaultReadinessThresholds`, omitting sections for unavailable dimensions and rendering `STALE` badges for stale ones, (g) GHE supported in V1 via optional `APIBaseURL`, (h) no multi-provider abstraction — `GitHubClient` is concrete.
- **Update backlog item** `execute/gct-github-release-api`: same locked contract, plus the GitHub Releases + Tags subset of the `GitHubClient` interface; reuse the same `FakeGitHubClient` for tests.
- **Update document** `scenarios/git-control-tower/PRD.md`: add a "GitHub Integration" sub-section under Tech Direction Snapshot noting non-goals (no GitHub App in V1, no LLM-generated PR prose in V1, no multi-VCS provider abstraction in V1) and the canonical credential field deltas (`Service`, `APIBaseURL`).
- **Create backlog item (chore)** "Document GitHub PAT scopes in credentials UI": link to GitHub's `/settings/tokens/new?scopes=repo` (classic) and the fine-grained PAT page with the right resource permissions pre-selected. Justification: this is a focused UI/docs task that doesn't fit inside `execute/gct-github-pr-api` (which is API surface) and isn't worth blocking the API work on.
- **Initiative-level** — no `depends_on` or `priority` changes. The two `execute/gct-github-*-api` siblings remain valid as designed; the `*-ui` siblings are unaffected. Upstream (`git-control-tower-ai-provenance`) and downstream (`gct-merge-and-conflicts`, `gct-release-pipeline`) initiatives are unaffected.
- **No deletions or invalidations** — all five initiative members remain valid; no sibling research has been superseded by these findings, and no separate "multi-VCS abstraction" research item is created (round 2 d4 = A explicitly closes that question).
