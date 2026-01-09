import { z } from 'zod';

/**
 * Playwright Driver Configuration Schema
 *
 * CONTROL SURFACE: This file defines the tunable levers for the Playwright Driver.
 *
 * **IMPORTANT**: Most users only need 5-7 options. See CONFIG-TIERS.md for guidance
 * on which options to prioritize.
 *
 * ## Configuration Tiers (see CONFIG-TIERS.md):
 * - **Tier 1 (Essential)**: port, maxConcurrent, defaultTimeoutMs, logLevel, metricsEnabled
 * - **Tier 2 (Advanced)**: Fine-grained timeouts, telemetry controls, session tuning
 * - **Tier 3 (Internal)**: Recording, frame streaming, performance debug
 *
 * ## Configuration Groups:
 * - **server**: HTTP server settings (port, host, request limits)
 * - **browser**: Browser launch options (headless, executable, args)
 * - **session**: Session lifecycle (concurrency, timeouts, pooling)
 * - **execution**: Instruction execution timeouts (the main performance/reliability tradeoff)
 * - **recording**: Record mode settings (buffer size, selector confidence)
 * - **telemetry**: Observability data collection (screenshots, DOM, network)
 * - **logging**: Log output configuration
 * - **metrics**: Prometheus metrics endpoint
 *
 * ## Adding New Levers:
 * 1. Add to appropriate group in ConfigSchema
 * 2. Add environment variable parsing in loadConfig()
 * 3. Document the tradeoff in comments
 * 4. Set a sane default that works for common usage
 * 5. Update CONFIG-TIERS.md if user-facing
 * 6. Add tier metadata to CONFIG_TIER_METADATA below
 */

// =============================================================================
// CONFIGURATION TIER METADATA
// =============================================================================

/**
 * Configuration tier levels.
 * - ESSENTIAL: Options most users will need to consider
 * - ADVANCED: Fine-tuning options for specific needs
 * - INTERNAL: Rarely changed, for developers and debugging
 */
export enum ConfigTier {
  ESSENTIAL = 1,
  ADVANCED = 2,
  INTERNAL = 3,
}

/**
 * Data types for configuration options.
 * Used for UI rendering and validation.
 */
export type ConfigDataType = 'boolean' | 'integer' | 'float' | 'string' | 'enum';

/**
 * Full metadata for a configuration option.
 */
export interface ConfigOptionMetadata {
  tier: ConfigTier;
  defaultValue: unknown;
  description: string;
  /** Data type for UI rendering and validation */
  dataType: ConfigDataType;
  /** For numeric types: minimum allowed value */
  min?: number;
  /** For numeric types: maximum allowed value */
  max?: number;
  /** For enum types: allowed values */
  enumValues?: string[];
  /**
   * Whether this option can be changed at runtime without restart.
   * Options marked as editable will have a setter in the runtime config.
   */
  editable?: boolean;
}

/**
 * Metadata for each configuration option.
 * Maps environment variable names to their tier, default value, type, and editability.
 * Used for warning when non-essential options are modified and for UI display.
 */
