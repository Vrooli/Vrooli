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

# Create Performance Monitoring Dashboard
Priority: LOW  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Build a comprehensive performance monitoring dashboard to track execution times, resource usage, and bottlenecks across the three-tier AI architecture. The dashboard will provide real-time visualization of system performance, tier interactions, and AI agent execution metrics.

**Current Implementation Analysis:**
- **Existing Infrastructure**: Basic metrics endpoint (`/metrics`), metrics service with system monitoring
- **Three-Tier Architecture**: Tier 1 (Coordination), Tier 2 (Process), Tier 3 (Execution) with performance tracking
- **Event Bus**: Redis-based event streaming with delivery guarantees
- **Performance Components**: PerformanceMonitor, ResourceMetrics, PerformanceTracker classes
- **Missing**: Real-time streaming, historical data persistence, dashboard UI components

**Key Deliverables:**

**Phase 1: Backend Infrastructure Enhancement**
- [ ] Enhance metrics collection across all three tiers:
  - [ ] Tier 1: Swarm coordination metrics (decision time, resource allocation efficiency)
  - [ ] Tier 2: Process orchestration metrics (run execution, step transitions, navigator performance)
  - [ ] Tier 3: Execution strategy metrics (tool usage, validation results, strategy comparisons)
- [ ] Implement metrics persistence layer:
  - [ ] Design metrics database schema (time-series optimized)
  - [ ] Create metrics storage service with retention policies
  - [ ] Implement aggregation for different time windows (1m, 5m, 1h, 1d)
- [ ] Add real-time metrics streaming:
  - [ ] WebSocket endpoint for live metrics updates
  - [ ] Server-sent events as fallback option
  - [ ] Subscription management for specific metric types

**Phase 2: API Development**
- [ ] Create dashboard-specific API endpoints:
  - [ ] `/api/dashboard/metrics/overview` - High-level system metrics
  - [ ] `/api/dashboard/metrics/tiers` - Per-tier performance breakdown
  - [ ] `/api/dashboard/metrics/executions` - Execution history and analytics
  - [ ] `/api/dashboard/metrics/resources` - Resource usage patterns
- [ ] Implement metric aggregation queries:
  - [ ] Time-series data with configurable granularity
  - [ ] Cross-tier performance correlation
  - [ ] Bottleneck detection algorithms
  - [ ] Cost analysis per execution type

**Phase 3: Dashboard UI Components**
- [ ] Create core dashboard layout:
  - [ ] Responsive grid system for metric cards
  - [ ] Navigation between different dashboard views
  - [ ] Time range selector for historical data
  - [ ] Auto-refresh toggle for real-time updates
- [ ] Implement visualization components:
  - [ ] Real-time line charts for performance metrics
  - [ ] Tier interaction flow diagram
  - [ ] Resource usage heat maps
  - [ ] Execution timeline with drill-down capability
- [ ] Add interactive features:
  - [ ] Click-through from metrics to detailed logs
  - [ ] Metric comparison tools
  - [ ] Custom dashboard configuration
  - [ ] Export functionality for reports

**Phase 4: Advanced Monitoring Features**
- [ ] Implement alerting system:
  - [ ] Configurable performance thresholds
  - [ ] Alert notification channels (UI, email, webhook)
  - [ ] Alert history and acknowledgment
- [ ] Add predictive analytics:
  - [ ] Performance trend analysis
  - [ ] Anomaly detection using rolling averages
  - [ ] Resource exhaustion predictions
- [ ] Create diagnostic tools:
  - [ ] Performance profiling triggers
  - [ ] Execution trace visualization
  - [ ] Bottleneck root cause analysis

**Phase 5: Integration and Testing**
- [ ] Integrate with existing monitoring infrastructure:
  - [ ] Connect to event bus for real-time updates
  - [ ] Leverage existing PerformanceMonitor classes
  - [ ] Ensure minimal performance overhead
