/**
 * Library Module Exports
 *
 * This module provides centralized exports for the lib/ directory.
 * Use these exports for cleaner imports in consuming code.
 */

// API Client - HTTP infrastructure
export {
  defaultApiClient as api,
  createApiClient,
  ApiClient,
  DEFAULT_API_BASE,
} from "./api-client";
export type { IApiClient } from "./api-client";

// API Errors - Structured error handling
export { ApiError, isApiError } from "./api-client";
export type { ApiErrorType } from "./api-client";

// API Endpoints - Route definitions
export { API_ENDPOINTS } from "./api-endpoints";
export type { ApiEndpoint } from "./api-endpoints";

// Utilities
export { cn } from "./utils";

// Formatting Utilities - Pure formatting functions
export { capitalize, formatDisplayText, formatFileSize, getFileExtension, formatRelativeTime } from "./format-utils";

// Backlog Utilities
export { sanitizeBacklogName, parseTagsInput, tagsToInput } from "./backlog-utils";
export { QUEUEABLE_BACKLOG_STATUSES, getBacklogNotQueueableReason, hasBlockingDeps, isBacklogQueueable } from "./backlog-queue-utils";

// Theme Utilities
export { applyTheme, resolveTheme, useResolvedTheme, watchSystemTheme } from "./theme-utils";
export type { ThemePreference, ResolvedTheme } from "./theme-utils";

// Error Utilities - Observability and diagnosis
export {
  categorizeError,
  createErrorLogEntry,
  generateCorrelationId,
  generateUniqueId,
  getRecoveryGuidance,
  logError,
  RECOVERY_PATHS,
} from "./error-utils";
export type { ErrorCategory, ErrorLogEntry } from "./error-utils";

// Workshop file utilities
export {
  WORKSHOP_FILE_PATHS,
  parseWorkshopRound,
  buildWorkshopRoundContent,
  getPendingDecisionCount,
  findBacklogFileByPath,
} from "./workshop-files";

// Readiness computation
export {
  READINESS_DIMENSIONS,
  DIMENSION_LABELS,
  SCORE_COLORS,
  buildReadinessData,
  computeNextNudge,
} from "./maturity";
export type { ReadinessIndicatorData } from "./maturity";

// Execution Utilities
export {
  EXECUTION_TAB_CONFIG,
  canCancelExecution,
  canFollowUpExecution,
  canRetryExecution,
  canStartExecution,
  isExecutionActive,
  isExecutionInTab,
  matchesExecutionFilters,
} from "./execution-utils";
export type { ExecutionFilters, ExecutionTabId } from "./execution-utils";

// Scenario Utilities
export { scenariosFromGlobs } from "./scenario-utils";

// Query Utilities - React Query configuration
export { defaultQueryOptions } from "./query-utils";
export type { DefaultQueryOptions } from "./query-utils";
