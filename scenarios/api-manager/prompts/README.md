# API Manager Prompts

This directory contains prompt templates for automated fixes via Claude Code agents.

## Important Notes

### No Caching Policy
**CRITICAL:** These prompt files are designed to be read fresh from disk every time they are used. DO NOT cache these prompts in memory or any other storage. This allows for:
- Real-time prompt refinement during development
- Quick iteration on fix strategies
- Testing different approaches without restarting services

### Prompt Files

1. **standards-compliance-fix.txt** - Template for fixing standards compliance violations
   - Used when triggering fixes from the Standards Compliance page
   - Variables replaced at runtime:
     - `{{SCENARIO_NAME}}` - The name of the scenario being fixed
     - `{{VIOLATIONS_SUMMARY}}` - Brief summary of violations
     - `{{DETAILED_VIOLATIONS}}` - Full violation details with line numbers

2. **vulnerability-fix.txt** - Template for fixing security vulnerabilities
   - Used when triggering fixes from the Vulnerability Scanner page
   - Variables replaced at runtime:
     - `{{SCENARIO_NAME}}` - The name of the scenario being fixed
     - `{{VULNERABILITIES_SUMMARY}}` - Brief summary of vulnerabilities
     - `{{DETAILED_VULNERABILITIES}}` - Full vulnerability details with code snippets

## Implementation Details

The prompts are loaded fresh on each request by:
1. Reading the appropriate .txt file from disk
2. Replacing template variables with actual data
3. Passing the prompt to `resource-claude-code` via stdin
4. Never storing the prompt content in variables beyond the request lifecycle

## Modifying Prompts

To modify prompts:
1. Edit the .txt files directly
2. Changes take effect immediately on the next fix request
3. No service restart required
4. Test changes by triggering a fix from the UI

## Adding New Prompts

To add a new prompt type:
1. Create a new .txt file in this directory
2. Use `{{VARIABLE_NAME}}` syntax for dynamic content
3. Update the handler to load and process the new template
4. Document the variables here