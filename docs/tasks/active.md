# Enhance Chat and Message Storage with Metadata
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

---

# Refactor Active Chat Store with Chat ID Identifier
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