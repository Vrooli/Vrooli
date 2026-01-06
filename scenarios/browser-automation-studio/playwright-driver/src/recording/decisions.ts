/**
 * Recording Decision Functions
 *
 * Named, testable decision functions that make recording behavior explicit.
 * Each function returns a decision object with:
 * - The decision result (boolean)
 * - The reason code (for logging and debugging)
 * - Human-readable message
 *
 * These replace implicit decision logic scattered throughout the codebase.
 * When debugging, you can log decisions to see exactly why events were dropped.
 *
 * @see context-initializer.ts - Uses these for injection decisions
 * @see pipeline-manager.ts - Uses these for event processing decisions
 */

// =============================================================================
// Decision Types
// =============================================================================

/** Result of an injection decision */
export interface InjectionDecision {
  shouldInject: boolean;
  reason:
    | 'yes_html_document'
    | 'skip_non_document'
    | 'skip_redirect'
    | 'skip_non_html'
    | 'skip_test_page';
  message: string;
  metadata?: {
    resourceType?: string;
    contentType?: string;
    status?: number;
    url?: string;
  };
}

/** Result of an event processing decision */
export interface EventProcessingDecision {
  shouldProcess: boolean;
  reason:
    | 'accepted'
    | 'not_recording'
    | 'no_handler'
    | 'stale_generation'
    | 'invalid_event';
  message: string;
  metadata?: {
    phase?: string;
    hasHandler?: boolean;
    eventType?: string;
  };
}

/** Result of a pipeline readiness decision */
export interface PipelineReadinessDecision {
  canStart: boolean;
  blockers: string[];
  suggestions: string[];
  metadata?: {
    phase?: string;
    scriptLoaded?: boolean;
    scriptReady?: boolean;
    inMainContext?: boolean;
    handlersCount?: number;
  };
}

// =============================================================================
// Injection Decisions
// =============================================================================

/**
 * Decide whether to inject the recording script into a request.
 *
 * @param resourceType - Playwright resource type (document, script, image, etc.)
 * @param url - Request URL
 * @param status - Response status code (if response available)
 * @param contentType - Response content-type header (if response available)
 */
export function shouldInjectScript(
  resourceType: string,
  url: string,
  status?: number,
  contentType?: string
): InjectionDecision {
  // Decision 1: Must be a document request
  if (resourceType !== 'document') {
    return {
      shouldInject: false,
      reason: 'skip_non_document',
      message: `Skipping non-document request (resourceType=${resourceType})`,
      metadata: { resourceType, url: url.slice(0, 80) },
    };
  }

  // Decision 2: Skip test page URLs (handled by dedicated route)
  if (url.includes('__vrooli_recording_test__')) {
    return {
      shouldInject: false,
      reason: 'skip_test_page',
      message: 'Skipping test page URL (handled by dedicated route)',
      metadata: { url: url.slice(0, 80) },
    };
  }

  // Decision 3: Skip redirects (let browser handle, inject at final destination)
  if (status !== undefined && status >= 300 && status < 400) {
    return {
      shouldInject: false,
      reason: 'skip_redirect',
      message: `Skipping redirect response (status=${status})`,
      metadata: { status, url: url.slice(0, 80) },
    };
  }

  // Decision 4: Must be HTML content
  if (contentType !== undefined && !contentType.includes('text/html')) {
    return {
      shouldInject: false,
      reason: 'skip_non_html',
      message: `Skipping non-HTML response (contentType=${contentType.slice(0, 50)})`,
      metadata: { contentType: contentType.slice(0, 50), url: url.slice(0, 80) },
    };
  }

  // All checks passed - inject!
  return {
    shouldInject: true,
    reason: 'yes_html_document',
    message: 'HTML document - injecting recording script',
    metadata: { resourceType, url: url.slice(0, 80), status, contentType: contentType?.slice(0, 50) },
  };
}

// =============================================================================
// Event Processing Decisions
// =============================================================================

/**
 * Decide whether to process an incoming recording event.
 *
 * @param phase - Current pipeline phase
 * @param hasHandler - Whether an event handler is set
 * @param eventType - Type of the event
 * @param generation - Current recording generation (optional)
 * @param eventGeneration - Generation the event was captured in (optional)
 */
