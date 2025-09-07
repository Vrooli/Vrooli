## 🔍 Pre-Generation Research Checklist

Before generating ANY new resource or scenario, you MUST complete this comprehensive research checklist. This ensures we don't duplicate work, learn from past efforts, and build on proven patterns.

### ✅ Mandatory Research Steps

#### 1. Search Qdrant Memory (REQUIRED)
```bash
# Search for similar implementations
vrooli resource-qdrant search-all "{{TARGET_NAME}} {{CATEGORY}}"
vrooli resource-qdrant search "similar to {{DESCRIPTION}}" all

# Check for existing work
vrooli resource-qdrant search "{{TARGET_TYPE}} {{CATEGORY}}" scenarios
vrooli resource-qdrant search "{{TARGET_TYPE}} {{CATEGORY}}" resources

# Learn from failures
vrooli resource-qdrant search "{{CATEGORY}} failed" all
vrooli resource-qdrant search "{{CATEGORY}} error" docs
vrooli resource-qdrant search "{{CATEGORY}} problem" all

# Find proven patterns
vrooli resource-qdrant search "{{CATEGORY}} pattern" code
vrooli resource-qdrant search "{{CATEGORY}} best practice" docs
vrooli resource-qdrant search "{{CATEGORY}} architecture" docs
```

#### 2. Check Existing Implementations
- [ ] Search for similar scenarios/resources in the codebase
- [ ] Review their PRDs for requirements overlap
- [ ] Analyze their implementation patterns
- [ ] Note their success metrics and adoption rates
- [ ] Identify reusable components or templates

#### 3. Analyze Market/Competition (for scenarios)
- [ ] Research similar commercial solutions
- [ ] Identify unique value propositions
- [ ] Estimate market size and revenue potential
- [ ] Analyze competitor pricing models
- [ ] Document differentiators

#### 4. Technical Feasibility Assessment
- [ ] Verify all required resources are available
- [ ] Check for resource conflicts (ports, names)
- [ ] Validate integration possibilities
- [ ] Assess performance requirements
- [ ] Identify potential technical blockers

#### 5. Pattern Identification
- [ ] Document similar successful implementations
- [ ] Note common architectural patterns
- [ ] Identify reusable workflows or templates
- [ ] List applicable design patterns
- [ ] Record relevant code snippets

### 📊 Research Summary Template

After completing research, summarize your findings:

```markdown
## Research Summary for {{TARGET_NAME}}

### Similar Existing Work
- **Found**: [List similar implementations]
- **Reusable**: [Components that can be adapted]
- **Differentiators**: [What makes this unique]

### Lessons from Past Attempts
- **Failures to Avoid**: [Known issues and failed approaches]
- **Success Patterns**: [Proven approaches that work]
- **Technical Gotchas**: [Hidden complexities discovered]

### Market Analysis (if applicable)
- **Competition**: [Similar solutions in market]
- **Value Proposition**: [Unique value we provide]
- **Revenue Potential**: [Estimated value]

### Technical Approach
- **Architecture Pattern**: [Recommended pattern]
- **Resource Selection**: [Resources needed and why]
- **Integration Points**: [How components connect]
- **Risk Factors**: [Technical risks identified]

### Reusable Assets
- **Templates**: [Existing templates to use]
- **Code**: [Reusable code components]
- **Workflows**: [Proven workflows to adapt]
- **Documentation**: [Relevant docs to reference]
```

### 🚫 Generation Blockers

DO NOT proceed with generation if:
- ❌ An identical or very similar implementation already exists
- ❌ Technical feasibility cannot be confirmed
- ❌ Required resources are unavailable or conflicting
- ❌ No clear value proposition or differentiation exists
- ❌ Critical technical blockers are identified

### ✨ Research Quality Metrics

Your research is complete when you can answer:
1. **Uniqueness**: How is this different from existing work?
2. **Value**: What specific problem does this solve?
3. **Feasibility**: Can this be built with available resources?
4. **Patterns**: What proven patterns will you follow?
5. **Risks**: What are the main technical risks?
6. **Reusability**: What existing work can be leveraged?
7. **Revenue**: What's the business value? (for scenarios)
8. **Timeline**: How long will implementation take?

### 💡 Research Tips

- **Cast a wide net initially**: Start with broad searches, then narrow down
- **Look for negative signals**: Failed attempts often teach the most
- **Check multiple categories**: Solutions might exist under different names
- **Review recent work first**: Newer implementations likely use better patterns
- **Document everything**: Your research helps future developers

### 🔄 Continuous Learning

After generation, update Qdrant with:
- New patterns discovered during implementation
- Challenges encountered and solutions
- Performance metrics and optimizations
- Integration insights
- Reusable components created

Remember: **Every hour of research saves days of rework.**