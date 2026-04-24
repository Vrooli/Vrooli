# Web-Console Release Readiness — Orchestration Summary

## Strategic Rationale

Web-console is the business bundle's second headliner per `docs/monetization/catalog/base/business.md`, paired with Git Control Tower. The bundle cannot ship paid until both headliners are release-ready. GCT has six dedicated initiatives tracking its readiness; web-console previously had none — its specific items were scattered either under `continuous-audio-platform` (which is really about audio extraction, not web-console paid-release) or standalone.

This initiative exists to provide a single tracking surface for web-console's paid-release readiness, without cannibalizing `continuous-audio-platform` by relocating its items.

Origin: director-swarm portfolio-manager decision `dec-1776982737575948642` accepted on 2026-04-24 vision walk (option A — create the initiative). Decision was deferred from the 2026-04-23 walk pending the coverage-visibility plan (`docs/plans/portfolio-manager-scenario-coverage-implementation-plan.md`, shipped 2026-04-24 11:46 EDT).

## Cross-Item Decisions

- **Dependency, not relocation.** The portfolio-manager's original proposal recommended *relocating* five web-console items out of `continuous-audio-platform`. The operator rejected relocation on the grounds that `continuous-audio-platform` is a coherent concept and stripping items from it would weaken the tracking surface. Instead, this initiative declares `depends_on: ["continuous-audio-platform"]`. The audio items stay where they are; web-console-readiness won't complete until continuous-audio-platform does. See friction F16 — initiative `depends_on` display gap in CLI/UI, not a data-model issue.
- **Only truly standalone items are pulled in.** `execute/web-console-tts-code-tick-handling` was the only web-console-specific item outside `continuous-audio-platform` and not attached to another initiative.
- **Scope includes monetization, auth, LPBS, docs-in-UI.** Pricing/billing integration is a must-have for paid release. LPBS registration is required because the business bundle's subscription-tier pipeline runs through LPBS — without declaring web-console there, tiers cannot gate audio features by allocation. Auth/security review and in-UI documentation are standard paid-release concerns. These should be expanded into items during the workshop phase, not pre-authored here.
- **Convergence items (spec-sync, documentation-health) are always-run, not case-by-case.** Operator's initial instinct was that documentation approach might depend on the scenario's state; on closer reading these are pattern-constant regardless of the branch taken by the production-readiness audit gate.

## Sequencing Notes

The intended item order (to be workshopped — this is not prescriptive):

1. **GATE:** `research/web-console-production-readiness-audit` — classifies whether the remaining work is progress-path (features not yet built) or monetization-hardening-path (ready to wrap for paid release). Must run first; branches the rest.
2. **Dependency wait:** continuous-audio-platform completion + validation. Expressed via initiative-level `depends_on`, not an item.
3. **Branch: monetization/hardening items** — pricing/billing integration, LPBS registration + tier config, auth/security review, in-UI docs.
4. **Convergence:** spec-sync run, documentation-health run.

## Pattern Note for Future Release-Readiness Initiatives

This is the FIRST release-readiness initiative shaped at this grain. Per the operator's `feedback_duplicate_before_extract.md` rule, we are intentionally NOT authoring a reusable release-readiness template skill today. The next release-readiness initiative (likely GCT's post-cluster readiness phase) gives us data point #2. After two or three instances we extract the pattern into a skill. Until then, the shape above is a breadcrumb, not a contract.
