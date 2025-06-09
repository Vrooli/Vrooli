# Verify and Test Rate Limiting Implementation
Priority: HIGH  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Audit existing rate limiting and API quota management system. Create comprehensive tests to ensure it's working correctly and preventing abuse.

**Current Implementation Analysis:**
- Token bucket algorithm with burst support and sliding window
- Request-level middleware with API/IP/User rate limits
- Per-endpoint configuration with custom limits
- Credit system integration for usage-based billing
- Existing test coverage in request.test.ts

**Key Deliverables:**

**Phase 1: Audit Current Implementation**
- [ ] Review all rate limiting code for correctness
- [ ] Identify any missing rate limits on endpoints
- [ ] Check for bypass vulnerabilities
- [ ] Verify Redis failure fallback behavior
- [ ] Audit credit system integration

**Phase 2: Enhance Test Coverage**
- [ ] Test distributed rate limiting across multiple servers
- [ ] Test burst capacity handling
- [ ] Test rate limit recovery over time
- [ ] Test edge cases (clock skew, Redis failures, Lua errors, concurrent requests)
- [ ] Test all endpoint rate limits

**Phase 3: Security Testing**
- [ ] Test for rate limit bypass attempts
- [ ] Test IP spoofing resistance
- [ ] Test safe origin validation
- [ ] Test API key rotation impact
- [ ] Test DDoS resistance

**Phase 4: Monitoring & Alerts**
- [ ] Verify rate limit violation logging
- [ ] Test metrics collection
- [ ] Create alert thresholds
- [ ] Test event emission for violations

**Phase 5: Documentation**
- [ ] Document all rate limits per endpoint
- [ ] Create rate limiting best practices guide
- [ ] Document monitoring procedures
- [ ] Create incident response playbook

---

# Fix Commenting System
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

---

# Implement and Test API Key Generation and Validation
Priority: HIGH  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Ensure internal and external API key generation and validation is fully implemented with proper encryption. Create comprehensive tests for key generation, validation, rotation, and secure storage. Verify keys are properly encrypted at rest and in transit.

**Current Implementation Analysis:**
- 32-character key generation with AES-256-CBC encryption
- Dual storage system (site keys and external keys)
- Multi-source authentication (header, query, env var)
- Permission-based access control system
- Basic unit tests exist but lack comprehensive coverage

**Key Deliverables:**

**Phase 1: Complete Core Implementation**
- [ ] Fix raw key return in createOne endpoint (TODO at line 24)
- [ ] Implement key rotation mechanism with history and grace period
- [ ] Add key expiration support with cleanup jobs

**Phase 2: Security Enhancements**
- [ ] Verify encryption at rest (database, backups)
- [ ] Implement encryption in transit (HTTPS, TLS)
- [ ] Add usage restrictions (IP whitelist, domain limits, time windows)

**Phase 3: Comprehensive Testing**
- [ ] Unit tests for generation, encryption, permissions
- [ ] Integration tests for full lifecycle and multi-tenant isolation
- [ ] Security tests for brute force, timing attacks, enumeration

**Phase 4: Usage Tracking & Analytics**
- [ ] Implement detailed per-endpoint usage tracking
- [ ] Add usage alerts and automated suspension
- [ ] Create real-time usage dashboard

**Phase 5: Documentation & Developer Experience**
- [ ] Complete API authentication guide
- [ ] Document security best practices
- [ ] Create migration tools and guides

---

# Multi-Agent Workflow Functionality
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

---

# Advanced Input Component Refactoring
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
- [ ] Look into removing TextInput and TranslatedTextInput components
- [ ] Remove associated hooks including useTagDropdown
- [ ] Update documentation to reflect new input component architecture

---

# Enhance IntegerInput Component
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

---

# Model Selector Improvements in Dashboard
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

---

# File Upload and Storage System
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

---

# Component Revamp: DataConverterView
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

---

# Component Revamp: DataStructureView
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

---

# Component Revamp: ProjectCrud
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

---

# Component Revamp: PromptView
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

---

# Storybook Tests Initialization
Priority: MEDIUM  
Status: TODO

**Description:**  
Initialize storybook tests for multiple components following the established patterns in ApiView.stories.tsx and ApiUpsert.stories.tsx. Ensure that mocked data is realistic, accurate, and matches the correct type.