- [ ] Comprehensive testing:
  - [ ] Load testing for metrics collection
  - [ ] UI component testing with Storybook
  - [ ] End-to-end dashboard functionality tests
  - [ ] Performance impact assessment

**Technical Implementation Notes:**
- Use React with Material-UI for dashboard components
- Leverage D3.js or Recharts for data visualization
- Implement efficient data fetching with React Query
- Use WebSocket for real-time updates with automatic reconnection
- Follow existing metrics patterns from `packages/server/src/services/metrics.ts`
- Ensure dashboard doesn't impact system performance

**UI Mock-up Structure:**
```
┌─────────────────────────────────────────────────────────┐
│  Performance Dashboard                    [Time Range ▼] │
├─────────────────────────────────────────────────────────┤
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐     │
│ │ Total Runs   │ │ Avg Duration │ │ Success Rate │     │
│ │    1,234     │ │   3.45s      │ │    98.5%     │     │
│ └──────────────┘ └──────────────┘ └──────────────┘     │
│                                                          │
│ ┌────────────────────────────┐ ┌──────────────────────┐│
│ │ Tier Performance           │ │ Resource Usage       ││
│ │ [Line Chart]              │ │ [Stacked Area Chart] ││
│ └────────────────────────────┘ └──────────────────────┘│
│                                                          │
│ ┌────────────────────────────────────────────────────┐ │
│ │ Execution Timeline                                  │ │
│ │ [Interactive Timeline with Tier Breakdown]          │ │
│ └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

**Success Criteria:**
- Real-time performance visibility across all tiers
- Historical data analysis with configurable time ranges
- Bottleneck identification within 5 seconds
- Dashboard load time under 2 seconds
- Support for 100+ concurrent dashboard users
- Actionable insights for performance optimization

---

# Develop Comprehensive API Documentation with OpenAPI/Swagger
Priority: MEDIUM  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Create interactive API documentation with automatic client SDK generation capabilities using OpenAPI/Swagger. The project already has a comprehensive OpenAPI schema generator but lacks a live documentation endpoint. This task will implement swagger-ui-express to serve interactive API documentation and enhance the existing documentation infrastructure.

**Current Implementation Analysis:**
- **Existing Schema Generator**: `/packages/shared/src/__tools/api/buildOpenAPISchema.ts` generates OpenAPI 3.0.0 spec
- **Generated Schema Location**: `/docs/docs/assets/openapi.json`
- **API Structure**: Well-organized endpoints in `/packages/server/src/endpoints/`
- **Documentation**: Comprehensive markdown docs exist but no interactive API explorer
- **Missing**: Live API documentation endpoint, swagger-ui integration, SDK generation

**Key Deliverables:**

**Phase 1: Swagger UI Integration**
- [ ] Install swagger-ui-express and related dependencies
- [ ] Create `/packages/server/src/endpoints/docs/` directory structure
- [ ] Implement swagger-ui middleware to serve interactive documentation
- [ ] Add authentication/security for documentation endpoint (configurable)
- [ ] Configure swagger-ui with custom theme matching Vrooli branding
- [ ] Add route `/api-docs` to serve the documentation

**Phase 2: Enhance OpenAPI Schema Generation**
- [ ] Update `buildOpenAPISchema.ts` to include:
  - [ ] Request/response examples from actual test data
  - [ ] Enhanced descriptions from JSDoc comments
  - [ ] Authentication schemes (API keys, JWT tokens)
  - [ ] Rate limiting information per endpoint
  - [ ] Webhook definitions and callbacks
- [ ] Add validation to ensure schema stays in sync with code
- [ ] Create build script to auto-generate docs on changes

**Phase 3: SDK Generation Setup**
- [ ] Research and select SDK generation tool (e.g., OpenAPI Generator, Swagger Codegen)
- [ ] Configure SDK generation for popular languages:
  - [ ] TypeScript/JavaScript (primary)
  - [ ] Python
  - [ ] Go
  - [ ] Java
- [ ] Create GitHub Actions workflow for automatic SDK publishing
- [ ] Set up SDK versioning strategy aligned with API versions

**Phase 4: Documentation Enhancements**
- [ ] Add interactive examples with "Try it out" functionality
- [ ] Create getting started guide within swagger-ui
- [ ] Add code snippets for common use cases
- [ ] Implement request/response logging for debugging
- [ ] Add API changelog and versioning information
- [ ] Create postman collection export functionality

**Phase 5: Developer Experience**
- [ ] Add hot-reload for documentation during development
- [ ] Create VSCode snippets for adding new endpoints with proper docs
- [ ] Implement API mocking for frontend development
- [ ] Add API health check endpoint with documentation
- [ ] Create migration guides for API version changes

**Technical Implementation Notes:**
- Use existing OpenAPI schema generator as foundation
- Ensure documentation is version-controlled but generated files are gitignored
- Follow existing project patterns for middleware and routing
- Documentation should be accessible in development but protected in production
- Consider performance impact of serving documentation in production
- Integrate with existing authentication system for protected endpoints

**File Structure:**
```
packages/server/src/endpoints/docs/
├── index.ts              # Swagger UI setup and configuration
├── middleware.ts         # Authentication and rate limiting for docs
├── customizations/       # Custom CSS and UI modifications
│   ├── theme.css        # Vrooli-branded swagger theme
│   └── logo.svg         # Project logo for docs
└── examples/            # Request/response examples
    └── [endpoint].json  # Example data per endpoint
