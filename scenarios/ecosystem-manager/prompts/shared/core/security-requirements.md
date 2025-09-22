# Security & Standards Enforcement

For resources:
```bash
resource-auditor audit <resource-name> --timeout 240 > /tmp/audit-<resource-name>.json
jq '{security: .security.outcome, standards: .standards.outcome}' /tmp/audit-<resource-name>.json
```

For scenarios:
```bash
scenario-auditor audit <scenario-name> --timeout 240 > /tmp/audit-<scenario-name>.json
jq '{security: .security.outcome, standards: .standards.outcome}' /tmp/audit-<scenario-name>.json
```