**Key Deliverables:**
- [ ] Create storybook test files for dataConverter component
- [ ] Create storybook test files for dataStructure component
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

---

# Team Schema Enhancement
Priority: HIGH
Status: TODO

Implement comprehensive schema support for Teams to enable structured organization tools and visualizations.

- [ ] Design and implement schema for Business Model Canvas, like we've done previously (e.g. /shared/src/run/configs/run.ts)
- [ ] Create schema for organizational chart
- [ ] Add UI components for displaying Business Model Canvas
- [ ] Add interactive org chart visualization
- [ ] Implement data persistence (stringified field in team prisma table) for team schemas
- [ ] Create routine for generating team schema

---

# Authentication Settings UI Enhancement
Priority: MEDIUM
Status: TODO

Improve the user experience and functionality of the authentication settings interface.

- [ ] Add "Sign out all devices" feature
- [ ] Redesign SettingsAuthenticationView for better UX
- [ ] Add confirmation dialogs for sensitive actions
- [ ] Implement loading states for async operations
- [ ] Add success/error notifications
- [ ] Update authentication settings documentation

---

# Enhanced Testing Infrastructure
Priority: MEDIUM
Status: TODO

Improve the testing setup across all packages.

- [ ] Add more comprehensive unit tests
- [ ] Implement E2E testing with Playwright
- [ ] Add performance testing
- [ ] Improve test documentation

---

# Documentation Improvements
Priority: MEDIUM
Status: TODO

Enhance project documentation for better developer and user experience.

- [ ] Create comprehensive API documentation
- [ ] Add more code examples
- [ ] Improve setup instructions
- [ ] Add troubleshooting guides
- [ ] Update main README to link to project-level tasks, docs, ARCHITECTURE.md, etc.
- [ ] Enhance .env-example with detailed comments explaining each environment variable

---

# Performance Optimization
Priority: LOW
Status: TODO

Optimize application performance and resource usage.

- [ ] Implement caching strategies
- [ ] Optimize database queries
- [ ] Add performance monitoring
- [ ] Implement lazy loading where beneficial

---

# Local Installation Support
Priority: LOW
Status: TODO

Enable users to run Vrooli locally with their preferred storage and AI models.

- [ ] Implement configurable storage locations (local/cloud/decentralized)
- [ ] Add support for local AI models
- [ ] Create native desktop applications (.exe, .dmg, .deb)
- [ ] Implement offline-first capabilities
- [ ] Ensure cross-platform compatibility

---

### Create Routine Runner Component
*Priority:** HIGH  
**Priority:**ODO  
**Status:**ies:** None

#### Description
C#te a component for running routines that's displayed in a chat interface. The component will provide real-time feedback about routine execution progress.

#### Technical Specifications

The component should be a responsive UI element with the following structure:
- A rounded container with distinct sections
- Top section containing:
  - Loading spinner for active routines
  - Routine title
  - Seconds elapsed counter
- Two main content sections:
  - Left section: Steps list with status indicators
    - Each step should display a loading spinner when active
    - Completed/failed steps should display appropriate status icons
  - Right section: Information panel for the active task

#### Implementation Details

1. **Component Structure:**
   - Create a new functional React component using TypeScript
   - Use Material UI components for styling
   - Support responsive layouts

2. **State Management:**
   - Track active steps and their states
   - Implement elapsed time counter
   - Handle routine status updates

3. **UI Elements:**
   - Include loading animations
   - Create status indicators for different step states
   - Implement a clean, intuitive layout

4. **Integration:**
   - Ensure the component can be embedded in chat interfaces
   - Support both mobile and desktop view modes

5. **Testing:**
   - Write unit tests using Mocha, Chai, and Sinon
   - Include tests for all state transitions and UI behaviors

