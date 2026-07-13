import { RunMode, ExecutionMode } from "@vrooli/proto-types/agent-manager/v1/domain/types_pb";

// Re-export proto type for reading Task objects from API
export type { ContextAttachment } from "@vrooli/proto-types/agent-manager/v1/domain/task_pb";

// Plain object interface for form editing (no proto Message metadata required)
export interface ContextAttachmentData {
  type: string;           // "file" | "link" | "note" | "image"
  key?: string;           // Unique identifier
  tags?: string[];        // Categorization tags
  path?: string;          // For "file" type
  url?: string;           // For "link" type
  content?: string;       // For "note" type, or descriptions
  label?: string;         // Optional human-readable label
  attachment_id?: string; // For "image" type - reference to uploaded Attachment
}

export {
  NetworkAccess,
  SandboxMode,
  TaskStatus,
  RunStatus,
  RunFinalizationStatus,
  ApprovalState,
  RunMode,
  ExecutionMode,
  RunPhase,
  RunEventType,
  RecoveryAction,
} from "@vrooli/proto-types/agent-manager/v1/domain/types_pb";

export { RunnerType } from "@vrooli/proto-types/agent-manager/v1/domain/types_pb";

export type {
  AgentProfile,
  RunConfig,
} from "@vrooli/proto-types/agent-manager/v1/domain/profile_pb";

export type {
  Task,
} from "@vrooli/proto-types/agent-manager/v1/domain/task_pb";

export type {
  Run,
  RunActions,
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

export interface ProfileFormData {
  name: string;
  profileKey?: string;
  description?: string;
  roleRef: string;
  maxTurns?: number;
  timeoutMinutes?: number;
  allowedTools?: string[];
  deniedTools?: string[];
  skipPermissionPrompt?: boolean;
  // Sandbox mode for the run. Empty preserves the server-side default
  // (Tracking); "off" disables sandboxing entirely.
  sandboxMode?: "off" | "tracking" | "protected";
  networkAccess?: "none" | "localhost" | "full";
  allowedPaths?: string[];
  deniedPaths?: string[];
  features?: {
    enableBrowser?: boolean;
  };
  extraFlags?: Record<string, string[]>;
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
  roleRef?: string;
  maxTurns?: number;
  timeoutMinutes?: number;
  allowedTools?: string[];
  deniedTools?: string[];
  skipPermissionPrompt?: boolean;
  // Sandbox mode override for this run. Empty preserves the profile /
  // server default.
  sandboxMode?: "off" | "tracking" | "protected";
  networkAccess?: "none" | "localhost" | "full";
  allowedPaths?: string[];
  deniedPaths?: string[];
  features?: {
    enableBrowser?: boolean;
  };
  extraFlags?: Record<string, string[]>;
  prompt?: string;
  runMode?: RunMode;
  /** Execution substrate. Interactive launches the real CLI in a live
   *  web-console session and is rejected for protected/sandboxed runs. Empty
   *  preserves the server default (codec-pipe). */
  executionMode?: ExecutionMode;
  idempotencyKey?: string;
  /** Conversation linkage per Decision D7 of the auditability contract.
   *  Spawn surfaces SHOULD populate at least one explicitly. */
  conversationId?: string;
  parentRunId?: string;
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

/** Investigation depth controls how thorough the investigation should be. */
export type InvestigationDepth = "quick" | "standard" | "deep";

export interface CreateInvestigationRunRequest {
  runIds: string[];
  customContext?: string;
  /** Investigation depth: quick (fast analysis), standard (balanced), or deep (thorough) */
  depth?: InvestigationDepth;
  /** Project root - where the agent can look at code (explicit, no guessing) */
  projectRoot?: string;
  /** Scope paths - where the agent can make changes */
  scopePaths?: string[];
}

export interface ApplyInvestigationRunRequest {
  investigationRunId: string;
  customContext?: string;
}

// =============================================================================
// Investigation Settings Types
// =============================================================================

/** Context flags for investigation - what to include as context attachments */
export interface InvestigationContextFlags {
  /** Include run summary data (always lightweight) */
  runSummaries: boolean;
  /** Include run events (can be large but essential for debugging) */
  runEvents: boolean;
  /** Include code changes made during runs */
  runDiffs: boolean;
  /** Include full run logs (can be very large) */
  fullLogs: boolean;
}

/** Investigation settings - configuration for investigation agents */
export interface InvestigationSettings {
  /** Plain text prompt template for Investigation agents - no variables/templating */
  promptTemplate: string;
  /** Plain text prompt template for Apply Investigation agents - no variables/templating */
  applyPromptTemplate: string;
  /** Default investigation depth */
  defaultDepth: InvestigationDepth;
  /** Default context flags */
  defaultContext: InvestigationContextFlags;
  /** Allowlist for which run tags are eligible for Apply Fixes and recommendation extraction */
  investigationTagAllowlist: InvestigationTagRule[];
  /** When settings were last modified */
  updatedAt: string;
}

export interface InvestigationTagRule {
  pattern: string;
  isRegex: boolean;
  caseSensitive: boolean;
}


/** Default context flags */
export const DEFAULT_INVESTIGATION_CONTEXT: InvestigationContextFlags = {
  runSummaries: true,
  runEvents: true,
  runDiffs: true,
  fullLogs: false,
};

// Update the investigation run request to include context flags
export interface CreateInvestigationRunRequestV2 extends CreateInvestigationRunRequest {
  /** Context flags - what to include in the investigation */
  context?: InvestigationContextFlags;
  /** Manual scenario path override */
  scenarioOverride?: string;
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

// =============================================================================
// Recommendation Extraction Types
// =============================================================================

/** Status of recommendation extraction for investigation runs */
export type RecommendationStatus =
  | "none"       // Not applicable (non-investigation run or not yet complete)
  | "pending"    // Awaiting extraction (queued for background processing)
  | "extracting" // Extraction in progress
  | "complete"   // Successfully extracted and cached
  | "failed";    // Extraction failed after max retries

/** A single recommendation extracted from an investigation */
export interface Recommendation {
  id: string;
  text: string;
  selected: boolean;
  /** User-added note (appended inline when serializing) */
  note?: string;
  /** Indicates this is a newly added item */
  isNew?: boolean;
  /** Indicates text was edited */
  isEdited?: boolean;
}

/** A category grouping related recommendations */
export interface RecommendationCategory {
  id: string;
  name: string;
  recommendations: Recommendation[];
  /** Indicates this is a newly added category */
  isNew?: boolean;
}

/** Result of extracting recommendations from an investigation run */
export interface ExtractionResult {
  success: boolean;
  categories: RecommendationCategory[];
  rawText: string;
  extractedFrom: "summary" | "events" | "pending"; // "pending" indicates in-progress
  error?: string;
}

/**
 * Extended Run type with recommendation extraction fields.
 * The base Run type comes from proto-types, but these fields are added
 * by the backend when returning run data.
 */
export interface RunWithRecommendations {
  recommendationStatus?: RecommendationStatus;
  recommendationResult?: ExtractionResult;
  recommendationError?: string;
  recommendationAttempts?: number;
  recommendationQueuedAt?: string;
}
