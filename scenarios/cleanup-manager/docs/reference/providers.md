# Providers — Cleanup Manager

Cleanup providers are preview-first units of reclaimable-space logic. A
provider can inspect usage, produce a preview, and apply only the exact
previewed items after policy and approval gates pass.

## Provider Contract

Every provider implements `api/internal/cleanup.Provider`:

| Method | Purpose |
|---|---|
| `Metadata` | Declares provider id, version, owner scenario, safety tier, default mode, default approval, supported platforms, privileges, irreversible effects, and test substitute. |
| `Estimate` | Computes a reclaim estimate from observations and policy. It may return a blocked reason instead of an error when policy disables the provider. |
| `Preview` | Returns concrete preview rows. Apply must use this preview and must not rediscover a broader target set. |
| `Apply` | Mutates only through injected seams and only after caller-supplied plan, provider version, approval mode, and idempotency key checks. |
| `Verify` | Reports whether the provider-specific post-apply check succeeded. |

Apply-only providers are invalid. `cleanup.ValidateProvider` rejects
providers with missing metadata, unsafe conditional defaults, or enabled
forbidden defaults.

## Safety Tiers

| Tier | Default posture | Examples |
|---|---|---|
| `safe` | Disabled or explicitly policy-enabled; may still require operator approval. | Temporary files under configured roots. |
| `safe_with_owner` | Owner or operator approval required. | Web-console old sessions, test-run retention owned by another scenario. |
| `conditional` | Disabled by conservative defaults; operator approval required when enabled. | Docker dangling images/build cache, language/build caches, journald, apt metadata. |
| `forbidden` | Always disabled. | Live databases, model stores, Docker volumes unless a future explicit provider reclassifies them with stronger controls. |

## Built-In Provider Set

The current conservative built-ins are constructed by
`providers.ConservativeBuiltIns`:

| Provider | Tier | Default | Seam |
|---|---|---|---|
| `trash` | `safe_with_owner` | disabled, owner approval | `FileSystem` |
| `tmp` | `safe` | disabled, operator approval | `FileSystem` |
| `go-build-cache` | `conditional` | disabled, operator approval | `FileSystem` |
| `playwright-cache` | `conditional` | disabled, operator approval | `FileSystem` |
| `docker` | `conditional` | disabled, operator approval | `DockerClient` |
| `journald` | `conditional` | disabled, operator approval | `JournalClient` |
| `apt-metadata` | `conditional` | disabled, operator approval | `ProcessRunner` metadata probe, preview-only apply |
| `workspace-sandbox-retention` | `safe_with_owner` | disabled, owner approval | `ScenarioProviderClient` |
| `test-genie-run-retention` | `safe_with_owner` | disabled, owner approval | `ScenarioProviderClient` |
| `web-console-sessions` | `safe_with_owner` | disabled, owner approval | `ScenarioProviderClient` |

Filesystem providers only walk configured roots and skip active-looking
paths such as lock files, dotfiles, sockets, and `/proc` descendants.
Docker previews include dangling images and build cache, but intentionally
exclude volumes. Command-backed metadata providers are preview-only until
a typed allowlisted executor is wired. Owner-scenario providers are
catalog entries only until an owner client is available; when enabled they
delegate Estimate/Preview/Apply to the owning scenario and never crawl
private storage directly.

## Adjacent Scenario Handoff

Cleanup Manager owns cleanup policy, planning, approval, replay safety,
and audit. Adjacent scenarios keep their narrower ownership:

| Scenario | Role | Cleanup-manager boundary |
|---|---|---|
| `system-monitor` | Observes disk pressure, attribution, and health signals. | It links operators to cleanup plans but does not mutate disk state. |
| `vrooli-autoheal` | Detects failures and offers recovery surfaces. | Broad disk/Docker cleanup actions hand off to `cleanup-manager cleanup plan`; autoheal does not run prune/vacuum/cache deletion directly. |
| `workspace-sandbox` | Owns sandbox-private lifecycle and orphan cleanup semantics. | `workspace-sandbox-retention` is a disabled-by-default owner hook that delegates Estimate/Preview/Apply through `ScenarioProviderClient`. |
| `test-genie` and `web-console` | Own run/session retention semantics. | `test-genie-run-retention` and `web-console-sessions` are disabled-by-default owner hooks, not filesystem crawlers in cleanup-manager. |

This keeps circular startup risk low: adjacent scenarios can degrade to
diagnostic links when cleanup-manager is unavailable, while cleanup-manager
never duplicates owner-private deletion logic.

## Policy Profiles

`api/internal/policy` defines three profile names:

| Profile | Behavior |
|---|---|
| `conservative` | Keeps provider metadata defaults, disables all conditional and forbidden providers, and uses a seven-day minimum age for file providers. |
| `balanced` | Enables `safe` and `safe_with_owner` providers with a three-day minimum age; conditional providers remain disabled. |
| `aggressive` | Enables every non-forbidden provider with a one-day minimum age but still requires operator approval for conditional providers. |

`policy.ValidateProviderPolicy` rejects unsafe overrides: forbidden
providers cannot be enabled, conditional providers cannot run without
operator approval, and age/byte limits cannot be negative.

## Test Substitutes

Provider tests use fakes from `api/internal/testutil/cleanup`:

| Fake | Protected effect |
|---|---|
| `FileSystem` | Blocks removal unless enabled and scoped under the fake root. |
| `ProcessRunner` | Records commands and rejects forbidden command strings. |
| `DockerClient` | Records prune requests and rejects volume prune. |
| `JournalClient` | Records fake vacuum requests. |
| `ScenarioProviderClient` | Records owner-scenario delegated apply requests. |

The no-real-cleanup drift test still scans production Go code for direct
cleanup side-effect constructors such as `os.RemoveAll`, `exec.Command`,
Docker prune commands, journal vacuum commands, and package-cache cleanup
commands.

## Cross-References

- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect seams
- [`../internal/INVARIANTS.md`](../internal/INVARIANTS.md) — cleanup and replay invariants
- [`configuration.md`](configuration.md) — provider policy control surface
