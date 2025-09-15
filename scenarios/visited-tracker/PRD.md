# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Persistent file visit tracking with staleness detection for systematic code analysis

### Intelligence Amplification
**How does this capability make future agents smarter?**
Enables maintenance scenarios to efficiently track which files have been analyzed, ensuring comprehensive coverage across large codebases without redundant work. Agents can prioritize stale files and systematically work through entire projects over multiple conversations.

### Recursive Value
**What new scenarios become possible after this exists?**
- **API analysis scenarios** (like api-manager) can use this to systematically review all scenario APIs
- **Test coverage scenarios** (like test-genie) can track which files need testing attention
- **Documentation scenarios** can identify files that lack proper documentation
- **Code quality scenarios** can prioritize refactoring based on file staleness
- **Security audit scenarios** can ensure all files are regularly reviewed for vulnerabilities

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Campaign-based file tracking with patterns (*.go, *.js, etc.)
  - [x] Visit count tracking for each file in a campaign
  - [x] Staleness scoring based on visit frequency and modification time
  - [x] CLI interface for programmatic integration
  - [x] JSON file storage for persistence and portability
  
- **Should Have (P1)**
  - [x] HTTP API for web interface and external integrations
  - [x] Web interface for manual campaign management
  - [x] File synchronization with glob pattern matching
  - [x] Least visited and most stale file prioritization
  - [x] Campaign export/import capabilities
  
- **Nice to Have (P2)**
  - [ ] Advanced analytics and staleness trend analysis
  - [ ] Integration with git history for enhanced staleness detection
  - [ ] Multi-project campaign management
  - [ ] Automated file discovery and pattern suggestions

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| File sync performance | < 2s for 1000 files | CLI timing logs |
| API response time | < 100ms for common operations | HTTP monitoring |
| Data persistence | 100% reliability | Automated testing |
| CLI response time | < 1s for common operations | Performance testing |

## 🔗 Dependencies

### Required Resources
- Local file system (for tracking file modifications and storing campaign data)

### Optional Resources
- postgres (for enhanced data storage and querying capabilities)
- redis (for caching and performance optimization)

### Integration Points
- **Input**: Called by maintenance scenarios (api-manager, test-genie, etc.)
- **Output**: Provides prioritized file lists and staleness metrics
- **Data Flow**: Receives file patterns and directory paths, returns visit recommendations

## 📋 Implementation Notes

### Key Design Decisions
- File-based JSON storage for simplicity and portability
- Campaign-based organization for different analysis projects
- Staleness scoring algorithm based on visit frequency and file modification time
- HTTP API + CLI dual interface for flexibility

### Technical Constraints
- Must work across different file types and directory structures
- Should handle large codebases (1000+ files) efficiently
- Must maintain state across multiple agent conversations
- Designed as internal developer tool, not end-user application

### Future Considerations
- Machine learning for intelligent file prioritization based on change patterns
- Integration with version control systems for enhanced context
- Support for remote repositories and distributed development teams
- Advanced analytics for development workflow optimization