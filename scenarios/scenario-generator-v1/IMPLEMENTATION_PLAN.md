# Scenario Generator V1 - Implementation Plan

## Overview
This scenario creates an autonomous scenario development pipeline that uses Claude Code to iteratively generate, refine, validate, and deploy complete Vrooli scenarios. It provides a campaign-based dashboard for managing multiple scenario development projects.

## Core Architecture

### Resources Required
- **n8n** (5678): Orchestrate the entire generation pipeline 
- **postgres** (5433): Store campaigns, scenarios, generation logs, and improvement analytics
- **minio** (9000): Store generated scenario files, plans, and artifacts
- **claude-code**: AI-driven scenario generation and refinement
- **redis** (6380): Queue management and session caching

### File Structure
```
scenario-generator-v1/
├── service.json                      # Resource configuration
├── IMPLEMENTATION_PLAN.md            # This plan (for tracking progress)
├── initialization/
│   ├── automation/
│   │   ├── n8n/
│   │   │   ├── main-pipeline.json           # Master orchestration workflow
│   │   │   ├── planning-workflow.json       # Plan generation & refinement
│   │   │   ├── building-workflow.json       # Scenario implementation
│   │   │   └── validation-workflow.json     # Testing & deployment
│   ├── configuration/
│   │   ├── app-config.json           # UI settings, generation parameters
│   │   ├── claude-prompts.json       # All prompt templates
│   │   ├── pipeline-config.json      # Generation pipeline settings
│   │   └── resource-urls.json        # Resource endpoint configuration
│   └── storage/
│       ├── schema.sql                # Campaigns, scenarios, generation logs
│       └── seed.sql                  # Sample campaigns and data
├── prompts/
│   ├── initial-planning-prompt.md    # Detailed scenario analysis prompt
│   ├── plan-refinement-prompt.md     # Plan improvement prompt
│   ├── implementation-prompt.md      # Code generation prompt
│   └── validation-prompt.md          # Bug analysis & fixing prompt
├── deployment/
│   └── startup.sh                    # Initialize pipeline and UI
└── test.sh                          # Integration validation
```

## Implementation Phases

### Phase 1: Foundation Setup
- [x] Store implementation plan
- [ ] Create service.json with resource requirements
- [ ] Design PostgreSQL schema for campaigns, scenarios, analytics
- [ ] Create basic configuration files

### Phase 2: Core Pipeline Workflows
- [ ] Create master orchestration n8n workflow
- [ ] Build planning workflow (generation + refinement loops)
- [ ] Build implementation workflow (coding + bug fixing loops) 
- [ ] Build validation workflow (dry-run + deployment)

### Phase 3: User Interface
- [ ] Complete React UI with scenario management
- [ ] Create scenario management cards and controls
- [ ] Implement real-time progress tracking
- [ ] Add configuration panels for iteration counts

### Phase 4: Claude Code Integration
- [ ] Create detailed prompt templates
- [ ] Build Claude Code CLI wrapper scripts
- [ ] Implement structured output parsing
- [ ] Add error handling and retry logic

### Phase 5: Validation & Analytics
- [ ] Integrate scenario-to-app.sh validation
- [ ] Build pattern analysis for common issues
- [ ] Create improvement suggestion system
- [ ] Add performance monitoring

## Detailed Workflows

### 1. Scenario Generation Pipeline

**Step 1: Initial Planning**
- User enters scenario request in React web interface
- n8n triggers Claude Code with comprehensive planning prompt
- Generated plan stored in MinIO with version control
- Plan metadata stored in PostgreSQL

**Step 2: Plan Refinement Loop** (configurable 1-5 iterations)
- Claude Code analyzes and improves previous plan
- Each iteration compared and scored
- Best refinements merged into final plan
- All iterations stored for pattern analysis

**Step 3: Implementation Loop** (configurable 1-5 iterations)
- Claude Code generates complete scenario files from plan
- Files validated for structure and completeness
- Bug analysis and fixing performed each iteration
- All findings stored in improvement database

