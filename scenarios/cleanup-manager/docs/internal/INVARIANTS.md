# Invariants — Cleanup Manager

## Cleanup Safety

- Providers must implement Estimate and Preview before Apply. Apply-only
  providers are not part of the contract.
- Providers declare safety tier, default mode, default approval,
  irreversible effects, privileges, platforms, owner metadata, version,
  and test substitute.
- Forbidden providers are disabled and cannot be enabled by default.
- Conditional providers require operator approval or are disabled.
- Tests use fake filesystems, fake process runners, fake Docker clients,
  fake journal clients, fake clocks, and fake owner-scenario clients.
- Production code must not instantiate filesystem deletion, Docker prune,
  journald vacuum, apt cleanup, or language cache cleanup outside approved
  seam constructors.
- Built-in conservative providers are disabled by default when the cleanup
  target is conditional or owner-scoped. Docker provider previews exclude
  volumes and applies only dangling images/build cache through DockerClient.
- Owner-scenario providers delegate private deletion to the owner scenario
  after owner/operator approval; cleanup-manager records orchestration and
  policy but does not duplicate private deletion logic.

## Replay/Idempotency Invariants

- Apply requires the exact plan id, policy version, provider version,
  approval mode, and idempotency key.
- Replaying the same apply request with the same idempotency key returns
  an already-done/no-op result rather than applying again.
- Audit records are append-only: retries add attempt context without
  rewriting prior events.
- Provider previews are inputs to Apply. Apply does not rediscover a
  broader target set than the approved preview.
- File providers apply only previewed paths that remain under configured
  roots. Active-looking paths such as lock files, dotfiles, sockets, and
  `/proc` descendants are skipped during preview.

## Checkpoint Flows

- Scan is repeatable and can be discarded.
- Plan is a durable checkpoint: it captures provider estimates, preview
  rows, policy version, and provider versions.
- Apply is a guarded checkpoint: it either commits an apply attempt and
  audit event or fails before provider mutation.
