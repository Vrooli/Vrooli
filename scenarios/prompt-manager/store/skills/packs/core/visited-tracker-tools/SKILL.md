## Visited Tracker Tools

Use `visited-tracker` to record file coverage, avoid duplicate work, and leave handoff notes that persist across conversations.

---

### 1. Core Commands

**Get the next files to review (use `--pattern` that is applicable to current situation):**
```bash
visited-tracker least-visited \
  --location {{LOCATION}} \
  --pattern "**/*.{ts,tsx,go}" \
  --tag {{TAG}} \
  --limit 5
```

**Check campaign status:**
```bash
visited-tracker status \
  --location {{LOCATION}} \
  --tag {{TAG}}
```

**Record multiple visits with per-file notes (single command, globs supported):**
```bash
visited-tracker visit \
  --location {{LOCATION}} \
  --tag {{TAG}} \
  --file-note "src/pages/**/*.tsx" "Updated headings + ARIA labels" \
  --file-note "src/lib/error-utils.ts" "Replaced unsafe JSON.parse"
```

**Exclude files with reasons (typically done to remove irrelevant or fully completed files from campaign):**
```bash
visited-tracker exclude \
  --location {{LOCATION}} \
  --tag {{TAG}} \
  --file-reason "src/pages/**/*.tsx" "Already clean" \
  --file-reason "src/lib/error-utils.ts" "Non-target utility"
```

**Campaign handoff note:**
```bash
visited-tracker campaigns note \
  --location {{LOCATION}} \
  --tag {{TAG}} \
  --note "Handoff: remaining risks, TODOs, and hotspots"
```

---

### 2. Interpretation + Note Quality

- Prioritize high `staleness_score` and low `visit_count` files first.
- Use `coverage_percent` to track progress toward full coverage.
- **File notes** should capture what changed and what remains.
  - ✅ Good: "Replaced unsafe JSON.parse, still need runtime validation for nested payloads."
  - ❌ Bad: "Made improvements"
- **Campaign notes** should capture global progress and remaining risk hotspots.
  - ✅ Good: "Completed 12/40 files (30%). Watchouts: Settings page state logic, API retry path."
  - ❌ Bad: "Made progress"

---

### 3. Guardrails

- Keep tags consistent per skill (e.g., `ux`, `react-stability`, `cli-steer`) to avoid mixing coverage.
- Prefer a single `visited-tracker visit` command with repeated `--file-note` entries (or `--note` + `--` list) over multiple single-file commands.
- Use glob patterns relative to `--location`; verify they match expected files.