```

**Success Criteria:**
- Interactive API documentation accessible at `/api-docs`
- All endpoints documented with examples and schemas
- SDK generation automated for at least 3 languages
- Documentation stays in sync with code changes
- Improved developer onboarding time
- Reduced support requests about API usage

---

# Build Agent Performance Profiling System
Priority: LOW  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Implement a comprehensive performance profiling system for the emergent agent infrastructure to measure, benchmark, and optimize agent performance metrics including response time, accuracy, resource consumption, and learning efficiency. The system will provide insights into agent behavior patterns and enable data-driven optimization of the three-tier AI architecture.

**Current Implementation Analysis:**
- **Agent Infrastructure**: Emergent agents in `/packages/server/src/services/execution/cross-cutting/agents/`
- **Existing Monitoring**: TelemetryShim (5ms overhead limit), PerformanceMonitor, PerformanceTracker
- **Event-Driven Architecture**: Agents subscribe to events and learn from patterns
- **Performance Events**: Already emitted (`execution/metrics`, `routine/completed`, `step/completed`)
- **Missing**: Agent-specific profiling, learning metrics, swarm coordination analytics

**Key Deliverables:**

**Phase 1: Agent Metrics Infrastructure**
- [ ] Extend existing metrics system for agent-specific data:
  - [ ] Agent lifecycle metrics (startup time, initialization, shutdown)
  - [ ] Event processing metrics (events/second, processing latency, queue depth)
  - [ ] Learning metrics (patterns detected, proposals generated, accuracy rates)
  - [ ] Resource consumption per agent (CPU, memory, event bus usage)
- [ ] Create agent profiling service:
  - [ ] Real-time agent performance tracking
  - [ ] Historical performance data with configurable retention
  - [ ] Agent comparison and benchmarking tools
  - [ ] Swarm-level performance aggregation

**Phase 2: Performance Benchmarking Framework**
- [ ] Develop standardized agent benchmarks:
  - [ ] Response time benchmarks for different event types
  - [ ] Accuracy benchmarks for pattern detection
  - [ ] Resource efficiency benchmarks
  - [ ] Learning speed benchmarks
- [ ] Create synthetic workload generator:
  - [ ] Configurable event streams for testing
  - [ ] Reproducible test scenarios
  - [ ] Load testing capabilities
  - [ ] Edge case simulation

**Phase 3: Profiling Tools and Visualization**
- [ ] Build agent profiling dashboard:
  - [ ] Real-time agent performance metrics
  - [ ] Event flow visualization
  - [ ] Resource utilization heat maps
  - [ ] Learning progress tracking
- [ ] Implement profiling CLI tools:
  - [ ] Agent performance snapshots
  - [ ] Comparative analysis between agents
  - [ ] Performance regression detection
  - [ ] Export capabilities for further analysis

**Phase 4: Optimization Insights Engine**
- [ ] Create performance analysis algorithms:
  - [ ] Bottleneck detection in event processing
  - [ ] Inefficient pattern matching identification
  - [ ] Resource waste detection
  - [ ] Optimal agent configuration recommendations
- [ ] Implement automated optimization:
  - [ ] Dynamic event subscription tuning
  - [ ] Priority adjustment based on performance
  - [ ] Resource allocation optimization
  - [ ] Learning rate adjustments

**Phase 5: Integration with Emergent Capabilities**
- [ ] Enable performance-aware agent deployment:
  - [ ] Performance-based agent selection
  - [ ] Automatic scaling based on metrics
  - [ ] Load balancing across agent swarms
  - [ ] Performance SLAs for critical agents
- [ ] Create feedback loops:
  - [ ] Performance metrics influence agent learning
  - [ ] Agents propose their own optimizations
  - [ ] Swarm-level performance evolution
  - [ ] Self-healing based on performance degradation

**Technical Implementation Notes:**
- Leverage existing TelemetryShim for minimal overhead profiling
- Use event bus for performance event collection
- Store metrics in time-series optimized format
- Ensure profiling overhead stays under 2% of agent processing time
- Follow emergent pattern - let agents learn to optimize themselves
- Integration with existing MetricsService at `/packages/server/src/services/metrics.ts`

**File Structure:**
```
packages/server/src/services/execution/cross-cutting/profiling/
├── agentProfiler.ts           # Core profiling service
├── benchmarks/                # Standardized benchmark suite
│   ├── responseBenchmark.ts
│   ├── accuracyBenchmark.ts
│   └── resourceBenchmark.ts
├── analysis/                  # Performance analysis tools
│   ├── bottleneckDetector.ts
│   ├── optimizationEngine.ts
│   └── trendAnalyzer.ts
└── visualization/             # Dashboard components
    ├── AgentMetricsCard.tsx
    ├── SwarmPerformance.tsx
    └── ProfilerDashboard.tsx
