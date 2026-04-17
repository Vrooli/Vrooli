# Tool & Safeguard Migration Candidates

Extracted from `master` (`git worktree add ../vrooli-master master`) via Phase C
of the tools/safeguards formalization effort. All files here are frozen copies
of bash originals from the pre-Go-migration era — none are wired into the
current branch. Use this as a reference pool when deciding what to port into
`internal/runtime/` (tool/safeguard handlers) and/or the new top-level
`tools/` and `safeguards/` folders.

## Legend

- **PORTED** — capability already exists as a Go handler in `internal/runtime/`.
  Keep the bash copy for capability-comparison only.
- **UNPORTED** — no Go equivalent; if the capability is still wanted, this is
  implementation input.
- **PARTIAL** — Go version covers some but not all of the bash behavior;
  comparison is worthwhile.
- **DEAD** — bash was only wired through `scripts/lib/setup.sh` (the
  monolithic bash setup orchestrator) and has no live resource/scenario
  consumers; if we don't consciously port it, the capability is simply gone.

## Master consumers

"Master consumer" = a file in `resources/`, `scenarios/`, or a non-setup
`scripts/` entrypoint that sourced the candidate. `scripts/lib/setup.sh` is
noted separately because it's the historical top-level orchestrator —
everything it did is now the Go `vrooli setup` flow.

---

## tools/

Installable programs/CLIs. Analog: `internal/runtime/tool_*.go` handlers
registered in `internal/runtime/registry.go`.

| File | Purpose | Status | Go handler | Master consumers | Notes |
|---|---|---|---|---|---|
| `ast-grep.sh` | Install `ast-grep` (syntax-aware search) | UNPORTED | — | `scripts/lib/setup.sh` only | CLAUDE.md references ast-grep as preferred dev tool; worth porting |
| `ajv.sh` | Install `ajv-cli` npm package (JSON-Schema validator) | UNPORTED | — | `scripts/lib/setup.sh` only | Used to validate schema files; may already be unused |
| `bats.sh` | Install `bats` (bash test runner) | PORTED | `tool_bats.go` | `scripts/lib/setup.sh` only | Compare for feature parity |
| `buf.sh` | Install `buf` CLI (Protobuf tooling) | UNPORTED | — | `scripts/lib/setup.sh` only | Relevant if we move to proto |
| `cloudflare-tunnel.sh` | Install `cloudflared` via apt | UNPORTED | — | `scripts/lib/setup.sh` only | Linux/apt-only; Go port needs cross-platform (brew/choco). User explicitly called this out as a gap |
| `docker.sh` | Install/configure Docker + ensure daemon running | PORTED | `tool_docker.go` | `scripts/lib/setup.sh`, `scripts/resources/common.sh` | Bash is 22KB — rich daemon-management logic; compare |
| `go.sh` | Install Go toolchain, configure GOPATH/GOBIN | PORTED | `tool_go.go` | `scripts/lib/setup.sh` only | 21KB — version-pinning & path setup; compare |
| `helm.sh` | Install Helm CLI | PORTED | `tool_helm.go` | `scripts/lib/setup.sh` only | 24KB — chart/repo helpers beyond just install; compare |
| `js-yaml.sh` | Install `js-yaml` npm package | UNPORTED | — | `scripts/lib/setup.sh` only | Small; probably drop |
| `lychee.sh` | Install `lychee` (link checker) | UNPORTED | — | `scripts/lib/setup.sh` only | Relevant for doc-link validation |
| `nodejs.sh` | Install Node + npm | PORTED | `tool_node.go` | `scripts/lib/setup.sh` only | 23KB — version-pinning via nvm or binary; compare |
| `python.sh` | Install Python + pip | PORTED | `tool_python.go` | `scripts/lib/setup.sh` only | 24KB — venv + distro detection; compare |
| `shellcheck.sh` | Install `shellcheck` | UNPORTED | — | `scripts/lib/setup.sh` only | Shellcheck is mostly irrelevant now that scenario CLIs are Go |
| `sqlite.sh` | Install SQLite CLI | PORTED | `tool_sqlite.go` | `scripts/lib/setup.sh` only | Compare; resource `sqlite` also exists as its own thing |
| `stripe_cli.sh` | Install Stripe CLI | PORTED | `tool_stripe.go` | `scripts/lib/setup.sh` only | Compare |
| `vault.sh` | Install Vault CLI + auth/kv helpers | PARTIAL | — | **5 live resources** (openrouter, gemini, litellm configuration helpers) | **Critical**: live resources source this for `vault kv` operations. Current branch keeps a `vault` resource separately; this bash file is a *CLI tool* wrapper distinct from the resource |

## safeguards/

Machine-state modifications for protection/security/operation. Analog:
`internal/runtime/safeguard_*.go` handlers (only 1 exists today).

