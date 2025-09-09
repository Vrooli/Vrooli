# Task Completion Protocol

## Purpose
When you complete your assigned task, properly document your work and update the task status to enable future improvements.

## Task Completion Steps

### 1. Final Validation
Verify your work actually functions:
- Test key features you implemented/improved
- Confirm health checks pass (if applicable)
- Verify no regressions in existing functionality

### 2. Update Documentation
Follow the `scripts/resources/contracts/v2.0/universal.yaml` guidelines for proper documentation structure:

**For Resources:**
- Update PRD.md with completed requirements using: `[x] Description ✅ YYYY-MM-DD`
- Include test commands that prove functionality works
- Update README.md to reflect current capabilities
- Document any new configuration or dependencies

**For Scenarios:**
- Update PRD.md progress checkboxes with completion dates
- Document new features in README.md
- Update any UI workflow documentation
- Note any API changes or new endpoints

### 3. Capture Learnings
Document insights in your response that will help future agents:
- What approaches worked well
- What challenges you encountered and how you solved them
- Specific test commands that validate the work
- Recommendations for future improvements

### 4. Task Status Update
The ecosystem-manager API handles moving your task file automatically. Your final response should clearly state:
- What was accomplished
- Current status (working/improved/partially complete)
- Any remaining issues or limitations
- Specific evidence that validates the work

## Documentation Organization is Critical

Proper documentation enables the semantic knowledge system to understand and build upon your work. Always:
- Use consistent naming and structure
- Include specific examples and test commands  
- Reference the v2.0 contract standards
- Write for future agents who haven't seen your work

Well-documented work multiplies in value - future agents can build on it effectively instead of starting from scratch.