# Research Conclusion: GitHub API Integration for Git Control Tower

## Research Question
How should Git Control Tower integrate with the GitHub API so that (a) every GitHub call is mockable behind a single Go interface, (b) credentials reuse the existing encrypted credential store, (c) tests make zero real GitHub API calls, and (d) review-panel results can be transformed into PR descriptions? Specifically: which Go library to use, which auth modes (PAT, GitHub App, SSH for transport vs. token for API), what PR metadata to surface, and how to wire all of this into the existing handler/service/runner pattern.

## Summary
GCT already has the foundations needed: `CredentialTypeHTTPS` (token-based), an encrypted `~/.config/git-control-tower/credentials.enc` store, a hand-rolled-fake mock convention, and structured `ReviewDimensions` data ready to feed PR descriptions. SSH stays the transport for git push/pull; a new token-based path is needed only for REST/GraphQL calls. The recommended direction is `github.com/google/go-github/v66` behind a narrow `GitHubClient` interface with a `FakeGitHubClient` that mirrors the existing `FakeGitRunner` pattern, V1 scoped to GitHub PAT only, and a deterministic template that maps `ReviewDimensions` into a PR body (LLM-assisted variant deferred). Open scope question: whether this single research item should also lock down the design for tag/release APIs (sibling `execute/gct-github-release-api`) since they share the same client and auth surface.

## Methodology
- Read item spec, initiative neighborhood (`gct-github-integration` — 5 members, upstream `git-control-tower-ai-provenance`, downstreams `gct-merge-and-conflicts` and `gct-release-pipeline`), and PRD at `scenarios/git-control-tower/PRD.md`.
- Inventoried existing primitives in `scenarios/git-control-tower/api/`:
  - Credential storage: `credentials_store.go`, `credentials_model.go`, `credential_resolve.go`.
  - Git execution seam: `git_runner.go` interface + `git_runner_fake_test.go` hand-rolled fake.
  - Review data shape: `review_model.go` (`ReviewDimensions`, `CodeQualityDimension`, `TestsDimension`, `StandardsDimension`, `VisualDimension`, `ProvenanceDimension`).
  - Routing & handler pattern: `routes.go`, `http_handler.go` (`RepoRead`/`RepoWrite`).
  - Push/pull auth: `push_pull_service.go` + `git_runner.go` `gitCredentialEnv()`.
- Confirmed go.mod has no `go-github`/`go-gh` today, so a new dependency is required.

## Findings

### Finding 1: The credential store already supports GitHub tokens
`CredentialTypeHTTPS` (`api/credentials_model.go:10`) carries `Username + Token`, persisted AES-256-GCM-encrypted at `~/.config/git-control-tower/credentials.enc` (`api/credentials_store.go:53,170-189`). A GitHub PAT is exactly an HTTPS token. **Implication:** no new credential type is required; we add a small "service tag" (e.g. `service: "github.com"` or a `provider` enum) so credential lookup distinguishes a GitHub API token from a generic git HTTPS token tied to a remote URL. SSH continues to handle git transport via `GIT_SSH_COMMAND` (`api/git_runner.go:185-243`); the GitHub API path is independent and token-only.

### Finding 2: A `GitRunner`-style seam is the right model for `GitHubClient`
`GitRunner` (`api/git_runner.go:19-161`) is explicitly designated the test seam (file header comment) and is mocked by a hand-rolled `FakeGitRunner` (`api/git_runner_fake_test.go`, ~600 lines). Every fake_test.go in the repo follows this pattern (`audit_logger_fake_test.go`, `db_checker_fake_test.go`, `fileio_fake_test.go`, `workspace_sandbox_fake_test.go`) — error-injection fields plus in-memory state, no codegen. **Implication:** introduce `GitHubClient` interface with the minimum surface needed (PR create/get/list/merge, repo info, branch protection read, release create/list/publish, tag create/list, rate-limit ping) and a `FakeGitHubClient` in a `_test.go` file with the same shape.

