/**
 * Loop Detection Module
 *
 * STABILITY: STABLE CORE
 *
 * Intelligent loop detection for AI browser navigation. Instead of naively
 * counting repeated actions, this module tracks whether actions have an effect.
 *
 * Key insight: "scroll:down" x5 is fine if the page scrolls each time.
 * It's only a loop if the scroll has no effect (e.g., at bottom of page).
 *
 * CONTROL LEVERS:
 * - LoopDetectionConfig: Per-action-type thresholds and behaviors
 * - ActionContext: Metadata about action effectiveness
 * - Detection strategies can be tuned per action type
 */

import type { BrowserAction, ScrollAction, ClickAction } from '../action/types';

// =============================================================================
// CONFIGURATION TYPES - The "Control Levers"
// =============================================================================

/**
 * Configuration for loop detection behavior.
 * All thresholds are configurable per action type.
 *
 * USAGE: Pass to createLoopDetector() to customize behavior.
 */
export interface LoopDetectionConfig {
  /**
   * Scroll-specific loop detection settings.
   */
  scroll: ScrollLoopConfig;

  /**
   * Click-specific loop detection settings.
   */
  click: ClickLoopConfig;

  /**
   * Default settings for action types not explicitly configured.
   * Used as fallback for: type, navigate, hover, select, wait, keypress
   */
  default: DefaultLoopConfig;

  /**
   * Whether to include detailed context in loop detection results.
   * Useful for debugging but adds overhead.
   */
  includeDebugInfo?: boolean;
}

/**
 * Scroll loop detection configuration.
 */
export interface ScrollLoopConfig {
  /**
   * Max consecutive scrolls in the same direction when scroll HAS NO EFFECT.
   * "No effect" = scroll position didn't change (stuck at boundary).
   * Default: 2 (stop after 2 ineffective scrolls)
   */
  maxIneffectiveRepeats: number;

  /**
   * Max consecutive scrolls in the same direction when scroll HAS EFFECT.
   * Set to -1 for unlimited (recommended - effective scrolls should continue).
   * Default: -1 (unlimited)
   */
  maxEffectiveRepeats: number;

  /**
   * Minimum scroll delta (in pixels) to consider a scroll "effective".
   * Helps avoid false positives from sub-pixel movements.
   * Default: 10
   */
  minScrollDelta: number;
}

/**
 * Click loop detection configuration.
 */
export interface ClickLoopConfig {
  /**
   * Max consecutive clicks on the SAME element before flagging as loop.
   * "Same element" = same elementId or same coordinates.
   * Default: 3
   */
  maxSameElementRepeats: number;

  /**
   * If true, clicking the same element is allowed if URL changed between clicks.
   * Useful for pagination (clicking "Next" multiple times).
   * Default: true
   */
  allowIfUrlChanged: boolean;

  /**
   * If true, don't count clicks on different elements as part of same loop.
   * Clicking element 1, then 2, then 1 again resets the counter.
   * Default: true
   */
  resetOnDifferentElement: boolean;
}

/**
 * Default loop detection for unconfigured action types.
 */
export interface DefaultLoopConfig {
  /**
   * Max consecutive identical actions before flagging as loop.
   * Default: 3
   */
  maxRepeats: number;
}

/**
 * Default configuration values.
 * These are sensible defaults that can be overridden.
 */
export const DEFAULT_LOOP_CONFIG: LoopDetectionConfig = {
  scroll: {
    maxIneffectiveRepeats: 2,   // Stop quickly when stuck at boundary
    maxEffectiveRepeats: -1,    // Unlimited effective scrolls
    minScrollDelta: 10,         // 10px minimum to count as effective
  },
  click: {
    maxSameElementRepeats: 3,   // Allow some retries on same element
    allowIfUrlChanged: true,    // Pagination pattern is common
    resetOnDifferentElement: true,
  },
  default: {
    maxRepeats: 3,              // Conservative default for other actions
  },
  includeDebugInfo: false,
};