```

**Success Criteria:**
- Agent performance metrics available within 100ms
- Profiling overhead under 2% of agent processing time
- Identify performance bottlenecks within 5 minutes
- 20% improvement in agent response times after optimization
- Automated performance regression detection
- Self-optimizing agent swarms based on profiling data

---

# Implement Distributed Tracing for Swarm Execution
Priority: LOW  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Add distributed tracing capabilities to the three-tier swarm execution architecture using OpenTelemetry. This will provide end-to-end visibility into multi-agent workflows, enabling better debugging, performance analysis, and bottleneck identification across Tier 1 (Coordination), Tier 2 (Process), and Tier 3 (Execution) layers.

**Current Implementation Analysis:**
- **Existing Infrastructure**: TelemetryShim (5ms overhead), MetricsService, MonitoringTools
- **Event Bus**: Redis-backed with correlation IDs already present
- **Logging**: Winston-based structured logging with trace IDs
- **Three-Tier Architecture**: Clear boundaries and event-based communication
- **Missing**: Distributed trace context propagation, span collection, trace visualization

**Key Deliverables:**

**Phase 1: Foundation Setup**
- [ ] Install and configure OpenTelemetry SDK (@opentelemetry/api, @opentelemetry/sdk-node)
- [ ] Create tracing configuration service at `/packages/server/src/services/execution/cross-cutting/tracing/`
- [ ] Set up trace exporter configuration (support Jaeger, Zipkin, OTLP)
- [ ] Implement trace context propagation utilities
- [ ] Add distributed tracing environment variables to .env-example

**Phase 2: Event Bus Instrumentation**
- [ ] Extend IntelligentEventBus to propagate trace context in event headers
- [ ] Add span creation for event publishing and consumption
- [ ] Implement trace context extraction from incoming events
- [ ] Ensure trace continuity across event-driven boundaries
- [ ] Add trace-aware event correlation

**Phase 3: Tier Instrumentation**
- [ ] Instrument Tier 1 (TierOneCoordinator):
  - [ ] Swarm lifecycle spans (initialization, execution, completion)
  - [ ] Resource allocation tracking
  - [ ] Team management operations
  - [ ] Strategic decision points
- [ ] Instrument Tier 2 (TierTwoOrchestrator):
  - [ ] Run lifecycle spans
  - [ ] Routine navigation tracking
  - [ ] Branch selection and coordination
  - [ ] Step orchestration
- [ ] Instrument Tier 3 (TierThreeExecutor):
  - [ ] Strategy selection and execution
  - [ ] Tool orchestration spans
  - [ ] I/O processing tracking
  - [ ] Resource utilization

**Phase 4: Integration with Existing Telemetry**
- [ ] Enhance TelemetryShim to emit trace spans alongside telemetry events
- [ ] Correlate traces with existing metrics and logs using trace IDs
- [ ] Add trace context to Winston logger
- [ ] Integrate with MonitoringTools for trace-based analysis
- [ ] Ensure minimal overhead (maintain <5ms telemetry target)

**Phase 5: Visualization and Analysis**
- [ ] Deploy trace backend (Jaeger or Zipkin) in Docker environment
- [ ] Configure trace sampling strategies (head-based and tail-based)
- [ ] Create trace analysis dashboards
- [ ] Implement trace-based alerting for performance degradation
- [ ] Add developer documentation for trace analysis

**Phase 6: Advanced Features**
- [ ] Implement custom span attributes for swarm-specific data:
  - [ ] Agent types and roles
  - [ ] Strategy selections
  - [ ] Resource constraints
  - [ ] Execution outcomes
- [ ] Add baggage propagation for cross-cutting concerns
- [ ] Create trace-based performance baselines
- [ ] Implement trace archival and retention policies

**Technical Implementation Notes:**
- Use OpenTelemetry for vendor-neutral instrumentation
- Leverage existing correlation IDs as trace IDs where possible
- Implement sampling to control overhead (start with 10% sampling rate)
- Use async span processing to minimize latency impact
- Follow OpenTelemetry semantic conventions for span naming
- Ensure trace context survives Redis serialization

**File Structure:**
```
packages/server/src/services/execution/cross-cutting/tracing/
├── tracingService.ts           # Core tracing configuration and setup
├── spanProcessors.ts           # Custom span processors
├── propagators.ts              # Trace context propagation utilities
├── instrumentations/           # Auto-instrumentation configs
│   ├── redis.ts               # Redis instrumentation
│   ├── http.ts                # HTTP instrumentation
│   └── eventBus.ts            # Event bus instrumentation
└── exporters/                  # Trace exporter configurations
    ├── jaeger.ts              # Jaeger exporter setup
    ├── zipkin.ts              # Zipkin exporter setup
    └── otlp.ts                # OTLP exporter setup