#### Files to Create/Modify:
- ackages/ui/src/components/RoutineRunner/RoutineRunner.tsx` - Main component
- `packages/ui/src/components/RoutineRunner/types.ts` - TypeScript interfaces
- `packages/ui/src/components/RoutineRunner/RoutineRunner.test.tsx` - Unit tests
- `packages/ui/src/components/RoutineRunner/RoutineRunner.stories.tsx` - Storybook examples

#### Technical Approach
Bud the component using functional React patterns with hooks for state management. Leverage existing UI components when possible for layout. The component should be thoroughly tested and well-documented to ensure maintainability.
Build the component using functional React patterns with hooks for state management. The component should be thoroughly tested and well-documented to ensure maintainability.

---

# Update ChatBubbleTree to Display Message-Triggered Tools and Routines
Priority: MEDIUM  
Status: TODO
Dependencies: [Enhance Chat and Message Storage with Metadata]
ParentTask: None

**Description:**  
Enhance the ChatBubbleTree component to display not only the message tree itself but also show tools and routines that were triggered by messages. Messages can trigger various tools including web search, code interpreter, and routines. This improvement requires integrating with the metadata stored in chat messages to identify and properly render these message-triggered events.

**Key Deliverables:**
- [ ] Modify the ChatBubbleTree component to retrieve and process metadata related to tools and routines triggered by messages
- [ ] Update the message rendering logic to visually display triggered tools and routines
- [ ] Implement UI for showing the relationship between messages and their triggered tools/routines
- [ ] Add visual indicators for different types of triggered tools (web search, code interpreter, routine execution, etc.)
- [ ] Create a collapsible/expandable view for tool execution results
- [ ] Ensure proper organization of the chat view to maintain conversation clarity
- [ ] Add tooltips or other UI elements to show additional information about the triggered tools
- [ ] Update tests to verify correct rendering of message-triggered tools and routines
- [ ] Ensure backward compatibility with existing message structures

---

# Document User Authentication API Endpoints
Priority: MEDIUM  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Create comprehensive documentation for all user authentication API endpoints in the system. This documentation should cover email registration, login, password reset, wallet authentication, and account management processes. The documentation will help developers understand and use these API endpoints effectively.

**Key Deliverables:**
- [ ] Document all authentication endpoints in `packages/server/src/endpoints/logic/auth.ts`:
  - [ ] Email signup and login
  - [ ] Email verification
  - [ ] Password reset
  - [ ] Wallet authentication
  - [ ] Guest login
- [ ] Document account management endpoints in `packages/server/src/endpoints/logic/user.ts`:
  - [ ] Profile update
  - [ ] Email update
  - [ ] Password change
- [ ] Document account deletion process in `packages/server/src/endpoints/logic/actions.ts`
- [ ] For each endpoint, include:
  - [ ] Endpoint purpose and description
  - [ ] Required inputs and expected outputs
  - [ ] Authentication requirements
  - [ ] Error codes and responses
  - [ ] Usage examples
- [ ] Organize documentation in a logical, user-friendly format
- [ ] Add the documentation to a dedicated API documentation section

---

# Fix API Key Raw Value Return on Creation
Priority: HIGH  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Fix the issue where raw API keys are not properly returned during creation. The current implementation has a TODO comment indicating that the unencrypted key value must be included in the create response so users can save it (as it's only shown once). This is critical for API key functionality to work properly.

**Current Implementation Analysis:**
- API key infrastructure is largely complete with encryption, permissions, and authentication
- The main issue is in `packages/server/src/endpoints/logic/apiKey.ts` line 24
- Keys are properly generated and encrypted, but the raw value isn't returned
- UI already expects to display the key once on creation

**Key Deliverables:**

**Phase 1: Fix Core Implementation**
- [ ] Modify the createOne endpoint to include the raw key in the response
- [ ] Ensure the raw key is only included during creation (never on updates/reads)
- [ ] Update the response type to include the temporary raw key field
- [ ] Verify the UI properly displays and handles the one-time key display

**Phase 2: Security Verification**
- [ ] Ensure raw keys are never logged or stored
- [ ] Verify keys are properly encrypted before database storage
- [ ] Confirm raw keys are only transmitted over HTTPS
- [ ] Add security warnings in the UI about saving the key

**Phase 3: Testing**
- [ ] Create unit tests for key creation with raw value return
- [ ] Test that subsequent reads never include the raw key
- [ ] Test UI flow for copying and saving the key
- [ ] Verify error handling if key creation fails

**Phase 4: Enhancement (Optional)**
- [ ] Add key rotation mechanism
- [ ] Implement key expiration dates
- [ ] Add IP whitelisting support
- [ ] Create usage analytics dashboard

**Technical Notes:**
- The fix is straightforward but security-critical
- Must coordinate with UI to ensure proper one-time display
- Consider adding a "download key" option for better UX

---