# Silent Founder OS - Sales Qualification Script

## Purpose

This script helps a support agent (human or AI) quickly understand a prospect's needs and recommend appropriate workflows or features. The goal is efficient qualification, not persuasion—if Silent Founder OS isn't a fit, acknowledge that early.

---

## Qualification Questions

### 1. What's your role?

**Why ask:** Different roles have different primary use cases.

| Answer | Likely Focus |
|--------|--------------|
| Solo founder / solopreneur | Automation + marketing replays |
| Freelance developer | Client automation + testing |
| QA engineer | End-to-end testing + evidence |
| Marketing / growth | Demo videos + onboarding content |
| DevOps / platform | CI/CD integration + scheduled tests |
| Other technical role | Varies—probe further |

**Follow-up if unclear:** "Are you primarily building products, testing them, or creating content about them?"

---

### 2. What's your main goal with automation?

**Why ask:** Identifies primary use case and feature emphasis.

| Answer | Recommended Focus |
|--------|-------------------|
| "Automate repetitive browser tasks" | Workflow recording, scheduling |
| "Test my app automatically" | Assertions, CI/CD integration, evidence capture |
| "Create product demos" | Replay styling, video export |
| "All of the above" | Full workflow: record → test → export |
| "Not sure yet" | Start with recording and explore |

**Follow-up:** "Can you give me an example of a task you'd want to automate?"

---

### 3. How big is your team?

**Why ask:** Affects collaboration needs and pricing considerations.

| Answer | Notes |
|--------|-------|
| Just me | Core use case—solo founder |
| 2-5 people | Small team, async workflows |
| 6-20 people | Consider whether shared workflow libraries are needed |
| 20+ | May need enterprise features not yet available |

**If large team:** "Silent Founder OS is currently designed for individuals and small teams. We don't have enterprise features like role-based access or shared workflow libraries yet. Is that a blocker for you?"

---

### 4. What tools are you currently using?

**Why ask:** Understand existing investment and migration considerations.

| Answer | Response |
|--------|----------|
| Playwright / Selenium / Cypress | "Vrooli Ascension complements those for visual QA and demo generation. For testing, it's less code-centric—good for teams that want to reduce maintenance." |
| Screen recording tools (Loom, etc.) | "Vrooli Ascension generates replays from real workflows, so you don't need to record separately." |
| No-code tools (Zapier, etc.) | "We're browser-focused, not API-focused. If you need browser automation specifically, we're a fit." |
| Nothing yet | "Great—you can start fresh with a unified workflow." |

---

### 5. What's your primary use case?

**Why ask:** Map to specific features and workflows.

**Option A: Automation**
- Emphasize: Recording, scheduling, timeline editing
- Example workflows: Lead research, CRM updates, data entry, invoice downloads

**Option B: Testing**
- Emphasize: Assertions, evidence capture, CLI/API integration
- Example workflows: Smoke tests, regression checks, cross-browser validation

**Option C: Marketing**
- Emphasize: Replay styling, video export, professional output
- Example workflows: Product demos, onboarding walkthroughs, feature announcements

**Option D: All three**
- Emphasize: Unified workflow (record once, test with assertions, export for marketing)
- This is the core value proposition

---

### 6. What's your timeline?

**Why ask:** Gauge urgency and trial intent.

| Answer | Approach |
|--------|----------|
| Exploring / researching | Point to docs and FAQ, let them self-serve |
| Need something this week | Help with specific first workflow |
| Long-term evaluation | Explain subscription model and guarantee |

---

### 7. Do you have a specific workflow in mind?

**Why ask:** Concrete use cases lead to faster activation.

**If yes:** Walk through how they'd build that workflow:
1. Navigate to site
2. Perform actions
3. Select and save as workflow
4. Add assertions if testing
5. Schedule if recurring
6. Export replay if marketing

**If no:** Suggest starting with something simple:
- "Try automating your morning check of a dashboard or news site"
- "Record yourself filling out a form you do repeatedly"

---

### 8. Any concerns or questions?

**Why ask:** Surface objections and address directly.

**Common concerns:**

| Concern | Response |
|---------|----------|
| Privacy / data security | "Runs locally, no data collection, open source. Your data never leaves your machine." |
| Learning curve | "No code required. Record by browsing, edit visually." |
| Pricing | "Single subscription covers all apps, including future releases. 30-day guarantee." |
| Maintenance burden | "Visual timeline editing is simpler than maintaining code-based scripts." |
| Missing features | "Check the roadmap in the docs. Submit requests at vrooli.com/feedback." |

---

## Qualification Outcomes

### Good Fit
- Solo or small team
- Browser-based workflow needs
- Wants automation + testing or marketing in one tool
- Comfortable with desktop software

**Next step:** Point to download at vrooli.com, offer to answer specific setup questions.

### Possible Fit
- Larger team but individual contributor use case
- Already invested in Playwright/Selenium but wants visual QA
- Unsure of specific use case

**Next step:** Suggest trying with one specific workflow, mention the 30-day guarantee.

### Not a Fit
- Needs cloud-only or mobile automation
- Requires real-time collaboration features
- Enterprise compliance requirements we don't meet
- Needs API automation (not browser-based)

**Next step:** Acknowledge honestly, suggest they look at specialized tools for their needs.

---

## Quick Qualification Summary

| Question | Purpose |
|----------|---------|
| Role? | Identify persona and primary use case |
| Main goal? | Determine feature emphasis |
| Team size? | Assess fit with solo/small team focus |
| Current tools? | Understand migration or complement needs |
| Primary use case? | Map to automation / testing / marketing |
| Timeline? | Gauge urgency |
| Specific workflow? | Accelerate activation |
| Concerns? | Surface and address objections |
