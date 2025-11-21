# Known Issues & Deferred Decisions

**Last Updated**: 2025-11-21 (Generator Phase)

---

## Open Issues

### 1. scenario-to-* Packager Availability
**Status**: ⚠️ Blocker for full P0 validation
**Priority**: Critical
**Description**: deployment-manager orchestrates scenario-to-desktop, scenario-to-ios, scenario-to-android, scenario-to-saas, and scenario-to-enterprise packagers. Currently only scenario-to-extension (browser extensions) and scenario-to-android exist in the repo.

**Impact**:
- Cannot fully validate P0 Module 06 (Deployment Orchestration) until at least scenario-to-desktop is implemented
- Multi-tier deployment testing (P1 Module 10) blocked for unavailable tiers

**Workaround**:
- Start with Tier 2 (desktop) focus using scenario-to-desktop once available
- Mock scenario-to-* API responses for early testing
- Document packager interface contract in deployment-manager so packagers can implement it

**Resolution Plan**:
- Track scenario-to-desktop implementation progress
- Once scenario-to-desktop exists, use it as reference implementation for other packagers
- deployment-manager should gracefully handle missing packagers (auto-discovery + clear error messages)

---

### 2. Fitness Scoring Rules Engine Design
**Status**: 🔵 Architecture Decision Needed
**Priority**: High (P0 Module 01)
**Description**: Fitness scoring engine needs pluggable rules that can be customized per scenario, per tier, and potentially per organization (enterprise tier).

**Open Questions**:
1. **Storage**: Should fitness rules be:
   - Hard-coded in Go structs (fast, version-controlled with code)?
   - Stored in postgres (flexible, runtime-configurable)?
   - Hybrid approach (defaults in code, overrides in DB)?

2. **Rule Format**: What's the DSL for expressing fitness rules?
   - JSON-based rule definitions?
   - Go functions registered in a rule registry?
   - Expression language (e.g., CEL, expr)?

3. **Sub-Score Weighting**: How are portability, resources, licensing, platform support weighted?
   - Fixed weights (e.g., 25% each)?
   - Tier-specific weights (e.g., mobile heavily weights portability)?
   - User-configurable weights per profile?

**Recommendation** (for improvers):
- Start with hard-coded Go functions for P0 (fastest path to validation)
- Store overrides in postgres JSONB for enterprise tier (P2)
- Document rule interface so custom rules can be added later

---

### 3. Swap Database Seeding Strategy
**Status**: 🔵 Implementation Detail
**Priority**: Medium (P0 Module 02)
**Description**: Dependency swap suggestions require a database of known alternatives with trade-off metadata (postgres → sqlite, ollama → openrouter, etc.).

**Open Questions**:
1. **Initial Population**: Who seeds the swap database?
   - Generator agent creates initial seed.sql with common swaps?
   - Improver agents add swaps as they discover them?
   - Community-contributed swaps (future)?

2. **Swap Metadata Schema**:
   - What fields are required? (old_dep, new_dep, tiers_applicable, fitness_delta, pros[], cons[], migration_effort)
   - How do we version swaps as dependencies evolve?

3. **Automatic Swap Discovery**:
   - Can we automatically suggest swaps by analyzing resource/scenario capabilities?
   - Use LLM (claude-code/ollama) to generate swap suggestions based on dependency metadata?

**Recommendation** (for improvers):
- Create `initialization/storage/postgres/seed_swaps.sql` with 10-20 common swaps
- Schema: `swap_alternatives` table with columns (id, from_dep_type, from_dep_name, to_dep_type, to_dep_name, applicable_tiers[], fitness_delta, pros, cons, migration_effort, created_at)
- Document swap contribution process in PROBLEMS.md for future agents

---

## Deferred Ideas

### 1. Real-Time Collaboration on Deployment Profiles
**PRD Reference**: OT-P2-020 (Collaborative Deployment Approval)
**Defer Reason**: P2 feature, requires WebSocket presence system + comment threading infrastructure
**Future Considerations**: Consider using existing collaboration libraries (e.g., Yjs, automerge) for conflict-free replicated data types (CRDTs)

---

### 2. AI-Powered Tier Recommendation Engine
**PRD Reference**: OT-P2-021 (Tier Recommendation Engine)
**Defer Reason**: P2 feature, requires training data (scenario characteristics → optimal tiers)
**Future Considerations**: Could use LLM few-shot prompting with tier constraints as interim solution before building ML model

---

### 3. Cross-Tier Resource Sharing
**PRD Reference**: OT-P2-022 (Cross-Tier Resource Sharing)
**Defer Reason**: P2 feature, complex orchestration (mobile/desktop clients sharing SaaS backend)
**Future Considerations**: Needs service mesh or API gateway to coordinate shared backends, potentially out of scope for deployment-manager (should scenario-to-saas handle this?)

---

## Technical Debt

*No technical debt yet (initialization phase). Improvers should document any shortcuts or compromises here.*

---

## Testing Gaps

### 1. UI Automation for Dependency Graph
**Impact**: Cannot fully validate REQ-P0-035 (Interactive Dependency Graph) without browser automation
**Workaround**: Manual testing during development, document in requirement notes
**Resolution Plan**: Use Browser Automation Studio (BAS) workflows once available (see `docs/testing/guides/ui-automation-with-bas.md`)

---

### 2. Performance Testing for Large Dependency Trees
**Impact**: REQ-P0-037 requires rendering <3s for 100 dependencies, need realistic test scenarios
**Workaround**: Create synthetic dependency trees for performance testing
**Resolution Plan**: Once real scenarios have 50+ dependencies, use them for performance benchmarks

---

## Questions for Future Agents

1. **Fitness Threshold**: What's the minimum acceptable fitness score per tier before deployment is blocked? (Suggestion: 70 for P0, configurable per profile)
2. **Cost Estimation Accuracy**: OT-P0-027 requires ±20% accuracy for SaaS cost estimates. Which cloud pricing APIs should we integrate? (AWS/DigitalOcean/GCP?)
3. **Secret Rotation Automation**: OT-P2-024 (Secret Auto-Rotation Integration) - should redeployment be automatic or require approval?
4. **Licensing Database**: Where do we get license metadata for all resources/scenarios? Parse from each dependency's LICENSE file, or maintain central database?

---

**Note for Improvers**: When you resolve an issue or make an architectural decision, move it from "Open Issues" to a new "Resolved" section below with date + rationale.
