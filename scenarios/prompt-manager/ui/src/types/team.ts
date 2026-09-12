/**
 * Team types for the prompt-manager UI.
 *
 * API types are re-exported from @/lib/schemas (single source of truth).
 */

// Re-export API types from schemas (these include runtime validation)
export type {
  RuntimeMode,
  CoordinationPattern,
  ReportingMode,
  MessagingMode,
  QueuePolicy,
  CoordinationCapabilities,
  Runtime,
  Coordination,
  Execution,
  Team,
  TeamDetails,
  TeamRole,
  TeamMember,
  TeamMemberStatus,
  CreateTeamRequest,
  UpdateTeamRequest,
  AddMemberRequest,
  UpdateMemberRequest,
  SetRolesRequest,
  TeamSharedFileEntry,
  TeamSharedFileListResponse,
  TeamSharedFileContentResponse,
  TeamSharedFileWriteRequest,
  TeamSharedFileCreateRequest,
  TeamSharedFileRenameRequest,
  AvailableCCTeam,
  ImportCCRequest,
  ExportCCResponse,
} from '@/lib/schemas'
