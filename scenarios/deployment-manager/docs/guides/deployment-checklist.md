# Deployment Checklist

Use this checklist before declaring any scenario "deployable" for a tier. Items marked ✅ can be automated later by deployment-manager; until then, keep the checklist in PRDs/app-issue-tracker tasks.

## 1. Scenario Metadata

- [ ] `service.json` lists all resource + scenario dependencies.
- [ ] `deployment.platforms.<tier>` entries exist with fitness scores and requirements.
- [ ] Secrets have deployment strategies defined (see [Secrets Guide](secrets-management.md)).

## 2. Dependency Fitness

- [ ] `scenario-dependency-analyzer` report stored with the scenario (latest run date recorded).
- [ ] Each dependency scored >= tier threshold or has a documented swap/migration plan.
- [ ] Swap decisions tracked in app-issue-tracker if they require engineering changes.

## 3. Packaging Plan

- [ ] `scenario-to-*` packager identified (desktop/mobile/cloud).
- [ ] Bundle manifest (list of binaries/resources/assets) drafted.
- [ ] Installer/updater approach chosen (Electron auto-update, MSI, Terraform+Helm, etc.).

## 4. Secrets

- [ ] Infrastructure secrets stripped from bundles.
- [ ] Generated service secrets recorded in secrets-manager with rotation policies.
- [ ] User secrets prompt/workflow implemented.

## 5. Testing

- [ ] Tier-specific smoke tests defined (desktop launch script, cloud health checks, etc.).
- [ ] Manual validation plan documented until automation exists.

## 6. Documentation

- [ ] Scenario README updated with deployment notes per tier.
- [ ] Relevant example doc under `docs/deployment/examples/` created or updated.
- [ ] Known limitations called out explicitly.

## 7. Approval

- [ ] Deployment-manager (when available) marks the scenario as ready for the tier.
- [ ] Responsible engineer/agent signs off and links to supporting tickets.

If any box remains unchecked, do **not** claim the scenario is deployable for that tier. Capture the gap as an issue so future automation has clear TODOs.
