# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Identify the target scenario and its docs/ directory.
- Check if manifest.json exists.

## Workflow
1. **Check infrastructure** — Does manifest.json exist? Is docs/ structured correctly?
2. **Audit references** — Verify DOC: comments in code point to real docs.
3. **Audit back-references** — Verify CODE: references in docs point to real code.
4. **Check manifest** — Are all docs registered? Any orphans?
5. **Check for drift** — Do docs describe current behavior or stale behavior?
6. **Report to qa-lead** — Documentation health report with specific fixes.

## Skills
- `prompt-manager skill read documentation-health` — Full methodology.
- `prompt-manager skill read visited-tracker-tools` — Track visited files.

## Coordination
- Receive audit scope from qa-lead.
- Report documentation findings to qa-lead.
- Documentation improvements feed into future teams having better context.
