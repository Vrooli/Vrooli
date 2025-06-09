# Create Integration Tests for Cross-Tier Communication
Priority: HIGH  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Develop comprehensive integration tests for the three-tier AI architecture (Coordination/Process/Execution) to ensure proper communication, error handling, and resilience between tiers. The tests should verify event-driven coordination, MCP tool communication, and emergency control channels.

**Current Implementation Analysis:**
- Three-tier architecture with Redis-based event bus for communication
- Tier 1 (Coordination) → Tier 2 (Process) via MCP tools
- Tier 2 (Process) → Tier 3 (Execution) via direct service interface
- Emergency channel for Tier 1 ↔ Tier 3 bypassing Tier 2
- Existing integration test pattern in `executionFlow.test.ts` using testcontainers

**Key Deliverables:**

**Phase 1: Core Communication Tests**
- [ ] Test Tier 1 → Tier 2 MCP tool communication (`run_routine`, resource management)
- [ ] Test Tier 2 → Tier 3 step execution flow (request/response cycle)
- [ ] Test emergency channel (Tier 1 ↔ Tier 3 direct communication)
- [ ] Test event bus delivery guarantees (fire-and-forget, reliable, barrier sync)
- [ ] Test concurrent swarm execution and resource sharing

**Phase 2: Error Handling and Resilience Tests**
- [ ] Test error propagation across tiers (transient, permanent, system errors)
- [ ] Test circuit breaker activation and recovery
- [ ] Test fallback strategies for each tier
- [ ] Test resource exhaustion scenarios and graceful degradation
- [ ] Test recovery from tier failures (Redis down, service unavailable)

**Phase 3: State Management Tests**
- [ ] Test swarm state transitions and persistence
- [ ] Test run state machine transitions across tiers
- [ ] Test checkpoint and recovery mechanisms
- [ ] Test context propagation through execution chain
- [ ] Test state synchronization during concurrent operations

**Phase 4: Performance and Load Tests**
- [ ] Test high-volume event processing
- [ ] Test tier performance under concurrent load
- [ ] Test resource limit enforcement
- [ ] Test priority-based event routing
- [ ] Test backpressure handling

**Phase 5: Security and Validation Tests**
- [ ] Test cross-tier authentication and authorization
- [ ] Test input/output validation at tier boundaries
- [ ] Test security event propagation
- [ ] Test rate limiting across tiers
- [ ] Test malicious input handling

**Technical Implementation Notes:**
- Use testcontainers for Redis (never mock core infrastructure)
- Follow existing pattern in `__test/integration/` directory
- Test files should use `.test.ts` extension
- Use shared test setup from `__test/setup.ts`
- Verify both success and failure scenarios
- Test event subscriptions and unsubscriptions
- Ensure tests are idempotent and can run in parallel

**Test Structure:**
```
packages/server/src/services/execution/__test/integration/
├── tierCommunication.test.ts      # Core communication tests
├── errorHandling.test.ts           # Error and resilience tests
├── stateManagement.test.ts         # State and persistence tests
├── performance.test.ts             # Load and performance tests
└── security.test.ts                # Security boundary tests
```

---

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

# Implement Comprehensive Error Handling Strategy
Priority: HIGH  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Create a unified error handling system with custom error classes, consistent error codes, and proper error propagation across all tiers. The system should improve debugging, user experience, and provide better error tracking and recovery mechanisms throughout the application.

**Current Implementation Analysis:**
- **CustomError class** exists in `/packages/server/src/events/error.ts` with trace code system
- **Translation-based errors** using i18n keys from `/packages/shared/src/translations/locales/en/error.json`
- **Trace codes** are manually managed per file (4-digit location + 4-digit random)
- **Execution tiers** have sophisticated error classification and recovery strategies
- **HTTP errors** have specialized classes (FetchError, TimeoutError, NetworkError)
- **No centralized error utilities** - error handling is distributed across modules

**Key Deliverables:**

**Phase 1: Error Infrastructure Enhancement**
- [ ] Create centralized error utilities module at `/packages/shared/src/errors/`
  - [ ] Move and enhance CustomError class with additional metadata
  - [ ] Create error factory functions for common error types
  - [ ] Implement automatic trace code generation system
  - [ ] Add error context preservation (stack traces, request data, user context)
- [ ] Extend error classification system from execution tiers to general application
  - [ ] Apply ErrorSeverity, ErrorCategory, ErrorRecoverability enums application-wide
  - [ ] Create error classifier for non-execution errors
  - [ ] Implement recovery strategy selector for common errors

**Phase 2: Error Code Management**
- [ ] Implement automatic trace code assignment system
  - [ ] Create build-time script to assign unique codes to error locations
  - [ ] Generate trace code mapping file for debugging
  - [ ] Add VS Code extension for trace code lookup
- [ ] Standardize error code format across application
  - [ ] Document error code ranges for different modules
  - [ ] Create error code registry to prevent duplicates
  - [ ] Add validation for error code uniqueness

**Phase 3: Error Propagation and Handling**
- [ ] Implement consistent error propagation patterns
  - [ ] Create error boundary components for UI
  - [ ] Add error middleware for all API endpoints
  - [ ] Implement error aggregation for batch operations
  - [ ] Add context propagation through error chain
