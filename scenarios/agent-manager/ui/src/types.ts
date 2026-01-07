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

// Investigation types
export type InvestigationStatus = "pending" | "running" | "completed" | "failed" | "cancelled";

export interface AnalysisType {
  errorDiagnosis: boolean;
  efficiencyAnalysis: boolean;
  toolUsagePatterns: boolean;
}

export interface ReportSections {
  rootCauseEvidence: boolean;
  recommendations: boolean;
  metricsSummary: boolean;
}

export interface Evidence {
  runId: string;
  eventSeq?: number;
  description: string;
  snippet?: string;
}

export interface RootCauseAnalysis {
  description: string;
  evidence: Evidence[];
  confidence: "high" | "medium" | "low";
}

export interface Recommendation {
  id: string;
  priority: "critical" | "high" | "medium" | "low";
  title: string;
  description: string;
  actionType: "prompt_change" | "profile_config" | "code_fix";
}

export interface InvestigationReport {
  summary: string;
  rootCause?: RootCauseAnalysis;
  recommendations?: Recommendation[];
}

export interface MetricsData {
  totalRuns: number;
  successRate: number;
  avgDurationSeconds: number;
  totalTokensUsed: number;
  totalCost: number;
  toolUsageBreakdown: Record<string, number>;
  errorTypeBreakdown: Record<string, number>;
  customMetrics: Record<string, number>;
}

export interface Investigation {
  id: string;
  runIds: string[];
  status: InvestigationStatus;
  analysisType: AnalysisType;
  reportSections: ReportSections;
  customContext?: string;
  progress: number;
  agentRunId?: string;
  findings?: InvestigationReport;
  metrics?: MetricsData;
  errorMessage?: string;
  sourceInvestigationId?: string;
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
}

export interface CreateInvestigationRequest {
  runIds: string[];
  analysisType: AnalysisType;
  reportSections: ReportSections;
  customContext?: string;
}

export interface ApplyFixesRequest {
  recommendationIds: string[];
  note?: string;
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
