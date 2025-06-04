# Complete RunStateMachine Migration Integration
Priority: HIGH  
Status: IN_PROGRESS
Dependencies: None
ParentTask: None

**Description:**  
Complete the integration of RunStateMachine with the UnifiedExecutionEngine following the migration guide. The foundation has been established with all type systems, interfaces, and basic integration in place. The remaining work involves updating the execution flow to use the new unified system while maintaining backward compatibility.

**Progress Completed:**
- ✅ Enhanced `packages/shared/src/run/executor.ts` with new execution context types
- ✅ Created `packages/shared/src/run/executionContextManager.ts` with context integration
- ✅ Added strategy configuration to `packages/shared/src/shape/configs/routine.ts`
- ✅ Created `packages/server/src/services/execution/unifiedSubroutineExecutor.ts`
- ✅ Created `packages/server/src/services/execution/interfaces.ts` with server-side interfaces
- ✅ Fixed import issues and type conflicts in the unified subroutine executor
- ✅ Added ExecutionContextManager import to RunStateMachine
- ✅ Fixed variable naming issues in initNewRun method
- ✅ Fixed null check for resourceSubType
- ✅ All compilation errors resolved (except minor pre-existing linter issues)

**Key Deliverables Remaining:**
- [ ] Update RunStateMachine execution flow to use the new context system:
  - Add buildExecutionContext method to runIteration
  - Implement swarm context retrieval and integration
  - Add enhanced execution path with fallback to legacy system
- [ ] Implement state synchronization between swarm and routine execution:
  - Add context synchronization after execution
  - Update swarm blackboard with routine results
  - Ensure bidirectional state flow
- [ ] Add strategy-aware execution:
  - Check for unified execution engine availability
  - Route execution to appropriate strategy
  - Handle strategy override system
- [ ] Create comprehensive testing for the migration:
  - Test both legacy and unified execution paths
  - Verify context flow and state synchronization
  - Ensure backward compatibility

**Technical Implementation Notes:**
The migration follows a 3-tier architecture (Coordination/Process/Execution layers) with swarm-based AI execution. The RunStateMachine now supports both legacy SubroutineExecutor and the new UnifiedExecutionEngine, allowing for gradual migration and backward compatibility.

**Files Modified:**
- `packages/shared/src/run/executor.ts`
- `packages/shared/src/run/executionContextManager.ts`
- `packages/shared/src/shape/configs/routine.ts`
- `packages/shared/src/run/stateMachine.ts`
- `packages/server/src/services/execution/unifiedSubroutineExecutor.ts`
- `packages/server/src/services/execution/interfaces.ts`

---

# Enhance Chat and Message Storage with Metadata
Priority: MEDIUM  
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

---

# Refactor Active Chat Store with Chat ID Identifier
Priority: MEDIUM  
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