| File | Purpose | Status | Go handler | Master consumers | Notes |
|---|---|---|---|---|---|
| `clock.sh` | Ensure system clock is NTP-synced | UNPORTED | — | `scripts/lib/setup.sh` | Critical for TLS/JWT/Kerberos; explicit gap per user |
| `common_deps.sh` | Install common OS packages (build-essential, pkg-config, etc. via apt/brew) | UNPORTED | — | `scripts/lib/setup.sh` | Meta-package installer; decide whether this becomes one safeguard or many tools |
| `domain_check.sh` | DNS/domain reachability diagnostic | UNPORTED | — | None | Standalone diagnostic; zero callers — may never have been wired |
| `firewall_ufw.sh` | Configure UFW with Vrooli port allowances | UNPORTED | — | `scripts/lib/setup.sh` (+ sourced by `remote_session_protect.sh` + `firewall_network.sh`) | Linux UFW specific; explicit gap per user |
| `firewall_network.sh` | Lower-level iptables/nft rules | UNPORTED | — | `scripts/lib/setup.sh`, `system/remote_session_protect.sh`, `firewall/firewall.sh` | Overlaps with `firewall_ufw.sh`; decide unified port |
| `kernel_config.sh` | Sysctl tuning (swap, file handles, net backlog) | UNPORTED | — | `scripts/lib/setup.sh` | Linux-only sysctl config; explicit gap per user |
| `network_diagnostics.sh` | 38KB multi-phase network diagnostic (DNS, TCP, TLS, MTU, IPv4/IPv6, auto-fix) | UNPORTED | — | `scripts/lib/setup.sh` | Large; explicit gap per user. Splits into sub-safeguards (dns_check, tcp_tuning, tls_handshake, etc.)? |
| `remote_session_protect.sh` | Linux sysctl + systemd hardening for GUI/SSH sessions | PARTIAL | `safeguard_remote_session_protection.go` | None (included via sourcing) | 23KB bash vs the Go handler — compare thoroughness; bash also pulls firewall + network rules |
| `ssh_authorize_key.sh` | Add a public key to `authorized_keys` | UNPORTED | — | None | Manual-use bootstrap; maybe belongs as a CLI command rather than a safeguard |
| `ssh_enable.sh` | Enable sshd, configure password/key auth | UNPORTED | — | None | Manual-use bootstrap |
| `ssh_keyless.sh` | Set up passwordless SSH between nodes | UNPORTED | — | None | Manual-use bootstrap |

## unclassified/

Files that don't cleanly fit the tool or safeguard definitions.

| File | Purpose | Suggested home |
|---|---|---|
| `service_repository.sh` | Legacy helper for git repo setup / package repositories | Probably dead. Only matches were Go test files (false positives) |
| `k8s_health.sh` | Runtime health polling for k8s-backed resources | Belongs with resource runtime/health-framework code, not tools/safeguards |
| `k8s/dependencies.sh` | Validate k8s pods/services/configmaps exist | Runtime validator, not installer. Sourced by `scripts/lib/scenario/runner.sh` (deleted already) |
| `k8s/operators.sh` | Validate k8s operator CRDs + pods | Same — runtime validator |

---

## Cross-cutting observations

1. **Most candidates had zero live consumers outside `scripts/lib/setup.sh`.**
   The bash setup.sh has been fully replaced by `internal/setup/setup.go` +
   `internal/hostreq/Resolve` + `internal/runtime/EnsureRequirements`. If we
   don't port a capability, it's simply gone after these bash files are
   retired. That's fine for capabilities we don't want (e.g. `shellcheck`
   installer, `ssh_*` bootstrap); it's a problem for ones we do (firewall,
   clock, network_diagnostics).

2. **`vault.sh` is the notable exception** — live resources (openrouter,
   gemini) still source it for `vault kv` credential reads. Any cleanup
   plan for `scripts/migrate_candidates/` must either port this to a Go
   tool handler *plus* update resource consumers, or accept that those
   resources break.

3. **`cloudflared` install is Linux-only in bash.** A Go port must cover
   macOS (`brew install cloudflared`) and Windows (`choco` or direct
   download).

4. **`remote_session_protect.sh` (23KB) is dramatically richer than the
   Go port.** The bash version also invokes firewall + network rules as
   part of the safeguard. Worth reviewing to decide whether the Go
   handler should pull firewall config in too, or whether firewall
   should be its own separate safeguard that gets declared alongside
   remote_session_protection.

5. **`common_deps.sh` design choice** — one monolithic safeguard that
   installs the "standard" OS package set, versus N individual tool
   handlers (build-essential, pkg-config, etc.). Leaning one safeguard
   in Phase E discussion.

6. **SSH bootstrap scripts had no automation consumers.** They were manual
   operator tools. May belong as `vrooli bootstrap ssh-*` subcommands
   rather than declared safeguards.

## What was deliberately excluded

- `scripts/lib/setup-conditions/*.sh` — not tools/safeguards; these are
  lifecycle condition evaluators (binaries exist, dirs exist, etc.).
  Already superseded by Go lifecycle phase conditions.
- `scripts/lib/scenario/runner.sh`, `dependencies.sh` — scenario
  orchestration, superseded by `internal/lifecycle/`.
- `scripts/lib/process-manager.sh`, `ui-guard.sh`, `static-ui-build.js`
  — not tools or safeguards.
- `scripts/lib/utils/*` — shared shell helpers; the ones still live
  remain in the current branch's `scripts/lib/utils/`.

## Next step (Phase D)

Audit `internal/runtime/` to inventory what already exists Go-side and
map it against this candidate list. Then Phase E decides scope for the
top-level `tools/` and `safeguards/` folders.