- [ ] Create tier-specific error handlers
  - [ ] Tier 1: Strategic error handling with fallback coordination
  - [ ] Tier 2: Process error handling with retry and circuit breaking
  - [ ] Tier 3: Execution error handling with detailed logging

**Phase 4: User-Facing Error Improvements**
- [ ] Enhance error messages for better user experience
  - [ ] Create user-friendly error message templates
  - [ ] Add actionable error messages with suggested fixes
  - [ ] Implement error message personalization based on user role
- [ ] Add error recovery UI components
  - [ ] Create retry mechanisms for failed operations
  - [ ] Add fallback UI states for critical errors
  - [ ] Implement progressive degradation for feature failures

**Phase 5: Error Monitoring and Analytics**
- [ ] Implement comprehensive error tracking
  - [ ] Add error telemetry with anonymized data
  - [ ] Create error dashboards for monitoring
  - [ ] Implement error alerting for critical issues
- [ ] Add error pattern detection
  - [ ] Identify recurring error patterns
  - [ ] Create automated error reports
  - [ ] Implement predictive error prevention

**Phase 6: Testing and Documentation**
- [ ] Create comprehensive error handling tests
  - [ ] Unit tests for all error classes and utilities
  - [ ] Integration tests for error propagation
  - [ ] E2E tests for user-facing error scenarios
- [ ] Document error handling patterns
  - [ ] Create error handling best practices guide
  - [ ] Document all error codes and their meanings
  - [ ] Add troubleshooting guides for common errors

**Technical Implementation Notes:**
- Build on existing CustomError and execution tier error handling
- Maintain backward compatibility with existing error codes
- Ensure errors are properly logged but don't expose sensitive data
- Follow TypeScript strict error typing patterns
- Consider performance impact of error tracking

**File Structure:**
```
packages/shared/src/errors/
├── index.ts                    # Main exports
├── CustomError.ts              # Enhanced CustomError class
├── errorFactory.ts             # Error creation utilities
├── errorClassifier.ts          # Error classification logic
├── errorCodes.ts               # Error code constants
└── types.ts                    # Error-related types

packages/server/src/middleware/
├── errorHandler.ts             # Global error middleware
└── errorLogger.ts              # Error logging middleware

packages/ui/src/components/errors/
├── ErrorBoundary.tsx           # React error boundary
├── ErrorDisplay.tsx            # User-friendly error display
└── ErrorRecovery.tsx           # Error recovery UI
```

**Success Criteria:**
- All errors use consistent error classes and codes
- Error messages are user-friendly and actionable
- Debugging is simplified with trace codes and context
- Error patterns are tracked and analyzed
- Recovery mechanisms reduce user frustration
- Documentation enables quick error resolution

---

# Implement Code Duplication Detection and Refactoring
Priority: MEDIUM  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Refactor duplicate code patterns in endpoint logic files to reduce maintenance burden and improve code reusability. Analysis shows ~36 out of 47 endpoint files follow nearly identical CRUD patterns with only minor configuration differences. This refactoring could reduce the codebase by 15,000-20,000 lines while standardizing behavior across all endpoints.

**Key Deliverables:**
- [ ] Create endpoint factory function for standard CRUD operations:
  - Design configuration interface for rate limits, permissions, and visibility
  - Implement `createStandardCrudEndpoints` factory function
  - Support custom overrides for endpoints with special logic
- [ ] Migrate standard endpoints to use the factory:
  - Start with 5 simple endpoints as proof of concept
  - Create migration script to convert remaining endpoints
  - Preserve existing functionality and type safety
- [ ] Extract common configuration constants:
  - Rate limit presets (100, 250, 500, 1000, 2000)
  - Permission combinations
  - Default visibility settings
- [ ] Update endpoint tests to use shared patterns:
  - Continue migration to shared fixtures as per README-SharedFixtures.md
  - Create test factory for common CRUD test scenarios
  - Ensure test coverage remains at 100%
- [ ] Document the new patterns:
  - Add usage examples to developer documentation
  - Create migration guide for future endpoints
  - Update contribution guidelines

**Technical Implementation Notes:**
The factory approach should maintain full TypeScript type safety and allow for progressive migration. Endpoints with custom logic (auth, user, etc.) should be able to override specific methods while still benefiting from the standardization. The implementation should follow the existing pattern of using helper functions (createOneHelper, readOneHelper, etc.) which already handle the business logic well.

**Acceptance Criteria:**
- All migrated endpoints maintain exact same functionality
- TypeScript types remain fully intact with no loss of type safety
- Performance characteristics remain unchanged
- Code reduction of at least 50% in migrated endpoint files
- All existing tests pass without modification
- New factory is well-documented with examples

**Files to Focus On:**
- `packages/server/src/endpoints/logic/*.ts` (primary refactoring targets)
- `packages/server/src/endpoints/util/` (location for new factory)
- `packages/server/src/__test/fixtures/` (test pattern updates)
- `packages/server/src/endpoints/logic/README-SharedFixtures.md` (existing guide)

---