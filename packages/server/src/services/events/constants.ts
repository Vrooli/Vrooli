/**
 * Event System Constants
 * 
 * Centralized constants for the unified event system to avoid magic numbers
 * and strings throughout the codebase.
 */

/**
 * Event bus configuration constants
 */
export const EVENT_BUS_CONSTANTS = {
  /** Maximum number of events to keep in history */
  MAX_EVENT_HISTORY: 1000,
  /** Maximum number of listeners for a single event pattern */
  MAX_EVENT_LISTENERS: 1000,
  /** Length of pattern suffix for wildcard matching */
  PATTERN_SUFFIX_LENGTH: 2,
  /** Base delay for retry attempts in milliseconds */
  RETRY_BASE_DELAY_MS: 100,
  /** Exponential backoff factor for retries */
  RETRY_EXPONENTIAL_FACTOR: 2,
  /** Default timeout for barrier sync in milliseconds */
  DEFAULT_BARRIER_TIMEOUT_MS: 30000,
} as const;

/**
 * Event catalog configuration constants
 */
export const CATALOG_CONSTANTS = {
  /** Maximum number of top events to return */
  MAX_TOP_EVENTS: 10,
  /** Minimum length for event proposal justification */
  MIN_JUSTIFICATION_LENGTH: 50,
  /** Initial confidence score for proposed events */
  INITIAL_CONFIDENCE_SCORE: 0.5,
  /** Threshold for auto-accepting proposed events */
  AUTO_ACCEPT_CONFIDENCE_THRESHOLD: 0.8,
  /** Maximum number of proposals to keep in history */
  MAX_PROPOSAL_HISTORY: 100,
} as const;

/**
 * Component names used in event sources
 */
export const COMPONENT_NAMES = {
  EVENT_BUS: "event-bus",
  EVENT_CATALOG: "event-catalog",
  APPROVAL_SYSTEM: "tool-approval-system",
  TIER_ONE_COORDINATOR: "tier-1-coordinator",
  TIER_TWO_ORCHESTRATOR: "tier-2-orchestrator",
  TIER_THREE_EXECUTOR: "tier-3-executor",
} as const;

/**
 * Event type prefixes for consistent naming
 */
export const EVENT_PREFIXES = {
  // Tier 1 - Coordination
  SWARM: "swarm/",
  GOAL: "goal/",
  TEAM: "team/",
  RESOURCE: "resource/",
  
  // Tier 2 - Process
  ROUTINE: "routine/",
  STATE: "state/",
  CONTEXT: "context/",
  
  // Tier 3 - Execution
  STEP: "step/",
  TOOL: "tool/",
  STRATEGY: "strategy/",
  
  // Cross-cutting
  SAFETY: "safety/",
  EMERGENCY: "emergency/",
  THREAT: "threat/",
  EXECUTION: "execution/",
  RECOVERY: "recovery/",
  FALLBACK: "fallback/",
  CIRCUIT_BREAKER: "circuit_breaker/",
} as const;

/**
 * Event action suffixes for consistent naming
 */
export const EVENT_ACTIONS = {
  // Lifecycle
  CREATED: "created",
  UPDATED: "updated",
  STARTED: "started",
  COMPLETED: "completed",
  FAILED: "failed",
  CANCELLED: "cancelled",
  
  // Approvals
  APPROVAL_REQUIRED: "approval_required",
  APPROVAL_GRANTED: "approval_granted",
  APPROVAL_REJECTED: "approval_rejected",
  APPROVAL_TIMEOUT: "approval_timeout",
  APPROVAL_CANCELLED: "approval_cancelled",
  
  // State changes
  STATE_CHANGED: "state_changed",
  THRESHOLD_CROSSED: "threshold_crossed",
  
  // Resource management
  ALLOCATED: "allocated",
  EXHAUSTED: "exhausted",
  OPTIMIZED: "optimized",
  RATE_LIMITED: "rate_limited",
  
  // Team operations
  FORMED: "formed",
  MEMBER_ADDED: "member_added",
  MEMBER_REMOVED: "member_removed",
  DISBANDED: "disbanded",
  
  // Error handling
  ERROR_OCCURRED: "error/occurred",
  ERROR_REPORTING_FAILED: "error/reporting_failed",
  
  // Performance
  PERFORMANCE_MEASURED: "performance/measured",
  PERFORMANCE_FAILED: "performance/failed",
  FEEDBACK_RECEIVED: "feedback/received",
  
  // Recovery
  ATTEMPT: "attempt",
  OUTCOME: "outcome",
  EXECUTED: "executed",
} as const;

/**
 * Delivery guarantee levels
 */
export const DELIVERY_GUARANTEES = {
  FIRE_AND_FORGET: "fire-and-forget" as const,
  RELIABLE: "reliable" as const,
  BARRIER_SYNC: "barrier-sync" as const,
} as const;

/**
 * Priority levels for event processing
 */
export const PRIORITY_LEVELS = {
  LOW: "low" as const,
  MEDIUM: "medium" as const,
  HIGH: "high" as const,
  CRITICAL: "critical" as const,
} as const;

/**
 * Risk levels for approval decisions
 */
export const RISK_LEVELS = {
  LOW: "low" as const,
  MEDIUM: "medium" as const,
  HIGH: "high" as const,
  CRITICAL: "critical" as const,
} as const;

/**
 * Barrier sync timeout actions
 */
export const TIMEOUT_ACTIONS = {
  AUTO_APPROVE: "auto-approve" as const,
  AUTO_REJECT: "auto-reject" as const,
  KEEP_PENDING: "keep-pending" as const,
} as const;

/**
 * Fallback actions for safety events
 */
export const FALLBACK_ACTIONS = {
  EMERGENCY_STOP: "emergency_stop" as const,
  SAFE_FALLBACK: "safe_fallback" as const,
  USER_PROMPT: "user_prompt" as const,
} as const;

/**
 * Type alias for all constant values
 */
export type EventBusConstant = typeof EVENT_BUS_CONSTANTS[keyof typeof EVENT_BUS_CONSTANTS];
export type CatalogConstant = typeof CATALOG_CONSTANTS[keyof typeof CATALOG_CONSTANTS];
export type ComponentName = typeof COMPONENT_NAMES[keyof typeof COMPONENT_NAMES];
export type EventPrefix = typeof EVENT_PREFIXES[keyof typeof EVENT_PREFIXES];
export type EventAction = typeof EVENT_ACTIONS[keyof typeof EVENT_ACTIONS];
export type DeliveryGuarantee = typeof DELIVERY_GUARANTEES[keyof typeof DELIVERY_GUARANTEES];
export type PriorityLevel = typeof PRIORITY_LEVELS[keyof typeof PRIORITY_LEVELS];
export type RiskLevel = typeof RISK_LEVELS[keyof typeof RISK_LEVELS];
export type TimeoutAction = typeof TIMEOUT_ACTIONS[keyof typeof TIMEOUT_ACTIONS];
export type FallbackAction = typeof FALLBACK_ACTIONS[keyof typeof FALLBACK_ACTIONS];
