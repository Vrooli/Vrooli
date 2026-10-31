# Quality Health

Quality Health is Vrooli's static-quality authority for lint, type-safety, quality contracts, suppressions, command execution evidence, and safe config autofix previews.

It is being built as a dedicated scenario so Test Genie can delegate one `quality` phase instead of carrying native lint/type logic and hidden Scenario Auditor -> Tidiness Manager policy chains.

## What You Get

- Code Facts based surface discovery.
- Language/framework contract registry for static quality.
- Rule-parity preservation for the existing Tidiness Manager / Scenario Auditor type-safety checks.
- Agent-readable findings with stable IDs, evidence, remediation, and maturity.
- Safe `fix-config` preview/apply workflow for deterministic config edits.
- Future Test Genie `quality` phase provider.
- Operator UI for audit overview, surface breakdown, findings triage, contract detail, command evidence, and autofix preview.

## Documentation Map

- [PRD](PRD.md) defines operational targets.
- [Start Here](docs/START-HERE.md) tracks generated-scenario orientation.
- [Architecture](docs/concepts/ARCHITECTURE.md) defines the system boundary.
- [Domains](docs/concepts/DOMAINS.md) defines implementation domains.
- [Quality Contracts](docs/reference/quality-contracts.md) preserves rule-parity policy.
- [Finding Schema](docs/reference/finding-schema.md) defines normalized output.
- [Autofix](docs/reference/autofix.md) defines mutation safety.

## Local Commands

```bash
make orient
make test
vrooli scenario requirements validate quality-health
```

The scenario CLI exposes:

```bash
quality-health audit <scenario> --json
quality-health contracts list --json
quality-health explain <finding-id> --scenario <scenario>
quality-health fix-config <scenario> --dry-run --json
```

## UI

The generated `notes` reference domain has been removed. The dashboard now opens directly to the Quality Health audit workbench and consumes the same `AuditService` model used by the CLI and future Test Genie provider.