// =============================================================================
// ACTION CONTEXT TYPES - Tracking Action Effectiveness
// =============================================================================

/**
 * Context captured during/after action execution.
 * Used to determine if an action had any effect.
 */
export interface ActionContext {
  /**
   * Type of the action for quick filtering.
   */
  actionType: BrowserAction['type'];

  /**
   * Scroll-specific context (only for scroll actions).
   */
  scroll?: ScrollContext;

  /**
   * Click-specific context (only for click actions).
   */
  click?: ClickContext;

  /**
   * URL before action execution.
   */
  urlBefore: string;

  /**
   * URL after action execution.
   */
  urlAfter: string;

  /**
   * Whether the URL changed.
   */
  urlChanged: boolean;

  /**
   * Timestamp when action was executed.
   */
  timestamp: number;
}

/**
 * Scroll-specific context for effectiveness tracking.
 */
export interface ScrollContext {
  /**
   * Scroll direction attempted.
   */
  direction: ScrollAction['direction'];

  /**
   * Scroll position before action.
   */
  positionBefore: { x: number; y: number };

  /**
   * Scroll position after action.
   */
  positionAfter: { x: number; y: number };

  /**
   * Actual pixels scrolled (computed).
   */
  delta: { x: number; y: number };

  /**
   * Whether the scroll had any effect (position changed).
   */
  wasEffective: boolean;
}

/**
 * Click-specific context for loop tracking.
 */
export interface ClickContext {
  /**
   * Element ID that was clicked (if element-based click).
   */
  elementId?: number;

  /**
   * Coordinates that were clicked (if coordinate-based click).
   */
  coordinates?: { x: number; y: number };

  /**
   * A stable identifier for the click target.
   * Used for comparing whether same target is being clicked.
   */
  targetKey: string;
}

// =============================================================================
// HISTORY ENTRY - What We Track Per Action
// =============================================================================

/**
 * Entry in the action history for loop detection.
 */
export interface ActionHistoryEntry {
  /**
   * The action that was executed.
   */
  action: BrowserAction;

  /**
   * Serialized string representation for basic comparison.
   */
  serialized: string;

  /**
   * Execution context with effectiveness data.
   */
  context: ActionContext;

  /**
   * Step number when this action occurred.
   */
  stepNumber: number;
}

// =============================================================================
// DETECTION RESULT
// =============================================================================

/**
 * Result of loop detection check.
 */
export interface LoopDetectionResult {
  /**
   * Whether a loop was detected.
   */
  isLoop: boolean;

  /**
   * Human-readable description of why it's a loop (or why not).
   */
  reason: string;

  /**
   * The repeated action (if loop detected).
   */
  repeatedAction?: string;

  /**
   * How many times the action was repeated.
   */
  repeatCount?: number;

  /**
   * Which strategy detected the loop.
   */
  detectionStrategy?: 'scroll_ineffective' | 'click_same_element' | 'default_repeat';

  /**
   * Debug information (if includeDebugInfo is true).
   */
  debug?: {
    historyLength: number;
    lastActions: string[];
    contexts: ActionContext[];
  };
}

// =============================================================================
// LOOP DETECTOR - The Main Interface
// =============================================================================

/**
 * Loop detector interface.
 */
export interface LoopDetector {
  /**
   * Check if the current action history indicates a loop.
   */
  detect(history: ActionHistoryEntry[]): LoopDetectionResult;

  /**
   * Get the current configuration.
   */
  getConfig(): LoopDetectionConfig;

  /**
   * Update configuration (useful for runtime tuning).
   */
  updateConfig(partial: Partial<LoopDetectionConfig>): void;
}

/**
 * Create a loop detector with the given configuration.
 */
