This file tracks and manages tasks, features, and improvements for the Vrooli project. Each task is designed for autonomous execution with clear deliverables and step-by-step instructions to ensure that if a task is picked using the "go" command, it can be completed with minimal clarification.

# Task Operations
**organize:**  
- For the first unstructured task in the list (not all tasks at once):
  - Thoroughly explore the codebase to understand task context:
    - Search for related files and code patterns to gain deeper context
    - Analyze existing implementations of similar features
    - Review code architecture to understand integration points
    - Identify potential dependencies not explicitly mentioned
  - Ask clarifying questions if the task description is incomplete or ambiguous
  - Conduct research on any technical terms or concepts that need clarification
  - Extract key details and enhance them with additional insights from codebase exploration
  - Reformat the task directly in TASKS.md (not in a scratch file) following these guidelines:
    - Use proper task template format with title, priority, status, dependencies
    - Expand descriptions to include implementation approaches and technical considerations
    - Add detailed context about where in the codebase the task applies
    - Identify specific files that will need to be modified
    - Suggest potential implementation strategies based on existing code patterns
    - Break down deliverables into more granular steps where appropriate
  - Remove the original unstructured task from the list after organizing it
  - Flag any task that still lacks critical information after research, with specific questions

**reprioritize:**  
- Review all tasks and adjust priorities (HIGH/MEDIUM/LOW) based on business impact, technical urgency, dependencies, and current project goals. Provide brief justifications for changes, and reorder tasks accordingly.

**suggest:**  
- Identify gaps in technical infrastructure, user experience, documentation, testing, or security. Based on these gaps and dependencies, suggest new tasks formatted with the task template and add them for review.

**update:**  
- Review each task's deliverables and progress indicators, update the status (TODO/IN_PROGRESS/BLOCKED/DONE), and reflect any new blockers or dependencies.

**cleanup:**  
- Merge duplicate tasks, split tasks that are too large, and archive completed items. Ensure all task entries have consistent formatting and sufficient detail.

**go:**  
- Select the highest-priority task from the backlog, prioritizing those related to recently completed work when appropriate to reduce context switching. First explore the codebase to identify possible implementation approaches, then draft a brief plan with options. After confirmation, automatically update the task status to IN_PROGRESS. When you think the task is completed, wait for confirmation before changing the task to DONE and moving it to the "Completed Tasks" section.

**engage:**  
- Prompt: "Would you like me to review and update our task list or start working on a backlog item?" and wait for input.

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
- Update ChatBubbleTree to render not only the message tree, but also show the messages triggered by routines. More on that in another task
- Create a component for running routines, that's displayed in a chat. It should be a rounded box with a top box and 2 main sections. The top should have a loading spinner, the routine title, and the seconds elapsed. Under the top box is a section to the left and a secton to the right. The left section should display the steps taken. Each step should have a loading spinner if active, or an icon to represent the state if completed/failed/etc. The right section should be any information about the active tasks, which we'll update later.
---

# Active Tasks

### Enhance Chat and Message Storage with Metadata
Priority: HIGH  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Implement functionality to store additional contextual information in chats and chat messages using the existing metadata field. Chats need to store information about connected projects and other related data, while chat messages need to store information about triggered routines, tools called, and other execution context. This information will be stored as stringified JSON with appropriate serialization/deserialization functions similar to the approach used in packages/shared/src/run/configs.

**Key Deliverables:**
- [ ] Design and implement a metadata structure for chats that can store:
  - Connected project information
  - Reference information to other Vrooli objects
  - Configuration and context data
- [ ] Design and implement a metadata structure for chat messages that can store:
  - Routines triggered by the message
  - Tools called during message processing
  - Execution results and state information
- [ ] Create utility functions in packages/shared/src/run/configs or a new appropriate location for:
  - Serializing chat metadata to JSON strings
  - Parsing metadata JSON strings back to their typed objects
  - Validating metadata structure
- [ ] Update the ChatMessage model (packages/server/src/models/base/chatMessage.ts) to handle the metadata field:
  - Add metadata to the create and update operations
  - Add validation for metadata content
- [ ] Update the Chat model (packages/server/src/models/base/chat.ts) with similar metadata handling
- [ ] Add TypeScript types and interfaces for the metadata structures
- [ ] Implement test cases for serialization/deserialization

### Refactor Active Chat Store with Chat ID Identifier
Priority: HIGH  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Refactor the active chat Zustand store and its accompanying hook to use an identifier-based approach instead of tracking a single active chat object. This change will make the architecture more maintainable and will support multiple chat scenarios. Additionally, modify the AdvancedInput component to store tools and contexts in the chat Zustand store rather than receiving them as parameters.

**Key Deliverables:**
- [ ] Update `activeChatStore.ts` to track active chats by identifier (either an actual chat ID or DUMMY_ID)
- [ ] Modify the `useActiveChat` hook to support the identifier-based approach
- [ ] Refactor the store to maintain a map/dictionary of chats indexed by their identifiers
- [ ] Update all code that accesses the current active chat to use the new identifier-based mechanism
- [ ] Modify the AdvancedInput component to retrieve tools and contexts from the chat Zustand store
- [ ] Create comprehensive unit tests for the updated store and hook functionality
- [ ] Update any related components that depend on the active chat store

---

# ⌚ Backlog

## Fix Commenting System
Priority: MEDIUM  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Fix the commenting functionality throughout the application, starting with the UI components and then implementing necessary server-side changes.

**Key Deliverables:**
- [ ] Identify and fix UI issues in the commenting system
- [ ] Implement necessary server-side fixes
- [ ] Test commenting functionality across different parts of the application
- [ ] Ensure proper error handling for comment operations

