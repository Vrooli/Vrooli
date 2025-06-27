# Smart Subroutine Resolution System

## Overview

This system intelligently resolves subroutine dependencies for routine generation, avoiding TODO placeholders and duplicate generation.

## Multi-Pass Resolution Process

### Pass 1: Dependency Analysis
1. **Parse Routine Requirements**: Extract needed subroutine capabilities from backlog item
2. **Build Dependency Tree**: Identify which subroutines depend on others
3. **Categorize by Type**: Group by Generate, Code, API, etc.

### Pass 2: Subroutine Discovery & Matching
For each needed subroutine capability:

1. **Semantic Search Database**:
   ```bash
   vrooli routine search "text sentiment analysis" --format ids
   ```

2. **Check Staged Files**: Scan `staged/` directory for matching functionality

3. **Embeddings Matching**: Use CLI to find semantically similar routines

4. **Capability Scoring**: Rate how well existing routines match requirements

### Pass 3: Smart Resolution
1. **Reuse High-Confidence Matches**: Use existing subroutines with >80% match
2. **Generate Missing Leaf Nodes**: Create simple subroutines with no dependencies first  
3. **Work Up Dependency Tree**: Generate complex subroutines that depend on simpler ones
4. **Update Staging Index**: Track generated subroutines to avoid duplicates

### Pass 4: Main Routine Generation
1. **Map All Dependencies**: Replace capability descriptions with actual subroutine IDs
2. **Validate Data Flow**: Ensure all inputs/outputs align properly
3. **Generate Final Routine**: Create main routine with complete subroutine references

## CLI Commands Used

```bash
# Semantic search for existing subroutines
vrooli routine search "analyze sentiment" --format ids

# Get all available routines with metadata  
vrooli routine discover --format mapping

# Validate generated routines
vrooli routine validate staged/subroutine.json

# Export routine for template analysis
vrooli routine export <id> -o template.json
```

## File Structure Updates

```
docs/ai-creation/routine/
├── staged/
│   ├── subroutines/          # Generated subroutines
│   └── main-routines/        # Main routines that use subroutines
├── cache/
│   ├── search-results.json   # Cached search results
│   ├── staged-index.json     # Index of staged subroutines
│   └── capability-map.json   # Capability to ID mappings
└── templates/
    └── subroutine-templates.json  # Common subroutine patterns
```

## Subroutine Capability Taxonomy

### Text Processing
- `sentiment-analysis`: Analyze text emotion/sentiment
- `text-summarization`: Condense long text to key points  
- `keyword-extraction`: Extract important terms
- `language-detection`: Identify text language

### Data Analysis  
- `data-correlation`: Find relationships in datasets
- `statistical-summary`: Calculate stats and distributions
- `anomaly-detection`: Identify outliers and unusual patterns

### Content Generation
- `text-generation`: Create written content
- `response-formatting`: Structure AI responses
- `template-filling`: Populate templates with data

### Research & Information
- `web-search`: Search for information online
- `fact-checking`: Verify claims against sources
- `source-compilation`: Gather information from multiple sources

## Resolution Algorithm

```python
def resolve_subroutines(capability_requirements):
    resolved = {}
    generation_queue = []
    
    for capability in capability_requirements:
        # 1. Search existing database
        matches = search_database(capability.description, threshold=0.8)
        
        if matches:
            resolved[capability.id] = select_best_match(matches)
        else:
            # 2. Check staged files  
            staged_match = search_staged_files(capability)
            
            if staged_match:
                resolved[capability.id] = staged_match.id
            else:
                # 3. Add to generation queue
                generation_queue.append(capability)
    
    # 4. Generate missing subroutines
    for capability in generation_queue:
        subroutine_id = generate_subroutine(capability)
        resolved[capability.id] = subroutine_id
        
    return resolved
```

## Integration with Generation Scripts

The `routine-generate.sh` script will be enhanced to:

1. **Pre-Process**: Extract subroutine requirements from backlog item
2. **Resolve Dependencies**: Run the smart resolution system  
3. **Generate Hierarchically**: Create subroutines before main routines
4. **Post-Validate**: Ensure all references are valid

## Error Handling

- **Search Timeouts**: Fall back to generation if search takes too long
- **Invalid Matches**: Validate semantic matches actually work for the use case
- **Circular Dependencies**: Detect and resolve circular subroutine dependencies
- **Generation Failures**: Retry with simpler subroutine designs

## Performance Optimizations

- **Caching**: Store search results and capability mappings
- **Batch Operations**: Group similar searches together
- **Incremental Updates**: Only search for new capabilities
- **Parallel Generation**: Generate independent subroutines concurrently