export function createLoopDetector(
  config: Partial<LoopDetectionConfig> = {}
): LoopDetector {
  // Merge with defaults
  let currentConfig: LoopDetectionConfig = mergeConfig(DEFAULT_LOOP_CONFIG, config);

  return {
    detect(history: ActionHistoryEntry[]): LoopDetectionResult {
      if (history.length === 0) {
        return { isLoop: false, reason: 'Empty history' };
      }

      const lastEntry = history[history.length - 1];
      const actionType = lastEntry.action.type;

      // Route to appropriate strategy based on action type
      switch (actionType) {
        case 'scroll':
          return detectScrollLoop(history, currentConfig);
        case 'click':
          return detectClickLoop(history, currentConfig);
        case 'done':
          // 'done' actions are never loops
          return { isLoop: false, reason: 'Done action - not a loop' };
        default:
          return detectDefaultLoop(history, currentConfig);
      }
    },

    getConfig(): LoopDetectionConfig {
      return { ...currentConfig };
    },

    updateConfig(partial: Partial<LoopDetectionConfig>): void {
      currentConfig = mergeConfig(currentConfig, partial);
    },
  };
}

// =============================================================================
// DETECTION STRATEGIES
// =============================================================================

/**
 * Detect scroll loops using effectiveness tracking.
 */
function detectScrollLoop(
  history: ActionHistoryEntry[],
  config: LoopDetectionConfig
): LoopDetectionResult {
  const scrollConfig = config.scroll;

  // Get consecutive scroll actions in the same direction from the end
  const scrollSequence: ActionHistoryEntry[] = [];
  let currentDirection: string | null = null;

  for (let i = history.length - 1; i >= 0; i--) {
    const entry = history[i];
    if (entry.action.type !== 'scroll') break;

    const scrollAction = entry.action as ScrollAction;
    const direction = scrollAction.direction;

    if (currentDirection === null) {
      currentDirection = direction;
    } else if (direction !== currentDirection) {
      break; // Different direction, stop counting
    }

    scrollSequence.unshift(entry);
  }

  if (scrollSequence.length === 0) {
    return { isLoop: false, reason: 'No scroll sequence' };
  }

  // Count effective vs ineffective scrolls
  let ineffectiveCount = 0;
  let effectiveCount = 0;

  for (const entry of scrollSequence) {
    const scrollCtx = entry.context.scroll;
    if (!scrollCtx) {
      // No context - treat as potentially ineffective for safety
      ineffectiveCount++;
      continue;
    }

    // Check if scroll was effective (moved more than minScrollDelta)
    const deltaAbs = Math.abs(
      scrollCtx.direction === 'up' || scrollCtx.direction === 'down'
        ? scrollCtx.delta.y
        : scrollCtx.delta.x
    );

    if (deltaAbs >= scrollConfig.minScrollDelta) {
      effectiveCount++;
      // Reset ineffective count when we see an effective scroll
      // This prevents flagging: scroll(effective), scroll(ineffective), scroll(effective)
      ineffectiveCount = 0;
    } else {
      ineffectiveCount++;
    }
  }

  // Check for ineffective scroll loop
  if (ineffectiveCount > scrollConfig.maxIneffectiveRepeats) {
    const result: LoopDetectionResult = {
      isLoop: true,
      reason: `Scroll ${currentDirection} repeated ${ineffectiveCount} times with no effect (stuck at boundary)`,
      repeatedAction: `scroll:${currentDirection}`,
      repeatCount: ineffectiveCount,
      detectionStrategy: 'scroll_ineffective',
    };

    if (config.includeDebugInfo) {
      result.debug = {
        historyLength: history.length,
        lastActions: scrollSequence.map(e => e.serialized),
        contexts: scrollSequence.map(e => e.context),
      };
    }

    return result;
  }

  // Check for effective scroll limit (if configured)
  if (scrollConfig.maxEffectiveRepeats !== -1 && effectiveCount > scrollConfig.maxEffectiveRepeats) {
    return {
      isLoop: true,
      reason: `Scroll ${currentDirection} repeated ${effectiveCount} times (exceeded effective limit)`,
      repeatedAction: `scroll:${currentDirection}`,
      repeatCount: effectiveCount,
      detectionStrategy: 'scroll_ineffective',
    };
  }

  return {
    isLoop: false,
    reason: `Scroll sequence of ${scrollSequence.length} (${effectiveCount} effective, ${ineffectiveCount} ineffective) within limits`,
  };
}

