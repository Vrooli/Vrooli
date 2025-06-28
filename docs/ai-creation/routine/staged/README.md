# Staged Routines Organization

This directory contains AI-generated routines organized by category and properly tagged for the Vrooli platform.

## Directory Structure

```
staged/
├── meta-intelligence/      # System-level intelligence, memory, capabilities
├── personal/              # Personal development, habits, wellness
├── professional/          # Career, leadership, team building
├── innovation/            # Creative problem-solving, brainstorming
├── research/              # Analysis, synthesis, documentation
├── technical/             # Code, APIs, data processing
├── financial/             # Budgets, investments, planning
├── decision-making/       # Crisis management, risk, scenarios
├── productivity/          # Tasks, time management, workflows
├── learning/              # Education, skills, knowledge retention
├── communication/         # Writing, content, documentation
├── infrastructure/        # Systems, processes, operations
├── executive/             # C-suite, governance, strategy
├── business/              # Sales, partnerships, growth
├── crm/                   # Customer lifecycle, support
├── lifestyle/             # Health, family, home, travel
└── content-creation/      # Digital content, tutorials
```

## Tag System

Each routine includes tags from two sources:

### 1. Primary Category Tags
Each category folder has a `_tags.json` file listing available tags specific to that domain.

### 2. Cross-Cutting Feature Tags
Available across all categories:
- `analysis` - Analytical capabilities
- `automation` - Process automation
- `coaching` - Guidance and mentoring
- `monitoring` - Tracking and observation
- `planning` - Organization and strategy
- `optimization` - Performance improvement
- `synthesis` - Information combination
- `generation` - Content creation
- `ai-agent` - AI orchestration
- `real-time` - Live processing

## Tag Structure

Tags in routines follow the Vrooli API response format:

```json
{
  "tags": [
    {
      "id": "7829564732190847001",
      "tag": "meta-intelligence",
      "translations": [{
        "id": "7829564732190847101",
        "language": "en",
        "description": "System-level intelligence and self-improvement"
      }]
    }
  ]
}
```

## Adding New Routines

1. Determine the primary category
2. Select 2-3 appropriate tags:
   - 1 primary category tag
   - 1-2 feature/capability tags
3. Use tags from `tags-reference.json` or the category's `_tags.json`
4. Place in the appropriate category folder

## Tag Guidelines

- **Minimum**: 1 tag (primary category)
- **Maximum**: 4 tags (avoid over-tagging)
- **Balance**: Mix category-specific and cross-cutting tags
- **Accuracy**: Choose tags that accurately represent functionality

## Import Considerations

When importing to Vrooli:
- Tags help with discovery and search
- Category organization aids navigation
- Proper tagging enables better AI recommendations
- Tags will be validated against the platform's tag system

## Maintenance

- Keep `tags-reference.json` updated with new tags
- Ensure category folders have current `_tags.json`
- Review and update tags as routines evolve
- Consider user feedback on tag effectiveness