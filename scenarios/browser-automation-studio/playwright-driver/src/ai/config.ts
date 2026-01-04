/**
 * AI Module Configuration
 *
 * CONTROL SURFACE: Tunable levers for AI-powered browser navigation.
 *
 * This module consolidates configuration for:
 * - Vision Agent (observe-decide-act loop)
 * - Screenshot Capture (for AI analysis)
 * - Action Executor (browser action execution)
 * - Callback Emitter (step event delivery)
 *
 * ## Configuration Philosophy:
 * - All values have sane defaults that work for common use cases
 * - Each lever controls a meaningful tradeoff
 * - Levers are grouped by domain for discoverability
 *
 * ## Adding New Levers:
 * 1. Add to the appropriate interface
 * 2. Add a default value in DEFAULTS
 * 3. Document the tradeoff in comments
 * 4. Update merge function if nested
 */

// =============================================================================
// VISION AGENT CONFIGURATION
// =============================================================================

/**
 * Vision Agent tunable settings.
 *
 * These control the observe-decide-act loop behavior.
 */
export interface VisionAgentSettings {
  /**
   * Maximum navigation steps before giving up.
   * Trade-off: Higher = more attempts to achieve goal, more token cost.
   * Default: 20
   */
  maxSteps: number;

  /**
   * Delay between steps in ms (for rate limiting).
   * Trade-off: Higher = slower navigation, less API pressure.
   * Default: 0 (no artificial delay)
   */
  stepDelayMs: number;

  /**
   * Maximum conversation history messages to retain.
   * Trade-off: Higher = more context for AI, more token cost per request.
   * Default: 20
   */
  maxHistoryMessages: number;

  /**
   * Maximum element labels to include in prompts.
   * Trade-off: Higher = AI sees more elements, more token cost.
   * Default: 50
   */
  maxElementLabels: number;

  /**
   * Delay after action execution to let page settle (ms).
   * Trade-off: Higher = more stable screenshots, slower navigation.
   * Default: 100
   */
  postActionSettleMs: number;
}

// =============================================================================
// SCREENSHOT CAPTURE CONFIGURATION
// =============================================================================

/**
 * Screenshot capture settings for AI analysis.
 */
export interface AIScreenshotSettings {
  /**
   * JPEG quality for screenshots (1-100).
   * Trade-off: Higher = better image quality, larger payloads.
   * Default: 80
   */
  quality: number;

  /**
   * Whether to capture full page screenshots.
   * Trade-off: True = see entire page, may cause viewport issues.
   * Default: false
   */
  fullPage: boolean;

  /**
   * Maximum screenshot size in bytes.
   * Trade-off: Higher = larger images allowed, more memory/bandwidth.
   * Default: 5MB
   */
  maxSizeBytes: number;
}

// =============================================================================
// ACTION EXECUTOR CONFIGURATION
// =============================================================================

/**
 * Action executor settings for browser actions.
 */
export interface ActionExecutorSettings {
  /**
   * Default timeout for action execution (ms).
   * Trade-off: Higher = more tolerant of slow pages, longer waits on failure.
   * Default: 30000 (30s)
   */
  defaultTimeoutMs: number;

  /**
   * Default wait duration for explicit wait actions (ms).
   * Trade-off: Higher = longer pauses, more time for dynamic content.
   * Default: 1000 (1s)
   */
  defaultWaitMs: number;
}

// =============================================================================
// CALLBACK EMITTER CONFIGURATION
// =============================================================================

/**
 * Callback emitter settings for step event delivery.
 */
export interface CallbackEmitterSettings {
  /**
   * Timeout for callback HTTP requests (ms).
   * Trade-off: Higher = more tolerant of slow servers, longer waits on failure.
   * Default: 5000 (5s)
   */
  requestTimeoutMs: number;

  /**
   * Maximum retry attempts for failed callbacks.
   * Trade-off: Higher = more resilient delivery, more latency on failures.
   * Default: 3
   */
  maxRetries: number;

  /**
   * Base delay between retries (ms, with exponential backoff).
   * Trade-off: Higher = less server pressure, slower retry recovery.
   * Default: 500
   */
  retryDelayMs: number;
}

// =============================================================================
// ELEMENT EXTRACTION CONFIGURATION
// =============================================================================

/**
 * Element extraction settings for interactive element detection.
 */
export interface ElementExtractionSettings {
  /**
   * Maximum elements to extract per page.
   * Trade-off: Higher = more complete element list, more processing time.
   * Default: 50
   */
  maxElements: number;

  /**
   * Maximum text length per element (characters).
   * Trade-off: Higher = more context per element, more token cost.
   * Default: 100
   */
  maxTextLength: number;
}

// =============================================================================
// COMBINED AI CONFIGURATION
// =============================================================================

/**
 * Complete AI module configuration.
 */
export interface AIConfig {
  visionAgent: VisionAgentSettings;
  screenshot: AIScreenshotSettings;
  actionExecutor: ActionExecutorSettings;
  callbackEmitter: CallbackEmitterSettings;
  elementExtraction: ElementExtractionSettings;
}

