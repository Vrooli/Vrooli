/**
 * Barrel re-export file for backward compatibility.
 *
 * All API functions and types have been split into domain-specific modules:
 * - api-base.ts       - Shared base utilities (API_BASE, jsonResponse, resolveAttachmentUrl)
 * - api-types.ts      - Core domain types (Chat, Message, ToolCallRecord, etc.)
 * - api-chat.ts       - Chat CRUD, toggles, bulk ops
 * - api-misc.ts       - Labels, search, export, health, path validation
 * - api-messages.ts   - Message add, edit, regenerate, branch selection
 * - api-completion.ts - Streaming completion, SSE, StreamingEvent
 * - api-models.ts     - Models and usage tracking
 * - api-tools.ts      - Tool discovery, config, execution
 * - api-approvals.ts  - Tool approvals and pending approvals
 * - api-templates.ts  - Templates CRUD, import/export
 * - api-skills.ts     - Skills CRUD, suggestions, sync
 * - api-settings.ts   - YOLO mode, web search, suggestions settings, link preview
 * - api-uploads.ts    - Attachment upload functions
 * - api-agent.ts      - Agent mode types and functions
 *
 * Existing imports from "@/lib/api" or "../lib/api" continue to work unchanged.
 */

// Base utilities
export { resolveAttachmentUrl } from "./api-base";

// Core domain types
export type {
  Chat,
  ToolCall,
  Attachment,
  Message,
  ToolCallRecord,
  ApprovalOverride,
  Label,
  ChatWithMessages,
  ValidatePathResult,
  ExportFormat,
} from "./api-types";

// Chat operations
export {
  fetchChats,
  fetchChat,
  createChat,
  updateChat,
  deleteChat,
  deleteAllChats,
  deleteArchivedChats,
  markAllChatsAsRead,
  setActiveTemplate,
  bulkOperateChats,
  forkChat,
  toggleRead,
  toggleArchive,
  toggleStar,
} from "./api-chat";
export type {
  BulkOperation,
  BulkOperationResult,
} from "./api-chat";

// Labels, search, export, health, validation
export {
  fetchLabels,
  createLabel,
  deleteLabel,
  assignLabel,
  removeLabel,
  searchChats,
  autoNameChat,
  exportChat,
  validatePath,
  fetchProjectRoot,
  fetchHealth,
} from "./api-misc";
export type {
  SearchResult,
  ContentSearchOptions,
} from "./api-misc";

// Message operations
export {
  addMessage,
  regenerateMessage,
  editMessage,
  selectBranch,
  getMessageSiblings,
} from "./api-messages";
export type {
  AddMessageData,
  EditMessageData,
} from "./api-messages";

// Completion & streaming
export {
  completeChat,
  processSSEStream,
} from "./api-completion";
export type {
  StreamingEvent,
  SkillPayloadForAPI,
} from "./api-completion";

// Models & usage
export {
  fetchModels,
  fetchUsageStats,
  fetchChatUsageStats,
  fetchUsageRecords,
} from "./api-models";
export type {
  ModelPricing,
  ModelArchitecture,
  Model,
  UsageRecord,
  ModelUsage,
  DailyUsage,
  UsageStats,
} from "./api-models";

// Tool discovery protocol types (re-exported from proto-contracts via api-tools)
export type {
  ScenarioInfo,
  ToolParameters,
  ParameterSchema,
  ToolMetadata,
  ToolCategory,
  DiscoveredTool,
} from "./proto-contracts";

// Tools
export {
  fetchTools,
  fetchChatToolCalls,
  fetchToolSet,
  fetchScenarioStatuses,
  setToolEnabled,
  resetToolConfig,
  fetchScenarioInfo,
  syncTools,
  executeToolManually,
} from "./api-tools";
export type {
  ToolConfigurationScope,
  EffectiveTool,
  ToolSet,
  ScenarioStatus,
  ToolConfigUpdate,
  DiscoveryResult,
  ToolDefinition,
  ManualToolExecuteRequest,
  ManualToolExecuteResponse,
} from "./api-tools";

// Tool approvals
export {
  setToolApproval,
  getPendingApprovals,
  approveToolCall,
  rejectToolCall,
} from "./api-approvals";
export type {
  PendingApproval,
  ApprovalResult,
} from "./api-approvals";

// Templates
export {
  fetchTemplates,
  fetchTemplate,
  createTemplate,
  updateTemplate,
  deleteTemplate,
  resetTemplate,
  updateDefaultTemplate,
  importTemplates,
  exportTemplates,
} from "./api-templates";
export type {
  TemplateVariable,
  Template,
  TemplateSource,
  TemplateResponse,
  TemplateListResponse,
  CreateTemplateInput,
  UpdateTemplateInput,
} from "./api-templates";

// Skills
export {
  fetchSkills,
  fetchSkill,
  createSkill,
  updateSkill,
  deleteSkill,
  importSkills,
  exportSkills,
  fetchSkillSuggestions,
  syncSkills,
} from "./api-skills";
export type {
  SkillResponse,
  SkillListResponse,
  CreateSkillInput,
  UpdateSkillInput,
  SuggestedSkill,
  SkillSuggestResponse,
  SyncStatus,
} from "./api-skills";

// Settings
export {
  getYoloMode,
  setYoloMode,
  getSuggestionsSettings,
  setSuggestionsSettings,
  getWebSearchEnabled,
  setWebSearchEnabled,
  fetchLinkPreview,
} from "./api-settings";
export type {
  SuggestionsAutoSuggestConfig,
  SuggestionsSettingsResponse,
  LinkPreviewData,
} from "./api-settings";

// Uploads
export {
  uploadAttachment,
  uploadAgentAttachment,
} from "./api-uploads";
export type {
  UploadResponse,
} from "./api-uploads";

// Agent mode
export {
  AgentModeError,
  RUNNER_OPTIONS,
  SUPPORTED_RUNNER_TYPES,
  isCompactionEvent,
  getCompactionReduction,
  startAgentMode,
  sendAgentMessage,
  getAgentEvents,
  getAgentStatus,
  stopAgentMode,
  clearAgentMode,
  listAgentRuns,
  getRunEvents,
  attachAgentRun,
} from "./api-agent";
export type {
  RunnerType,
  AgentRunStatus,
  AgentChatConfig,
  AgentModeResponse,
  AgentModeStatus,
  AgentEvent,
  AgentEventsResponse,
  AgentRunSummary,
  ListAgentRunsResponse,
  ListAgentRunsOptions,
} from "./api-agent";