### Finding 3: Review data is already PR-description-ready
`ReviewDimensions` (`api/review_model.go:29-45`) bundles code-quality scores, test pass/fail counts, standards violations with file/line/severity, visual capture metadata, and provenance. `review_readiness.go:34-51` produces a green/yellow/red status. **Implication:** PR description generation in V1 should be a deterministic template that consumes `ReviewDimensions` + commit range and emits a structured Markdown body (Summary, Readiness traffic-light, Tests N passed/M failed, Code Quality score, Top Standards Violations, Visual artifacts, Provenance). LLM-assisted prose can be a v2 layered on top — exactly mirrors how OT-P1-002 left commit-message AI as an optional layer.

### Finding 4: Routing and DI conventions slot new PR endpoints in cleanly
`routes.go:9-128` registers grouped `gorilla/mux` routes. New endpoints (`/api/v1/repo/pr/create|list|view|merge`, `/api/v1/repo/release/...`) plug in with `RepoRead`/`RepoWrite` handlers (`api/http_handler.go:36-107`). DI uses per-domain `Deps` structs (`CredentialsDeps`, `PushPullDeps`). **Implication:** add a `GitHubDeps { Client GitHubClient; Creds CredentialsService; Now func() time.Time }` and pass it through service constructors — no architectural surgery.

### Finding 5: Library shortlist
- **`github.com/google/go-github/v66`** *(recommended for V1)*: complete REST coverage (PRs, releases, tags, branches), `WithAuthToken()` and `WithEnterpriseURLs()` for GHE, `httpcache`-friendly transport, well-known mock approach (interface wrap; `httptest` server in tests).
- **`github.com/cli/go-gh/v2`**: thin client used by the `gh` CLI. Lower surface, less type-rich; better when piggybacking on an existing `gh auth` session, which we do not want (we own creds).
- **Hand-rolled HTTP**: maximal control, but pays a tax for every endpoint. Not justified.

### Finding 6: Initiative-graph implications
The initiative has two execute children that depend on this research: `execute/gct-github-pr-api` and `execute/gct-github-release-api`. Both consume the same `GitHubClient` interface and credential model. Conducting one research that resolves the *shared* design (auth, client, mocks, deps wiring) and leaves PR-vs-release-specific endpoint shape to each execute item is the lower-friction path. The alternative — splitting research into PR-design and release-design — duplicates the auth/mock decisions.

## Limitations
- Library choice not yet user-confirmed; recommendation is based on coverage and convention but the user may have a preference (e.g. self-hosted GitHub Enterprise constraints, license, vendoring).
- GitHub App vs PAT: PAT is fine for a single-user CLI/dev tool; if GCT will run as a shared service for multiple users/orgs, App-based auth and per-installation tokens are eventually required. Not yet clarified.
- No empirical look at rate-limit handling, retry/backoff, or pagination strategy — deferred to a later round once library is locked.
- PR description templating against real `ReviewDimensions` payloads has not been prototyped; field availability when reviews are partial/stale is an open question.
- Token revocation/rotation flow inside the encrypted store has not been examined for GitHub-specific concerns (e.g. fine-grained PAT scopes, expiry).
- Confidence: high on the *shape* of the design (interface, fake, deps wiring), medium on the right library, medium-low on auth-scope sufficiency.

## Actions

<!-- TBD — finalized after decisions resolve -->

### Pending direction
Concrete `Create`/`Update`/`Delete` actions will be filled in once the round-1 decisions resolve. The likely shape:
- **Update backlog item** `execute/gct-github-pr-api` and `execute/gct-github-release-api` with the locked client interface, auth model, and fake convention so each picks up the contract directly.
- Possibly **Create backlog item** (research or chore) for "GitHub App auth path" if V1 is PAT-only, deferred until multi-user scenarios are needed.
- **Update document** `scenarios/git-control-tower/PRD.md` to note the GitHub integration sub-plane and its non-goals (no GitHub App in V1, no UI-side token entry beyond existing credentials panel).
