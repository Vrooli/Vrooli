# Qdrant Semantic Knowledge System - Implementation Plan

## Executive Summary
Build a comprehensive embedding system that indexes project knowledge (workflows, code, documentation) into Qdrant vector collections, enabling semantic search across the main Vrooli app and all generated scenario apps. The system will automatically refresh embeddings on git commits and provide powerful cross-app knowledge discovery.

## Core Principles
1. **App Isolation**: Each app (main + generated) has its own collection namespace
2. **Full Refresh**: On changes, completely regenerate app's embeddings (simpler than incremental)
3. **Standardization**: Defined locations for different knowledge types
4. **Discoverability**: Validation shows what can be embedded vs what's missing
5. **Cross-Pollination**: Search across all apps to find patterns

## Phase 1: Foundation Infrastructure
**Goal**: Establish directory structure and configuration system

### 1.1 Directory Structure
```bash
scripts/resources/storage/qdrant/embeddings/
├── manage.sh                 # Main orchestrator
├── extractors/              # Content extraction modules
│   ├── workflows.sh         # N8n workflow extraction
│   ├── scenarios.sh         # Scenario PRD/config extraction
│   ├── code.sh             # Function/API extraction
│   ├── docs.sh             # Documentation extraction
│   └── resources.sh        # Resource capability extraction
├── indexers/               # Collection management
│   ├── project.sh          # Main vrooli indexing
│   └── app.sh              # Generated app indexing
├── search/                 # Search functionality
│   ├── single.sh           # Single app search
│   └── multi.sh            # Cross-app search
├── validation/             # Validation and discovery
│   ├── validator.sh        # Content validation
│   └── discovery.sh       # Missing content detection
└── config/
    ├── schema.yaml         # Knowledge source definitions
    └── templates/          # Document templates
```

### 1.2 Configuration Schema
Create `config/schema.yaml` defining what to extract:
```yaml
version: 1.0.0
knowledge_sources:
  workflows:
    pattern: "**/initialization/**/*.json"
    extractor: workflows
    collection_suffix: workflows
    
  scenarios:
    patterns:
      - "scenarios/*/PRD.md"
      - "scenarios/*/.scenario.yaml"
    extractor: scenarios
    collection_suffix: scenarios
    
  documentation:
    files:
      - path: "docs/ARCHITECTURE.md"
        sections: ["Design Decisions", "Trade-offs", "Patterns"]
      - path: "docs/SECURITY.md"
        sections: ["Security Principles", "Known Vulnerabilities"]
      - path: "docs/LESSONS_LEARNED.md"
        sections: ["What Worked", "What Failed", "Why"]
    extractor: docs
    collection_suffix: knowledge
```

### 1.3 App Identity System
Create `.vrooli/app-identity.json` for each app:
```json
{
  "app_id": "vrooli-main",
  "type": "main",
  "source_scenario": null,
  "collections": {
    "workflows": "vrooli-main-workflows",
    "scenarios": "vrooli-main-scenarios",
    "knowledge": "vrooli-main-knowledge",
    "code": "vrooli-main-code"
  },
  "last_indexed": "2025-01-22T12:00:00Z",
  "index_commit": "abc123def",
  "stats": {
    "total_embeddings": 1523,
    "last_duration_seconds": 45
  }
}
```

## Phase 2: Standard Knowledge Templates
**Goal**: Create templates for structured knowledge capture

### 2.1 Document Templates Location
`scripts/resources/storage/qdrant/embeddings/config/templates/`

Each template includes:
- Required sections with examples
- Metadata format specifications
- Embedding markers for granular indexing

### 2.2 Template Examples

**ARCHITECTURE.md**:
```markdown
# Architecture Decisions

## Design Decisions
<!-- EMBED:DECISION:START -->
### [2025-01-22] Three-Tier AI Architecture
**Context:** Need clear separation in AI system
**Decision:** Split into Coordination, Process, Execution
**Rationale:** Independent scaling, clear responsibilities
**Trade-offs:** Complexity vs separation
**Tags:** #architecture #ai #tiers
<!-- EMBED:DECISION:END -->
```

**LESSONS_LEARNED.md**:
```markdown
# Lessons Learned

## What Worked
<!-- EMBED:SUCCESS:START -->
### Thin Wrapper Pattern
**Context:** CLI commands becoming complex
**Implementation:** Move logic to lib functions
**Result:** 80% code reduction, better testing
**Reusable:** Yes - apply to all CLIs
<!-- EMBED:SUCCESS:END -->
```

## Phase 3: Content Extractors
**Goal**: Build modular extractors for each content type

### 3.1 Workflow Extractor (`extractors/workflows.sh`)
```bash
qdrant::extract::workflows() {
    local file="$1"
    
    # Extract workflow metadata
    local name=$(jq -r '.name' "$file")
    local description=$(jq -r '.description // ""' "$file")
    local nodes=$(jq -r '[.nodes[].type] | unique | join(", ")' "$file")
    
    # Create embeddable content
    echo "Workflow: $name
Description: $description
Node Types: $nodes
File: $file"
}
```