/**
 * Detect click loops on the same element.
 */
function detectClickLoop(
  history: ActionHistoryEntry[],
  config: LoopDetectionConfig
): LoopDetectionResult {
  const clickConfig = config.click;

  // Get consecutive clicks from the end
  const clickSequence: ActionHistoryEntry[] = [];
  let currentTargetKey: string | null = null;

  for (let i = history.length - 1; i >= 0; i--) {
    const entry = history[i];
    if (entry.action.type !== 'click') break;

    const clickCtx = entry.context.click;
    const targetKey = clickCtx?.targetKey ?? entry.serialized;

    if (currentTargetKey === null) {
      currentTargetKey = targetKey;
    } else if (targetKey !== currentTargetKey) {
      if (clickConfig.resetOnDifferentElement) {
        break; // Different element clicked, stop counting
      }
    }

    clickSequence.unshift(entry);
  }

  if (clickSequence.length === 0) {
    return { isLoop: false, reason: 'No click sequence' };
  }

  // Filter to only clicks on the same target
  const sameTargetClicks = clickSequence.filter(entry => {
    const clickCtx = entry.context.click;
    const targetKey = clickCtx?.targetKey ?? entry.serialized;
    return targetKey === currentTargetKey;
  });

  // Check if URL changed between any of the clicks (pagination pattern)
  if (clickConfig.allowIfUrlChanged && sameTargetClicks.length > 1) {
    const urls = sameTargetClicks.map(e => e.context.urlAfter);
    const uniqueUrls = new Set(urls);
    if (uniqueUrls.size > 1) {
      return {
        isLoop: false,
        reason: `Clicking same element but URL changed between clicks (pagination pattern)`,
      };
    }
  }

  // Check for same-element click loop
  if (sameTargetClicks.length > clickConfig.maxSameElementRepeats) {
    const result: LoopDetectionResult = {
      isLoop: true,
      reason: `Clicked same element ${sameTargetClicks.length} times without URL change`,
      repeatedAction: currentTargetKey ?? 'click',
      repeatCount: sameTargetClicks.length,
      detectionStrategy: 'click_same_element',
    };

    if (config.includeDebugInfo) {
      result.debug = {
        historyLength: history.length,
        lastActions: sameTargetClicks.map(e => e.serialized),
        contexts: sameTargetClicks.map(e => e.context),
      };
    }

    return result;
  }

  return {
    isLoop: false,
    reason: `Click sequence of ${sameTargetClicks.length} within limits`,
  };
}

/**
 * Default loop detection for other action types.
 * Falls back to simple consecutive repeat counting.
 */
