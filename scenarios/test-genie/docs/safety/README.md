# Testing Safety

> **CRITICAL**: Before writing any test scripts, read the [Safety Guidelines](GUIDELINES.md) to prevent accidental data loss.

## Quick Safety Checklist

Before committing test scripts:

- [ ] All `rm` commands are guarded with path validation
- [ ] BATS setup() sets variables before skip conditions
- [ ] BATS teardown() validates variables before cleanup
- [ ] Test files are created under `/tmp` or other safe location
- [ ] Wildcard patterns (`*`) are never used with empty variables

## Safety Documentation

| Document | Description |
|----------|-------------|
| [GUIDELINES.md](GUIDELINES.md) | Complete safety rules and patterns |
| [bats-teardown-bug.md](bats-teardown-bug.md) | Real incident case study |

## Critical Rules

1. **NEVER** use unguarded `rm` commands in test scripts
2. **ALWAYS** validate variables before file operations
3. **SET** critical variables before skip conditions in BATS
4. **PREFER** Go tests over bash scripts for new development
5. **REVIEW** shell scripts carefully before committing

> **Note**: For new development, prefer Go tests which have safer patterns. Shell scripts are still used for CLI testing (BATS) but require extra care.

---

*Remember: A test that accidentally deletes production files is worse than no test at all. **Safety first, always.***
