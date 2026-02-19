# Documentation Outlines: Tunnel Manager

> Outlines for scenario documentation to be created during the process step.
> Sources: docs/ARCHITECTURE.md, docs/PROBLEMS.md, docs/RESEARCH.md, README.md

## README.md Sections

The existing README.md is comprehensive. During processing, update it to reflect:

1. **Quick Start** — Add `tunnel-manager init` as the first command (interactive seeding)
2. **CLI Commands** — Add `init` command; update descriptions to match enhanced scope
3. **Management Modes** — Note that both modes are supported from day one with auto-detection
4. **Privilege Escalation** — New section explaining the D-Bus/polkit approach and UX-grantable fallback
5. **Coordination with vrooli-autoheal** — Add lock file details

## RESEARCH.md Updates

The existing docs/RESEARCH.md should be carried forward with additions:

1. **D-Bus systemd API research** — Document findings on using `godbus` for `RestartUnit()` and polkit rules for authorization
2. **User-level systemd for cloudflared** — Document feasibility of running cloudflared as a user service
3. **UX-grantable sudo patterns** — Document how other tools allow granting elevated permissions from a UI/CLI

## PROBLEMS.md Updates

Existing problems with resolution status updates:

| Problem | Resolution |
|---------|------------|
| Q1: Remote-managed tunnel migration | **Resolved**: Both modes from day one, auto-detect (Q2 answer) |
| Q2: cloudflared metrics port | **Action needed**: Configure `--metrics 127.0.0.1:20241` in systemd unit (prerequisite) |
| Q3: External probe reliability | **Resolved**: 3 consecutive failures threshold (Q6 answer confirms 60s interval) |
| Q4: Sudo access for systemctl restart | **Partially resolved**: D-Bus/polkit preferred; UX-grantable sudo as fallback (Q1 answer). Needs implementation research. |
| Q5: Route manifest initial seeding | **Resolved**: Interactive CLI prompt (Q3 answer) |

New problems to add:
- **Lock file cleanup**: What happens if tunnel-manager crashes while holding the restart lock? Need stale lock detection.
- **Cloudflare API read-modify-write races**: Need locking strategy for concurrent API config updates.

## PROGRESS.md Initial Entry

```markdown
# Progress Log

## 2026-02-18 — Enhanced Plan Completed
- All 6 clarifying questions answered and incorporated into plan
- Scope refined: both management modes promoted to P0 (from original P1)
- Privilege escalation strategy defined (D-Bus → user systemd → UX-grantable sudo)
- Lock file coordination with vrooli-autoheal specified
- Ready for processing
```

## docs/ARCHITECTURE.md Updates

The existing architecture doc is thorough. Key updates needed:
- Add mode auto-detection logic to Management Modes section
- Add lock file mechanism to Recovery Engine section
- Add privilege escalation hierarchy to Recovery Engine section
- Update Monitoring Loop to reference 60s default interval (confirmed by Q6)