export const CONFIG_TIER_METADATA: Record<string, ConfigOptionMetadata> = {
  // === Tier 1: Essential ===
  PLAYWRIGHT_DRIVER_PORT: { tier: ConfigTier.ESSENTIAL, defaultValue: 39400, description: 'HTTP server port', dataType: 'integer', min: 1, max: 65535, editable: false },
  MAX_SESSIONS: { tier: ConfigTier.ESSENTIAL, defaultValue: 10, description: 'Maximum parallel browser sessions', dataType: 'integer', min: 1, max: 100, editable: false },
  EXECUTION_DEFAULT_TIMEOUT_MS: { tier: ConfigTier.ESSENTIAL, defaultValue: 30000, description: 'Default timeout for operations (click, type, etc.)', dataType: 'integer', min: 1000, max: 300000, editable: true },
  LOG_LEVEL: { tier: ConfigTier.ESSENTIAL, defaultValue: 'info', description: 'Log verbosity level', dataType: 'enum', enumValues: ['debug', 'info', 'warn', 'error'], editable: true },
  METRICS_ENABLED: { tier: ConfigTier.ESSENTIAL, defaultValue: true, description: 'Prometheus metrics endpoint', dataType: 'boolean', editable: false },

  // === Tier 2: Advanced ===
  EXECUTION_NAVIGATION_TIMEOUT_MS: { tier: ConfigTier.ADVANCED, defaultValue: 45000, description: 'Navigation timeout (goto, reload)', dataType: 'integer', min: 5000, max: 300000, editable: true },
  EXECUTION_WAIT_TIMEOUT_MS: { tier: ConfigTier.ADVANCED, defaultValue: 30000, description: 'Wait timeout (waitForSelector)', dataType: 'integer', min: 1000, max: 300000, editable: true },
  EXECUTION_ASSERTION_TIMEOUT_MS: { tier: ConfigTier.ADVANCED, defaultValue: 15000, description: 'Assertion timeout', dataType: 'integer', min: 1000, max: 120000, editable: true },
  EXECUTION_REPLAY_TIMEOUT_MS: { tier: ConfigTier.ADVANCED, defaultValue: 10000, description: 'Replay action timeout', dataType: 'integer', min: 1000, max: 120000, editable: true },
  SCREENSHOT_ENABLED: { tier: ConfigTier.ADVANCED, defaultValue: true, description: 'Screenshot collection', dataType: 'boolean', editable: true },
  SCREENSHOT_QUALITY: { tier: ConfigTier.ADVANCED, defaultValue: 80, description: 'Screenshot JPEG quality', dataType: 'integer', min: 1, max: 100, editable: true },
  DOM_ENABLED: { tier: ConfigTier.ADVANCED, defaultValue: true, description: 'DOM snapshot collection', dataType: 'boolean', editable: true },
  CONSOLE_ENABLED: { tier: ConfigTier.ADVANCED, defaultValue: true, description: 'Console log collection', dataType: 'boolean', editable: true },
  NETWORK_ENABLED: { tier: ConfigTier.ADVANCED, defaultValue: true, description: 'Network event collection', dataType: 'boolean', editable: true },
  SESSION_POOL_SIZE: { tier: ConfigTier.ADVANCED, defaultValue: 5, description: 'Pre-warmed session pool', dataType: 'integer', min: 1, max: 50, editable: false },
  SESSION_IDLE_TIMEOUT_MS: { tier: ConfigTier.ADVANCED, defaultValue: 300000, description: 'Session idle TTL (5 min default)', dataType: 'integer', min: 10000, max: 3600000, editable: true },

  // === Tier 2: AI Navigation (Advanced) ===
  AI_MAX_STEPS: { tier: ConfigTier.ADVANCED, defaultValue: 20, description: 'Vision agent max steps per navigation', dataType: 'integer', min: 1, max: 100, editable: true },
  AI_SCREENSHOT_QUALITY: { tier: ConfigTier.ADVANCED, defaultValue: 80, description: 'AI screenshot JPEG quality', dataType: 'integer', min: 1, max: 100, editable: true },
  AI_ACTION_TIMEOUT_MS: { tier: ConfigTier.ADVANCED, defaultValue: 30000, description: 'AI action execution timeout', dataType: 'integer', min: 1000, max: 120000, editable: true },

  // === Tier 3: Internal ===
  PLAYWRIGHT_DRIVER_HOST: { tier: ConfigTier.INTERNAL, defaultValue: '127.0.0.1', description: 'HTTP server host', dataType: 'string', editable: false },
  REQUEST_TIMEOUT_MS: { tier: ConfigTier.INTERNAL, defaultValue: 300000, description: 'Request timeout', dataType: 'integer', min: 1000, max: 600000, editable: true },
  MAX_REQUEST_SIZE: { tier: ConfigTier.INTERNAL, defaultValue: 5242880, description: 'Max request body size (bytes)', dataType: 'integer', min: 1024, max: 52428800, editable: false },
  HEADLESS: { tier: ConfigTier.INTERNAL, defaultValue: false, description: 'Use headless_shell binary (false = regular Chromium with --headless=new)', dataType: 'boolean', editable: false },
  BROWSER_EXECUTABLE_PATH: { tier: ConfigTier.INTERNAL, defaultValue: undefined, description: 'Custom browser path', dataType: 'string', editable: false },
  BROWSER_ARGS: { tier: ConfigTier.INTERNAL, defaultValue: '', description: 'Extra browser arguments (comma-separated)', dataType: 'string', editable: false },
  IGNORE_HTTPS_ERRORS: { tier: ConfigTier.INTERNAL, defaultValue: false, description: 'Ignore HTTPS certificate errors', dataType: 'boolean', editable: false },
  RECORDING_MAX_BUFFER_SIZE: { tier: ConfigTier.INTERNAL, defaultValue: 10000, description: 'Recording buffer size (actions)', dataType: 'integer', min: 100, max: 100000, editable: true },
  RECORDING_MIN_SELECTOR_CONFIDENCE: { tier: ConfigTier.INTERNAL, defaultValue: 0.3, description: 'Selector confidence threshold (0-1)', dataType: 'float', min: 0, max: 1, editable: true },
  RECORDING_INPUT_DEBOUNCE_MS: { tier: ConfigTier.INTERNAL, defaultValue: 500, description: 'Input debounce timing', dataType: 'integer', min: 50, max: 2000, editable: true },
  RECORDING_SCROLL_DEBOUNCE_MS: { tier: ConfigTier.INTERNAL, defaultValue: 150, description: 'Scroll debounce timing', dataType: 'integer', min: 50, max: 1000, editable: true },
  RECORDING_MAX_CSS_DEPTH: { tier: ConfigTier.INTERNAL, defaultValue: 5, description: 'CSS path max depth', dataType: 'integer', min: 2, max: 10, editable: true },
  RECORDING_INCLUDE_XPATH: { tier: ConfigTier.INTERNAL, defaultValue: true, description: 'Include XPath selectors', dataType: 'boolean', editable: true },
  RECORDING_DEFAULT_SWIPE_DISTANCE: { tier: ConfigTier.INTERNAL, defaultValue: 300, description: 'Default swipe distance (px)', dataType: 'integer', min: 50, max: 1000, editable: true },
  FRAME_STREAMING_USE_SCREENCAST: { tier: ConfigTier.INTERNAL, defaultValue: true, description: 'Use CDP screencast for streaming', dataType: 'boolean', editable: false },
  FRAME_STREAMING_FALLBACK: { tier: ConfigTier.INTERNAL, defaultValue: true, description: 'Fall back to polling if screencast fails', dataType: 'boolean', editable: false },
  FRAME_STREAMING_CDP_ACK_TIMEOUT_MS: { tier: ConfigTier.INTERNAL, defaultValue: 1000, description: 'CDP frame ACK timeout', dataType: 'integer', min: 100, max: 10000, editable: true },
  FRAME_STREAMING_CDP_MAX_ACK_FAILURES: { tier: ConfigTier.INTERNAL, defaultValue: 5, description: 'Max CDP ACK failures before error', dataType: 'integer', min: 1, max: 100, editable: true },
  FRAME_STREAMING_CDP_FRAME_LOG_INTERVAL: { tier: ConfigTier.INTERNAL, defaultValue: 60, description: 'Log stats every N frames (0 = disable)', dataType: 'integer', min: 0, max: 1000, editable: true },
  FRAME_STREAMING_CDP_PAGE_CHECK_INTERVAL_MS: { tier: ConfigTier.INTERNAL, defaultValue: 100, description: 'Multi-tab check interval', dataType: 'integer', min: 50, max: 5000, editable: true },
  PLAYWRIGHT_DRIVER_PERF_ENABLED: { tier: ConfigTier.INTERNAL, defaultValue: false, description: 'Performance debug mode', dataType: 'boolean', editable: true },
  PLAYWRIGHT_DRIVER_PERF_INCLUDE_HEADERS: { tier: ConfigTier.INTERNAL, defaultValue: true, description: 'Include timing headers', dataType: 'boolean', editable: true },
  PLAYWRIGHT_DRIVER_PERF_LOG_INTERVAL: { tier: ConfigTier.INTERNAL, defaultValue: 60, description: 'Performance log interval (frames)', dataType: 'integer', min: 0, max: 1000, editable: true },
  PLAYWRIGHT_DRIVER_PERF_BUFFER_SIZE: { tier: ConfigTier.INTERNAL, defaultValue: 100, description: 'Performance buffer size', dataType: 'integer', min: 1, max: 1000, editable: true },
  METRICS_PORT: { tier: ConfigTier.INTERNAL, defaultValue: 9090, description: 'Metrics server port', dataType: 'integer', min: 1, max: 65535, editable: false },
  CLEANUP_INTERVAL_MS: { tier: ConfigTier.INTERNAL, defaultValue: 60000, description: 'Session cleanup interval', dataType: 'integer', min: 5000, max: 600000, editable: true },
  SCREENSHOT_FULL_PAGE: { tier: ConfigTier.INTERNAL, defaultValue: false, description: 'Full page screenshots (may cause viewport oscillation)', dataType: 'boolean', editable: true },
  RECORDING_DIAGNOSTICS_ENABLED: { tier: ConfigTier.INTERNAL, defaultValue: false, description: 'Verbose recording diagnostics (injection, event flow)', dataType: 'boolean', editable: true },
  SCREENSHOT_MAX_SIZE: { tier: ConfigTier.INTERNAL, defaultValue: 512000, description: 'Max screenshot size (bytes)', dataType: 'integer', min: 1024, max: 10485760, editable: true },
  DOM_MAX_SIZE: { tier: ConfigTier.INTERNAL, defaultValue: 524288, description: 'Max DOM snapshot size (bytes)', dataType: 'integer', min: 1024, max: 10485760, editable: true },
  CONSOLE_MAX_ENTRIES: { tier: ConfigTier.INTERNAL, defaultValue: 100, description: 'Max console entries', dataType: 'integer', min: 1, max: 10000, editable: true },
  NETWORK_MAX_EVENTS: { tier: ConfigTier.INTERNAL, defaultValue: 200, description: 'Max network events', dataType: 'integer', min: 1, max: 10000, editable: true },
  HAR_ENABLED: { tier: ConfigTier.INTERNAL, defaultValue: false, description: 'HAR recording', dataType: 'boolean', editable: true },
  TRACING_ENABLED: { tier: ConfigTier.INTERNAL, defaultValue: false, description: 'Playwright tracing', dataType: 'boolean', editable: true },
  LOG_FORMAT: { tier: ConfigTier.INTERNAL, defaultValue: 'json', description: 'Log format', dataType: 'enum', enumValues: ['json', 'text'], editable: true },

  // === Tier 3: AI Navigation Internal ===
  AI_STEP_DELAY_MS: { tier: ConfigTier.INTERNAL, defaultValue: 0, description: 'Delay between AI steps', dataType: 'integer', min: 0, max: 10000, editable: true },
  AI_MAX_HISTORY_MESSAGES: { tier: ConfigTier.INTERNAL, defaultValue: 20, description: 'AI conversation history limit', dataType: 'integer', min: 1, max: 100, editable: true },
  AI_MAX_ELEMENT_LABELS: { tier: ConfigTier.INTERNAL, defaultValue: 50, description: 'Max elements in AI context', dataType: 'integer', min: 1, max: 500, editable: true },
  AI_POST_ACTION_SETTLE_MS: { tier: ConfigTier.INTERNAL, defaultValue: 100, description: 'Delay after AI action', dataType: 'integer', min: 0, max: 5000, editable: true },
  AI_CALLBACK_TIMEOUT_MS: { tier: ConfigTier.INTERNAL, defaultValue: 5000, description: 'AI callback request timeout', dataType: 'integer', min: 1000, max: 30000, editable: true },
  AI_CALLBACK_MAX_RETRIES: { tier: ConfigTier.INTERNAL, defaultValue: 3, description: 'AI callback retry attempts', dataType: 'integer', min: 0, max: 10, editable: true },
  AI_CALLBACK_RETRY_DELAY_MS: { tier: ConfigTier.INTERNAL, defaultValue: 500, description: 'AI callback retry delay', dataType: 'integer', min: 100, max: 5000, editable: true },
  AI_MAX_ELEMENTS: { tier: ConfigTier.INTERNAL, defaultValue: 50, description: 'Max elements to extract per page', dataType: 'integer', min: 1, max: 500, editable: true },
  AI_MAX_TEXT_LENGTH: { tier: ConfigTier.INTERNAL, defaultValue: 100, description: 'Max text length per element', dataType: 'integer', min: 10, max: 1000, editable: true },

  // === History Callback ===
  HISTORY_CALLBACK_URL: { tier: ConfigTier.INTERNAL, defaultValue: '', description: 'URL to POST history entries when navigation occurs', dataType: 'string', editable: false },
  HISTORY_THUMBNAIL_ENABLED: { tier: ConfigTier.INTERNAL, defaultValue: true, description: 'Capture thumbnails for history entries', dataType: 'boolean', editable: true },
  HISTORY_THUMBNAIL_QUALITY: { tier: ConfigTier.INTERNAL, defaultValue: 60, description: 'JPEG quality for history thumbnails', dataType: 'integer', min: 10, max: 100, editable: true },
};

