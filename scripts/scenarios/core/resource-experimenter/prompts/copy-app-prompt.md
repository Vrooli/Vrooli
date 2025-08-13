# Copy Existing App Prompt

You are Claude Code. Your task is to create an exact copy of an existing Vrooli scenario and prepare it for modification.

## Your Assignment

**Source Scenario:** {{TARGET_SCENARIO}}
**Target Location:** {{TARGET_LOCATION}}

## Task Details

1. **Read the entire scenario** from `/home/matthalloran8/Vrooli/scripts/scenarios/core/{{TARGET_SCENARIO}}/`

2. **Copy all files and directories** to the target location, maintaining the exact structure:
   ```
   {{TARGET_SCENARIO}}/
   ├── .vrooli/
   │   └── service.json
   ├── initialization/
   │   ├── automation/
   │   ├── configuration/  
   │   └── storage/
   ├── api/ (if present)
   ├── cli/ (if present)
   ├── ui/ (if present)
   ├── deployment/ (if present)
   ├── prompts/ (if present)
   ├── scenario-test.yaml
   └── test.sh (if present)
   ```

3. **Present the complete file structure** with all file contents

4. **Verify completeness** by listing all copied files and their purposes

## Output Format

```
## Copied Scenario: {{TARGET_SCENARIO}}

### File Structure:
[complete directory tree]

### Files Copied:

#### .vrooli/service.json
[complete file content]

#### initialization/[category]/[file]
[complete file content for each file]

[continue for all files]

### Copy Summary:
- Total files: X
- Directories: Y
- Configuration files: Z
- Automation workflows: A
- etc.
```

## Important Notes

- Preserve ALL existing functionality
- Maintain exact file permissions and structure
- Include ALL files (don't skip any)
- Keep original comments and documentation
- Verify no files are missing

Your goal is to create a perfect, identical copy that can serve as the base for resource integration experiments.