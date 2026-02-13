/**
 * Swarm Manager Configuration
 *
 * This module defines the control surface for the Swarm Manager UI.
 * All tunable levers are centralized here with clear groupings,
 * safe defaults, and documented impacts.
 *
 * Design Principles:
 * - Expose only high-value, low-risk levers
 * - Group related settings by concern
 * - Provide sane defaults that work for common usage
 * - Make impacts clear and monotonic where possible
 *
 * NOT configurable here (intentionally internal):
 * - HTTP client implementation details (cache policies)
 * - Component styling (Tailwind classes)
 * - Type definitions (domain types)
 *
 * DOC: docs/reference/configuration.md
 * DOC: docs/internal/SEAMS.md#control-surface--tunable-levers-design
 */

// ============================================================================
// Data Fetching Configuration
// ============================================================================

/**
 * Controls how data is fetched and refreshed across the UI.
 *
 * These settings affect network usage, responsiveness, and server load.
 * All values have been tested to provide good UX under typical conditions.
 */
export const dataFetchingConfig = {
  /**
   * Number of retry attempts for failed API requests.
   *
   * Impact: Higher values improve reliability on flaky networks but delay
   * error feedback to the user.
   *
   * Default: 2 (retry twice before showing error)
   * Range: 0-5 (0 = no retries, show errors immediately)
   */
  retryCount: 2,

  /**
   * Delay between retry attempts in milliseconds.
   *
   * Impact: Higher values reduce server hammering but slow recovery.
   * Uses exponential backoff: delay * 2^attempt
   *
   * Default: 1000ms
   * Range: 500-5000ms
   */
  retryDelayMs: 1000,

  /**
   * Time in milliseconds before considering cached data stale.
   *
   * Impact: Lower values ensure fresher data but increase API calls.
   * Higher values reduce load but may show outdated information.
   *
   * Default: 30000ms (30 seconds)
   * Range: 5000-300000ms (5 seconds to 5 minutes)
   */
  staleTimeMs: 30_000,

  /**
   * Time in milliseconds to keep unused data in cache.
   *
   * Impact: Higher values improve performance when navigating back
   * but use more memory.
   *
   * Default: 300000ms (5 minutes)
   * Range: 60000-600000ms (1 minute to 10 minutes)
   */
  cacheTimeMs: 300_000,

  /**
   * Whether to refetch data when the window regains focus.
   *
   * Impact: true = always fresh data when returning to tab
   *         false = relies on staleTime for freshness
   *
   * Default: true
   */
  refetchOnWindowFocus: true,
} as const;

// ============================================================================
// Display Limits Configuration
// ============================================================================

/**
 * Controls truncation and pagination of displayed items.
 *
 * These settings balance information density with readability.
 * They affect what users see at a glance vs. what requires drilling down.
 */
export const displayLimitsConfig = {
  /**
   * Maximum tags shown on backlog cards before showing "+N more".
   *
   * Impact: Higher values show more context but create visual clutter.
   *
   * Default: 3
   * Range: 1-10
   */
  backlogCardMaxTags: 3,

  /**
   * Maximum tags shown on scenario cards before truncation.
   *
   * Impact: Scenario cards are wider, so they can accommodate more tags.
   *
   * Default: 5
   * Range: 1-10
   */
  scenarioCardMaxTags: 5,

  /**
   * Number of lines to show for descriptions before truncation.
   *
   * Impact: Higher values show more detail but create uneven card heights.
   * Uses CSS line-clamp.
   *
   * Default: 2
   * Range: 1-5
   */
  descriptionLineClamp: 2,

  /**
   * Default page size for paginated lists.
   *
   * Impact: Higher values reduce navigation clicks but increase load time.
   * Currently used for future pagination implementation.
   *
   * Default: 20
   * Range: 10-100
   */
  defaultPageSize: 20,
} as const;

// ============================================================================
// Insights Engine Configuration
// ============================================================================