const ConfigSchema = z.object({
  server: z.object({
    port: z.number().min(1).max(65535).default(39400),
    host: z.string().default('127.0.0.1'),
    requestTimeout: z.number().min(1000).max(600000).default(300000), // 5 minutes - playwright operations can be slow
    maxRequestSize: z.number().min(1024).max(50 * 1024 * 1024).default(5 * 1024 * 1024),
  }),
  browser: z.object({
    /**
     * Headless mode configuration.
     *
     * IMPORTANT: We use headless: false + --headless=new arg to get the "new headless" mode.
     * This makes Playwright use the regular Chromium binary (not headless_shell) with
     * the --headless=new flag, which has full service worker support.
     *
     * - headless: true  → Uses chromium_headless_shell (broken SW, causes Google loops)
     * - headless: false + --headless=new → Uses regular Chromium in new headless mode (CORRECT)
     */
    headless: z.boolean().default(false),
    executablePath: z.string().optional(),
    /**
     * Additional browser arguments.
     *
     * Default includes --headless=new for new headless mode with full SW support.
     */
    args: z.array(z.string()).default(['--headless=new']),
    ignoreHTTPSErrors: z.boolean().default(false),
  }),
  session: z.object({
    maxConcurrent: z.number().min(1).max(100).default(10),
    idleTimeoutMs: z.number().min(10000).max(3600000).default(300000),
    poolSize: z.number().min(1).max(50).default(5),
    cleanupIntervalMs: z.number().min(5000).max(600000).default(60000),
  }),
  /**
   * Execution Timeouts
   *
   * These control the reliability vs speed tradeoff for instruction execution.
   * Higher values = more tolerant of slow pages, but longer waits on failures.
   * Lower values = faster failure detection, but may fail on slow networks/pages.
   *
   * Recommended tuning:
   * - Fast local testing: reduce timeouts by 50%
   * - Flaky/slow sites: increase timeouts by 50-100%
   * - CI environments: use defaults or slightly higher
   */
  execution: z.object({
    /** Default timeout for general operations (click, type, etc.) - ms */
    defaultTimeoutMs: z.number().min(1000).max(300000).default(30000),
    /** Navigation timeout (goto, reload) - typically longer due to network - ms */
    navigationTimeoutMs: z.number().min(5000).max(300000).default(45000),
    /** Wait timeout (waitForSelector, waitForTimeout) - ms */
    waitTimeoutMs: z.number().min(1000).max(300000).default(30000),
    /** Assertion timeout - typically shorter for faster feedback - ms */
    assertionTimeoutMs: z.number().min(1000).max(120000).default(15000),
    /** Replay action timeout during recording preview - ms */
    replayActionTimeoutMs: z.number().min(1000).max(120000).default(10000),
  }),
  /**
   * Recording Configuration
   *
   * Controls how Record Mode captures and processes user actions.
   *
   * Buffer size tradeoff: larger = more actions stored, higher memory usage.
   * Selector confidence: higher = stricter selector selection, fewer candidates.
   */
  recording: z.object({
    /** Maximum actions to buffer in memory per session (FIFO eviction when full) */
    maxBufferSize: z.number().min(100).max(100000).default(10000),
    /** Minimum confidence score (0-1) for selector candidates to be included */
    minSelectorConfidence: z.number().min(0).max(1).default(0.3),
    /** Default swipe gesture distance in pixels */
    defaultSwipeDistance: z.number().min(50).max(1000).default(300),
    /**
     * Enable verbose recording diagnostics.
     * When true, logs detailed information about:
     * - Script injection (success/failure, injection method)
     * - Event flow (events received, parsed, converted)
     * - Context isolation debugging
     *
     * Controlled by RECORDING_DIAGNOSTICS_ENABLED environment variable.
     */
    diagnosticsEnabled: z.boolean().default(false),
    /**
     * Debounce timings for event capture (ms).
     * Lower = more responsive, more events. Higher = more batching, fewer events.
     */
    debounce: z.object({
      /** Input event debounce - batches keystrokes into single type actions */
      inputMs: z.number().min(50).max(2000).default(500),
      /** Scroll event debounce - reduces scroll event noise */
      scrollMs: z.number().min(50).max(1000).default(150),
    }),
    /**
     * Selector Generation Options
     * Controls how selectors are generated for recorded elements.
     */
    selector: z.object({
      /** Maximum CSS path depth for traversal (lower = shorter selectors, may be less unique) */
      maxCssDepth: z.number().min(2).max(10).default(5),
      /** Whether to include XPath as a fallback selector strategy */
      includeXPath: z.boolean().default(true),
    }),
  }),
  telemetry: z.object({
    screenshot: z.object({
      enabled: z.boolean().default(true),
      fullPage: z.boolean().default(false), // Default false to prevent viewport oscillation during execution
      quality: z.number().min(1).max(100).default(80),
      maxSizeBytes: z.number().min(1024).max(10 * 1024 * 1024).default(512000),
    }),
    dom: z.object({
      enabled: z.boolean().default(true),
      maxSizeBytes: z.number().min(1024).max(10 * 1024 * 1024).default(524288),
    }),
    console: z.object({
      enabled: z.boolean().default(true),
      maxEntries: z.number().min(1).max(10000).default(100),
    }),
    network: z.object({
      enabled: z.boolean().default(true),
      maxEvents: z.number().min(1).max(10000).default(200),
    }),
    har: z.object({
      enabled: z.boolean().default(false),
    }),
    tracing: z.object({
      enabled: z.boolean().default(false),
    }),
  }),
  logging: z.object({
    level: z.enum(['debug', 'info', 'warn', 'error']).default('info'),
    format: z.enum(['json', 'text']).default('json'),
  }),
  metrics: z.object({
    enabled: z.boolean().default(true),
    port: z.number().min(1).max(65535).default(9090),
  }),
  /**
   * Frame Streaming Configuration
   *
   * Controls how frames are captured and streamed during recording.
   * The default uses CDP startScreencast for push-based frame delivery,
   * with fallback to polling-based screenshot capture.
   *
   * CDP screencast provides 30-60 FPS vs 10-15 FPS with polling.
   *
   * CONTROL LEVERS:
   * - useScreencast: Strategy selection (CDP push vs polling pull)
   * - fallbackToPolling: Resilience strategy (auto-fallback on failure)
   * - cdp.ackTimeoutMs: CDP responsiveness vs tolerance tradeoff
   * - cdp.maxAckFailures: Failure tolerance before logging errors
   * - cdp.frameLogInterval: Observability vs log noise tradeoff
   * - cdp.pageCheckIntervalMs: Multi-tab responsiveness vs CPU tradeoff
   */
  frameStreaming: z.object({
    /** Use CDP screencast (true) or legacy polling (false) */
    useScreencast: z.boolean().default(true),
    /** Fall back to polling if screencast fails */
    fallbackToPolling: z.boolean().default(true),
    /**
     * CDP-specific tuning options.
     * These control the behavior of the CDP screencast strategy.
     */
    cdp: z.object({
      /**
       * Timeout (ms) for acknowledging each screencast frame.
       * Chrome stops sending frames if ACKs are not received.
       * Trade-off: Lower = faster failure detection, but may cause spurious timeouts on slow systems.
       * Higher = more tolerant of system load spikes, but delays error detection.
       */
      ackTimeoutMs: z.number().min(100).max(10000).default(1000),
      /**
       * Maximum consecutive ACK failures before logging an error.
       * Trade-off: Lower = earlier warning, more noise. Higher = quieter logs, later detection.
       */
      maxAckFailures: z.number().min(1).max(100).default(5),
      /**
       * Log frame statistics every N frames.
       * Trade-off: Lower = more visibility, more log volume. Higher = less noise, less observability.
       * Set to 0 to disable periodic logging.
       */
      frameLogInterval: z.number().min(0).max(1000).default(60),
      /**
       * Interval (ms) for checking if the active page has changed (multi-tab support).
       * Screencast must be restarted when tabs change.
       * Trade-off: Lower = faster tab switch detection, more CPU. Higher = less overhead, slower detection.
       */
      pageCheckIntervalMs: z.number().min(50).max(5000).default(100),
    }),
  }),
  /**
   * Performance Debug Mode
   *
   * Controls timing instrumentation for the frame streaming pipeline.
   * When enabled, detailed timing data is collected and can be sent
   * to the API for aggregation and analysis.
   *
   * Tradeoff: Enabling adds ~1-2ms overhead per frame for timing collection.
   */
  performance: z.object({
    /** Enable debug performance mode for frame streaming */
    enabled: z.boolean().default(false),
    /** Include timing data in WebSocket frame headers (prepended to JPEG) */
    includeTimingHeaders: z.boolean().default(true),
    /** Log performance summaries every N frames (0 = disabled) */
    logSummaryInterval: z.number().min(0).max(1000).default(60),
    /** Number of frame timings to retain for percentile analysis */
    bufferSize: z.number().min(1).max(1000).default(100),
  }),
  /**
   * History Callback Configuration
   *
   * Controls how navigation history is reported to the Go API for persistence.
   * When a callback URL is configured, navigation events are POSTed to that URL
   * with page title, URL, timestamp, and optional thumbnail.
   */
  history: z.object({
    /** URL to POST history entries when navigation occurs (empty = disabled) */
    callbackUrl: z.string().default(''),
    /** Whether to capture thumbnails for history entries */
    thumbnailEnabled: z.boolean().default(true),
    /** JPEG quality for history thumbnails (10-100) */
    thumbnailQuality: z.number().min(10).max(100).default(60),
  }),
});

