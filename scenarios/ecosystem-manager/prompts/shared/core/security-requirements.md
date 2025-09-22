# Security & Standards Enforcement
**Run the violation scanner CLI.**
```bash
scenario-auditor audit <scenario-name> --timeout 240 > /tmp/audit.json
jq '{security: .security.outcome, standards: .standards.outcome}' /tmp/audit.json
```
- The CLI command kicks off security (`/scenarios/<name>/scan`) and standards (`/standards/check/<name>`) jobs, polls until completion or timeout, and captures both results in one JSON blob.
- Outcomes are `completed`, `failed`, `timeout`, or `unknown`; each block includes the raw API payload under `.status`, plus the job id and poll endpoint for later reference.
```