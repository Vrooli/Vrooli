# Swarm-Manager Prompts

This directory contains structured prompt templates for swarm-manager backlog item processing. Unlike prompt-manager skills (which are principle-based transferable guidance), these are execution templates with specific schemas and outputs.

## Directory Structure

```
prompts/
├── research/           # Deep research prompts for different item kinds
│   ├── deep-research-idea.md
│   ├── deep-research-fix.md
│   └── deep-research-other.md
├── workflow/           # Clarify/Suggest/Enhance workflow prompts
│   ├── clarify.md
│   ├── suggest.md
│   └── enhance.md
├── processing/         # Execution prompts for processing items
│   ├── processing-guidance.md  # Shared guidance (read by agents)
│   ├── process-idea.md
│   ├── process-fix.md
│   └── process-execute.md
└── README.md
```

## Prompt Structure

Each prompt file follows a consistent structure:

```markdown
# [Prompt Name]

## Purpose
[1-2 sentences describing the goal]

## Input Context
- What files/data the agent receives

## Output Requirements
- Target file(s) to create/update
- Schema (if structured output)

## Success Criteria
- [ ] Specific, measurable criteria

## Instructions
[Detailed agent instructions]

## Quality Guidelines
[What makes good vs poor output]

## Anti-Patterns
[What NOT to do]

## Template Variables
{{ITEM_NAME}}, {{ITEM_TITLE}}, etc.
```

## Template Variables

The PromptLoader substitutes these variables at runtime:

| Variable | Description |
|----------|-------------|
| `{{ITEM_NAME}}` | Sanitized folder name |
| `{{ITEM_TITLE}}` | Human-readable title |
| `{{ITEM_DESCRIPTION}}` | Full description text |
| `{{ITEM_KIND}}` | idea, fix, execute, research |
| `{{ITEM_STATUS}}` | Current status |
| `{{ITEM_PRIORITY}}` | Priority (0-100) |
| `{{ITEM_TAGS}}` | Comma-separated tags |
| `{{ITEM_FOLDER}}` | Absolute path to item folder |
| `{{RESEARCH_TARGET}}` | For research kind: target to research |

## Output Schemas

### Clarify Questions (clarify/questions.json)

```json
{
  "questions": [{
    "id": "q1",
    "question": "What is the expected user experience?",
    "category": "users|technical|scope|constraints|integration",
    "importance": "critical|important|nice-to-have",
    "options": ["Option A", "Option B", "Other"],
    "answer": ""
  }],
  "generated_at": "2024-01-15T10:30:00Z",
  "max_questions": 10
}
```

### Suggestions (suggest/suggestions.json)

```json
{
  "suggestions": [{
    "id": "s1",
    "suggestion": "Consider using WebSocket for real-time updates",
    "details": "This would improve UX by eliminating polling...",
    "category": "architecture|ux|scope|risk|opportunity",
    "impact": "high|medium|low",
    "status": "pending|accepted|rejected",
    "rejection_reason": ""
  }],
  "generated_at": "2024-01-15T10:30:00Z",
  "max_suggestions": 7
}
```

## Usage

Prompts are loaded by `api/internal/backlog/prompt_loader.go`:

```go
loader := NewPromptLoader(rootDir)
template, err := loader.Load("workflow", "clarify")
prompt := loader.Build(template, item)
```

## Modifying Prompts

1. Edit the relevant `.md` file
2. Test with a sample backlog item
3. Verify output matches expected schema
4. Update this README if adding new template variables