export type Config = z.infer<typeof ConfigSchema>;

/**
 * Parse integer from environment variable with validation
 *
 * Hardened assumptions:
 * - Environment variable may be undefined, empty, or invalid
 * - parseInt may return NaN for invalid input
 * - Default value must be returned for any invalid input
 */
/**
 * Parse a required integer environment variable.
 * Throws if not set or invalid - used for ports that must be set by the lifecycle system.
 */
function parseRequiredEnvInt(envName: string): number {
  const envVar = process.env[envName];
  if (!envVar || envVar.trim() === '') {
    throw new Error(`Required environment variable ${envName} is not set. This should be set by the vrooli lifecycle system.`);
  }

  const parsed = parseInt(envVar, 10);
  if (Number.isNaN(parsed)) {
    throw new Error(`Environment variable ${envName} has invalid value "${envVar}" - expected an integer.`);
  }

  return parsed;
}

function parseEnvInt(envVar: string | undefined, defaultValue: number): number {
  if (!envVar || envVar.trim() === '') {
    return defaultValue;
  }

  const parsed = parseInt(envVar, 10);

  // Check for NaN and return default if invalid
  if (Number.isNaN(parsed)) {
    // Log warning about invalid config (but don't fail startup)
    console.warn(`Invalid numeric config value: "${envVar}", using default: ${defaultValue}`);
    return defaultValue;
  }

  return parsed;
}

