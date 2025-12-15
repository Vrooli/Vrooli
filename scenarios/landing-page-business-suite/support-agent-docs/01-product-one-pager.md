# Silent Founder OS - Product One-Pager

## Headline

**Build and run a full business solo. No team, no meetings, no social skills required.**

### Subheadline

Silent Founder OS is a suite of cross-platform desktop apps that give solopreneurs the software engineering, automation, and marketing capabilities of a full team—without the overhead.

---

## Who This Is For

- **Solo founders** building products without a co-founder or employees
- **Freelance developers** who want to automate repetitive client work
- **Small teams (1-5 people)** that need enterprise-grade automation without enterprise complexity
- **Technical marketers** who want to produce professional demos without video editing skills
- **QA engineers and developers** who need reliable end-to-end testing without writing Playwright/Selenium code

## Who This Is NOT For

- Large enterprises needing custom integrations or dedicated support contracts
- Non-technical users expecting a no-code drag-and-drop website builder
- Anyone needing real-time collaboration features (this is built for async, solo work)
- Users who require cloud-only solutions (apps run locally on your machine)

---

## Core Problems Solved

1. **Automation is too hard to set up.** Traditional tools require writing code, debugging selectors, and maintaining brittle scripts. Silent Founder OS records your normal browser activity and lets you turn it into automation retroactively.

2. **E2E testing is expensive and fragile.** Selenium and Playwright require dedicated QA engineers. We make testing a byproduct of working—record once, assert forever.

3. **Marketing content takes too long.** Product demos require screen recording, video editing, and design skills. We generate polished, professional replays directly from your automation runs.

4. **Tool sprawl drains productivity.** Separate tools for automation, testing, and marketing mean context switching and duplicate work. Silent Founder OS unifies these workflows.

---

## Silent Founder OS Overview

Silent Founder OS is a subscription bundle of desktop applications designed for individuals who want to build real businesses without hiring teams. Each app in the suite addresses a specific capability gap that solo founders face.

**Current apps:**
- Vrooli Ascension (browser automation, testing, and demo generation)

**Planned apps:**
- Marketing video generator (uses Vrooli Ascension replays as A-roll footage)
- Additional tools for common founder workflows

All current and future apps are included in your subscription at no additional cost.

---

## Vrooli Ascension - Deep Dive

### What It Does

Vrooli Ascension is a desktop application with a built-in browser. As you browse normally, it builds a complete timeline of every action you take—clicks, typing, navigation, scrolling. You can then select any range of actions and save them as a reusable workflow.

These workflows can:
- Run on demand or on a schedule
- Collect screenshots, logs, and network requests at every step
- Include assertions for automated testing
- Generate stylized replays for demos and marketing

### How It Works

1. **Browse normally** inside the Vrooli Ascension browser
2. **Review the timeline** of captured actions
3. **Select and edit** any range to create a workflow
4. **Add assertions** if you want to use it for testing
5. **Run workflows** manually or schedule them
6. **View replays** with professional styling, export as video

The underlying engine supports both Browserless (default) and Playwright for execution, with full telemetry capture including console logs, network requests, and DOM snapshots.

### Primary Use Cases

#### Friction-Free Automation
- Automate lead research, CRM updates, and data entry
- No coding required—just work normally, then automate what you did
- Edit workflows retroactively if something changes

#### End-to-End Testing
- Turn any recorded browser session into a test
- Add assertions for element existence, text content, or attributes
- Run tests on schedule with full screenshot evidence
- Integrate into CI/CD through CLI or API

#### Demos and Marketing Replays
- Generate professional product walkthroughs from real usage
- Apply styling themes (highlights, zoom, cursor trails)
- Export as HTML packages or video (MP4/WEBM)
- No screen recording or video editing required

### Key Differentiators

| Traditional Tools | Vrooli Ascension |
|-------------------|------------------|
| Write code first, then test | Work first, automate later |
| Maintain selector scripts | Visual timeline editing |
| Separate tools for automation vs. testing vs. demos | Unified workflow for all three |
| Steep learning curve | Record → Edit → Run |
| Headless execution only | Full visual replay with styling |

---

## Security, Privacy, and Open Source

**No data collection.** Vrooli Ascension runs entirely on your local machine. Your browser data, recordings, and workflows never leave your computer unless you explicitly export them.

**Open source (AGPL v3).** The full source code is available at [github.com/Vrooli/Vrooli](https://github.com/Vrooli/Vrooli). You can audit the code, contribute improvements, or fork for your own purposes under the license terms.

**No cloud dependency.** Screenshots, logs, and workflow data are stored locally. Optional MinIO integration is available for teams that want object storage, but it runs on your infrastructure.

---

## Cross-Platform Support

- **Linux** (primary development platform)
- **macOS** (Intel and Apple Silicon)
- **Windows** (Windows 10+)

All platforms have feature parity. The app bundles its own Chromium-based browser, so no additional browser installation is required.

---

## Subscription Model

- **Single subscription** covers all apps in the Silent Founder OS bundle
- **Flat pricing** with no per-seat fees
- **Cancel anytime** without penalty; access continues through the end of your billing period
- **30-day money-back guarantee** if you're not satisfied
- **Future apps included** at no additional cost

Pricing details and plan tiers are available at [vrooli.com](https://vrooli.com).

---

## Resources

- **Landing page:** [vrooli.com](https://vrooli.com)
- **Source code:** [github.com/Vrooli/Vrooli](https://github.com/Vrooli/Vrooli)
- **Vrooli Ascension docs:** [github.com/Vrooli/Vrooli/tree/master/scenarios/browser-automation-studio/docs](https://github.com/Vrooli/Vrooli/tree/master/scenarios/browser-automation-studio/docs)
- **License:** [GNU AGPL v3](https://github.com/Vrooli/Vrooli/blob/master/LICENSE)
- **Feedback:** [vrooli.com/feedback](https://vrooli.com/feedback)
- **X/Twitter:** [@vrooliofficial](https://x.com/vrooliofficial)
