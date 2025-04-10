This file tracks and manages tasks, features, and improvements for the Vrooli project. Each task is designed for autonomous execution with clear deliverables and step-by-step instructions to ensure that if a task is picked using the “go” command, it can be completed with minimal clarification.

# Task Operations

**organize:**  
_Scan the unstructured tasks, extract key details, and reformat them using the task template. If any task lacks critical information, flag it for clarification._

**reprioritize:**  
_Review all tasks and adjust priorities based on business impact, technical urgency, dependencies, and current project goals. Provide brief justifications for changes, and reorder tasks accordingly._

**suggest:**  
_Identify gaps in technical infrastructure, user experience, documentation, testing, or security. Based on these gaps and dependencies, suggest new tasks formatted with the task template and add them for review._

**update:**  
_Review each task's deliverables and progress indicators, update the status (TODO, IN_PROGRESS, BLOCKED, DONE), and reflect any new blockers or dependencies._

**cleanup:**  
_Merge duplicate tasks, split tasks that are too large, and archive completed items. Ensure all task entries have consistent formatting and sufficient detail._

**go:**  
_Select a task from the backlog and transition it to active work. Begin by drafting a brief plan and then update the task’s status as work progresses._

**engage:**  
_Prompt for user input or confirmation whenever process deviations or ambiguities are detected; record feedback for continuous improvement._

---

# Task Template

Each task must follow this format:

```markdown
### [Task Title]
Priority: [HIGH | MEDIUM | LOW]  
Status: [TODO | IN_PROGRESS | BLOCKED | DONE]
Dependencies: [None | string[]]
ParentTask: [None | string[]]

**Description:**  
Brief explanation of the task including specific steps or deliverables required for completion.

**Key Deliverables:**
- [ ] Deliverable 1
- [ ] Deliverable 2

---

# Unstructured Tasks

Add your unstructured tasks here:

---

# Active Tasks

## Improve AI Agent Integration
Priority: HIGH
Status: IN_PROGRESS

Enhance the project's AI-readability and automation capabilities.

- [x] Create ARCHITECTURE.md
- [x] Add package-level documentation
- [x] Create TASKS.md for better task tracking
- [ ] Enhance code documentation for better AI understanding

# ⌚ Backlog

## Storybook Tests Initialization
Priority: MEDIUM  
Status: TODO

**Description:**  
Initialize storybook tests for multiple components following the established patterns in ApiView.stories.tsx and ApiUpsert.stories.tsx. Ensure that mocked data is realistic, accurate, and matches the correct type.

**Key Deliverables:**
- [ ] Create storybook test files for dataConverter component
- [ ] Create storybook test files for dataStructure component
- [ ] Create storybook test files for focusMode component
- [ ] Create storybook test files for meeting component
- [ ] Create storybook test files for memberInvite component
- [ ] Create storybook test files for project component
- [ ] Create storybook test files for prompt component
- [ ] Create storybook test files for smartContract component
- [ ] Create storybook test files for team component
- [ ] Create storybook test files for user component
- [ ] Configure session parameters for all test files
- [ ] Set up MSW handlers for all required API routes
- [ ] Implement route parameters for all test scenarios

## Team Schema Enhancement
Priority: HIGH
Status: TODO

Implement comprehensive schema support for Teams to enable structured organization tools and visualizations.

- [ ] Design and implement schema for Business Model Canvas, like we've done previously (e.g. /shared/src/run/configs/run.ts)
- [ ] Create schema for organizational chart
- [ ] Add UI components for displaying Business Model Canvas
- [ ] Add interactive org chart visualization
- [ ] Implement data persistence (stringified field in team prisma table) for team schemas
- [ ] Create routine for generating team schema

### Authentication Settings UI Enhancement
Priority: MEDIUM
Status: TODO

Improve the user experience and functionality of the authentication settings interface.

- [ ] Add "Sign out all devices" feature
- [ ] Redesign SettingsAuthenticationView for better UX
- [ ] Add confirmation dialogs for sensitive actions
- [ ] Implement loading states for async operations
- [ ] Add success/error notifications
- [ ] Update authentication settings documentation

## Enhanced Testing Infrastructure
Priority: MEDIUM
Status: TODO

Improve the testing setup across all packages.

- [ ] Add more comprehensive unit tests
- [ ] Implement E2E testing with Playwright
- [ ] Add performance testing
- [ ] Improve test documentation

## Documentation Improvements
Priority: MEDIUM
Status: TODO

Enhance project documentation for better developer and user experience.

- [ ] Create comprehensive API documentation
- [ ] Add more code examples
- [ ] Improve setup instructions
- [ ] Add troubleshooting guides
- [ ] Update main README to link to project-level tasks, docs, ARCHITECTURE.md, etc.
- [ ] Enhance .env-example with detailed comments explaining each environment variable

## Performance Optimization
Priority: LOW
Status: TODO

Optimize application performance and resource usage.

- [ ] Implement caching strategies
- [ ] Optimize database queries
- [ ] Add performance monitoring
- [ ] Implement lazy loading where beneficial

## Local Installation Support
Priority: LOW
Status: TODO

Enable users to run Vrooli locally with their preferred storage and AI models.

- [ ] Implement configurable storage locations (local/cloud/decentralized)
- [ ] Add support for local AI models
- [ ] Create native desktop applications (.exe, .dmg, .deb)
- [ ] Implement offline-first capabilities
- [ ] Ensure cross-platform compatibility
