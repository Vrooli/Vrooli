# Problems & Known Issues

> **Last Updated**: 2025-11-24 (Current Agent)  
> **Status**: Factory/template split hardened; generator produces runnable samples; requirements schema now clean

## Open Issues

### 🟠 Factory validation coverage gaps
- **What**: Factory UI/API flows aren’t yet validated end-to-end in automation; manual checks covered template list and generation, but no UI smoke or generated-scenario runtime validation is recorded.
- **Impact**: Drift between factory dashboard guidance and actual generation/customization behavior may go unnoticed.
- **Next**: Run `vrooli scenario ui-smoke landing-manager` and manually trigger generation/customize paths; record results in PROGRESS.md.

### 🟠 Agent customization dependencies
- **What**: `customize` endpoint files issues to app-issue-tracker and is exposed in the factory UI; end-to-end agent flow (issue creation + investigation) hasn’t been validated recently.
- **Impact**: Users may believe an agent is working when downstream automation is unhealthy; silent failure risk.
- **Next**: Keep app-issue-tracker healthy, run a full customize request against a generated scenario, and capture the resulting issue/run IDs for verification. No on-box model dependency.

### 🟡 Template validation drift
- **What**: Template PRD and requirements were out of sync on P1 design/branding status; checkboxes now updated, but remaining targets (multi-template, advanced analytics, etc.) still lack validation.
- **Impact**: Risk of over-reporting readiness if future edits don’t land in the template payload.
- **Next**: Keep runtime changes inside `scripts/scenarios/templates/saas-landing-page/payload`; ensure requirement updates remain paired with PRD status changes.