/**
 * Validate log level from environment variable
 */
function parseLogLevel(envVar: string | undefined): 'debug' | 'info' | 'warn' | 'error' {
  const validLevels = ['debug', 'info', 'warn', 'error'];
  const level = (envVar || 'info').toLowerCase();

  if (!validLevels.includes(level)) {
    console.warn(`Invalid LOG_LEVEL "${envVar}", using default: info`);
    return 'info';
  }

  return level as 'debug' | 'info' | 'warn' | 'error';
}

/**
 * Validate log format from environment variable
 */
function parseLogFormat(envVar: string | undefined): 'json' | 'text' {
  const validFormats = ['json', 'text'];
  const format = (envVar || 'json').toLowerCase();

  if (!validFormats.includes(format)) {
    console.warn(`Invalid LOG_FORMAT "${envVar}", using default: json`);
    return 'json';
  }

  return format as 'json' | 'text';
}

/**
 * Parse float from environment variable with validation
 */
function parseEnvFloat(envVar: string | undefined, defaultValue: number): number {
  if (!envVar || envVar.trim() === '') {
    return defaultValue;
  }

  const parsed = parseFloat(envVar);

  if (Number.isNaN(parsed)) {
    console.warn(`Invalid numeric config value: "${envVar}", using default: ${defaultValue}`);
    return defaultValue;
  }

  return parsed;
}