```

**Docker Compose Addition:**
```yaml
# Add to docker-compose.yml
jaeger:
  image: jaegertracing/all-in-one:latest
  ports:
    - "16686:16686"  # Jaeger UI
    - "14268:14268"  # Jaeger collector
  environment:
    - COLLECTOR_ZIPKIN_HOST_PORT=:9411
```

**Success Criteria:**
- End-to-end trace visibility for all swarm executions
- Trace overhead under 2% of execution time
- Ability to diagnose performance issues within 5 minutes
- Correlation between traces, logs, and metrics
- Support for high-cardinality trace analysis
- Developer adoption through clear documentation

**Testing Approach:**
- Unit tests for trace propagation logic
- Integration tests for cross-tier tracing
- Performance tests to verify overhead limits
- End-to-end tests with sample swarm executions
- Chaos testing to ensure trace continuity during failures

# Create Accessibility Audit and Enhancement System
Priority: HIGH  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Implement automated accessibility testing, add missing ARIA labels, ensure keyboard navigation works properly, and create accessibility checklist for new components. Build upon the existing Material-UI foundation to achieve WCAG 2.1 AA compliance and provide an inclusive user experience.

**Current Implementation Analysis:**
- **Strong Foundation**: Material-UI provides built-in accessibility for basic components
- **Icon Accessibility**: Well-implemented with proper `aria-hidden` and `aria-label` handling
- **Dialog System**: Proper ARIA labeling with `aria-labelledby` and `aria-describedby`
- **Tab Navigation**: Correct tab role structure and ARIA attributes
- **Keyboard Support**: Basic keyboard event handlers exist but incomplete coverage
- **Testing**: React Testing Library used but no specialized accessibility testing tools

**Key Deliverables:**

**Phase 1: Automated Testing Infrastructure**
- [ ] Install and configure axe-core and jest-axe for automated accessibility testing
- [ ] Add accessibility tests to existing Storybook stories
- [ ] Create accessibility test utilities for common patterns
- [ ] Integrate accessibility testing into CI/CD pipeline
- [ ] Set up accessibility linting rules (eslint-plugin-jsx-a11y)

**Phase 2: Keyboard Navigation Enhancement**
- [ ] Audit all interactive components for keyboard accessibility
- [ ] Implement focus traps for modals and dialogs
- [ ] Add escape key handling for dismissible components
- [ ] Create keyboard navigation documentation and patterns
- [ ] Test tab order and focus management across the application

**Phase 3: ARIA Enhancement**
- [ ] Add missing ARIA landmarks (`main`, `navigation`, `banner`, `contentinfo`)
- [ ] Implement ARIA live regions for dynamic content updates
- [ ] Enhance complex widgets with proper ARIA patterns:
  - [ ] Dropdown menus (`role="menu"`, `role="menuitem"`)
  - [ ] Data tables (`role="grid"`, `role="gridcell"`)
  - [ ] Search/filter components with proper announcements
- [ ] Add comprehensive ARIA state management (expanded, selected, disabled)

**Phase 4: Form Accessibility**
- [ ] Enhance form validation with accessible error announcements
- [ ] Add proper fieldset and legend usage for grouped form controls
- [ ] Implement accessible form instructions and help text
- [ ] Create accessible multi-step form patterns
- [ ] Add form completion status announcements

**Phase 5: Visual and Design Accessibility**
- [ ] Audit color contrast ratios and fix violations (WCAG AA: 4.5:1 normal, 3:1 large)
- [ ] Implement high contrast theme option
- [ ] Add reduced motion support (`prefers-reduced-motion`)
- [ ] Ensure focus indicators are visible and consistent
- [ ] Test with different zoom levels (up to 200%)

**Phase 6: Content and Navigation**
- [ ] Add skip navigation links for keyboard users
- [ ] Implement semantic HTML structure (`section`, `article`, `aside`)
- [ ] Create accessible breadcrumb navigation
- [ ] Add page title updates for single-page application navigation
- [ ] Implement accessible search result announcements

**Phase 7: Screen Reader Optimization**
- [ ] Add screen reader only content for context (`sr-only` class usage)
- [ ] Implement accessible data table headers and captions
- [ ] Create screen reader friendly loading states
- [ ] Add descriptive text for complex UI interactions
- [ ] Test with actual screen readers (NVDA, JAWS, VoiceOver)

**Phase 8: Documentation and Guidelines**
- [ ] Create comprehensive accessibility checklist for new components
- [ ] Document accessible coding patterns and examples
- [ ] Create accessibility testing procedures and tools
- [ ] Add accessibility considerations to component design guidelines
- [ ] Create accessibility audit schedule and procedures

**Technical Implementation Notes:**
- Build upon existing Material-UI accessibility features
- Leverage React Testing Library's accessibility-focused testing approach
- Use semantic HTML5 elements where possible
- Implement progressive enhancement patterns
- Ensure compatibility with assistive technologies
- Follow WCAG 2.1 AA guidelines as minimum standard

**File Structure:**
```
packages/ui/src/accessibility/
├── testing/
│   ├── axeConfig.ts           # Axe-core configuration
│   ├── testUtils.ts           # Accessibility test utilities
│   └── commonTests.ts         # Reusable accessibility tests
├── components/
│   ├── SkipLink.tsx           # Skip navigation component
│   ├── ScreenReaderOnly.tsx   # Screen reader only content
│   └── FocusTrap.tsx          # Focus management utility
├── hooks/
│   ├── useKeyboardNavigation.ts
│   ├── useFocusManagement.ts
│   └── useAccessibilityAnnouncements.ts
└── utils/
    ├── ariaUtils.ts           # ARIA helper functions
    ├── focusUtils.ts          # Focus management utilities
    └── colorContrast.ts       # Color contrast validation
