# Research: Deployment Manager

**Date**: 2025-11-21
**Researcher**: Generator Agent
**Purpose**: Establish uniqueness, identify integration points, and validate deployment-manager as a viable capability

---

## Uniqueness Check

### Repository-Wide Search
**Query**: Scenarios containing "deployment" or "deploy" keywords

**Results**:
- ✅ **No existing "deployment-manager" scenario** found in `/home/matthalloran8/Vrooli/scenarios/`
- 🔍 Found 2 related scenarios:
  - `scenario-to-extension`: Browser extension packager (P0 complete, production-ready)
  - `scenario-to-android`: Android app packager (exists but status unknown)

**Conclusion**: deployment-manager is a unique capability. No overlap with existing scenarios.

---

## Related Scenarios

### Integration Dependencies
These scenarios will be consumed by deployment-manager:

1. **scenario-dependency-analyzer** (`/scenarios/scenario-dependency-analyzer/`)
   - **Status**: Deployed and operational
   - **Purpose**: Provides dependency tree data (resources + scenarios) for fitness scoring
   - **Integration**: API endpoint `/api/v1/health/analysis` (15s timeout)
   - **Data Source**: Reads `.vrooli/service.json` from each scenario
   - **Key Capability**: Recursive dependency traversal, resource/scenario detection
   - **Fitness Contribution**: Provides raw dependency data for platform fitness calculations

2. **secrets-manager** (`/scenarios/secrets-manager/`)
   - **Status**: Active scenario (recent updates visible in git status)
   - **Purpose**: Secret classification, template generation, vault management
   - **Integration**: API + CLI for secret mapping and rotation
   - **Key Capability**: Distinguishes dev-only, user-supplied, vault-managed secrets
   - **Fitness Contribution**: Validates secret availability per deployment tier

3. **app-issue-tracker** (`/scenarios/app-issue-tracker/`)
   - **Status**: Deployed (auto-detected as dependency by scenario-dependency-analyzer)
   - **Purpose**: Creates migration tasks when dependency swaps are approved
   - **Integration**: API for issue creation with metadata (deployment profile ID linking)
   - **Key Capability**: Webhook notifications when linked issues close
   - **Fitness Contribution**: Tracks technical debt from dependency compromises

### Downstream Packagers (scenario-to-*)
deployment-manager orchestrates these specialized packaging scenarios:

1. **scenario-to-extension** (`/scenarios/scenario-to-extension/`)
   - **Status**: ✅ Production-ready (100% P0 complete, 0 security vulnerabilities)
   - **Purpose**: Generates browser extensions (Chrome, Firefox, Edge)
   - **Interface**: API POST `/api/v1/extension/generate`, CLI `scenario-to-extension generate`
   - **Deployment Target**: Tier 2 (Desktop) - browser extension deployment
   - **Fitness Alignment**: High portability, low resource requirements

2. **scenario-to-android** (`/scenarios/scenario-to-android/`)
   - **Status**: Exists in repo (detected by grep)
   - **Purpose**: Generates Android APK/AAB packages
   - **Interface**: Expected API + CLI pattern (to be validated)
   - **Deployment Target**: Tier 3 (Mobile) - Android platform
   - **Fitness Alignment**: Mobile-optimized resource requirements

3. **scenario-to-desktop** (referenced in docs, not yet found)
   - **Status**: Referenced in `/docs/deployment/README.md` as validation driver
   - **Purpose**: Tauri/Electron desktop app packager
   - **Deployment Target**: Tier 2 (Desktop)
   - **Historical Context**: Initial failures exposed need for deployment-manager

4. **scenario-to-ios** (future)
   - **Status**: Not yet implemented
   - **Purpose**: Xcode project + IPA generation
   - **Deployment Target**: Tier 3 (Mobile) - iOS platform