function detectDefaultLoop(
  history: ActionHistoryEntry[],
  config: LoopDetectionConfig
): LoopDetectionResult {
  const defaultConfig = config.default;

  // Count consecutive identical actions from the end
  const lastSerialized = history[history.length - 1].serialized;
  let repeatCount = 0;

  for (let i = history.length - 1; i >= 0; i--) {
    if (history[i].serialized === lastSerialized) {
      repeatCount++;
    } else {
      break;
    }
  }

  if (repeatCount > defaultConfig.maxRepeats) {
    const result: LoopDetectionResult = {
      isLoop: true,
      reason: `Action "${lastSerialized}" repeated ${repeatCount} times`,
      repeatedAction: lastSerialized,
      repeatCount,
      detectionStrategy: 'default_repeat',
    };

    if (config.includeDebugInfo) {
      result.debug = {
        historyLength: history.length,
        lastActions: history.slice(-repeatCount).map(e => e.serialized),
        contexts: history.slice(-repeatCount).map(e => e.context),
      };
    }

    return result;
  }

  return {
    isLoop: false,
    reason: `Action repeated ${repeatCount} times, within limit of ${defaultConfig.maxRepeats}`,
  };
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

/**
 * Deep merge configuration objects.
 */
function mergeConfig(
  base: LoopDetectionConfig,
  override: Partial<LoopDetectionConfig>
): LoopDetectionConfig {
  return {
    scroll: { ...base.scroll, ...override.scroll },
    click: { ...base.click, ...override.click },
    default: { ...base.default, ...override.default },
    includeDebugInfo: override.includeDebugInfo ?? base.includeDebugInfo,
  };
}

/**
 * Serialize an action for comparison purposes.
 * Moved from agent.ts for reuse.
 */
export function serializeAction(action: BrowserAction): string {
  switch (action.type) {
    case 'click':
      if ('elementId' in action && action.elementId !== undefined) {
        return `click:element:${action.elementId}`;
      }
      if ('coordinates' in action && action.coordinates) {
        return `click:coords:${action.coordinates.x},${action.coordinates.y}`;
      }
      return 'click:unknown';
    case 'type':
      if ('elementId' in action && action.elementId !== undefined) {
        return `type:element:${action.elementId}:${action.text}`;
      }
      return `type:focused:${action.text}`;
    case 'scroll':
      return `scroll:${action.direction}`;
    case 'navigate':
      return `navigate:${action.url}`;
    case 'hover':
      if ('elementId' in action && action.elementId !== undefined) {
        return `hover:element:${action.elementId}`;
      }
      if ('coordinates' in action && action.coordinates) {
        return `hover:coords:${action.coordinates.x},${action.coordinates.y}`;
      }
      return 'hover:unknown';
    case 'select':
      return `select:${action.elementId}:${action.value}`;
    case 'wait':
      return `wait:${action.ms ?? 'default'}`;
    case 'keypress':
      return `keypress:${action.key}`;
    case 'done':
      return `done:${action.success}`;
    default:
      return `unknown:${JSON.stringify(action)}`;
  }
}

/**
 * Create an action context with default values.
 * Use this as a builder pattern for creating contexts.
 */
export function createActionContext(
  action: BrowserAction,
  urlBefore: string,
  urlAfter: string
): ActionContext {
  const context: ActionContext = {
    actionType: action.type,
    urlBefore,
    urlAfter,
    urlChanged: urlBefore !== urlAfter,
    timestamp: Date.now(),
  };

  // Add click context if applicable
  if (action.type === 'click') {
    const clickAction = action as ClickAction;
    context.click = {
      elementId: clickAction.elementId,
      coordinates: clickAction.coordinates,
      targetKey: clickAction.elementId !== undefined
        ? `element:${clickAction.elementId}`
        : clickAction.coordinates
          ? `coords:${clickAction.coordinates.x},${clickAction.coordinates.y}`
          : 'unknown',
    };
  }

  return context;
}

/**
 * Create scroll context from before/after positions.
 */
export function createScrollContext(
  direction: ScrollAction['direction'],
  positionBefore: { x: number; y: number },
  positionAfter: { x: number; y: number },
  minDelta: number = 10
): ScrollContext {
  const delta = {
    x: positionAfter.x - positionBefore.x,
    y: positionAfter.y - positionBefore.y,
  };

  // Determine if scroll was effective based on direction
  let wasEffective: boolean;
  switch (direction) {
    case 'up':
      wasEffective = delta.y < -minDelta;
      break;
    case 'down':
      wasEffective = delta.y > minDelta;
      break;
    case 'left':
      wasEffective = delta.x < -minDelta;
      break;
    case 'right':
      wasEffective = delta.x > minDelta;
      break;
    default:
      wasEffective = Math.abs(delta.x) > minDelta || Math.abs(delta.y) > minDelta;
  }

  return {
    direction,
    positionBefore,
    positionAfter,
    delta,
    wasEffective,
  };
}

/**
 * Create a history entry from action and context.
 */
export function createHistoryEntry(
  action: BrowserAction,
  context: ActionContext,
  stepNumber: number
): ActionHistoryEntry {
  return {
    action,
    serialized: serializeAction(action),
    context,
    stepNumber,
  };
}
