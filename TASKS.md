# Vrooli Tasks

This file tracks tasks, features, and improvements for the Vrooli project. Tasks are organized by status and priority.

## 🤖 AI Actions and Prompts

### 1. Organize Unstructured Tasks
Use this prompt to convert unstructured tasks into properly formatted entries:
```markdown
Given the unstructured tasks:
1. Analyze each task to determine:
   - A clear, concise title
   - Priority (HIGH/MEDIUM/LOW) based on impact and urgency
   - Status (TODO/IN_PROGRESS/BLOCKED/DONE)
   - Appropriate section (Active Tasks or Backlog)
2. Break down tasks into clear deliverables
3. Format using the task template
4. Place in appropriate sections
5. Remove from unstructured section
6. If any task lacks critical information, ask for clarification
```

### 2. Reprioritize Tasks
Use this prompt to review and adjust task priorities:
```markdown
Review all tasks and:
1. Evaluate each task's priority based on:
   - Business impact (high/medium/low)
   - Technical urgency (high/medium/low)
   - Dependencies on other tasks
   - Current project goals
2. Adjust priority levels (HIGH/MEDIUM/LOW)
3. Reorder tasks within sections based on new priorities
4. Provide brief justification for any priority changes
5. Move tasks between Active and Backlog if needed
```

### 3. Suggest New Tasks
Use this prompt to suggest new tasks based on existing ones:
```markdown
Analyze the current tasks and:
1. Identify potential gaps in:
   - Technical infrastructure
   - User experience
   - Documentation
   - Testing coverage
   - Security measures
2. Consider dependencies of existing tasks
3. Review industry best practices
4. Suggest new tasks that would:
   - Complement existing tasks
   - Address identified gaps
   - Improve project quality
5. Format suggestions using the task template
6. Add to unstructured section for review
```

### 4. Update Task Status
Use this prompt to review and update task status:
```markdown
For each task:
1. Review deliverables and completion criteria
2. Check for:
   - Completed deliverables
   - Blocked items
   - Dependencies
   - Progress indicators
3. Update status to reflect current state
4. Move between sections if needed
5. Add any new blockers or dependencies
6. Update deliverable checkboxes
```

### 5. Clean Up and Consolidate
Use this prompt to maintain task list clarity:
```markdown
Review all tasks and:
1. Identify and merge duplicate tasks
2. Break down overly large tasks
3. Remove or archive completed tasks
4. Update outdated information
5. Ensure consistent formatting
6. Add missing details to incomplete entries
7. Validate all task dependencies
```

## 📄 Task Template
```markdown
### [Task Title]
Priority: [HIGH|MEDIUM/LOW]
Status: [TODO/IN_PROGRESS/BLOCKED/DONE]

Description of the task and any relevant details.

- [ ] Key deliverable 1
- [ ] Key deliverable 2
```

## 📝 Unstructured Tasks

Add your unstructured tasks below this line:
-------------------------------------------------------------------------------

## 🏃‍♂️ Active Tasks

### Improve AI Agent Integration
Priority: HIGH
Status: IN_PROGRESS

Enhance the project's AI-readability and automation capabilities.

- [x] Create ARCHITECTURE.md
- [x] Add package-level documentation
- [x] Create TASKS.md for better task tracking
- [ ] Enhance code documentation for better AI understanding

## ⌚ Backlog

### Team Schema Enhancement
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

### Enhanced Testing Infrastructure
Priority: MEDIUM
Status: TODO

Improve the testing setup across all packages.

- [ ] Add more comprehensive unit tests
- [ ] Implement E2E testing with Playwright
- [ ] Add performance testing
- [ ] Improve test documentation

### Documentation Improvements
Priority: MEDIUM
Status: TODO

Enhance project documentation for better developer and user experience.

- [ ] Create comprehensive API documentation
- [ ] Add more code examples
- [ ] Improve setup instructions
- [ ] Add troubleshooting guides
- [ ] Update main README to link to project-level tasks, docs, ARCHITECTURE.md, etc.
- [ ] Enhance .env-example with detailed comments explaining each environment variable

### Performance Optimization
Priority: LOW
Status: TODO

Optimize application performance and resource usage.

- [ ] Implement caching strategies
- [ ] Optimize database queries
- [ ] Add performance monitoring
- [ ] Implement lazy loading where beneficial

### Local Installation Support
Priority: LOW
Status: TODO

Enable users to run Vrooli locally with their preferred storage and AI models.

- [ ] Implement configurable storage locations (local/cloud/decentralized)
- [ ] Add support for local AI models
- [ ] Create native desktop applications (.exe, .dmg, .deb)
- [ ] Implement offline-first capabilities
- [ ] Ensure cross-platform compatibility