### 3.2 Code Extractor (`extractors/code.sh`)
```bash
qdrant::extract::code() {
    local file="$1"
    
    # Extract functions with docstrings
    grep -A 5 "^[a-zA-Z_][a-zA-Z0-9_]*\(\)" "$file" | \
    while read -r line; do
        # Parse function signature and comments
        # Generate embedding text
    done
}
```

### 3.3 Documentation Extractor (`extractors/docs.sh`)
```bash
qdrant::extract::docs() {
    local file="$1"
    local sections="$2"
    
    # Extract marked sections
    sed -n '/<!-- EMBED:.*:START -->/,/<!-- EMBED:.*:END -->/p' "$file" | \
    while read -r block; do
        # Process each embeddable block
    done
}
```

## Phase 4: Embedding Management
**Goal**: Core functions for generating and storing embeddings

### 4.1 Main Orchestrator (`manage.sh`)
```bash
#!/usr/bin/env bash
# Main entry point for embedding operations

qdrant::embeddings::refresh() {
    local app_id="${1:-$(qdrant::embeddings::get_app_id)}"
    
    log::info "Refreshing embeddings for app: $app_id"
    
    # Get app collections
    local collections=$(jq -r '.collections | to_entries[] | .value' .vrooli/app-identity.json)
    
    # Clear existing embeddings for this app
    for collection in $collections; do
        qdrant::collections::delete "$collection" --force || true
        qdrant::collections::create "$collection" --model mxbai-embed-large
    done
    
    # Re-index all content
    qdrant::embeddings::index_workflows "$app_id"
    qdrant::embeddings::index_scenarios "$app_id"
    qdrant::embeddings::index_documentation "$app_id"
    qdrant::embeddings::index_code "$app_id"
    
    # Update app identity
    qdrant::embeddings::update_identity "$app_id"
}
```

### 4.2 Batch Embedding Generator
```bash
qdrant::embeddings::batch_generate() {
    local content_file="$1"
    local collection="$2"
    local model="${3:-mxbai-embed-large}"
    
    # Process in batches for efficiency
    local batch_size=10
    local count=0
    local batch_content=""
    
    while IFS= read -r content; do
        batch_content+="$content"$'\n---SEPARATOR---\n'
        ((count++))
        
        if [[ $count -eq $batch_size ]]; then
            # Generate embeddings for batch
            local embeddings=$(ollama embed "$batch_content" --model "$model")
            # Store in collection
            count=0
            batch_content=""
        fi
    done < "$content_file"
}
```

## Phase 5: Validation System
**Goal**: Validate content and discover missing documentation

### 5.1 Validation Function (`validation/validator.sh`)
```bash
qdrant::embeddings::validate() {
    local app_path="${1:-.}"
    
    echo "=== Embedding Validation Report ==="
    echo "App: $(basename "$app_path")"
    echo ""
    
    # Check each knowledge source
    echo "📄 Documentation Status:"
    for doc in ARCHITECTURE SECURITY LESSONS_LEARNED BREAKING_CHANGES PERFORMANCE PATTERNS; do
        if [[ -f "docs/${doc}.md" ]]; then
            local sections=$(qdrant::validate::check_sections "docs/${doc}.md")
            echo "  ✅ ${doc}.md - $sections sections found"
        else
            echo "  ❌ ${doc}.md - MISSING (would improve discoverability)"
        fi
    done
    
    echo ""
    echo "📦 Embeddable Content Found:"
    echo "  • Workflows: $(find . -name "*.json" -path "*/initialization/*" | wc -l)"
    echo "  • Scenarios: $(find . -name "PRD.md" -path "*/scenarios/*" | wc -l)"
    echo "  • Code files: $(find . -name "*.sh" -o -name "*.ts" -o -name "*.js" | wc -l)"
    
    echo ""
    echo "💡 Recommendations:"
    qdrant::validate::generate_recommendations
}
```

### 5.2 Discovery Report
```bash
qdrant::validate::generate_recommendations() {
    local missing=()
    
    [[ ! -f "docs/ARCHITECTURE.md" ]] && missing+=("Create ARCHITECTURE.md for design decisions")
    [[ ! -f "docs/SECURITY.md" ]] && missing+=("Add SECURITY.md for security considerations")
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        echo "To improve embedding coverage:"
        for rec in "${missing[@]}"; do
            echo "  • $rec"
        done
    else
        echo "  All recommended documentation present!"
    fi
}
```

## Phase 6: Search System
**Goal**: Enable powerful semantic search across apps

### 6.1 Single App Search (`search/single.sh`)
```bash
qdrant::search::single_app() {
    local query="$1"
    local app_id="$2"
    local type="${3:-all}"  # all, workflows, scenarios, knowledge, code
    
    # Generate query embedding
    local query_embedding=$(ollama embed "$query" --model mxbai-embed-large)
    
    # Search appropriate collections
    if [[ "$type" == "all" ]]; then
        # Search all collections for this app
        for collection in $(qdrant::get_app_collections "$app_id"); do
            qdrant::search::collection "$collection" "$query_embedding"
        done
    else
        # Search specific type
        local collection="${app_id}-${type}"
        qdrant::search::collection "$collection" "$query_embedding"
    fi
}
```