/**
 * Default values for all AI configuration.
 */
export const AI_DEFAULTS: AIConfig = {
  visionAgent: {
    maxSteps: 20,
    stepDelayMs: 0,
    maxHistoryMessages: 20,
    maxElementLabels: 50,
    postActionSettleMs: 100,
  },
  screenshot: {
    quality: 80,
    fullPage: false,
    maxSizeBytes: 5 * 1024 * 1024, // 5MB
  },
  actionExecutor: {
    defaultTimeoutMs: 30_000,
    defaultWaitMs: 1000,
  },
  callbackEmitter: {
    requestTimeoutMs: 5000,
    maxRetries: 3,
    retryDelayMs: 500,
  },
  elementExtraction: {
    maxElements: 50,
    maxTextLength: 100,
  },
};

// =============================================================================
// ENVIRONMENT VARIABLE MAPPING
// =============================================================================

/**
 * Load AI configuration from environment variables.
 *
 * Environment variables (all optional, with defaults):
 * - AI_MAX_STEPS: Vision agent max steps
 * - AI_STEP_DELAY_MS: Delay between steps
 * - AI_MAX_HISTORY_MESSAGES: Conversation history limit
 * - AI_MAX_ELEMENT_LABELS: Element labels limit
 * - AI_SCREENSHOT_QUALITY: Screenshot JPEG quality
 * - AI_ACTION_TIMEOUT_MS: Action execution timeout
 * - AI_CALLBACK_TIMEOUT_MS: Callback request timeout
 * - AI_CALLBACK_MAX_RETRIES: Callback retry attempts
 */
export function loadAIConfig(): AIConfig {
  return {
    visionAgent: {
      maxSteps: parseEnvInt('AI_MAX_STEPS', AI_DEFAULTS.visionAgent.maxSteps),
      stepDelayMs: parseEnvInt('AI_STEP_DELAY_MS', AI_DEFAULTS.visionAgent.stepDelayMs),
      maxHistoryMessages: parseEnvInt('AI_MAX_HISTORY_MESSAGES', AI_DEFAULTS.visionAgent.maxHistoryMessages),
      maxElementLabels: parseEnvInt('AI_MAX_ELEMENT_LABELS', AI_DEFAULTS.visionAgent.maxElementLabels),
      postActionSettleMs: parseEnvInt('AI_POST_ACTION_SETTLE_MS', AI_DEFAULTS.visionAgent.postActionSettleMs),
    },
    screenshot: {
      quality: parseEnvInt('AI_SCREENSHOT_QUALITY', AI_DEFAULTS.screenshot.quality),
      fullPage: process.env.AI_SCREENSHOT_FULL_PAGE === 'true',
      maxSizeBytes: parseEnvInt('AI_SCREENSHOT_MAX_SIZE', AI_DEFAULTS.screenshot.maxSizeBytes),
    },
    actionExecutor: {
      defaultTimeoutMs: parseEnvInt('AI_ACTION_TIMEOUT_MS', AI_DEFAULTS.actionExecutor.defaultTimeoutMs),
      defaultWaitMs: parseEnvInt('AI_DEFAULT_WAIT_MS', AI_DEFAULTS.actionExecutor.defaultWaitMs),
    },
    callbackEmitter: {
      requestTimeoutMs: parseEnvInt('AI_CALLBACK_TIMEOUT_MS', AI_DEFAULTS.callbackEmitter.requestTimeoutMs),
      maxRetries: parseEnvInt('AI_CALLBACK_MAX_RETRIES', AI_DEFAULTS.callbackEmitter.maxRetries),
      retryDelayMs: parseEnvInt('AI_CALLBACK_RETRY_DELAY_MS', AI_DEFAULTS.callbackEmitter.retryDelayMs),
    },
    elementExtraction: {
      maxElements: parseEnvInt('AI_MAX_ELEMENTS', AI_DEFAULTS.elementExtraction.maxElements),
      maxTextLength: parseEnvInt('AI_MAX_TEXT_LENGTH', AI_DEFAULTS.elementExtraction.maxTextLength),
    },
  };
}

/**
 * Merge partial config with defaults.
 */
export function mergeAIConfig(partial: Partial<AIConfig>): AIConfig {
  return {
    visionAgent: { ...AI_DEFAULTS.visionAgent, ...partial.visionAgent },
    screenshot: { ...AI_DEFAULTS.screenshot, ...partial.screenshot },
    actionExecutor: { ...AI_DEFAULTS.actionExecutor, ...partial.actionExecutor },
    callbackEmitter: { ...AI_DEFAULTS.callbackEmitter, ...partial.callbackEmitter },
    elementExtraction: { ...AI_DEFAULTS.elementExtraction, ...partial.elementExtraction },
  };
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

function parseEnvInt(envName: string, defaultValue: number): number {
  const value = process.env[envName];
  if (!value) return defaultValue;

  const parsed = parseInt(value, 10);
  if (Number.isNaN(parsed)) {
    console.warn(`Invalid AI config value for ${envName}: "${value}", using default: ${defaultValue}`);
    return defaultValue;
  }

  return parsed;
}
