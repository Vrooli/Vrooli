# Architectural Decisions Record (ADR)

This document records major architectural decisions in Vrooli's evolution and the rationale behind them.

## Decision Log Format

Each decision follows this structure:
- **Date**: When the decision was made
- **Status**: Accepted/Superseded/Deprecated
- **Context**: The situation and problem we faced
- **Decision**: What we decided to do
- **Consequences**: The results and trade-offs
- **Superseded By**: If replaced, what replaced it

---

## ADR-001: From Monolithic Application to Resource Orchestration Platform

**Date**: Q3 2023  
**Status**: Accepted  
**Context**: 
- Building every feature into a single application created massive complexity
- Difficult to maintain, test, and scale individual capabilities
- Limited ability to leverage existing open-source tools
- Monolithic architecture prevented modular deployment

**Decision**: 
Transform Vrooli from a monolithic application into a resource orchestration platform that coordinates 30+ independent services (resources) to create emergent business capabilities.

**Consequences**:
-  Dramatically reduced codebase complexity
-  Leveraged best-in-class tools for each capability
-  Enabled modular deployment and scaling
-  Improved testing through isolation
- � Increased operational complexity
- � Required new orchestration patterns

---

## ADR-002: Scenario-Based Application Architecture

**Date**: Q4 2023  
**Status**: Accepted  
**Context**:
- Needed a way to package resource combinations into deployable business applications
- Traditional app templates weren't sufficient for resource orchestration
- Required a mechanism to define business value alongside technical configuration

**Decision**:
Create "scenarios" - complete business application definitions that orchestrate multiple resources to deliver specific business value ($10K-50K per deployment).

**Consequences**:
-  Clear mapping between technical implementation and business value
-  Reusable patterns for common business needs
-  Simplified deployment of complex multi-resource applications
-  Enabled AI-driven scenario generation

---

## ADR-003: Dual-Purpose Scenarios (Test = Deployment)

**Date**: Q1 2024  
**Status**: Accepted  
**Context**:
- Traditional separation between tests and production code created drift
- Maintaining separate test suites and deployment configurations doubled work
- Tests didn't prove actual deployment readiness
- AI couldn't generate complete, deployment-ready solutions

**Decision**:
Design scenarios to serve dual purposes: integration tests AND production deployments. Every scenario that passes tests is deployment-ready.

**Consequences**:
-  Eliminated test/production drift
-  Tests now prove deployment readiness
-  Reduced maintenance overhead by 50%
-  AI can generate complete solutions in one pass
-  Every scenario proves both technical and business viability

---

## ADR-004: From Scenario Conversion to Direct Execution

**Date**: Q2 2024  
**Status**: Accepted  
**Context**:
- Initial approach converted scenarios into standalone applications before deployment
- Conversion process added complexity and build time
- Generated applications created duplication and maintenance issues
- Build artifacts consumed significant disk space
- Changes required regeneration and redeployment

**Decision**:
Eliminate the conversion step entirely. Run scenarios directly from their source location in the scenarios/ directory.

**Consequences**:
-  Instant startup (2-5 seconds vs 30+ seconds)
-  Zero build artifacts or generated code
-  Single source of truth (scenarios/ folder)
-  Edit and run immediately without regeneration
-  Saved gigabytes of disk space
-  Simplified debugging with direct source access
- � Required new process isolation mechanisms
- � Changed deployment packaging approach

**Supersedes**: ADR-002 (partially - scenarios remain, but execution model changed)

---

## ADR-006: Local-First Resource Strategy

**Date**: Q3 2024  
**Status**: Accepted  
**Context**:
- Cloud APIs create dependencies, costs, and privacy concerns
- Rate limits and availability issues affected reliability
- Customers wanted complete data control
- Needed unlimited experimentation capability

**Decision**:
Prioritize local resource deployment over cloud APIs. All core capabilities must be available through local resources.

**Consequences**:
-  Complete data privacy and control
-  No API rate limits or usage costs
-  Unlimited local experimentation
-  Works offline after initial setup
- � Higher initial setup complexity
- � Requires more local compute resources

---

## ADR-008: Unified CLI Over Multiple Tools

**Date**: Q4 2024  
**Status**: Accepted  
**Context**:
- Multiple scripts and tools created confusion
- Inconsistent interfaces across different operations
- Difficult onboarding for new users

**Decision**:
Create unified `vrooli` CLI that provides consistent interface for all operations (resources, scenarios, development, deployment).

**Consequences**:
-  Single tool to learn and use
-  Consistent command structure
-  Improved discoverability
-  Simplified documentation
- � Required significant refactoring
- � Backward compatibility considerations

---

## ADR-009: Process Isolation for Scenarios

**Date**: Q4 2024  
**Status**: Accepted  
**Context**:
- Direct execution required isolation between scenarios
- Needed to prevent scenarios from interfering with each other
- Required separate logging and monitoring per scenario

**Decision**:
Implement process isolation with separate PM_HOME and PM_LOG_DIR for each scenario, while sharing common resources.

**Consequences**:
-  Complete isolation between scenarios
-  Independent logging and monitoring
-  Clean process management
-  Scenarios can run simultaneously
- � Slightly increased resource usage
- � Required new process management patterns

---

## ADR-010: Meta-Scenarios for Self-Improvement

**Date**: Q4 2024  
**Status**: Accepted  
**Context**:
- Platform needed ability to improve itself
- Manual creation of scenarios was limiting growth
- Required mechanisms for automated enhancement

**Decision**:
Create meta-scenarios that improve Vrooli itself: Scenario Generator, System Monitor, App Issue Tracker, Resource Experimenter.

**Consequences**:
-  Platform can generate new scenarios
-  Self-monitoring and debugging capability
-  Automated resource integration testing
-  Continuous platform improvement
-  Reduced manual maintenance
- � Increased system complexity
- � Required careful access controls

---

## Future Considerations

### Under Evaluation
- **Resource versioning**: Support multiple versions of same resource

### Principles Going Forward
1. **Simplicity over complexity**: Direct execution proved simpler is better
2. **Business value focus**: Every technical decision must map to business value
3. **Local-first philosophy**: Prioritize local control and privacy
4. **Emergent capabilities**: Let complex behavior emerge from simple resource combinations
5. **Single source of truth**: Avoid duplication and generated artifacts

---

*This document is living and will be updated as new architectural decisions are made.*