5. **scenario-to-saas** (future)
   - **Status**: Not yet implemented
   - **Purpose**: Docker Compose + Terraform + cloud provider SDKs
   - **Deployment Target**: Tier 4 (SaaS/Cloud)

6. **scenario-to-enterprise** (future)
   - **Status**: Not yet implemented
   - **Purpose**: Helm charts + Kubernetes + compliance overlays
   - **Deployment Target**: Tier 5 (Enterprise/Appliance)

---

## Resource Dependencies

### Required Resources
deployment-manager will integrate with these local resources:

1. **postgres**
   - **Purpose**: Store deployment profiles, configuration history, fitness scores
   - **Schema**: Will need tables for profiles, swaps, validation results, audit logs
   - **Portability Note**: postgres itself has low mobile/desktop fitness (swap candidates: sqlite, duckdb)

2. **claude-code** (or other LLM resource)
   - **Purpose**: AI-powered migration strategy suggestions for dependency swaps
   - **Use Case**: Agent integration for proposing alternative architectures
   - **Portability Note**: High cloud fitness, low mobile fitness (swap candidate: openrouter, groq)

3. **redis** (optional)
   - **Purpose**: Cache fitness scores, dependency tree calculations
   - **Portability Note**: Low mobile fitness (swap candidate: in-memory cache, local storage)

4. **n8n** (optional)
   - **Purpose**: Deployment orchestration workflows, notification hooks
   - **Portability Note**: Low mobile/desktop fitness (swap candidate: direct API calls)

---

## Deployment Tier System

### Tier Definitions (from `/docs/deployment/README.md`)

| Tier | Name | Description | Viability |
|------|------|-------------|-----------|
| 1 | Local/Dev Stack | Full Vrooli + app-monitor Cloudflare tunnel | ✅ Production |
| 2 | Desktop | Windows/macOS/Linux bundles (Tauri/Electron) | ⚠️ Thin client only |
| 3 | Mobile | iOS/Android packages | 🚧 Not started |
| 4 | SaaS/Cloud | DigitalOcean/AWS/bare metal | ⚠️ Needs fitness + secrets |
| 5 | Enterprise | Hardware appliance deployments | 🧭 Vision stage |

### Key Insights from Deployment Hub Docs

1. **The Genesis Problem**:
   - scenario-to-desktop failures exposed that "build an app" is useless without:
     - Dependency intelligence (what resources/scenarios must travel together)
     - Fitness scoring (which dependencies work on target platform)
     - Secret strategy (how to provision credentials per tier)
   - This failure validated the need for deployment-manager

2. **Current State**:
   - Tier 1 (local stack) works via app-monitor + Cloudflare tunnel
   - Tiers 2-5 require deployment-manager to become viable
   - scenario-dependency-analyzer exists but lacks fitness scoring engine
   - secrets-manager exists but lacks tier-specific secret strategies

3. **The Orchestration Loop** (from deployment hub docs):
   ```
   deployment-manager (this scenario)
     → queries scenario-dependency-analyzer (dependency DAG)
     → scores fitness for target tier
     → suggests swaps for blockers
     → coordinates with secrets-manager (secret classification)
     → triggers scenario-to-* packager (platform-specific build)
     → files app-issue-tracker tasks (manual work required)
   ```

4. **Fitness Scoring Requirements**:
   - Must extend `.vrooli/service.json` with `deployment.platforms` metadata
   - Needs resource tallies (memory, CPU, GPU, storage, network)
   - Cascade fitness scores through dependency tree
   - Identify blockers (fitness = 0) and warnings (fitness < 50)

---

## External References

### Deployment Management Systems (Industry Research)

1. **Kubernetes + Helm**
   - **Relevance**: Multi-tier deployment orchestration, health checks, rollbacks
   - **Lesson**: Declarative configuration + templating enables reproducible deployments
   - **Adoption**: Use similar profile-based configuration for deployment-manager

