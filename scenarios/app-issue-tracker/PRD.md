# PRD: App Issue Tracker

## 1. Product Overview

The App Issue Tracker is a lightweight, file-based issue tracking system designed for software development teams. It enables easy creation, management, and investigation of issues using YAML files for storage, eliminating the need for a traditional database. The system integrates with AI tools like Claude Code to provide automated investigation and resolution suggestions.

### Key Value Proposition
- **Zero Setup**: No database configuration required – just files
- **AI-Powered**: Automatic issue analysis and fix suggestions
- **Portable**: Works anywhere Git works
- **Extensible**: Easy to integrate with existing workflows

## 2. Target Users
- Software developers tracking bugs and features
- Team leads managing project issues
- AI agents automating investigation workflows
- Small teams avoiding heavy tools like Jira

## 3. Goals and Objectives
### Business Goals
- Reduce issue tracking overhead by 80%
- Enable AI-assisted resolution for 50% of issues
- Provide semantic search across all historical issues

### Technical Goals
- 100% file-based storage with YAML schema
- REST API for programmatic access
- Web UI for visual management
- CLI for terminal operations

## 4. User Stories

### Core Stories
- As a developer, I want to create a new issue from a template so that I can standardize reporting
- As a team member, I want to assign issues to myself or others so that responsibilities are clear
- As a lead, I want to search issues by keyword or semantic similarity so that I can find related problems
- As an AI agent, I want to investigate an issue automatically so that I can generate fix suggestions

### Advanced Stories
- As a maintainer, I want to close issues with resolution notes so that knowledge is preserved
- As an analyst, I want to generate reports on issue trends so that I can identify patterns
- As a collaborator, I want to integrate with GitHub so that issues sync with PRs

## 5. Functional Requirements

### Issue Management
- Create issues with title, description, priority, labels, assignee
- Update issue status (open, investigating, in-progress, fixed, closed)
- Delete or archive issues
- Bulk operations on multiple issues

### Search and Discovery
- Keyword search across all issue fields
- Semantic search using vector embeddings (optional Qdrant integration)
- Filter by status, priority, assignee, date range
- Advanced query syntax support

### AI Integration
- Automatic investigation workflow triggered by commands
- Generate fix suggestions using Claude Code
- Auto-categorize issues by type (bug, feature, documentation)
- Predict issue severity based on description

### Interfaces
- **CLI**: `app-issue-tracker create/update/list/search/investigate`
- **API**: REST endpoints for all operations (/issues, /search, /investigate)
- **UI**: Simple web interface for browsing and editing issues

### Storage
- Folder bundles in `data/issues/<status>/<issue-id>/metadata.yaml`
- `artifacts/` subdirectories for logs, screenshots, HAR files, etc.
- Template metadata files for manual creation
- Backup and migration scripts

## 6. Non-Functional Requirements

### Performance
- Handle 1000+ issues with <500ms search times
- File operations optimized for frequent updates
- Caching layer for repeated queries (Redis optional)

### Security
- File permissions management
- Input sanitization for descriptions and titles
- Audit log for issue changes
- No sensitive data in YAML files

### Reliability
- 99.9% uptime for API/UI
- Graceful degradation without vector search
- Automated backups of issue files
- Recovery from corrupted YAML files

### Usability
- Intuitive CLI with help and examples
- Clean, responsive web UI
- Keyboard shortcuts in UI
- Export issues to CSV/Markdown

### Compatibility
- Linux/macOS/Windows (via WSL)
- Go 1.21+, Node.js 18+
- Git integration for version control
- Docker support for containerized deployment

## 7. Technical Architecture

### Components
- **API Server**: Go-based REST API handling file I/O and business logic
- **Web UI**: Node.js server with static file serving and optional SSR
- **CLI Tool**: Bash script wrapping API calls and direct file operations
- **AI Investigator**: Script integrating with Claude Code API
- **Search Engine**: Optional Qdrant for semantic search, fallback to keyword

### Data Flow
1. User creates issue via CLI/UI/API
2. System validates and saves YAML to appropriate status directory
3. Search queries scan files or query vector store
4. Investigation triggers AI analysis on issue content
5. Results stored as comments or separate investigation files

### Deployment
- Local development: `make start`
- Production: Docker containers or direct binary execution
- Resources: Optional Redis/Qdrant for enhanced features

## 8. Success Metrics
- 90% user satisfaction with issue creation speed
- 70% issues resolved with AI assistance
- <5% error rate in file operations
- 100% test coverage for core functionality

## 9. Risks and Mitigations
- **Risk**: YAML parsing errors – **Mitigation**: Schema validation + error recovery
- **Risk**: Performance with large issue volumes – **Mitigation**: Indexing + caching
- **Risk**: AI investigation inaccuracies – **Mitigation**: Human review workflow + feedback loop
- **Risk**: File conflicts in teams – **Mitigation**: Git integration + locking mechanism

## 10. Timeline
- MVP (CLI + basic API): 2 weeks
- UI Integration: 1 week
- AI Features: 2 weeks
- Testing & Polish: 1 week
- Total: 6 weeks