```

**Success Criteria:**
- 100% of components pass automated accessibility tests
- All interactive elements are keyboard accessible
- WCAG 2.1 AA compliance achieved across the application
- Screen reader users can navigate and use all features
- Accessibility checklist integrated into development workflow
- Zero accessibility violations in production
- Comprehensive accessibility testing coverage

**Testing Strategy:**
- Automated testing with axe-core in unit and integration tests
- Manual testing with keyboard navigation
- Screen reader testing with NVDA, JAWS, and VoiceOver
- Color contrast validation tools
- Focus management testing
- User testing with accessibility community

---

# Create Database Migration Rollback System
Priority: HIGH  
Status: TODO
Dependencies: None
ParentTask: None

**Description:**  
Implement a robust rollback system for Prisma database migrations to improve deployment safety. Currently, Vrooli relies on manual backup restoration for migration rollbacks, which is time-consuming and error-prone. This task will create an automated rollback mechanism that generates down migrations, tracks migration state, and provides safe rollback capabilities for both development and production environments.

**Current Implementation Analysis:**
- **Migration System**: Prisma migrations stored in `/packages/server/src/db/migrations/`
- **Execution**: Migrations run automatically during server startup via `start.sh`
- **Backup System**: Daily backups via `/scripts/main/backup.sh` but no pre-migration backups
- **Rollback Process**: Manual - restore from backup and mark as rolled back
- **Missing**: Down migrations, automated rollback commands, migration validation, pre-migration backups

**Key Deliverables:**

**Phase 1: Down Migration Generation**
- [ ] Create script to analyze Prisma migrations and generate corresponding down migrations
  - [ ] Parse SQL from existing migrations to identify reversible operations
  - [ ] Generate inverse SQL for CREATE TABLE → DROP TABLE, ADD COLUMN → DROP COLUMN, etc.
  - [ ] Handle non-reversible operations with warnings (data modifications, drops)
  - [ ] Store down migrations alongside up migrations with naming convention
- [ ] Implement migration analyzer to categorize migrations:
  - [ ] Safe (fully reversible) - schema additions
  - [ ] Risky (data loss possible) - column drops, type changes
  - [ ] Irreversible (manual intervention required) - data transformations
- [ ] Add validation to ensure down migrations are syntactically correct

**Phase 2: Rollback Command Infrastructure**
- [ ] Create rollback CLI command in `/packages/server/src/cli/`:
  - [ ] `pnpm prisma:rollback` - Roll back last migration
  - [ ] `pnpm prisma:rollback --to <migration>` - Roll back to specific migration
  - [ ] `pnpm prisma:rollback --dry-run` - Preview rollback without execution
- [ ] Implement rollback state tracking:
  - [ ] Create `_prisma_rollback_history` table to track rollback operations
  - [ ] Store rollback metadata (timestamp, user, reason, affected tables)
  - [ ] Prevent re-applying rolled back migrations without explicit override
- [ ] Add rollback hooks for custom logic:
  - [ ] Pre-rollback validation
  - [ ] Post-rollback cleanup
  - [ ] Data integrity checks

**Phase 3: Automated Backup Integration**
- [ ] Enhance backup system for migration safety:
  - [ ] Automatic backup before any migration execution
  - [ ] Backup naming includes migration identifier
  - [ ] Configurable retention for migration backups (separate from daily backups)
- [ ] Create restore utilities:
  - [ ] Fast restore from migration-specific backups
  - [ ] Partial restore for specific tables
  - [ ] Backup validation before migration
- [ ] Implement backup cleanup:
  - [ ] Remove old migration backups after successful rollback window
  - [ ] Compress older backups to save space

**Phase 4: Testing Framework**
- [ ] Create migration testing utilities:
  - [ ] Test harness for up/down migration pairs
  - [ ] Data integrity validation after rollback
  - [ ] Performance impact assessment
- [ ] Add migration tests to CI/CD:
  - [ ] Automatic testing of new migrations
  - [ ] Rollback testing for all migrations
  - [ ] Multi-version compatibility testing
- [ ] Create test data generators:
  - [ ] Generate realistic data for migration testing
  - [ ] Test edge cases and large datasets

**Phase 5: Production Safety Features**
- [ ] Implement migration safety checks:
  - [ ] Lock detection to prevent concurrent migrations
  - [ ] Connection pool draining before migrations
  - [ ] Health check integration to prevent migrations during issues
- [ ] Add monitoring and alerting:
  - [ ] Migration execution metrics
  - [ ] Rollback trigger alerts
  - [ ] Performance degradation detection
- [ ] Create rollback approval workflow:
  - [ ] Required approvals for production rollbacks
  - [ ] Audit trail for all rollback operations
  - [ ] Integration with existing monitoring tools

**Phase 6: Kubernetes Integration**
- [ ] Create Kubernetes migration job templates:
  - [ ] Separate job for migrations (not part of server startup)
  - [ ] Pre-upgrade and post-upgrade hooks
  - [ ] Automatic rollback on job failure
- [ ] Implement blue-green migration strategy:
  - [ ] Apply migrations to inactive environment first
  - [ ] Validation before switching traffic
  - [ ] Quick rollback by switching back
- [ ] Add Helm chart enhancements:
  - [ ] Migration job configuration
  - [ ] Rollback hooks
  - [ ] Backup volume management

**Technical Implementation Notes:**
- Use Prisma's introspection capabilities to analyze schema changes
- Leverage PostgreSQL's transactional DDL for safe rollbacks
- Ensure compatibility with existing backup system
- Maintain zero-downtime deployment capability
- Follow Prisma's migration resolution system for tracking
- Consider using `pg_dump` for schema-only backups as lighter alternative

**File Structure:**
```
packages/server/src/
├── cli/
│   ├── migration/
│   │   ├── rollback.ts         # Main rollback command
│   │   ├── analyzer.ts         # Migration analysis utilities
│   │   └── generator.ts        # Down migration generator
├── db/
│   ├── migrations/
│   │   └── [timestamp]_[name]/
│   │       ├── migration.sql   # Up migration (existing)
│   │       └── down.sql        # Down migration (new)
│   └── rollback/
│       ├── history.ts          # Rollback history tracking
│       └── validator.ts        # Migration validation
└── scripts/
    ├── backup-migration.sh     # Pre-migration backup
    └── test-migration.sh       # Migration testing
```

**Success Criteria:**
- All migrations have corresponding down migrations
- Rollback completes within 5 minutes for typical migrations
- Zero data loss for safe migrations
- Clear warnings for risky operations
- 100% test coverage for rollback logic
- Production rollbacks require < 30 seconds of downtime
- Comprehensive audit trail for all operations

**Risk Mitigation:**
- Always test rollbacks in staging environment first
- Maintain manual rollback procedures as fallback
- Document all non-reversible migrations clearly
- Implement gradual rollout for the rollback system itself
- Create runbooks for common rollback scenarios