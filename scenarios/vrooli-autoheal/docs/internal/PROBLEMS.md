# Known Issues and Problems

## Open Issues

### P0 Blockers
- Host-level recovery grant application still requires an operator-authenticated `sudo vrooli setup`; the existing `/etc/sudoers.d/vrooli-autoheal` is stale, mode `0644`, and does not match the managed wildcard-free grant.

### P1 Issues
- The UI dependency/test-utils gate is not green in the current checkout; Go and platform seam suites are green.
- The full 24-hour data-retention soak remains outstanding.
- Test Genie storage/workflow evidence is unavailable while `storage-manager` fails its own API build at `internal/validation/hygiene_common.go:19`.

### P2 Minor Issues
_None at initialization_

## Deferred Ideas

### Not In Scope (Intentionally Excluded)
- **Full metrics collection**: system-monitor handles detailed metrics; autoheal focuses on health/recovery
- **Alerting/paging**: Use dedicated alerting scenarios for notifications
- **Non-Vrooli service management**: Only manages Vrooli resources, scenarios, and OS watchdog

### Future Consideration
- **AI-powered root cause analysis** (OT-P2-005): Use Ollama to analyze failure patterns
- **Custom check plugins** (OT-P2-004): Allow external health check definitions
- **Certificate monitoring** (OT-P2-001): Check SSL cert expiration
- **Display manager health** (OT-P2-002): GDM/lightdm/sddm monitoring

## Architecture Decisions

### ADR-001: CLI-based resource/scenario management
**Decision**: Use `vrooli resource/scenario` CLI commands instead of direct API calls
**Rationale**: CLI already handles authentication, port discovery, and error handling; reduces code duplication
**Trade-offs**: Slightly higher latency than direct calls; dependency on vrooli CLI being installed

### ADR-002: In-memory state with SQLite persistence
**Decision**: Keep current health state in memory and persist structured history to SQLite.
**Rationale**: Fast local status queries, durable history across restarts, no separate database resource required.
**Trade-offs**: SQLite write volume and retention must be bounded for small hosts; snapshots are fingerprint-deduplicated.

### ADR-003: OS watchdog keeps autoheal running
**Decision**: Install OS-level service (systemd/launchd/Windows) that monitors autoheal itself
**Rationale**: Ensures autoheal survives even its own crashes; brings system back after reboots
**Trade-offs**: Requires elevated privileges for installation on some platforms

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| WSL platform detection edge cases | Medium | Low | Test explicitly on WSL1 and WSL2 |
| Watchdog installation fails without sudo | Medium | Medium | Graceful degradation with warning |
| Circular dependency (autoheal monitors scenarios that monitor autoheal) | Low | Medium | Exclude autoheal from monitored scenario list |
| Too-aggressive restarts cause thrashing | Low | High | Add restart cooldown and max retry limits |

## UX Issues

### 2026-05-15: App-shell overlap and mobile overflow regression fix

#### Fixed in this pass
- Replaced the dense inline header/tab implementation with an explicit app-shell layout contract so header, navigation, and main content have stable ownership and selector-backed landmarks.
- Added mobile layout regression coverage for shell/header/content ordering, summary-grid responsiveness, selector-backed tabs, and browser-level bounding-box validation at 390px, 360px, and desktop widths.
- Removed the visible horizontal overflow seen in the mobile dashboard by tightening root overflow containment, dashboard summary-card sizing, health-group containers, and variable-length check-card callouts.
- Migrated high-traffic tick notices and check-card importance callouts to the shared `Notice` composite, reducing one-off callout styling in the dashboard path.
- Moved additional shared status visuals (`StatusIcon`, `StatusSparkline`, `ActionHistory`, action result icons) onto semantic theme tokens instead of raw palette classes.

#### Remaining UX debt
- The tab rail is touch-scrollable on very narrow screens; a future pass could add a compact overflow affordance or segmented menu if users miss offscreen tabs.
- `SystemProtection.tsx` and `CheckDetailModal.tsx` remain high-density components and should still be split by responsibility in a follow-up architecture pass.
- The comprehensive scenario runner still depends on healthy external orchestration (`test-genie` + runtime URL discovery); this pass records its environment failure separately from the UI gates.

### 2026-02-18: Header clarity + mobile health-check list pass

