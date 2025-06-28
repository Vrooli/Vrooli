/**
 * Event System Utilities
 * 
 * Common utility functions for the unified event system to reduce code duplication
 * and improve consistency.
 */

import { nanoid } from "@vrooli/shared";
import type {
  BaseEvent,
  EventSource,
  EventMetadata,
  BarrierSyncConfig,
} from "./types.js";
import {
  DELIVERY_GUARANTEES,
  PRIORITY_LEVELS,
  EVENT_BUS_CONSTANTS,
} from "./constants.js";

/**
 * Retry strategy configuration
 */
export interface RetryStrategy {
  maxAttempts: number;
  backoffMs: (attempt: number) => number;
  shouldRetry?: (error: Error, attempt: number) => boolean;
}

/**
 * Default retry strategies
 */
export const DEFAULT_RETRY_STRATEGIES = {
  /** Linear backoff with fixed delay */
  linear: (delayMs = 1000): RetryStrategy => ({
    maxAttempts: 3,
    backoffMs: () => delayMs,
  }),
  
  /** Exponential backoff starting from base delay */
  exponential: (baseDelayMs: number = EVENT_BUS_CONSTANTS.RETRY_BASE_DELAY_MS): RetryStrategy => ({
    maxAttempts: 3,
    backoffMs: (attempt) => baseDelayMs * Math.pow(EVENT_BUS_CONSTANTS.RETRY_EXPONENTIAL_FACTOR, attempt - 1),
  }),
  
  /** No retry */
  none: (): RetryStrategy => ({
    maxAttempts: 1,
    backoffMs: () => 0,
  }),
} as const;

/**
 * Execute a function with retry logic
 */
export async function withRetry<T>(
  fn: () => Promise<T>,
  strategy: RetryStrategy,
  logger?: { warn: (message: string, meta?: any) => void },
): Promise<T> {
  let lastError: Error | undefined;
  
  for (let attempt = 1; attempt <= strategy.maxAttempts; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error as Error;
      
      if (attempt === strategy.maxAttempts) {
        break;
      }
      
      if (strategy.shouldRetry && !strategy.shouldRetry(lastError, attempt)) {
        break;
      }
      
      const delayMs = strategy.backoffMs(attempt);
      
      logger?.warn(`Retry attempt ${attempt}/${strategy.maxAttempts} after ${delayMs}ms`, {
        error: lastError.message,
        attempt,
        delayMs,
      });
      
      await delay(delayMs);
    }
  }
  
  throw lastError || new Error("Retry failed with unknown error");
}

/**
 * Simple delay helper
 */
