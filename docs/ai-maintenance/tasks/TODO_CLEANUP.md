## 🚀 Overview of the new workflow

You will **scan, classify, and act** on every warning comment.
*If the fix is quick* (≤10 min) — do it now and delete the comment.
*If not* — create a short **Plan-of-Action (PoA)** entry in `docs/tasks/staged.md`, give the TODO a machine-readable **slug**, and replace the bare comment with the enriched form.
This keeps the codebase clean while pushing heavier work into a visible staging queue. ([gist.github.com][1], [jetbrains.com][2], [reddit.com][3], [gitlab.com][4])

---

## 1 Standard comment pattern (revised)

```ts
// TODO[SLUG]: one-line summary (see docs/tasks/staged.md for PoA)
```

* **SLUG** → `TD-YYYYMMDD-<kebab-summary>` (e.g. `TD-20250616-queue-retry-logic`).
  *TD = Tech Debt.*
* Keep the summary under 60 chars; details live in the PoA record.

---

## 2 Plan-of-Action file format

Append to `docs/tasks/staged.md`:

```md
# <Title Case Summary>
Slug: TD-YYYYMMDD-kebab-summary
Priority: HIGH | MEDIUM | LOW
Status: TODO | IN_PROGRESS | DONE
Dependencies: <comma-separated slugs or “None”>
ParentTask: <slug or “None”>

**Description:**  
One or two concise paragraphs explaining *why* the task matters.

**Current Implementation Analysis:**  
- Bullet list of relevant files, patterns, drawbacks.

**Key Deliverables:**

**Phase 1: <name>**
- [ ] Task checkbox list

**Phase 2: <name>**
- [ ] …

**Technical Notes:**  
(optional) Design choices, trade-offs, libraries.

**Success Criteria:**  
- Observable, testable outcomes.

**Migration Strategy:**  
(optional) Steps for rolling out safely.

```

Place newest PoAs at the **top** of the file to keep the list fresh.

---

## 3 Step-by-step workflow

| # | What you do                                             | Commands / notes                                                |                    |     |      |            |                      |
| - | ------------------------------------------------------- | --------------------------------------------------------------- | ------------------ | --- | ---- | ---------- | -------------------- |
| 1 | **Scan** for raw warning comments                       | \`rg -n "(TODO                                                  | FIXME              | BUG | HACK | DEPRECATED | XXX)" packages/...\` |
| 2 | **Decide** quick-fix vs PoA                             | If ≤10 min → fix & delete comment. Otherwise continue.          |                    |     |      |            |                      |
| 3 | **Generate slug**                                       | `date +%Y%m%d` + terse summary → `TD-YYYYMMDD-…`                |                    |     |      |            |                      |
| 4 | **Write PoA** to `docs/tasks/staged.md` (template §2)   | Keep it concrete & testable.                                    |                    |     |      |            |                      |
| 5 | **Replace original comment** with enriched pattern (§1) | e.g. `// TODO[TD-20250616-queue-retry-logic]: improve back-off` |                    |     |      |            |                      |
| 6 | **Update AI\_CHECK** at top of the edited file          | \`// AI\_CHECK: TODO\_CLEANUP=\<count+1>                        | LAST: 2025-06-16\` |     |      |            |                      |
| 7 | **Verify**                                              | `pnpm test && pnpm lint` (must include `no-warning-comments`).  |                    |     |      |            |                      |

---

## 4 Automation & CI gates

### Linting

* ESLint rule `no-warning-comments` is enabled **error-level** for bare TODOs (`TODO `, `FIXME `, …). ([eslint.org][5])
* Acceptable pattern is `TODO[SLUG]:` — anything else breaks the build.

### Staged-plan checker

Add `scripts/check-staged.js` that:

1. Parses every `TODO[SLUG]`.
2. Confirms a corresponding `## SLUG` header exists in `docs/tasks/staged.md`.
3. Fails if any PoA is >90 days old (configurable). ([softwareengineering.stackexchange.com][8], [softwareengineering.stackexchange.com][8])

### Optional static tools

* **todocheck** (`preslavmihaylov/todocheck`) validates TODO ↔ issue IDs; can be adapted for PoA slugs. ([github.com][6])
* **todo-reminder** CLI can run on a cron branch to nag owners. ([github.com][9])

---

## 5 Prioritisation heuristic

1. **Expired PoAs** (>90 days) = top priority. ([softwareengineering.stackexchange.com][8])
2. Bare TODOs that touch **critical paths** (payments, auth).
3. Largest impact/effort ratio based on PoA “Effort” estimate.
4. Finally, low-impact chores and refactors.

---

## 6 Example in practice

**Before**

```ts
// TODO handle Redis reconnect logic better
await redisClient.connect();
```

**After**

```ts
// TODO[TD-20250616-redis-reconnect]: handle Redis reconnect logic better
await redisClient.connect();
```

`docs/tasks/staged.md` gains:

```
## Redis reconnect logic better
Slug: TD-20250616-redis-reconnect
Priority: HIGH
Status: TODO
Dependencies: None
ParentTask: None

# Continued...
```

---

## 7 Fast checklist before moving on

