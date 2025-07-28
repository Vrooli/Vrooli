# SearXNG Enhancement Plan

This document outlines the planned enhancements to the SearXNG manage.sh script to make it more powerful and user-friendly for AI agents and automation workflows.

## Current Capabilities

The manage.sh script currently supports:
- Basic search with parameters (`--query`, `--format`, `--category`, `--language`)
- Interactive search mode
- API testing & benchmarking
- Management actions (status, logs, info, config, diagnose)

## Enhancement Roadmap

### High Priority Features

#### 1. Advanced Search Parameters
```bash
./manage.sh --action search \
  --query "tech news" \
  --pageno 2 \
  --time-range day \
  --safesearch 0 \
  --engines "google,bing"
```

**Implementation Details:**
- Add `--pageno` for pagination support
- Add `--time-range` with values: hour, day, week, month, year
- Add `--safesearch` with values: 0 (off), 1 (moderate), 2 (strict)
- Add `--engines` to filter specific search engines

#### 2. Formatted Output Options
```bash
./manage.sh --action search --query "AI" --output-format "title-only"
./manage.sh --action search --query "AI" --output-format "title-url"
./manage.sh --action search --query "AI" --limit 5
```

**Output Formats:**
- `json` (default) - Full JSON response
- `title-only` - Just titles, one per line
- `title-url` - Title and URL pairs
- `csv` - CSV format for spreadsheet import
- `markdown` - Markdown formatted results
- `compact` - Condensed human-readable format

#### 3. Save Results
```bash
./manage.sh --action search --query "AI research" --save results.json
./manage.sh --action search --query "AI" --append search-history.jsonl
```

**Features:**
- `--save filename` - Save results to file
- `--append filename` - Append to existing file (JSONL format)
- Auto-timestamping for append mode
- Error handling for file permissions

#### 4. Batch Search Capabilities
```bash
./manage.sh --action batch-search --file queries.txt --output results/
./manage.sh --action batch-search --queries "AI,ML,quantum" --format csv
```

**Features:**
- Read queries from file (one per line)
- Read queries from comma-separated string
- Save each query result to separate file
- Progress indicators for large batches

### Medium Priority Features

#### 5. Quick Actions
```bash
./manage.sh --action headlines [--topic "tech"]
./manage.sh --action lucky --query "Python documentation"
./manage.sh --action trending
```

**Actions:**
- `headlines` - Latest news headlines (news category + recent time range)
- `lucky` - Return first result URL only (like "I'm Feeling Lucky")
- `trending` - Popular search terms (if available from API)

#### 6. Result Filtering/Extraction
```bash
./manage.sh --action search --query "data" --extract domains
./manage.sh --action search --query "research" --filter-engine wikipedia
./manage.sh --action search --query "news" --min-score 0.8
```

**Filters:**
- `--extract domains` - Extract unique domains from results
- `--extract emails` - Extract email addresses
- `--filter-engine name` - Show only results from specific engine
- `--min-score N` - Filter by relevance score

#### 7. Export Formats for Integrations
```bash
./manage.sh --action search --query "data" --export-for n8n
./manage.sh --action search --query "data" --export-for obsidian
./manage.sh --action search --query "AI news" --generate-rss
```

**Export Formats:**
- `n8n` - Format optimized for n8n workflows
- `obsidian` - Markdown format for Obsidian notes
- `rss` - Generate RSS feed from results
- `csv` - Spreadsheet-friendly format

#### 8. Search Presets
```bash
./manage.sh --action research --topic "quantum computing"
./manage.sh --action shop --query "laptop"
./manage.sh --action academic --query "machine learning"
```

**Presets:**
- `research` - Academic sources, extended time range, science category
- `shop` - Shopping-focused engines and categories
- `academic` - Scholarly articles and papers
- `news` - Recent news with time filtering
- `images` - Image search with large result sets

## Implementation Strategy

### Phase 1: Core Parameter Expansion
1. Extend `searxng::search()` function to accept additional parameters
2. Update argument parsing in `manage.sh`
3. Add parameter validation and error handling

### Phase 2: Output Formatting
1. Create output formatting functions
2. Add post-processing for different formats
3. Implement file saving capabilities

### Phase 3: Advanced Features
1. Implement batch operations
2. Add quick action presets
3. Create filtering and extraction utilities

### Phase 4: Integration & Testing
1. Update comprehensive test suite
2. Update README documentation
3. Add usage examples and tutorials

## Technical Considerations

### Backward Compatibility
- All new parameters are optional
- Existing functionality remains unchanged
- Default behavior preserved

### Error Handling
- Graceful degradation for unsupported features
- Clear error messages for invalid parameters
- Validation of file paths and permissions

### Performance
- Efficient batch processing with proper rate limiting
- Memory-conscious handling of large result sets
- Progress indicators for long-running operations

### Security
- Input sanitization for all parameters
- Safe file operations with proper permissions
- No shell injection vulnerabilities

## Testing Requirements

### Unit Tests
- Parameter parsing validation
- Output formatting accuracy
- File operation safety
- Error condition handling

### Integration Tests
- End-to-end search workflows
- Batch operation reliability
- Format conversion accuracy
- Real API interaction validation

### Performance Tests
- Large batch processing
- Memory usage monitoring
- Response time benchmarks
- Rate limiting compliance

## Documentation Updates

### README.md Updates
- New parameter reference table
- Updated examples section
- Integration workflow examples
- Troubleshooting for new features

### Help System
- Extended `--help` output
- Parameter validation messages
- Example usage hints
- Best practice recommendations

---

## Implementation Checklist

- [ ] Core parameter expansion
- [ ] Output formatting system
- [ ] File save/append functionality
- [ ] Batch search implementation
- [ ] Quick action presets
- [ ] Result filtering utilities
- [ ] Export format handlers
- [ ] Comprehensive test coverage
- [ ] Documentation updates
- [ ] Performance optimization

---

*This plan ensures SearXNG becomes a powerful, flexible tool for AI agents while maintaining simplicity and reliability.*