export function loadConfig(): Config {
  const config = {
    server: {
      port: parseRequiredEnvInt('PLAYWRIGHT_DRIVER_PORT'),
      host: process.env.PLAYWRIGHT_DRIVER_HOST || '127.0.0.1',
      requestTimeout: parseEnvInt(process.env.REQUEST_TIMEOUT_MS, 300000), // 5 minutes
      maxRequestSize: parseEnvInt(process.env.MAX_REQUEST_SIZE, 5242880),
    },
    browser: {
      // Default to false to use regular Chromium with --headless=new (not headless_shell)
      headless: process.env.HEADLESS === 'true',
      executablePath: process.env.BROWSER_EXECUTABLE_PATH,
      // Hardened: Simple comma-split can break args containing commas (rare but possible)
      // Browser args that contain commas are uncommon, but for robustness we trim each arg
      // and filter out empty strings that might result from trailing/leading commas
      // Note: If you need args with commas, use a different delimiter in the env var
      // e.g., BROWSER_ARGS="--arg1;;--arg2=value" with split(';;')
      //
      // Default includes:
      // - --headless=new: Uses new headless mode with full service worker support
      // - --disable-blink-features=AutomationControlled: Hides automation detection signals
      // These fix Google redirect loop issues caused by bot detection
      args: [
        '--headless=new',
        '--disable-blink-features=AutomationControlled',
        ...(process.env.BROWSER_ARGS
          ? process.env.BROWSER_ARGS.split(',')
              .map((arg) => arg.trim())
              .filter((arg) => arg.length > 0)
          : []),
      ],
      ignoreHTTPSErrors: process.env.IGNORE_HTTPS_ERRORS === 'true',
    },
    session: {
      maxConcurrent: parseEnvInt(process.env.MAX_SESSIONS, 10),
      idleTimeoutMs: parseEnvInt(process.env.SESSION_IDLE_TIMEOUT_MS, 300000),
      poolSize: parseEnvInt(process.env.SESSION_POOL_SIZE, 5),
      cleanupIntervalMs: parseEnvInt(process.env.CLEANUP_INTERVAL_MS, 60000),
    },
    // Execution timeouts - the main performance/reliability tradeoff
    execution: {
      defaultTimeoutMs: parseEnvInt(process.env.EXECUTION_DEFAULT_TIMEOUT_MS, 30000),
      navigationTimeoutMs: parseEnvInt(process.env.EXECUTION_NAVIGATION_TIMEOUT_MS, 45000),
      waitTimeoutMs: parseEnvInt(process.env.EXECUTION_WAIT_TIMEOUT_MS, 30000),
      assertionTimeoutMs: parseEnvInt(process.env.EXECUTION_ASSERTION_TIMEOUT_MS, 15000),
      replayActionTimeoutMs: parseEnvInt(process.env.EXECUTION_REPLAY_TIMEOUT_MS, 10000),
    },
    // Recording configuration
    recording: {
      maxBufferSize: parseEnvInt(process.env.RECORDING_MAX_BUFFER_SIZE, 10000),
      minSelectorConfidence: parseEnvFloat(process.env.RECORDING_MIN_SELECTOR_CONFIDENCE, 0.3),
      defaultSwipeDistance: parseEnvInt(process.env.RECORDING_DEFAULT_SWIPE_DISTANCE, 300),
      diagnosticsEnabled: process.env.RECORDING_DIAGNOSTICS_ENABLED === 'true',
      debounce: {
        inputMs: parseEnvInt(process.env.RECORDING_INPUT_DEBOUNCE_MS, 500),
        scrollMs: parseEnvInt(process.env.RECORDING_SCROLL_DEBOUNCE_MS, 150),
      },
      selector: {
        maxCssDepth: parseEnvInt(process.env.RECORDING_MAX_CSS_DEPTH, 5),
        includeXPath: process.env.RECORDING_INCLUDE_XPATH !== 'false',
      },
    },
    telemetry: {
      screenshot: {
        enabled: process.env.SCREENSHOT_ENABLED !== 'false',
        // Default to false to prevent viewport oscillation during execution
        // Set SCREENSHOT_FULL_PAGE=true to enable full page screenshots
        fullPage: process.env.SCREENSHOT_FULL_PAGE === 'true',
        quality: parseEnvInt(process.env.SCREENSHOT_QUALITY, 80),
        maxSizeBytes: parseEnvInt(process.env.SCREENSHOT_MAX_SIZE, 512000),
      },
      dom: {
        enabled: process.env.DOM_ENABLED !== 'false',
        maxSizeBytes: parseEnvInt(process.env.DOM_MAX_SIZE, 524288),
      },
      console: {
        enabled: process.env.CONSOLE_ENABLED !== 'false',
        maxEntries: parseEnvInt(process.env.CONSOLE_MAX_ENTRIES, 100),
      },
      network: {
        enabled: process.env.NETWORK_ENABLED !== 'false',
        maxEvents: parseEnvInt(process.env.NETWORK_MAX_EVENTS, 200),
      },
      har: {
        enabled: process.env.HAR_ENABLED === 'true',
      },
      tracing: {
        enabled: process.env.TRACING_ENABLED === 'true',
      },
    },
    logging: {
      level: parseLogLevel(process.env.LOG_LEVEL),
      format: parseLogFormat(process.env.LOG_FORMAT),
    },
    metrics: {
      enabled: process.env.METRICS_ENABLED !== 'false',
      port: parseEnvInt(process.env.METRICS_PORT, 9090),
    },
    frameStreaming: {
      useScreencast: process.env.FRAME_STREAMING_USE_SCREENCAST !== 'false', // Default true
      fallbackToPolling: process.env.FRAME_STREAMING_FALLBACK !== 'false', // Default true
      cdp: {
        ackTimeoutMs: parseEnvInt(process.env.FRAME_STREAMING_CDP_ACK_TIMEOUT_MS, 1000),
        maxAckFailures: parseEnvInt(process.env.FRAME_STREAMING_CDP_MAX_ACK_FAILURES, 5),
        frameLogInterval: parseEnvInt(process.env.FRAME_STREAMING_CDP_FRAME_LOG_INTERVAL, 60),
        pageCheckIntervalMs: parseEnvInt(process.env.FRAME_STREAMING_CDP_PAGE_CHECK_INTERVAL_MS, 100),
      },
    },
    performance: {
      enabled: process.env.PLAYWRIGHT_DRIVER_PERF_ENABLED === 'true',
      includeTimingHeaders: process.env.PLAYWRIGHT_DRIVER_PERF_INCLUDE_HEADERS !== 'false',
      logSummaryInterval: parseEnvInt(process.env.PLAYWRIGHT_DRIVER_PERF_LOG_INTERVAL, 60),
      bufferSize: parseEnvInt(process.env.PLAYWRIGHT_DRIVER_PERF_BUFFER_SIZE, 100),
    },
    history: {
      callbackUrl: process.env.HISTORY_CALLBACK_URL || '',
      thumbnailEnabled: process.env.HISTORY_THUMBNAIL_ENABLED !== 'false',
      thumbnailQuality: parseEnvInt(process.env.HISTORY_THUMBNAIL_QUALITY, 60),
    },
  };

  const parsed = ConfigSchema.parse(config);

  // Hardened: Deep freeze config to prevent accidental mutation during runtime
  return deepFreeze(parsed);
}

