import { ModelPreset, RunMode, RunnerType } from "@vrooli/proto-types/agent-manager/v1/domain/types_pb";

// Re-export proto type for reading Task objects from API
export type { ContextAttachment } from "@vrooli/proto-types/agent-manager/v1/domain/task_pb";

// Plain object interface for form editing (no proto Message metadata required)
export interface ContextAttachmentData {
  type: string;      // "file" | "link" | "note"
  key?: string;      // Unique identifier
  tags?: string[];   // Categorization tags
  path?: string;     // For "file" type
  url?: string;      // For "link" type
  content?: string;  // For "note" type, or descriptions
  label?: string;    // Optional human-readable label
}

export {
  RunnerType,
  ModelPreset,
  TaskStatus,
  RunStatus,
  ApprovalState,
  RunMode,
  RunPhase,
  RunEventType,
  RecoveryAction,
} from "@vrooli/proto-types/agent-manager/v1/domain/types_pb";

export type {
  AgentProfile,
  RunConfig,
} from "@vrooli/proto-types/agent-manager/v1/domain/profile_pb";

export type {
  Task,
} from "@vrooli/proto-types/agent-manager/v1/domain/task_pb";

export type {
  Run,
  RunSummary,
  RunnerStatus,
  ProbeResult,
  ApproveResult,
  StopAllResult,
  RunDiff,
} from "@vrooli/proto-types/agent-manager/v1/domain/run_pb";

export type {
  RunEvent,
} from "@vrooli/proto-types/agent-manager/v1/domain/events_pb";

export type {
  HealthResponse,
  ErrorResponse,
  JsonValue,
  JsonObject,
} from "@vrooli/proto-types/common/v1/types_pb";

export { HealthStatus } from "@vrooli/proto-types/common/v1/types_pb";

export type ModelOption = string | { id: string; description?: string };

export interface RunnerModelRegistry {
  models: ModelOption[];
  presets: Record<string, string>;
}

export interface ModelRegistry {
  version: number;
  fallbackRunnerTypes?: string[];
  runners: Record<string, RunnerModelRegistry>;
}

export interface ProfileFormData {
  name: string;
  profileKey?: string;
  description?: string;
  runnerType: RunnerType;
  model?: string;
  modelPreset?: ModelPreset;
  maxTurns?: number;
  timeoutMinutes?: number;
  fallbackRunnerTypes?: RunnerType[];
  allowedTools?: string[];
  deniedTools?: string[];
  skipPermissionPrompt?: boolean;
  requiresSandbox?: boolean;
  requiresApproval?: boolean;
  allowedPaths?: string[];
  deniedPaths?: string[];
}

export interface TaskFormData {
  title: string;
  description?: string;
  scopePath: string;
  projectRoot?: string;
  contextAttachments?: ContextAttachmentData[];
}

export interface RunFormData {
  taskId: string;
  agentProfileId?: string;
  tag?: string;
  existingSandboxId?: string;
  runnerType?: RunnerType;
  model?: string;
  modelPreset?: ModelPreset;
  maxTurns?: number;
  timeoutMinutes?: number;
  fallbackRunnerTypes?: RunnerType[];
  allowedTools?: string[];
  deniedTools?: string[];
  skipPermissionPrompt?: boolean;
  requiresSandbox?: boolean;
  requiresApproval?: boolean;
  allowedPaths?: string[];
  deniedPaths?: string[];
  prompt?: string;
  runMode?: RunMode;
  idempotencyKey?: string;
}

export interface ApproveFormData {
  actor?: string;
  commitMsg?: string;
  force?: boolean;
}

export interface RejectFormData {
  actor?: string;
  reason?: string;
}

export interface CreateInvestigationRunRequest {
  runIds: string[];
  customContext?: string;
}

export interface ApplyInvestigationRunRequest {
  investigationRunId: string;
  customContext?: string;
}

// =============================================================================
// Pricing Types
// =============================================================================

export type PricingSource = "manual_override" | "provider_api" | "historical_average" | "unknown";

export type PricingComponent =
  | "input_tokens"
  | "output_tokens"
  | "cache_read"
  | "cache_creation"
  | "web_search"
  | "server_tool_use";

export interface ModelPricingListItem {
  model: string;
  canonicalName?: string;
  provider: string;
  inputPricePer1M: number;
  outputPricePer1M: number;
  cacheReadPricePer1M?: number;
  cacheCreatePricePer1M?: number;
  inputSource: PricingSource;
  outputSource: PricingSource;
  cacheReadSource?: PricingSource;
  cacheCreateSource?: PricingSource;
  fetchedAt?: string;
  expiresAt?: string;
  pricingVersion?: string;
}

export interface ModelPricingListResponse {
  models: ModelPricingListItem[];
  total: number;
}

export interface PriceOverride {
  component: PricingComponent;
  priceUsd: number;
  expiresAt?: string;
  createdAt: string;
}

export interface OverridesResponse {
  overrides: PriceOverride[];
}

export interface SetOverrideRequest {
  component: PricingComponent;
  priceUsd: number;
  expiresAt?: string;
}

export interface ModelAlias {
  runnerModel: string;
  runnerType: string;
  canonicalModel: string;
  provider: string;
  createdAt: string;
  updatedAt: string;
}

export interface AliasesResponse {
  aliases: ModelAlias[];
  total: number;
}

export interface CreateAliasRequest {
  runnerModel: string;
  runnerType: string;
  canonicalModel: string;
  provider?: string;
}

export interface PricingSettings {
  historicalAverageDays: number;
  providerCacheTtlSeconds: number;
}

export interface UpdatePricingSettingsRequest {
  historicalAverageDays?: number;
  providerCacheTtlSeconds?: number;
}

export interface ProviderCacheStatus {
  provider: string;
  modelCount: number;
  lastFetchedAt: string;
  expiresAt: string;
  isStale: boolean;
}

export interface CacheStatusResponse {
  totalModels: number;
  expiredCount: number;
  providers: ProviderCacheStatus[];
}
