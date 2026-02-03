/**
 * Agent types for the prompt-manager UI.
 *
 * API types are re-exported from @/lib/schemas (single source of truth).
 */

// Re-export API types from schemas (these include runtime validation)
export type {
  Agent,
  AgentStatus,
  AgentAppearance,
  AgentCapability,
  AgentCapabilities,
  AgentConnector,
  ConnectorType,
  AgentHeartbeat,
  CreateAgentRequest,
  UpdateAgentRequest,
  SoulRequest,
  SoulResponse,
  AgentFileEntry,
  AgentFileListResponse,
  AgentFileContentResponse,
  AgentFileWriteRequest,
  AgentFileCreateRequest,
  AgentFileRenameRequest,
  PromptPreviewResponse,
} from '@/lib/schemas'

// Re-export default colors constant
export { DEFAULT_AGENT_COLORS } from '@/lib/schemas'
