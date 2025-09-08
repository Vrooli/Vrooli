# Qdrant Deep Search Protocol

## Comprehensive Memory Search Strategy

Before creating any new resource or scenario, perform exhaustive Qdrant searches:

### 1. Direct Name Searches
```
- Exact name match: "resource_name"
- Partial matches: "resource*", "*name"
- Acronym expansion: "ML" → "machine learning"
- Synonym searches: "chat" → "messaging, communication"
```

### 2. Semantic Similarity Searches
```
- Concept similarity (threshold: 0.7)
- Feature similarity (threshold: 0.75)
- Use case similarity (threshold: 0.8)
- Technical stack similarity (threshold: 0.65)
```

### 3. Pattern Mining
```
Query patterns from successful implementations:
- Architecture patterns
- Error handling approaches
- Configuration structures
- API design patterns
- Testing strategies
```

### 4. Failure Analysis
```
Search for failed attempts and issues:
- "failed + [concept]"
- "issue + [technology]"
- "deprecated + [approach]"
- "migration from + [old solution]"
```

### 5. Cross-Reference Searches
```
Find related components:
- Dependencies: "requires + [resource]"
- Integrations: "integrates with + [scenario]"
- Conflicts: "incompatible with + [component]"
- Replacements: "replaces + [old resource]"
```

## Search Results Processing

1. **Deduplication**: Remove redundant findings
2. **Relevance Scoring**: Rate 0-10 for applicability
3. **Pattern Extraction**: Identify common successful patterns
4. **Anti-Pattern Detection**: Note what to avoid
5. **Knowledge Synthesis**: Combine insights into actionable guidance

## Required Search Coverage

Must complete the core 5 searches:
1. Exact concept match
2. Functional equivalents  
3. Related technologies
4. Similar use cases
5. Existing resources/scenarios for overlap

## Documentation in PRD

Include in PRD.md:
- "Prior Work" section with Qdrant findings
- "Lessons Learned" from similar implementations
- "Patterns to Follow" from successful examples
- "Pitfalls to Avoid" from failed attempts