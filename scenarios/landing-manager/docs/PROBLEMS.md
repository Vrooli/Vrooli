# Problems & Known Issues

> **Last Updated**: 2025-11-21
> **Status**: Initialization phase - no implementation yet

## Open Issues

### 🔴 Blockers (Prevent Progress)
*None at this time*

### 🟠 Critical (High Priority)
*None at this time*

### 🟡 Important (Medium Priority)
*None at this time*

### 🟢 Minor (Low Priority)
*None at this time*

---

## Deferred Ideas

### Template System Architecture
**Question**: Should landing-manager manage templates as files in `scripts/scenarios/templates/` or as database records?

**Current Decision**: File-based in `scripts/scenarios/templates/saas-landing-page/` for MVP.

**Reasoning**:
- Simpler to version control
- Easier to review and iterate
- Aligns with existing Vrooli template system

**Future Consideration**: Database-backed templates for dynamic editing in P2 (template marketplace).

---

### Agent Customization Safety
**Risk**: AI agents may generate poor-quality or breaking customizations.

**Mitigations Planned**:
1. Preview mode before applying changes
2. Rollback capability (git-based or snapshot-based)
3. Validation checks before deployment
4. Agent should only modify config files/APIs, never arbitrary code

**Status**: Documented in RESEARCH.md, deferred to implementation phase.

---

### A/B Test Statistical Validity
**Risk**: Users may draw conclusions from insufficient sample sizes.

**Mitigations Planned**:
1. Display confidence intervals in admin dashboard (P2)
2. Require minimum sample size before showing results
3. Show "Not enough data" message when appropriate

**Status**: P2 requirement (OT-P2-003), not blocking MVP.

---

### Video Content Performance
**Question**: Should landing pages support self-hosted video or only external embeds?

**Current Decision**: External embeds only (YouTube, Vimeo) for MVP.

**Reasoning**:
- Avoids bandwidth/storage costs
- Leverages CDN performance of video platforms
- Simpler implementation

**Future Consideration**: Self-hosted video with adaptive bitrate streaming in P2.

---

### GDPR/CCPA Compliance
**Risk**: Analytics tracking may require cookie consent banners.

**Mitigations Planned**:
1. Document compliance requirements in generated README
2. Provide optional cookie banner component
3. Respect DNT (Do Not Track) headers

**Status**: Documented in RESEARCH.md, implementation details deferred.

---

### Multi-Tenant Admin Portal
**Question**: Should each landing page have separate admin accounts, or unified admin across all landing pages?

**Current Decision**: Separate admin accounts per landing page for MVP.

**Reasoning**:
- Simpler implementation (no cross-scenario auth)
- Better isolation for deployments
- Aligns with "each landing page is its own scenario" model

**Future Consideration**: Unified admin portal via scenario-authenticator integration in P2.

---

## Lessons Learned

*This section will be populated by improver agents as they work on the scenario.*

### Template Structure
*TBD - Document lessons about template organization, file structure, etc.*

### A/B Testing Implementation
*TBD - Document lessons about variant selection logic, metrics tracking, etc.*

### Stripe Integration
*TBD - Document lessons about webhook handling, subscription management, etc.*

---

## Questions for Future Agents

1. **Template Variables**: What variables should be customizable in the saas-landing-page template? (hero copy, CTA text, colors, fonts, etc.)

2. **Metrics Storage**: Should metrics be stored in PostgreSQL or a dedicated time-series DB (e.g., InfluxDB)?

3. **Admin Portal Framework**: Should admin portal use the same React+Vite stack as the public landing page, or a separate framework?

4. **Subscription Model**: Should generated landing pages support multiple pricing tiers, or single-tier subscriptions only for MVP?

5. **Agent Prompt Engineering**: What's the optimal prompt structure for agents customizing landing pages? (See RESEARCH.md for initial thoughts)

---

## Resolution Log

*This section tracks resolved issues and their solutions.*

### Resolved Issues
*None yet - scenario is in initialization phase*

---

## Instructions for Future Agents

**When you encounter a problem**:
1. Document it in the appropriate section above (Blocker/Critical/Important/Minor)
2. Include context: what were you trying to do, what went wrong, error messages, etc.
3. Note any workarounds you attempted
4. Update `docs/PROGRESS.md` with the blocker

**When you resolve a problem**:
1. Move it from Open Issues to Resolution Log
2. Document the solution and any lessons learned
3. Update `docs/PROGRESS.md` with the resolution
4. Consider if the solution warrants a PRD update (consult prd-control-tower first)

**When you defer a problem**:
1. Move it to Deferred Ideas with justification
2. Note any dependencies or prerequisites for addressing it later
3. Link to relevant requirements or PRD sections