export function shouldProcessEvent(
  phase: string,
  hasHandler: boolean,
  eventType: string,
  generation?: number,
  eventGeneration?: number
): EventProcessingDecision {
  // Decision 1: Must be in recording phase
  if (phase !== 'recording') {
    return {
      shouldProcess: false,
      reason: 'not_recording',
      message: `Event dropped: not in recording phase (phase=${phase})`,
      metadata: { phase, eventType },
    };
  }

  // Decision 2: Must have a handler
  if (!hasHandler) {
    return {
      shouldProcess: false,
      reason: 'no_handler',
      message: `Event dropped: no event handler set`,
      metadata: { phase, hasHandler: false, eventType },
    };
  }

  // Decision 3: Check generation (stale event detection)
  if (
    generation !== undefined &&
    eventGeneration !== undefined &&
    eventGeneration !== generation
  ) {
    return {
      shouldProcess: false,
      reason: 'stale_generation',
      message: `Event dropped: stale generation (event=${eventGeneration}, current=${generation})`,
      metadata: { phase, eventType },
    };
  }

  // All checks passed - process!
  return {
    shouldProcess: true,
    reason: 'accepted',
    message: `Event accepted: ${eventType}`,
    metadata: { phase, hasHandler: true, eventType },
  };
}

// =============================================================================
// Pipeline Readiness Decisions
// =============================================================================

/**
 * Decide whether the pipeline is ready to start recording.
 *
 * @param phase - Current pipeline phase
 * @param verification - Verification result (if available)
 */
export function canStartRecording(
  phase: string,
  verification?: {
    scriptLoaded: boolean;
    scriptReady: boolean;
    inMainContext: boolean;
    handlersCount: number;
  }
): PipelineReadinessDecision {
  const blockers: string[] = [];
  const suggestions: string[] = [];

  // Check phase
  if (phase !== 'ready') {
    blockers.push(`Pipeline not ready (phase=${phase})`);

    if (phase === 'uninitialized') {
      suggestions.push('Call initialize() first');
    } else if (phase === 'verifying') {
      suggestions.push('Wait for verification to complete');
    } else if (phase === 'error') {
      suggestions.push('Check error state and attempt recovery');
    }
  }

  // Check verification details (if available)
  if (verification) {
    if (!verification.scriptLoaded) {
      blockers.push('Recording script not loaded on page');
      suggestions.push('Check HTML injection route - page may not have been navigated via HTTP(S)');
      suggestions.push('Service workers or CSP headers might be blocking injection');
    } else if (!verification.scriptReady) {
      blockers.push('Recording script loaded but not initialized');
      suggestions.push('Script may have crashed during initialization - check browser console');
    } else if (!verification.inMainContext) {
      blockers.push('Script running in ISOLATED context (must be MAIN for History API)');
      suggestions.push('Script must be injected via HTML, not page.evaluate()');
    } else if (verification.handlersCount < 7) {
      blockers.push(`Low handler count (${verification.handlersCount}), expected 7+`);
      suggestions.push('Some event types may not be captured');
    }
  }

  return {
    canStart: blockers.length === 0,
    blockers,
    suggestions,
    metadata: verification
      ? {
          phase,
          scriptLoaded: verification.scriptLoaded,
          scriptReady: verification.scriptReady,
          inMainContext: verification.inMainContext,
          handlersCount: verification.handlersCount,
        }
      : { phase },
  };
}

// =============================================================================
// Logging Helpers
// =============================================================================

/**
 * Format a decision for logging.
 * Returns a structured object suitable for logger.info/warn/debug.
 */
export function formatDecisionForLog(
  decision: InjectionDecision | EventProcessingDecision | PipelineReadinessDecision
): Record<string, unknown> {
  if ('shouldInject' in decision) {
    return {
      decision: decision.shouldInject ? 'INJECT' : 'SKIP',
      reason: decision.reason,
      message: decision.message,
      ...decision.metadata,
    };
  }

  if ('shouldProcess' in decision) {
    return {
      decision: decision.shouldProcess ? 'PROCESS' : 'DROP',
      reason: decision.reason,
      message: decision.message,
      ...decision.metadata,
    };
  }

  // PipelineReadinessDecision
  return {
    decision: decision.canStart ? 'READY' : 'BLOCKED',
    blockers: decision.blockers,
    suggestions: decision.suggestions,
    ...decision.metadata,
  };
}
