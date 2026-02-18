# Known Problems and Open Questions

## Open Questions

### Q1: Remote-managed tunnel migration
**Status**: Needs decision
**Context**: The current tunnel is locally-managed (config.yml). Switching to remote-managed enables hot-reload but requires a one-time migration. Should this be handled by tunnel-manager's setup step, or documented as a manual prerequisite?
**Options**:
- A) Tunnel Manager handles the migration automatically during first setup
- B) Document as manual prerequisite; Tunnel Manager detects and works with whatever mode it finds
- C) Support both modes from day one, auto-detect current mode

### Q2: cloudflared metrics port
**Status**: Needs configuration
**Context**: cloudflared binds metrics to the first available port in 20241-20245. This makes scraping unreliable. We should set `--metrics 127.0.0.1:20241` in the systemd unit file.
**Action**: Update cloudflared systemd unit before Tunnel Manager development begins.

### Q3: External probe reliability
**Status**: Needs testing
**Context**: External probes go through the full internet path. They may fail due to transient network issues, Cloudflare edge instability, or DNS propagation delays — none of which indicate a tunnel problem. Need to determine appropriate timeout and failure thresholds.
**Proposed approach**: Require 3 consecutive external probe failures before classifying as "tunnel-issue". Single failures logged but not acted upon.

### Q4: Sudo access for systemctl restart
**Status**: Needs configuration
**Context**: `systemctl restart cloudflared` requires sudo. The tunnel-manager process needs passwordless sudo for this specific command. Options:
- A) sudoers rule: `vrooli ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart cloudflared`
- B) Run tunnel-manager API as root (not recommended)
- C) Use a helper script with setuid (not recommended)

### Q5: Route manifest initial seeding
**Status**: Design decision
**Context**: On first run, should the manifest be seeded from the current tunnel config, or should the user define routes manually? Auto-seeding is convenient but may import stale or incorrect routes.
**Proposed approach**: Auto-seed with user confirmation — show discovered routes and ask for approval before persisting.

## Deferred Ideas

### D1: Cloudflare Access integration
Tunnel Manager could also manage Cloudflare Access policies (authentication) for published routes. Deferred because it adds significant complexity and most routes are currently public.

### D2: Per-route bandwidth tracking
cloudflared metrics are aggregate, not per-route. True per-route analytics would require parsing access logs or using Cloudflare's analytics API. Deferred to P2.

### D3: Multi-server tunnel coordination
When Vrooli runs on multiple servers, tunnel routes need to be coordinated across instances. This requires a shared manifest (database or API) and replica-aware routing. Deferred to future architecture phase.
