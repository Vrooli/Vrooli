## Knowledge Observatory Tools

Use `knowledge-observatory docs` to read, write, search, and maintain scenario documentation through a single CLI.

---

### 1. Read & Write

**Read a scenario doc by type:**
```bash
knowledge-observatory docs read {{TARGET}} {{DOC_TYPE}}
```

Supported types: `problems`, `progress`, `seams`, `invariants`, `assumptions`, `error-semantics`, `security-posture`, `temporal-flows`, `coherence-notes`, `experience-audit`, `quickstart`, `architecture`, `glossary`, `prd`, `readme`, `manifest`.

**Add a structured entry (problems and progress only):**
```bash
knowledge-observatory docs add \
  --scenario {{TARGET}} --doc {{DOC_TYPE}} \
  --title "Brief summary" --body "Details and evidence"
```

For progress entries, also pass `--author` and `--status`.

For all other doc types, read via the CLI then edit the file directly.

**View a doc by path (for files outside the standard types):**
```bash
knowledge-observatory docs view --path "scenarios/{{TARGET}}/docs/internal/{{FILE}}"
```

---

### 2. Search

**Find files by pattern:**
```bash
knowledge-observatory docs search-files --pattern "**/*.md" --scope scenario --scenario {{TARGET}}
```

**Full-text search:**
```bash
knowledge-observatory docs search-text --query "search terms" --scope scenario --scenario {{TARGET}}
```

---

### 3. Health & Maintenance

**Check documentation health:**
```bash
knowledge-observatory docs health {{TARGET}}
```

**Preview stale entry cleanup (problems/progress):**
```bash
knowledge-observatory docs reset --path "scenarios/{{TARGET}}/docs/internal/PROBLEMS.md" \
  --max-age-days 30 --keep-min-entries 5 --preview
```

---

### 4. Quality Guidelines

- **Read before work.** Always read the relevant doc at the start of your loop. The code is the source of truth; verify existing claims before extending.
- **Write after work.** Use `docs add` for problems/progress entries. For all other doc types, edit the file directly after reading its current content.
- **Be specific.** Entries must capture what changed, what evidence supports it, and what remains.
  - Good: "Replaced unsafe JSON.parse in 3 API handlers; retry-path handler still unguarded."
  - Bad: "Fixed some issues."
- **One doc type per skill concern.** Use the doc type your skill designates; don't scatter findings across multiple files.