### 6.2 Cross-App Search (`search/multi.sh`)
```bash
qdrant::search::all_apps() {
    local query="$1"
    local type="${2:-all}"
    
    echo "=== Cross-App Search Results ==="
    echo "Query: $query"
    echo ""
    
    # Find all app identities
    for identity_file in $(find . -name "app-identity.json" -type f); do
        local app_id=$(jq -r '.app_id' "$identity_file")
        local results=$(qdrant::search::single_app "$query" "$app_id" "$type")
        
        if [[ -n "$results" ]]; then
            echo "📱 App: $app_id"
            echo "$results"
            echo ""
        fi
    done
}
```

## Phase 7: CLI Integration
**Goal**: Add user-friendly commands to resource-qdrant

### 7.1 CLI Commands
```bash
# In qdrant/cli.sh, add:
cli::register_command "embeddings" "Manage semantic embeddings" "qdrant_embeddings_dispatch"

qdrant_embeddings_dispatch() {
    local subcommand="${1:-help}"
    shift || true
    
    case "$subcommand" in
        refresh)
            qdrant::embeddings::refresh "$@"
            ;;
        validate)
            qdrant::embeddings::validate "$@"
            ;;
        search)
            qdrant::embeddings::search "$@"
            ;;
        status)
            qdrant::embeddings::status
            ;;
        gc)
            qdrant::embeddings::garbage_collect
            ;;
        help)
            qdrant::embeddings::show_help
            ;;
    esac
}
```

## Phase 8: Git Integration
**Goal**: Auto-refresh on git changes

### 8.1 Integration with manage.sh
```bash
# Add to manage.sh (main and generated apps)
check_embedding_updates() {
    if [[ ! -f .vrooli/app-identity.json ]]; then
        qdrant::embeddings::init_identity
    fi
    
    local last_commit=$(jq -r '.index_commit // ""' .vrooli/app-identity.json)
    local current_commit=$(git rev-parse HEAD 2>/dev/null || echo "no-git")
    
    if [[ "$current_commit" != "$last_commit" ]] || [[ ! -d .git ]]; then
        log::info "Updating embeddings (commit: $current_commit)"
        resource-qdrant embeddings refresh --quiet
    fi
}

# Hook into develop command
develop_with_embeddings() {
    check_embedding_updates
    original_develop_command
}
```

## Phase 9: Testing Strategy
**Goal**: Comprehensive test coverage

### 9.1 Test Categories
1. **Extractor Tests**: Verify each extractor properly parses content
2. **Embedding Tests**: Ensure embeddings are generated correctly
3. **Search Tests**: Validate search returns relevant results
4. **Validation Tests**: Check discovery of missing content
5. **Integration Tests**: End-to-end workflow testing

### 9.2 Test Fixtures
Create `__test__/fixtures/` with sample content for each type

## Phase 10: Documentation & Migration
**Goal**: Enable smooth adoption

### 10.1 Documentation Structure
- `README.md`: Overview and quick start
- `USAGE.md`: Detailed usage examples
- `MIGRATION.md`: Converting existing projects
- `AGENT_GUIDE.md`: How AI agents should use the system

### 10.2 Migration Script
```bash
#!/usr/bin/env bash
# One-time migration for existing projects

qdrant::migrate::existing_project() {
    # Create standard directories
    mkdir -p docs
    
    # Generate templates if missing
    for template in ARCHITECTURE SECURITY LESSONS_LEARNED; do
        if [[ ! -f "docs/${template}.md" ]]; then
            cp "$TEMPLATE_DIR/${template}.md" "docs/"
            echo "Created docs/${template}.md"
        fi
    done
    
    # Initialize app identity
    qdrant::embeddings::init_identity
    
    # Run first indexing
    qdrant::embeddings::refresh
}
```

## Implementation Timeline
- **Week 1**: Foundation (Phase 1-2) - Structure and templates
- **Week 2**: Extractors (Phase 3) - Content extraction modules
- **Week 3**: Core System (Phase 4-5) - Management and validation
- **Week 4**: Search (Phase 6) - Single and multi-app search
- **Week 5**: Integration (Phase 7-8) - CLI and git hooks
- **Week 6**: Testing & Polish (Phase 9-10) - Tests and documentation

## Success Metrics
1. **Coverage**: >80% of project knowledge embedded
2. **Performance**: Full refresh <60 seconds for typical app
3. **Accuracy**: Search returns relevant results in top 5
4. **Adoption**: All new scenarios include standard docs
5. **Discovery**: Agents successfully find and reuse patterns

## Risk Mitigation
1. **Performance**: Use batch processing, limit embedding size
2. **Storage**: Implement retention policies, clean old versions
3. **Accuracy**: Regular validation, user feedback loop
4. **Complexity**: Start simple, iterate based on usage

## Next Steps
1. Review and approve plan
2. Create foundation directories
3. Implement first extractor (workflows)
4. Build validation system
5. Test with main Vrooli app
6. Extend to generated apps
7. Add cross-app search
8. Document for agents