/**
 * Deep freeze an object to prevent any mutation.
 * Hardening measure to ensure config immutability.
 */
function deepFreeze<T extends Record<string, unknown>>(obj: T): T {
  const propNames = Object.getOwnPropertyNames(obj);

  for (const name of propNames) {
    const value = obj[name];
    if (value && typeof value === 'object' && !Object.isFrozen(value)) {
      deepFreeze(value as Record<string, unknown>);
    }
  }

  return Object.freeze(obj);
}

// =============================================================================
// CONFIGURATION TIER WARNINGS
// =============================================================================

/**
 * Result of checking modified configuration options.
 */
export interface ConfigTierCheckResult {
  /** Options modified from defaults */
  modified: Array<{
    envVar: string;
    tier: ConfigTier;
    tierName: string;
    description: string;
    currentValue: unknown;
    defaultValue: unknown;
  }>;
  /** Count of modified options by tier */
  byTier: {
    essential: number;
    advanced: number;
    internal: number;
  };
}

/**
 * Check which configuration options have been modified from their defaults.
 * Returns detailed information about modified options organized by tier.
 *
 * @returns Object with modified options and counts by tier
 */
export function checkModifiedConfig(): ConfigTierCheckResult {
  const modified: ConfigTierCheckResult['modified'] = [];
  const byTier = { essential: 0, advanced: 0, internal: 0 };

  const tierNames: Record<ConfigTier, string> = {
    [ConfigTier.ESSENTIAL]: 'essential',
    [ConfigTier.ADVANCED]: 'advanced',
    [ConfigTier.INTERNAL]: 'internal',
  };

  for (const [envVar, meta] of Object.entries(CONFIG_TIER_METADATA)) {
    const currentValue = process.env[envVar];

    // Skip if not set in environment
    if (currentValue === undefined) continue;

    // Check if value differs from default
    const defaultStr = String(meta.defaultValue);
    const currentStr = currentValue.trim();

    // Handle boolean defaults
    let isModified = false;
    if (typeof meta.defaultValue === 'boolean') {
      const currentBool = currentStr.toLowerCase() === 'true';
      isModified = currentBool !== meta.defaultValue;
    } else if (typeof meta.defaultValue === 'number') {
      const currentNum = parseFloat(currentStr);
      isModified = !isNaN(currentNum) && currentNum !== meta.defaultValue;
    } else if (meta.defaultValue === undefined) {
      // Any non-empty value for undefined default is a modification
      isModified = currentStr.length > 0;
    } else {
      isModified = currentStr !== defaultStr;
    }

    if (isModified) {
      const tierName = tierNames[meta.tier];
      modified.push({
        envVar,
        tier: meta.tier,
        tierName,
        description: meta.description,
        currentValue: currentStr,
        defaultValue: meta.defaultValue,
      });

      if (meta.tier === ConfigTier.ESSENTIAL) byTier.essential++;
      else if (meta.tier === ConfigTier.ADVANCED) byTier.advanced++;
      else byTier.internal++;
    }
  }

  // Sort by tier (essential first)
  modified.sort((a, b) => a.tier - b.tier);

  return { modified, byTier };
}

/**
 * Log warnings about modified configuration options.
 * Called at startup to inform operators about non-default settings.
 *
 * This function helps operators understand what they've configured:
 * - Tier 1 (Essential): Informational - expected to be configured
 * - Tier 2 (Advanced): Note - ensure these are intentional
 * - Tier 3 (Internal): Warning - rarely needed, may indicate misconfiguration
 *
 * @param logFn - Logging function to use (defaults to console.warn)
 */
