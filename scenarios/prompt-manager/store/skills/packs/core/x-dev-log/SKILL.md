## Tools focus: X Dev Log Generator

Generate engaging X/Twitter dev log threads by mining activity data from Vrooli scenarios for a given time period. Outputs draft content for review before posting.

---

### **1. Focus Statement**

Mine commits, agent runs, backlog transitions, and issue resolutions from Vrooli scenarios to generate authentic, builder-voice dev log threads for X/Twitter.

---

### **2. When to Use / When Not to Use**

| Use when | Don't use when |
|----------|----------------|
| Creating periodic dev logs (daily/weekly) | Need real-time posting |
| Generating content for launch announcements | Writing non-dev content |
| Summarizing sprint/milestone activity | Need marketing copy |
| Sharing interesting agent behaviors | Auto-posting without review |

---

### **3. Prerequisites**

Required scenarios (check with `vrooli scenario status`):
- **git-control-tower** - commit history and audit logs
- **agent-manager** - agent runs, events, costs
- **swarm-manager** - backlog items and transitions
- **app-issue-tracker** - issues and investigations

Required reading (every run):
- `docs/marketing/catalogs/post-types/text/dev-log.md` — type-level strategic canon (purpose, audience, conversion goal, voice rules, what→why framing, contrarian failure modes). **Load-bearing.**
- `docs/marketing/operating/OPERATING_MODEL.md` — team workflow canon: advertiser lanes draft from evidence, publisher releases, brand-manager owns canon and artifact requests.
- `docs/marketing/strategy/STRATEGY.md` — voice canon (Voice section, Voice samples, Anti-patterns).
- `docs/marketing/methods/post-techniques/essay-shape.md`
- `docs/marketing/methods/post-techniques/hook-vs-body-asymmetry.md`
- `docs/marketing/methods/post-techniques/intro-on-first-mention.md`
- `docs/marketing/methods/post-techniques/inter-post-linkage.md`
- `docs/marketing/methods/post-techniques/no-internal-numbering-externally.md`

---

### **4. Data Sources**

Query these APIs for the specified time period. Keep queries flexible - don't hardcode response shapes so enhancements auto-apply.

| Source | Endpoint | What to Mine |
|--------|----------|--------------|
| git-control-tower | `GET /api/v1/repo/history` | Commits with file changes |
| git-control-tower | `GET /api/v1/audit` | Who/what created commits |
| agent-manager | `GET /api/v1/runs` | Completed agent runs |
| agent-manager | `GET /api/v1/runs/{id}/events` | Tool calls, costs, context |
| swarm-manager | `GET /api/v1/backlog` | Ideas, status transitions |
| app-issue-tracker | `GET /api/v1/issues` | Bugs found/fixed |

Filter by timestamp fields to match the requested period.

---

### **5. Mining Strategy**

**From git-control-tower:**
- Commits with multi-file changes (interesting scope)
- Audit entries showing agent-created commits
- Conventional commit messages revealing intent

**From agent-manager:**
- Runs that failed then recovered (resilience stories)
- High token usage runs (complex tasks)
- Context attachments revealing thought processes
- Tool call patterns showing what agents were doing

**From swarm-manager:**
- Ideas that moved to `completed` status
- Research items that yielded insights
- Backlog items queued for agent execution

**From app-issue-tracker:**
- Bugs investigated and resolved
- Issues with rich artifacts (screenshots, logs)
- Cross-references to fixes/PRs

---

### **6. Interestingness Scoring**

Rank mined items to surface the most engaging content:

| Signal | Score | Why |
|--------|-------|-----|
| Agent recovered from failure | +3 | Shows resilience, interesting narrative |
| Multi-scenario coordination | +3 | Shows system sophistication |
| Large file change count (>10) | +2 | Significant work |
| Idea → completed in same period | +2 | Full arc story |
| High token usage run | +1 | Complex task |
| Bug fix with investigation | +1 | Problem-solving narrative |
| Routine single-file commit | 0 | Too mundane |

Select top items aiming for 3-7 tweets per thread. Balance variety - don't just show commits.

---

### **7. Output Contract**

Return structured JSON for review. The caller is responsible for turning an accepted result into the marketing team's draft/proposal workflow; this skill never publishes directly.

```json
{
  "period": { "from": "YYYY-MM-DD", "to": "YYYY-MM-DD" },
  "thread": [
    {
      "index": 1,
      "content": "Tweet text here...",
      "chars": 245,
      "sources": ["git-control-tower:commit:abc123"]
    }
  ],
  "stats": {
    "commits_analyzed": 47,
    "runs_analyzed": 12,
    "issues_analyzed": 5,
    "backlog_transitions": 8
  },
  "warnings": ["Any sensitive data concerns"]
}
```

---

### **8. Content Generation**

**Tone: Builder/hacker casual**
- Authentic dev voice - wins, struggles, learnings
- Not corporate or marketing-speak
- First person, conversational
- Technical enough to be credible, accessible enough to engage

**Thread structure:**

1. **Opener** (hook the reader)
   - "Been heads down on [scenario] this week. Here's what happened"
   - "Wild debugging session today. Let me walk you through it..."
   - "Shipped [feature] - here's the journey from idea to done:"

2. **Highlights** (2-5 tweets, one per interesting item)
   - Agent story: "Watched the agent [action]. It [unexpected thing]..."
   - Feature: "Finally got [feature] working. Key insight: [learning]"
   - Bug: "Tracked down a gnarly bug in [area]. Root cause was [thing]"
   - Arc: "Started as a note in swarm-manager, now it's [result]"

3. **Closer** (reflection or forward look)
   - "Next up: [what's coming]"
   - "Lessons learned: [insight]"
   - "Building in public is [reflection]"

---

### **9. Guardrails**

**Never expose:**
- API keys, tokens, credentials
- Absolute file paths (use relative)
- User emails or personal data
- Internal hostnames or ports

**Always:**
- Sanitize paths before including
- Output as draft only - no auto-posting
- Show character counts per tweet (warn only if the **hook** tweet exceeds 280)
- Cite sources for traceability

**Rate limiting:**
- Don't overwhelm scenario APIs
- Cache results when re-generating

---

### **10. Verification**

Before presenting output:
1. **Character count** - Flag the hook tweet if >280 chars. Do not flag body tweets for length — the account's tier in `docs/marketing/strategy/CHANNELS.md` sets the real cap, and per `docs/marketing/methods/post-techniques/hook-vs-body-asymmetry.md` body tweets routinely run 400-600+. Counts that stay flat near 280 across the thread indicate over-trimming, not compliance.
2. **Sensitive data scan** - Check for paths, keys, emails
3. **Source validation** - Verify cited sources exist
4. **Tone check** - Does it read like a builder talking?
5. **Thread coherence** - Does it tell a story?

---

### **11. Example Invocation**

```
Generate an X dev log for the last 7 days, focusing on prompt-manager work.
```

The skill will:
1. Query all data sources for 2026-01-22 to 2026-01-29
2. Score items by interestingness
3. Select top 5-6 items
4. Generate thread with opener, highlights, closer
5. Return JSON for review

---

### **12. Output Expectations**

You may:
- Query scenario APIs for activity data
- Generate multiple thread variations
- Suggest edits to improve engagement

You must:
- Output drafts only (never auto-post)
- Include character counts per tweet
- Cite sources for all claims
- Sanitize sensitive data
- Maintain builder/casual tone throughout