## Multi-Agent Workflow Functionality
Priority: HIGH  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Implement functionality to allow routines to specify role-based restrictions for each BPMN process. This will enable multi-agent workflows where different types of agents (project manager, software engineer, etc.) can work through parallel processes to complete tasks.

**Key Deliverables:**
- [ ] Design role-based restriction schema for BPMN processes
- [ ] Implement UI for configuring role-based restrictions
- [ ] Develop server-side logic to enforce restrictions
- [ ] Create documentation for multi-agent workflow configuration
- [ ] Test parallel processes with different agent roles

## Advanced Input Component Refactoring
Priority: MEDIUM  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Remove legacy input components (RichInput, RichInputBase, ChatMessageInput, etc.) after fully implementing AdvancedInput as their replacement.

**Key Deliverables:**
- [ ] Ensure AdvancedInput fully supports all features of legacy components
- [ ] Migrate all instances of legacy components to AdvancedInput
- [ ] Remove deprecated components: RichInput, RichInputBase, ChatMessageInput, RichInputLexical, RichInputMarkdown, RichInputTagDropdown
- [ ] Remove associated hooks including useTagDropdown
- [ ] Update documentation to reflect new input component architecture

## Enhance IntegerInput Component
Priority: LOW  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Extend the IntegerInput component to support currency symbols and other formatting options.

**Key Deliverables:**
- [ ] Add currency symbol support to IntegerInput
- [ ] Implement proper formatting for different currency types
- [ ] Add configuration options for currency display
- [ ] Update component documentation
- [ ] Add tests for currency formatting functionality

### Model Selector Improvements in Dashboard
Priority: MEDIUM  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Fix usability issues and enhance the model selector in the Dashboard with additional functionality and improved UI.

**Key Deliverables:**
- [ ] Fix dialog closing behavior (clicking outside or close button should close it)
- [ ] Implement toggles for chat tools (web search, command execution, file search, memory, etc.)
- [ ] Improve overall UI polish and visual design
- [ ] Ensure consistent behavior across the application

### File Upload and Storage System
Priority: HIGH  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Implement a file upload system with flexible storage options (S3, Google Drive, etc.) using dependency injection. Enable file attachments to projects with references for LLM retrieval.

**Key Deliverables:**
- [ ] Design abstraction layer for multiple storage providers
- [ ] Implement file upload UI components
- [ ] Develop project file attachment functionality
- [ ] Create integration with OpenAI's file retrieval API
- [ ] Implement secure file access controls
- [ ] Add comprehensive documentation for the feature

### Component Revamp: DataConverterView
Priority: MEDIUM  
Status: IN_PROGRESS
Dependencies: None
ParentTask: None

**Description:**  
Redesign and enhance the DataConverterView component to improve layout, usability, and incorporate test case functionality. Use ApiView as inspiration how to create a functional and polished component.

**Key Deliverables:**
- [ ] Improve component layout and visual design
- [ ] Add explanations or affordances to clarify the purpose of data converters
- [ ] Implement display for test cases from packages/shared/src/run/configs/code.ts
- [ ] Add functionality to run test cases in sandbox and display results
- [ ] Ensure responsive design for all screen sizes

### Component Revamp: DataStructureView
Priority: MEDIUM  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Fix and completely revamp the currently broken DataStructureView component. Use ApiView as inspiration how to create a functional and polished component.

**Key Deliverables:**
- [ ] Fix existing component issues
- [ ] Redesign UI for improved usability and visual appeal
- [ ] Ensure proper data handling and validation
- [ ] Add comprehensive documentation
- [ ] Implement proper error handling

### Component Revamp: ProjectCrud
Priority: HIGH  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Fix and restructure the broken ProjectCrud component with a new layout and improved functionality. Use ApiView as inspiration how to create a functional and polished component.

**Key Deliverables:**
- [ ] Implement editable title with HelpButton for description
- [ ] Add toggle between search input and AdvancedInput modes
- [ ] Develop search results display for search mode
- [ ] Create chat history list display for chat mode
- [ ] Ensure chat access to all project resources
- [ ] Fix all existing component issues

### Component Revamp: PromptView
Priority: MEDIUM  
Status: TODO
Dependencies: [Component Revamp: DataConverterView]
ParentTask: None

**Description:**  
Redesign and enhance the PromptView component with improved layout and additional functionality. Use ApiView and DataConverterView as inspiration how to create a functional and polished component.

**Key Deliverables:**
- [ ] Redesign so layout is similar to DataConverterView
- [ ] Use MarkdownDisplay to display prompt
- [ ] Add section with AdvancedInput for entering start messages
- [ ] Add action buttons below AdvancedInput (create new chat add to system message in new chat, add as context item to active chat, "customize" [user-friendly way of saying to fork it so you can customize it] etc.)
- [ ] Ensure proper integration with chat functionality

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

# ✅ Completed Tasks

### AdvancedInput Focus Enhancement
Priority: MEDIUM  
Status: DONE
Dependencies: None
ParentTask: None

**Description:**  
Implement functionality to focus the markdown/lexical text input component when clicking anywhere on AdvancedInput that's not a button/chip or other interactive element. Previous attempts were unsuccessful, so careful implementation with debugging is required.

**Key Deliverables:**
- [x] Analyze previous implementation attempts and identify issues
- [x] Implement event handling for clicks on non-interactive areas
- [x] Add temporary logging for debugging and verification
- [x] Test focus behavior across different scenarios and components
- [x] Document the solution for future reference