* [ ] Bare warning comments removed or upgraded to `TODO[SLUG]:`.
* [ ] PoA appended & committed (leave uncommitted for human review).
* [ ] ESLint & staged-checker green.
* [ ] `AI_CHECK` updated for `TODO_CLEANUP`.
* [ ] One-line stdout summary per file.

---

## 8 Reference & further reading

* ESLint `no-warning-comments` docs. ([eslint.org][5])
* Stack Exchange thread on TODO deadlines in CI. ([softwareengineering.stackexchange.com][8])
* **DEBT.md** pattern for visible tech-debt lists (Laserlemon Gist). ([gist.github.com][1])
* JetBrains TODO tool-window for quick scans. ([jetbrains.com][2])
* Reddit devs on linking TODOs to Jira tickets. ([reddit.com][3])
* GitHub issue improving TODO lint UX. ([github.com][10])
* Datadog rule: TODO/FIXME must have ownership. ([docs.datadoghq.com][7])
* GitLab discussion on auto-creating issues from TODOs. ([gitlab.com][4])
* Steve Grunwell blog on extracting TODOs → tickets. ([stevegrunwell.com][11])
* Visual Studio auto-generate doc comments (Copilot). ([devblogs.microsoft.com][12])
* Wiley “Code2Tree” paper on auto‐comment gen. ([onlinelibrary.wiley.com][13])
* DEV article “TODO: write a better comment”. ([dev.to][14])
* Pixelfreestudio guide on integrating code review tools in CI. ([blog.pixelfreestudio.com][15])
* Tenable guide on CI scanning for web-app code. ([tenable.com][16])
* Aikido security – automated CI/CD scanning. ([aikido.dev][17])

---

### Remember

Every TODO is either a **10-minute fix** or a **well-defined ticket**.
If it’s neither, make it so — and make sure the next person who sees the code knows exactly what to do next.

[1]: https://gist.github.com/laserlemon/211fd39fcb46dd71b48a?utm_source=chatgpt.com "Keeping Track of Technical Debt - GitHub Gist"
[2]: https://www.jetbrains.com/help/idea/todo-tool-window.html?utm_source=chatgpt.com "TODO tool window | IntelliJ IDEA Documentation - JetBrains"
[3]: https://www.reddit.com/r/ExperiencedDevs/comments/126bbfk/how_does_your_team_keep_track_of_ideastech/?utm_source=chatgpt.com "How does your team keep track of ideas/tech debt/easy wins? - Reddit"
[4]: https://gitlab.com/gitlab-org/gitlab-foss/-/issues/25200?utm_source=chatgpt.com "Detect TODOS in the code and create issues out of the list - GitLab"
[5]: https://eslint.org/docs/latest/rules/no-warning-comments?utm_source=chatgpt.com "no-warning-comments - ESLint - Pluggable JavaScript Linter"
[6]: https://github.com/preslavmihaylov/todocheck?utm_source=chatgpt.com "presmihaylov/todocheck: A static code analyser for annotated TODO ..."
[7]: https://docs.datadoghq.com/security/code_security/static_analysis/static_analysis_rules/python-best-practices/comment-fixme-todo-ownership/?utm_source=chatgpt.com "TODO and FIXME comments must have ownership - Datadog Docs"
[8]: https://softwareengineering.stackexchange.com/questions/349808/todo-comments-with-deadlines?utm_source=chatgpt.com "continuous integration - TODO comments with deadlines?"
[9]: https://github.com/leo108/todo-reminder?utm_source=chatgpt.com "leo108/todo-reminder: A command-line tool that scans ... - GitHub"
[10]: https://github.com/eslint/eslint/issues/12327?utm_source=chatgpt.com "no-warning-comments should include the comment itself in ... - GitHub"
[11]: https://stevegrunwell.com/blog/extract-todos-from-a-codebase/?utm_source=chatgpt.com "Steal This Idea: Extract TODOs from a Codebase | Steve Grunwell"
[12]: https://devblogs.microsoft.com/visualstudio/introducing-automatic-documentation-comment-generation-in-visual-studio/?utm_source=chatgpt.com "Introducing automatic documentation comment generation in Visual ..."
[13]: https://onlinelibrary.wiley.com/doi/10.1155/2022/6350686?utm_source=chatgpt.com "Code2tree: A Method for Automatically Generating Code Comments"
[14]: https://dev.to/adammc331/todo-write-a-better-comment-4c8c?utm_source=chatgpt.com "//TODO: Write a better comment - DEV Community"
[15]: https://blog.pixelfreestudio.com/how-to-integrate-code-review-tools-into-your-ci-cd-pipeline/?utm_source=chatgpt.com "How to Integrate Code Review Tools into Your CI/CD Pipeline"
[16]: https://www.tenable.com/blog/web-app-scanning-101-what-security-pros-need-to-know-about-cicd-pipelines?utm_source=chatgpt.com "CI/CD Pipelines and Web App Scanning - Tenable"
[17]: https://www.aikido.dev/features/ci-cd-pipeline-security?utm_source=chatgpt.com "Secure Your CI/CD Pipeline - Aikido"