**Step 4: Validation Pipeline**
- scenario-to-app.sh dry-run validation
- Parse validation errors and feed to Claude Code
- Iterative fixing until validation passes
- Final deployment via scenario-to-app.sh

**Step 5: Post-Deployment**
- Success/failure analytics recorded
- Generated scenario added to campaign dashboard
- Performance metrics collection setup

### 2. Campaign Management

**Campaign Organization**
- Scenarios grouped by business category or client
- Tab-based navigation in dashboard
- Progress tracking across all scenarios in campaign
- Batch operations for related scenarios

**Analytics & Learning**
- Success rate tracking per campaign
- Common issue pattern identification
- Resource utilization optimization
- Generation time improvements

### 3. Real-Time Progress Tracking

**Redis Integration**
- Job queue management for concurrent generation
- Real-time progress updates via pub/sub
- Session caching for UI state management
- Progress persistence across browser sessions

**UI Updates**
- Live status indicators on scenario cards
- Progress bars for long-running operations
- Error notifications with actionable feedback
- Detailed logs accessible from dashboard

## Database Schema Design

### Core Tables
- `campaigns`: Campaign metadata and organization
- `scenarios`: Individual scenario tracking and status
- `generation_logs`: Detailed logs of all generation steps
- `improvement_patterns`: Analysis of common issues and solutions
- `claude_interactions`: All Claude Code prompts and responses
- `validation_results`: scenario-to-app.sh results and fixes

### Analytics Tables
- `success_metrics`: Success rates by category and complexity
- `performance_data`: Generation time and resource usage
- `pattern_analysis`: Automated insights from improvement data
- `suggestion_queue`: AI-generated improvement recommendations

## Configuration Management

### Generation Pipeline Settings
- Default iteration counts for each phase
- Claude Code model selection and parameters
- Timeout values and retry logic
- Resource allocation per scenario complexity

### UI Customization
- Campaign color coding and icons
- Dashboard layout preferences
- Notification settings
- Progress display options

## Security & Performance

### Security Considerations
- Claude API keys stored in Vault
- Isolated execution environments for validation
- Audit logging for all generation activities
- Access control for campaign management

### Performance Optimization
- Redis queuing prevents resource conflicts
- MinIO versioning enables efficient rollback
- PostgreSQL indexing optimizes analytics queries
- Parallel processing where possible

## Success Metrics

### Primary KPIs
- Scenario generation success rate (target: >85%)
- Average generation time (target: <30 minutes)
- Validation pass rate on first attempt (target: >70%)
- User satisfaction with generated scenarios

### Learning Analytics
- Pattern recognition accuracy improvement over time
- Reduction in common error types
- Optimization of generation parameters
- Predictive success scoring for new requests

## Implementation Progress Tracking

Use this checklist to track implementation progress:

### Foundation (Phase 1)
- [ ] service.json created with all required resources
- [ ] PostgreSQL schema designed and tested
- [ ] Basic configuration files created
- [ ] MinIO bucket structure established

### Workflows (Phase 2)
- [ ] Master orchestration workflow implemented
- [ ] Planning workflow with refinement loops
- [ ] Implementation workflow with bug fixing
- [ ] Validation workflow with deployment

### UI (Phase 3)
- [ ] React application with tab navigation
- [ ] Scenario cards with status indicators
- [ ] Configuration panels functional
- [ ] Real-time progress updates working

### Integration (Phase 4)
- [ ] Claude Code CLI wrapper functional
- [ ] All prompt templates created and tested
- [ ] Error handling and retry logic implemented
- [ ] Structured output parsing working

### Analytics (Phase 5)
- [ ] scenario-to-app.sh integration complete
- [ ] Pattern analysis system functional
- [ ] Improvement suggestions generated
- [ ] Performance monitoring active

---

**Note**: This plan serves as our implementation roadmap. Update progress regularly and refer back when making design decisions to ensure consistency with the original vision.