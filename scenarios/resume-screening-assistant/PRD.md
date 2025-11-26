# Product Requirements Document (PRD)

## 🎯 Overview

**Purpose**: Resume Screening Assistant provides AI-powered resume screening and candidate evaluation capabilities with semantic matching and automated assessment workflows. This scenario transforms manual resume review into an intelligent, scalable hiring process that helps HR teams efficiently process large volumes of candidates and find the best matches for open positions.

**Primary Users**: HR teams, recruiters, hiring managers, talent acquisition specialists

**Deployment Surfaces**:
- Web UI (HyperRecruit 3000 retro-futuristic interface)
- REST API for programmatic access
- CLI for command-line workflows
- N8n workflows for automation integration

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Smart resume processing | Extract key information from resumes using Ollama LLM with structured output
- [ ] OT-P0-002 | Semantic candidate matching | Use vector embeddings via Qdrant for intelligent candidate-job matching
- [ ] OT-P0-003 | Job posting management | Create, read, update, and delete job postings with requirements tracking
- [ ] OT-P0-004 | Candidate database | Store and manage candidate profiles with structured data in PostgreSQL
- [ ] OT-P0-005 | Resume upload and parsing | Accept resume files and extract skills, experience, education, contact info
- [ ] OT-P0-006 | Semantic search API | Find best candidate matches for job postings using vector similarity
- [ ] OT-P0-007 | HyperRecruit 3000 UI | Retro-futuristic web interface with real-time processing stats and candidate views
- [ ] OT-P0-008 | Health monitoring | API health endpoints for service status and dependency checks

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Resume batch processing | Process multiple resumes in parallel with progress tracking
- [ ] OT-P1-002 | Candidate ranking | Score and rank candidates based on job fit with explainable AI reasoning
- [ ] OT-P1-003 | Assessment templates | Pre-built evaluation criteria for common job roles
- [ ] OT-P1-004 | Search filters | Advanced filtering by skills, experience level, location, salary requirements
- [ ] OT-P1-005 | Export capabilities | Export candidate lists and assessments to CSV/PDF formats
- [ ] OT-P1-006 | Workflow automation | N8n workflows for resume ingestion, processing, and notification pipelines

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Interview scheduling | Integration with calendar scenario for automated interview booking
- [ ] OT-P2-002 | Email templates | Automated candidate communication via email-triage scenario
- [ ] OT-P2-003 | Collaborative review | Multi-user review and commenting on candidate profiles
- [ ] OT-P2-004 | Analytics dashboard | Hiring funnel metrics and time-to-hire analytics
- [ ] OT-P2-005 | Resume versioning | Track candidate profile updates and resume revisions over time

## 🧱 Tech Direction Snapshot

**Architecture**:
- **Go API**: Coordination layer exposing REST endpoints for candidate/job/search operations
- **N8n Workflows**: Automation backbone for resume processing, job management, and semantic search
- **PostgreSQL**: Structured storage for candidates, jobs, and assessment data
- **Qdrant**: Vector database for semantic search and candidate-job matching
- **Ollama**: LLM for resume text extraction and skill identification
- **JavaScript UI**: Retro-futuristic "HyperRecruit 3000" interface with NodeJS server

**Data Flow**:
1. Resume upload → N8n resume-processing-pipeline → Ollama extraction → PostgreSQL + Qdrant embeddings
2. Job posting creation → PostgreSQL storage → Embedding generation → Qdrant indexing
3. Candidate search → Qdrant vector similarity → PostgreSQL enrichment → Ranked results

**Integration Pattern**: Workflows-first approach using N8n for core processing, with Go API providing lightweight coordination and HTTP endpoints for UI/CLI consumption

**Non-goals**:
- Applicant tracking system (ATS) features like offer management or onboarding
- Video interview hosting or assessment
- Background check integrations
- Direct integration with job boards (focus on internal resume processing)

## 🤝 Dependencies & Launch Plan

**Required Resources**:
- `postgres` - Candidate and job data storage (initialization/postgres/schema.sql)
- `qdrant` - Vector embeddings for semantic matching
- `ollama` - AI text analysis (models: llama3.2, nomic-embed-text)
- `n8n` - Workflow automation (3 workflows: resume-processing-pipeline, job-management-workflow, semantic-search-workflow)

**Scenario Dependencies**: None (standalone scenario)

**Launch Sequencing**:
1. **Phase 1**: Database schema setup (PostgreSQL tables for candidates, jobs, assessments)
2. **Phase 2**: Qdrant collection initialization for resume and job embeddings
3. **Phase 3**: N8n workflow deployment (resume processing, job management, semantic search)
4. **Phase 4**: Go API deployment with health checks and endpoint validation
5. **Phase 5**: UI build and deployment with HyperRecruit 3000 theme
6. **Phase 6**: CLI installation and verification

**Risks**:
- Ollama LLM extraction accuracy depends on resume format quality (PDF/DOCX parsing)
- Vector embedding quality impacts match relevance (requires tuning)
- N8n workflow complexity may require debugging for edge cases
- UI bundle size optimization needed for production deployment

## 🎨 UX & Branding

**Visual Style**: Retro-futuristic "HyperRecruit 3000" theme
- Neon green accents (#00ff41) on dark background (#0a0e27)
- Terminal-style monospace fonts (Courier New, Consolas)
- Animated scan lines and CRT monitor effects
- Glitch transitions and pixelated borders
- Dashboard with real-time processing stats

**Interface Philosophy**:
- Immediate visual feedback for all processing operations
- Clear candidate-to-job match confidence scores
- One-click actions for common workflows (upload resume, search candidates, view details)
- Progressive disclosure for detailed candidate profiles

**Accessibility Targets**:
- WCAG 2.1 AA compliance for core interactions
- Keyboard navigation for all primary functions
- High contrast mode toggle for reduced eye strain
- Screen reader compatibility for candidate lists

**Personality**:
- Tone: Professional yet playful, embracing the retro-tech aesthetic
- Mood: Efficient, intelligent, slightly futuristic
- Target feeling: Users should feel empowered by AI assistance while maintaining control

## 📎 Appendix

### Reference Implementation
- Database schema: `initialization/postgres/schema.sql`
- Qdrant collection config: `initialization/qdrant/collections.json`
- Assessment templates: `initialization/storage/assessment-templates.json`

### Inspiration
- Terminal UI aesthetics: Fallout series PIPBOY, Alien: Isolation interfaces
- Semantic search patterns: Algolia, Elasticsearch relevance tuning
- Resume parsing: Lever ATS, Greenhouse candidate views

### Future Integration Opportunities
- **product-manager-agent**: Hire product team members using semantic matching
- **personal-relationship-manager**: Track professional contacts extracted from resumes
- **mind-maps**: Build skill taxonomy and knowledge graphs from candidate data
- **calendar**: Schedule interviews and candidate pipeline reviews
- **email-triage**: Automate candidate communication and status updates