2. **Terraform**
   - **Relevance**: Infrastructure as code, dependency graphs, state management
   - **Lesson**: Explicit dependency declarations prevent deployment order issues
   - **Adoption**: Deployment profiles should declare all dependencies explicitly

3. **Docker Compose**
   - **Relevance**: Service orchestration, environment variable management, volume persistence
   - **Lesson**: Simple YAML format enables rapid local testing before cloud deployment
   - **Adoption**: Consider Compose-like syntax for deployment profile exports

4. **Vercel / Netlify**
   - **Relevance**: One-click deployments, preview environments, automatic rollbacks
   - **Lesson**: UX matters - make deployment approval/trigger frictionless
   - **Adoption**: deployment-manager UI should minimize clicks from analysis → deployment

5. **Mobile App Store Deployment Tools**
   - **Fastlane** (iOS/Android): Automated screenshot generation, metadata management, store submission
   - **Lesson**: Mobile deployments require platform-specific metadata (bundle IDs, provisioning profiles)
   - **Adoption**: Tier-specific settings in deployment profiles

### Fitness Scoring Approaches

1. **Google Lighthouse** (web performance scoring)
   - **Relevance**: 0-100 scoring with breakdown into sub-scores
   - **Lesson**: Users trust simple numeric scores with drill-down details
   - **Adoption**: deployment-manager fitness scores should mirror this UX

2. **npm audit** (dependency vulnerability scoring)
   - **Relevance**: Severity levels (low/medium/high/critical), remediation suggestions
   - **Lesson**: Actionable recommendations matter more than scores alone
   - **Adoption**: Each low-fitness dependency must include swap suggestions

3. **Cloud Provider Pricing Calculators**
   - **Relevance**: Estimate costs based on resource requirements
   - **Lesson**: Users need cost visibility before committing to deployments
   - **Adoption**: deployment-manager should estimate SaaS/cloud tier costs

### Dependency Swapping Precedents

1. **Homebrew** (macOS package manager)
   - **Relevance**: Optional dependencies, conflicts, alternatives
   - **Lesson**: Package managers already solve "swap X for Y based on platform"
   - **Adoption**: deployment-manager can borrow conflict resolution UX

2. **Cargo (Rust)** / **npm (Node.js)** - Feature Flags
   - **Relevance**: Conditional compilation based on target platform
   - **Lesson**: Same codebase can adapt to different deployment contexts
   - **Adoption**: Deployment profiles enable "same scenario, different dependencies"

---

## Gap Analysis

### What Exists
- ✅ scenario-dependency-analyzer (provides dependency tree data)
- ✅ secrets-manager (manages secrets, needs tier-specific strategies)
- ✅ app-issue-tracker (tracks migration tasks)
- ✅ scenario-to-extension (proven packager for browser extensions)
- ✅ Deployment tier documentation (clear 5-tier model)

### What's Missing (deployment-manager's role)
- ❌ Fitness scoring engine (platform-specific scoring rules)
- ❌ Dependency swap suggestion system (alternative matching)
- ❌ Deployment profile management (versioning, export/import)
- ❌ Orchestration layer (coordinates analyzer → secrets → packager)
- ❌ Interactive swap UI (visualize dependencies, approve swaps)
- ❌ Pre-deployment validation (fitness threshold, secret completeness, licensing)
- ❌ Post-deployment monitoring (health tracking, metrics aggregation)
- ❌ Update/rollback automation (blue-green, rolling deploys)

### What's Partially Implemented
- 🟡 service.json metadata schema (exists, needs `deployment.platforms` extension)
- 🟡 scenario-to-* packagers (extension works, others in progress)
- 🟡 CLI interfaces (scenarios have CLIs, need unified deployment CLI)

---

## Validation of Core Assumptions

### Assumption 1: Dependency data is accessible
**Status**: ✅ Validated
**Evidence**: scenario-dependency-analyzer exposes `/api/v1/health/analysis` endpoint (15s timeout)
**Source**: `/scenarios/scenario-dependency-analyzer/.vrooli/service.json` (lines 83-89)

