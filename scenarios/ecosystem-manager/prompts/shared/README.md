# 📚 Shared Prompts Library

This directory contains universal knowledge that ALL Vrooli scenarios and resources must understand and apply. These prompts ensure consistency, quality, and knowledge preservation across the entire system.

## Contents

### Core Knowledge

1. **core/memory-system.md** - Qdrant memory system usage (MANDATORY)
2. **core/prd-protocol.md** - PRD-driven development approach (MANDATORY)
3. **core/validation-gates.md** - Five-gate validation system (MANDATORY)
4. **scenario-specific/cross-impact.md** - Impact analysis methodology (MANDATORY)
5. **resource-specific/v2-contract.md** - v2.0 resource contract requirements

## Usage

### In Prompts

Reference these shared prompts using the include directive:

```markdown
## Critical Knowledge

### 🧠 Qdrant Memory System
{{INCLUDE: core/memory-system.md}}

### 📄 PRD Methodology
{{INCLUDE: core/prd-protocol.md}}
```

### In Code

When processing prompts programmatically:

```python
def load_prompt_with_includes(prompt_path):
    with open(prompt_path, 'r') as f:
        content = f.read()
    
    # Process includes
    import re
    include_pattern = r'{{INCLUDE: ([^}]+)}}'
    
    def replace_include(match):
        include_path = match.group(1).strip()
        with open(include_path, 'r') as inc:
            return inc.read()
    
    return re.sub(include_pattern, replace_include, content)
```

### In Shell Scripts

```bash
#!/bin/bash
# Include shared knowledge in prompts

process_prompt() {
    local prompt_file="$1"
    local output_file="$2"
    
    # Process includes
    while IFS= read -r line; do
        if [[ "$line" =~ \{\{INCLUDE:\ ([^}]+)\}\} ]]; then
            include_file="${BASH_REMATCH[1]// /}"
            cat "$include_file"
        else
            echo "$line"
        fi
    done < "$prompt_file" > "$output_file"
}
```

## Adding New Shared Prompts

When adding new universal knowledge:

1. **Create the prompt file** in this directory
2. **Follow the structure**:
   - Clear title with emoji
   - "Critical Understanding" section
   - Concrete examples
   - Commands/code samples
   - Do's and Don'ts
   - Integration instructions

3. **Update this README** with the new prompt

4. **Update relevant scenarios** to include it:
   ```markdown
   {{INCLUDE: your-new-prompt.md}}
   ```

## Enforcement

### Automated Validation

```bash
#!/bin/bash
# validate-prompt-includes.sh

validate_prompt() {
    local prompt="$1"
    local required_includes=(
        "memory-system.md"
        "prd-protocol.md"
        "validation-gates.md"
    )
    
    for include in "${required_includes[@]}"; do
        if ! grep -q "$include" "$prompt"; then
            echo "❌ Missing required include: $include"
            return 1
        fi
    done
    
    echo "✅ All required includes present"
    return 0
}
```

### CI/CD Integration

```yaml
# .github/workflows/prompt-validation.yml
name: Validate Prompts
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Check Prompt Includes
        run: |
          for prompt in scenarios/*/prompts/*.md; do
            ./scripts/validate-prompt-includes.sh "$prompt"
          done
```

## Prompt Categories

### Scenario Prompts
Must include:
- core/memory-system.md
- core/prd-protocol.md
- core/validation-gates.md
- scenario-specific/cross-impact.md

### Resource Prompts
Must include:
- core/memory-system.md
- resource-specific/v2-contract.md
- core/validation-gates.md

### Improvement Prompts
Must include:
- core/memory-system.md
- core/prd-protocol.md
- core/validation-gates.md
- scenario-specific/cross-impact.md

## Version History

### v1.0 (2025-01-03)
- Initial shared prompts library
- Core five prompts created
- Include system documented

## Best Practices

### Do's ✅
- Always include ALL mandatory prompts
- Keep shared prompts up to date
- Document new patterns discovered
- Use consistent formatting
- Test include processing

### Don'ts ❌
- Don't duplicate content - use includes
- Don't skip mandatory includes
- Don't modify shared prompts without review
- Don't hardcode knowledge that should be shared

## Quick Reference

### Include All Core Knowledge
```markdown
{{INCLUDE: core/memory-system.md}}
{{INCLUDE: core/prd-protocol.md}}
{{INCLUDE: core/validation-gates.md}}
{{INCLUDE: scenario-specific/cross-impact.md}}
```

### Include for Resources
```markdown
{{INCLUDE: core/memory-system.md}}
{{INCLUDE: resource-specific/v2-contract.md}}
{{INCLUDE: core/validation-gates.md}}
```

## Monitoring Usage

Check which scenarios use shared prompts:

```bash
# Find scenarios using shared prompts
find scenarios -name "*.md" -exec grep -l "INCLUDE.*shared/prompts" {} \;

# Count usage of each prompt
for prompt in prompts/shared/*/*.md; do
    basename="$(basename $prompt)"
    count=$(grep -r "$basename" scenarios/*/prompts | wc -l)
    echo "$basename: $count uses"
done
```

## Remember

- **Shared knowledge prevents repetition** - Define once, use everywhere
- **Consistency is critical** - All agents must think the same way
- **Updates propagate automatically** - Fix once, fixed everywhere
- **Memory makes it permanent** - Document discoveries here
- **Validation ensures compliance** - No scenario without core knowledge

This shared prompt library is the foundation of Vrooli's collective intelligence. Every scenario and resource builds on this knowledge.