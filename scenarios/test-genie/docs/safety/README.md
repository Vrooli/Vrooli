# Testing Safety

> **CRITICAL**: Before writing any test scripts, read the [Safety Guidelines](GUIDELINES.md) to prevent accidental data loss.

## Quick Safety Checklist

Before committing test scripts:

- [ ] All `rm` commands are guarded with path validation
- [ ] Cleanup functions set/validate variables before deleting
- [ ] Test files are created under `/tmp` or other safe location
- [ ] Wildcard patterns (`*`) are never used with empty variables

## Safety Documentation

| Document | Description |
|----------|-------------|
| [GUIDELINES.md](GUIDELINES.md) | Complete safety rules and patterns |
| [agent-spawning-security.md](agent-spawning-security.md) | Agent spawning security model and limitations |

## Critical Rules

1. **NEVER** use unguarded `rm` commands in test scripts
2. **ALWAYS** validate variables before file operations
3. **SET** critical variables before any early-exit conditions in shell scripts
4. **PREFER** Go tests over bash scripts for new development
5. **REVIEW** shell scripts carefully before committing

> **Note**: For new development, prefer Go tests. Any remaining shell scripts (install/lib helpers) require extra care around cleanup and file operations.

---

*Remember: A test that accidentally deletes production files is worse than no test at all. **Safety first, always.***