export function delay(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Create a typed event with proper structure
 */
export function createTypedEvent<TData = unknown>(
  type: string,
  data: TData,
  source: EventSource,
  options?: {
    id?: string;
    correlationId?: string;
    metadata?: EventMetadata;
  },
): BaseEvent {
  return {
    id: options?.id || nanoid(),
    type,
    timestamp: new Date(),
    source,
    correlationId: options?.correlationId,
    data,
    metadata: options?.metadata,
  };
}

/**
 * Pattern matching for MQTT-style event types
 */
export class EventPatternMatcher {
  private readonly pattern: string;
  private readonly regex: RegExp;
  
  constructor(pattern: string) {
    this.pattern = pattern;
    this.regex = this.patternToRegex(pattern);
  }
  
  /**
   * Check if an event type matches this pattern
   */
  matches(eventType: string): boolean {
    return this.regex.test(eventType);
  }
  
  /**
   * Get the original pattern string
   */
  getPattern(): string {
    return this.pattern;
  }
  
  /**
   * Convert MQTT-style pattern to regex
   */
  private patternToRegex(pattern: string): RegExp {
    // Special case: # matches everything
    if (pattern === "#") {
      return /^.*$/;
    }
    
    // Escape special regex characters except * and /
    let regexStr = pattern.replace(/[.+?^${}()|[\]\\]/g, "\\$&");
    
    // Replace MQTT wildcards with regex equivalents
    // * matches a single level (no slashes)
    regexStr = regexStr.replace(/\*/g, "[^/]+");
    
    // # matches multiple levels (must be at end)
    if (regexStr.endsWith("/#")) {
      regexStr = regexStr.slice(0, -2) + "(/.*)?";
    }
    
    return new RegExp(`^${regexStr}$`);
  }
}

/**
 * Create a pattern matcher for event subscriptions
 */
export function createPatternMatcher(pattern: string): EventPatternMatcher {
  return new EventPatternMatcher(pattern);
}

/**
 * Batch pattern matcher for multiple patterns
 */
export class BatchPatternMatcher {
  private readonly matchers: EventPatternMatcher[];
  
  constructor(patterns: string[]) {
    this.matchers = patterns.map(p => new EventPatternMatcher(p));
  }
  
  /**
   * Check if event type matches any pattern
   */
  matches(eventType: string): boolean {
    return this.matchers.some(m => m.matches(eventType));
  }
  
  /**
   * Get all patterns
   */
  getPatterns(): string[] {
    return this.matchers.map(m => m.getPattern());
  }
}

/**
 * Validate event structure
 */
export function validateEventStructure(event: unknown): event is BaseEvent {
  if (!event || typeof event !== "object") {
    return false;
  }
  
  const e = event as any;
  
  return (
    typeof e.id === "string" &&
    typeof e.type === "string" &&
    e.timestamp instanceof Date &&
    typeof e.source === "object" &&
    e.source !== null &&
    (typeof e.source.tier === "number" || typeof e.source.tier === "string") &&
    typeof e.source.component === "string" &&
    e.data !== undefined
  );
}

/**
 * Extract tier from event type
 */
export function getTierFromEventType(eventType: string): 1 | 2 | 3 | "cross-cutting" | "safety" | undefined {
  if (eventType.startsWith("swarm/") || eventType.startsWith("goal/") || 
      eventType.startsWith("team/") || eventType.startsWith("resource/")) {
    return 1;
  }
  
  if (eventType.startsWith("routine/") || eventType.startsWith("state/") || 
      eventType.startsWith("context/")) {
    return 2;
  }
  
  if (eventType.startsWith("step/") || eventType.startsWith("tool/") || 
      eventType.startsWith("strategy/")) {
    return 3;
  }
  
  if (eventType.startsWith("safety/") || eventType.startsWith("emergency/") || 
      eventType.startsWith("threat/")) {
    return "safety";
  }
  
  if (eventType.startsWith("execution/") || eventType.startsWith("recovery/") || 
      eventType.startsWith("fallback/") || eventType.startsWith("circuit_breaker/")) {
    return "cross-cutting";
  }
  
  return undefined;
}

/**
 * Create default metadata based on event type
 */
export function createDefaultMetadata(eventType: string): EventMetadata {
  // Safety events always use barrier-sync
  if (eventType.startsWith("safety/") || eventType.startsWith("emergency/")) {
    return {
      deliveryGuarantee: DELIVERY_GUARANTEES.BARRIER_SYNC,
      priority: PRIORITY_LEVELS.CRITICAL,
    };
  }
  
  // Approval events use barrier-sync
  if (eventType.includes("approval_required")) {
    return {
      deliveryGuarantee: DELIVERY_GUARANTEES.BARRIER_SYNC,
      priority: PRIORITY_LEVELS.HIGH,
    };
  }
  
  // Completion events use reliable delivery
  if (eventType.endsWith("/completed") || eventType.endsWith("/failed")) {
    return {
      deliveryGuarantee: DELIVERY_GUARANTEES.RELIABLE,
      priority: PRIORITY_LEVELS.MEDIUM,
    };
  }
  
  // Default to fire-and-forget
  return {
    deliveryGuarantee: DELIVERY_GUARANTEES.FIRE_AND_FORGET,
    priority: PRIORITY_LEVELS.MEDIUM,
  };
}

/**
 * Format event for logging
 */
export function formatEventForLogging(event: BaseEvent): Record<string, unknown> {
  return {
    id: event.id,
    type: event.type,
    timestamp: event.timestamp.toISOString(),
    source: `${event.source.tier}:${event.source.component}`,
    correlationId: event.correlationId,
    metadata: event.metadata,
    // Don't log full data payload for security/size reasons
    dataKeys: event.data && typeof event.data === "object" ? Object.keys(event.data) : undefined,
  };
}

/**
 * Create a barrier configuration with defaults
 */
export function createBarrierConfig(options: Partial<BarrierSyncConfig> = {}): BarrierSyncConfig {
  return {
    quorum: options.quorum || 1,
    timeoutMs: options.timeoutMs || EVENT_BUS_CONSTANTS.DEFAULT_BARRIER_TIMEOUT_MS,
    timeoutAction: options.timeoutAction || "auto-reject",
    requiredResponders: options.requiredResponders,
  };
}

/**
 * Check if event requires barrier synchronization
 */
export function requiresBarrierSync(event: BaseEvent): boolean {
  return event.metadata?.deliveryGuarantee === DELIVERY_GUARANTEES.BARRIER_SYNC;
}

/**
 * Calculate priority score for event ordering (higher = more important)
 */
export function calculatePriorityScore(event: BaseEvent): number {
  const priorityScores = {
    [PRIORITY_LEVELS.CRITICAL]: 1000,
    [PRIORITY_LEVELS.HIGH]: 100,
    [PRIORITY_LEVELS.MEDIUM]: 10,
    [PRIORITY_LEVELS.LOW]: 1,
  };
  
  const basePriority = priorityScores[event.metadata?.priority || PRIORITY_LEVELS.MEDIUM];
  
  // Boost score for barrier-sync events
  const barrierBoost = requiresBarrierSync(event) ? 500 : 0;
  
  // Boost score for safety events
  const safetyBoost = event.source.tier === "safety" ? 200 : 0;
  
  return basePriority + barrierBoost + safetyBoost;
}