### Assumption 2: Secret management infrastructure exists
**Status**: ✅ Validated
**Evidence**: secrets-manager scenario active (recent commits visible in git status)
**Source**: Git status shows `M scenarios/secrets-manager/PRD.md` and `.vrooli/deployment/deployment-report.json`

### Assumption 3: Issue tracking for migrations is available
**Status**: ✅ Validated
**Evidence**: app-issue-tracker detected as dependency by scenario-dependency-analyzer
**Source**: `/scenarios/scenario-dependency-analyzer/.vrooli/service.json` (lines 291-296)

### Assumption 4: At least one scenario-to-* packager works
**Status**: ✅ Validated
**Evidence**: scenario-to-extension is production-ready (100% P0 complete, 0 vulnerabilities)
**Source**: `/scenarios/scenario-to-extension/PRD.md` (extensive progress history)

### Assumption 5: Deployment tier model is defined
**Status**: ✅ Validated
**Evidence**: 5-tier model documented with current viability per tier
**Source**: `/docs/deployment/README.md` (lines 15-24)

---

## Uniqueness Statement

**deployment-manager is a fundamentally new capability** that does not duplicate existing functionality:

- **Not a packager**: Orchestrates scenario-to-* packagers, doesn't build apps itself
- **Not a dependency analyzer**: Consumes scenario-dependency-analyzer data, adds fitness scoring
- **Not a secret manager**: Integrates with secrets-manager for tier-specific strategies
- **Not an issue tracker**: Uses app-issue-tracker to manage migration work

**deployment-manager is the control tower** that:
1. Understands deployment contexts (5 tiers with different constraints)
2. Scores dependency fitness for each context
3. Guides users through swap decisions
4. Orchestrates the deployment pipeline
5. Monitors deployments across all tiers

No existing scenario performs this orchestration role.

---

## Next Steps for Improver Agents

1. **Implement fitness scoring engine** (core algorithm for deployment-manager)
   - Rules engine for portability, resource requirements, licensing, platform support
   - Cascade scoring through dependency trees
   - Sub-score breakdowns (memory, CPU, GPU, storage, network)

2. **Design swap suggestion system**
   - Maintain swap database (postgres → sqlite, ollama → openrouter, etc.)
   - Match blockers/low-fitness deps to alternatives
   - Calculate fitness delta and trade-offs

3. **Build deployment profile schema**
   - JSON schema for profiles (target tier, swaps, secrets, platform settings)
   - Versioning system with diffs
   - Export/import for CI/CD

4. **Create interactive swap UI**
   - Dependency graph visualization (React Flow or d3-force)
   - Real-time fitness recalculation on swap
   - Approve/reject/customize swap flows

5. **Integrate with existing scenarios**
   - Call scenario-dependency-analyzer API for dependency data
   - Call secrets-manager API for secret requirements
   - Trigger scenario-to-* APIs for actual packaging
   - Create app-issue-tracker issues for approved swaps

6. **Validate with real scenarios**
   - Test deployment-manager with picker-wheel (simple scenario)
   - Test with scenario-dependency-analyzer (complex, has many deps)
   - Test with scenario-to-extension (packager validation)

---

## References

- `/docs/deployment/README.md` - Deployment Hub (tier model, orchestration loop)
- `/docs/deployment/tiers/` - Individual tier documentation
- `/docs/deployment/guides/` - Fitness scoring, dependency swapping guides
- `/scenarios/scenario-dependency-analyzer/` - Dependency data source
- `/scenarios/secrets-manager/` - Secret management integration
- `/scenarios/app-issue-tracker/` - Migration task tracking
- `/scenarios/scenario-to-extension/` - Proven packager implementation

---

**Conclusion**: deployment-manager is a validated, unique, and necessary capability that fills a critical gap in Vrooli's deployment story. All required integration points exist. Ready for PRD and implementation.