export function logConfigTierWarnings(logFn: (msg: string, data?: Record<string, unknown>) => void = console.warn): void {
  const result = checkModifiedConfig();

  if (result.modified.length === 0) {
    return; // All defaults, nothing to log
  }

  // Group messages by tier
  const tier2Options = result.modified.filter(m => m.tier === ConfigTier.ADVANCED);
  const tier3Options = result.modified.filter(m => m.tier === ConfigTier.INTERNAL);

  // Log Tier 2 (Advanced) options as INFO
  if (tier2Options.length > 0) {
    logFn('[config] Advanced options configured (Tier 2):', {
      count: tier2Options.length,
      options: tier2Options.map(o => `${o.envVar}=${o.currentValue}`),
      hint: 'These are fine-tuning options. See CONFIG-TIERS.md for guidance.',
    });
  }

  // Log Tier 3 (Internal) options as WARN
  if (tier3Options.length > 0) {
    logFn('[config] Internal options configured (Tier 3):', {
      count: tier3Options.length,
      options: tier3Options.map(o => `${o.envVar}=${o.currentValue}`),
      hint: 'These are rarely-needed internal options. Consider if they are necessary.',
    });
  }
}

/**
 * Get a human-readable summary of the configuration tier status.
 * Useful for health checks and debugging.
 */
export function getConfigSummary(): string {
  const result = checkModifiedConfig();
  const total = result.modified.length;

  if (total === 0) {
    return 'All configuration options at defaults';
  }

  const parts: string[] = [];
  if (result.byTier.essential > 0) parts.push(`${result.byTier.essential} essential`);
  if (result.byTier.advanced > 0) parts.push(`${result.byTier.advanced} advanced`);
  if (result.byTier.internal > 0) parts.push(`${result.byTier.internal} internal`);

  return `${total} option(s) modified from defaults: ${parts.join(', ')}`;
}

/**
 * Configuration option for observability response.
 * Includes all metadata needed for UI rendering and editing.
 */
interface ObservabilityConfigOption {
  env_var: string;
  tier: 'essential' | 'advanced' | 'internal';
  description: string;
  current_value: string;
  default_value: string;
  is_modified: boolean;
  /** Data type for UI rendering and validation */
  data_type: ConfigDataType;
  /** For numeric types: minimum allowed value */
  min?: number;
  /** For numeric types: maximum allowed value */
  max?: number;
  /** For enum types: allowed values */
  enum_values?: string[];
  /** Whether this option can be changed at runtime without restart */
  editable: boolean;
}

/**
 * Get configuration summary in the format needed for observability.
 * Returns structured data for the diagnostics UI.
 */
export function getObservabilityConfigSummary(): {
  summary: string;
  modified_count: number;
  total_count: number;
  by_tier: { essential: number; advanced: number; internal: number };
  modified_options?: Array<{
    env_var: string;
    tier: 'essential' | 'advanced' | 'internal';
    description: string;
    current_value: string;
    default_value: string;
  }>;
  all_options?: {
    essential: ObservabilityConfigOption[];
    advanced: ObservabilityConfigOption[];
    internal: ObservabilityConfigOption[];
  };
} {
  const result = checkModifiedConfig();
  const summary = getConfigSummary();

  // Create a set of modified env vars for quick lookup
  const modifiedSet = new Set(result.modified.map(m => m.envVar));

  // Build all_options organized by tier
  const all_options: {
    essential: ObservabilityConfigOption[];
    advanced: ObservabilityConfigOption[];
    internal: ObservabilityConfigOption[];
  } = {
    essential: [],
    advanced: [],
    internal: [],
  };

  const tierNames: Record<ConfigTier, 'essential' | 'advanced' | 'internal'> = {
    [ConfigTier.ESSENTIAL]: 'essential',
    [ConfigTier.ADVANCED]: 'advanced',
    [ConfigTier.INTERNAL]: 'internal',
  };

  for (const [envVar, meta] of Object.entries(CONFIG_TIER_METADATA)) {
    const currentValue = process.env[envVar];
    const tierName = tierNames[meta.tier];

    // Determine current value (use env var if set, otherwise default)
    let currentStr: string;
    if (currentValue !== undefined && currentValue.trim() !== '') {
      currentStr = currentValue.trim();
    } else if (meta.defaultValue === undefined) {
      currentStr = '';
    } else {
      currentStr = String(meta.defaultValue);
    }

    const option: ObservabilityConfigOption = {
      env_var: envVar,
      tier: tierName,
      description: meta.description,
      current_value: currentStr,
      default_value: meta.defaultValue === undefined ? '' : String(meta.defaultValue),
      is_modified: modifiedSet.has(envVar),
      data_type: meta.dataType,
      min: meta.min,
      max: meta.max,
      enum_values: meta.enumValues,
      editable: meta.editable ?? false,
    };

    all_options[tierName].push(option);
  }

  // Sort each tier alphabetically by env_var
  all_options.essential.sort((a, b) => a.env_var.localeCompare(b.env_var));
  all_options.advanced.sort((a, b) => a.env_var.localeCompare(b.env_var));
  all_options.internal.sort((a, b) => a.env_var.localeCompare(b.env_var));

  // Only include modified_options if there are any
  const modified_options = result.modified.length > 0
    ? result.modified.map((opt) => ({
        env_var: opt.envVar,
        tier: opt.tierName as 'essential' | 'advanced' | 'internal',
        description: opt.description,
        current_value: String(opt.currentValue),
        default_value: String(opt.defaultValue),
      }))
    : undefined;

  const total_count = Object.keys(CONFIG_TIER_METADATA).length;

  return {
    summary,
    modified_count: result.modified.length,
    total_count,
    by_tier: result.byTier,
    modified_options,
    all_options,
  };
}