#### Fixed in this pass
- Removed the redundant global status badge from the app header so status emphasis stays in dashboard/trends where detailed context already exists.
- Removed the ambiguous header `Auto/Manual` toggle and moved refresh-mode control into `Settings > General` with explicit explanatory text.
- Added a dedicated `Dashboard Refresh` settings section with a labeled switch and mode state (`Enabled` / `Manual mode`) for clearer behavior.
- Reworked health-check group rendering for narrow viewports into a bordered, divided list pattern on mobile while preserving card presentation on `sm+` breakpoints.
- Updated app-level refresh UX test coverage to assert the header toggle removal and retained settings access.

#### Remaining UX debt
- Dashboard group headers still use fairly dense metadata rows; a follow-up could tighten icon/text hierarchy further for very small screens.
- The full comprehensive scenario test suite is long-running; this pass validated targeted UI tests and TypeScript type-checking only.

### 2026-02-18: Shared primitive/mobile containment cleanup pass

#### Fixed in this pass
- Updated shared modal primitive layout to improve small-screen behavior: bottom-aligned presentation on mobile, safe-area aware overlay padding, and tighter adaptive modal corner/height constraints.
- Refactored `CodePreview` to token-first styling and improved narrow-width interaction ergonomics (copy action spacing, label truncation, and border/surface consistency).
- Consolidated duplicate relative-time formatting in check details by reusing `ui/src/lib/utils.ts` for coherence and reduced drift risk.
- Improved docs reader containment for constrained viewports: tighter mobile padding, adaptive sidebar/document max heights using dynamic viewport units, and safe-area bottom spacing.
- Added explicit `aria-label` values to compact app-shell icon controls for clearer accessibility affordance on mobile.

#### Remaining UX debt
- Uptime chart tests still emit SVG-tag warnings in jsdom mocks (non-blocking but noisy for CI signal).
- Docs markdown still uses a custom regex-based parser; future hardening should move to a stricter parser/sanitizer path for long-term rendering reliability.

### 2026-02-18: Dashboard and modal responsiveness/coherence pass

#### Fixed in this pass
- Consolidated repeated relative-time formatting into shared utility (`ui/src/lib/utils.ts`) and reused in dashboard/trends surfaces.
- Refined app shell header for mobile: wrapped controls, reduced title density, scroll-safe tab row, and selector-backed tab test IDs.
- Reworked dashboard health-group headers and cards for clearer scan hierarchy and mobile-safe metadata wrapping.
- Updated events timeline and action buttons to use theme token styling instead of legacy slate/white color classes.
- Improved settings and check-detail modal layouts for constrained viewports: scroll-safe tabs, wrapped action rows, compact spacing, and responsive stats grids.
- Made docs and trends pages adaptive on narrow viewports (stacked layouts, overflow-safe controls, edge-safe padding).

### 2026-02-18: UX polish and coherence follow-up (mobile + dashboard cleanup)

#### Fixed in this pass
- Consolidated repeated dashboard health-group rendering (`critical`, `warning`, `ok`) into a single mapped section component for lower duplication and easier maintenance.
- Improved app-shell usability on small screens: action row now wraps cleanly, auto-refresh mode label remains visible on mobile, and tab test IDs are fully selector-registry backed.
- Improved docs navigation on mobile with progressive disclosure (`Browse documents` toggle), searchable scroll-constrained nav, and automatic nav collapse after document selection.
- Fixed content containment for variable-length text across dashboard and trends lists (check IDs, messages, sub-check details) to prevent clipping/truncation overflow.
- Updated system-protection install/uninstall action groups to stack safely on narrow viewports and expose a labeled uninstall action.
- Tuned docs markdown viewport behavior to keep article scrolling usable on mobile (`max-h` adaptive by breakpoint).

#### Remaining UX debt
- Uptime chart tests still emit SVG-tag warnings in jsdom mocks (non-blocking but noisy for CI signal).
- Some components outside this pass still retain legacy color classes and should be migrated to token-first styling for full coherence.

## 2026-02-17: UI lint cycle detection dependency gap

### Problem
React stability pass completed with runtime status normalization and defensive rendering guards. Remaining gap: eslint import cycle rule (import/no-cycle) is not enabled because eslint-plugin-import and eslint-import-resolver-typescript are not present in scenario UI dependencies, and dependency installation was not performed in this pass.

---
## Work ladder

- Rung: W0
- Evidence: no swarm-manager goal named or describing `vrooli-autoheal` was found by the required named-mention search; contract comparison is unverifiable.
- Blocker: no approved goal to compare against `PRD.md`.
- Measured: 2026-08-05