/**
 * Controls the behavior of the insights/self-improvement engine.
 *
 * These settings determine how the system learns from patterns
 * and suggests meta-level improvements.
 */
export const insightsConfig = {
  /**
   * Minimum number of completed scenarios before generating insights.
   *
   * Impact: Ensures enough data for meaningful pattern detection.
   *
   * Default: 3
   * Range: 1-10
   */
  minimumCompletedScenarios: 3,

  /**
   * Number of recent actions to analyze for pattern detection.
   *
   * Impact: Higher values detect longer-term patterns but use more memory.
   *
   * Default: 50
   * Range: 10-200
   */
  patternWindowSize: 50,

  /**
   * Confidence threshold (0-1) for surfacing an insight.
   *
   * Impact: Higher values show fewer but more reliable insights.
   * Lower values show more insights but with more noise.
   *
   * Default: 0.7 (70% confidence)
   * Range: 0.5-0.95
   */
  confidenceThreshold: 0.7,
} as const;

// ============================================================================
// UI Behavior Configuration
// ============================================================================

/**
 * Controls general UI behaviors and interactions.
 *
 * These settings affect the user experience but don't change business logic.
 */
export const uiBehaviorConfig = {
  /**
   * Debounce delay for search inputs in milliseconds.
   *
   * Impact: Higher values reduce API calls during typing but feel less
   * responsive. Lower values feel snappy but may hammer the server.
   *
   * Default: 300ms
   * Range: 100-1000ms
   */
  searchDebounceMs: 300,

  /**
   * Duration for toast notifications in milliseconds.
   *
   * Impact: How long success/error messages stay visible.
   *
   * Default: 5000ms (5 seconds)
   * Range: 2000-10000ms
   */
  toastDurationMs: 5000,

  /**
   * Whether to show loading skeletons vs spinners.
   *
   * Impact: Skeletons provide perceived performance but need more code.
   *
   * Default: true (use skeleton loading states)
   */
  useSkeletonLoading: true,

  /**
   * Enable keyboard shortcuts throughout the UI.
   *
   * Impact: Power users can navigate faster; may conflict with
   * browser shortcuts.
   *
   * Default: true
   */
  enableKeyboardShortcuts: true,

  /**
   * Show confirmation dialogs for destructive actions.
   *
   * Impact: true = safer but slower; false = faster but riskier
   *
   * Default: true (always confirm destructive actions)
   */
  confirmDestructiveActions: true,
} as const;

// ============================================================================
// API Configuration
// ============================================================================

/**
 * API-related configuration.
 *
 * Most API config comes from environment variables, but these provide
 * fallbacks and additional controls.
 */
export const apiConfig = {
  /**
   * Request timeout in milliseconds.
   *
   * Impact: Higher values allow for slower networks/operations.
   * Lower values fail faster on unresponsive servers.
   *
   * Default: 30000ms (30 seconds)
   * Range: 5000-120000ms
   */
  requestTimeoutMs: 30_000,

  /**
   * API version prefix for all requests.
   *
   * Impact: Changing this affects all API calls. Only change if the
   * server supports the new version.
   *
   * Default: "v1"
   */
  apiVersion: "v1" as const,
} as const;

// ============================================================================
// Combined Configuration Export
// ============================================================================

/**
 * Full configuration object combining all concerns.
 *
 * Import this for access to all configuration, or import individual
 * config objects for tree-shaking.
 */
export const config = {
  dataFetching: dataFetchingConfig,
  displayLimits: displayLimitsConfig,
  insights: insightsConfig,
  uiBehavior: uiBehaviorConfig,
  api: apiConfig,
} as const;

// ============================================================================
// Type Exports
// ============================================================================

export type DataFetchingConfig = typeof dataFetchingConfig;
export type DisplayLimitsConfig = typeof displayLimitsConfig;
export type InsightsConfig = typeof insightsConfig;
export type UIBehaviorConfig = typeof uiBehaviorConfig;
export type ApiConfig = typeof apiConfig;
export type Config = typeof config;
