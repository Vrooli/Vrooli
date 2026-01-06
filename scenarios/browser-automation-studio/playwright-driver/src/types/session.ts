import type { Page, BrowserContext, Browser, Frame } from 'rebrowser-playwright';
import type { RecordingContextInitializer, RecordingPipelineManager } from '../recording';
import type { BrowserProfile } from './browser-profile';
import type { ServiceWorkerControl } from './service-worker';
import type { ServiceWorkerController } from '../service-worker';

export type ReuseMode = 'fresh' | 'clean' | 'reuse';

export interface SessionSpec {
  execution_id: string;
  workflow_id: string;
  viewport: {
    width: number;
    height: number;
  };
  reuse_mode: ReuseMode;
  base_url?: string;
  labels?: Record<string, string>;
  required_capabilities?: {
    tabs?: boolean;
    iframes?: boolean;
    uploads?: boolean;
    downloads?: boolean;
    har?: boolean;
    video?: boolean;
    tracing?: boolean;
    viewport_width?: number;
    viewport_height?: number;
  };
  artifact_paths?: {
    root?: string;
    video_dir?: string;
    har_path?: string;
    trace_path?: string;
  };
  // Browser context configuration
  user_agent?: string;
  locale?: string;
  timezone?: string;
  geolocation?: {
    latitude: number;
    longitude: number;
    accuracy?: number;
  };
  permissions?: string[];
  storage_state?: {
    cookies: Array<{
      name: string;
      value: string;
      domain: string;
      path: string;
      expires: number;
      httpOnly: boolean;
      secure: boolean;
      sameSite: 'Strict' | 'Lax' | 'None';
    }>;
    origins: Array<{
      origin: string;
      localStorage: Array<{ name: string; value: string }>;
    }>;
  };
  /**
   * Service worker control configuration.
   * Controls how service workers are managed during the session.
   * Defaults to 'allow' mode if not specified.
   */
  service_worker_control?: ServiceWorkerControl;
  // Anti-detection and human-like behavior configuration
  browser_profile?: BrowserProfile;
}

/**
 * Session lifecycle phases.
 * These phases represent the high-level state of a session and are exposed
 * in API responses and logs for observability.
 *
 * ## STATE MACHINE
 *
 * ```
 *                          ┌─────────────────────────────────────┐
 *                          │                                     │
 *    startSession()        │       NORMAL OPERATION              │
 *         │                │                                     │
 *         ▼                │   ┌────────────────────────────┐    │
 *  ┌──────────────┐        │   │                            │    │
 *  │ initializing │────────┼──▶│           ready            │◀───┼────┐
 *  └──────────────┘        │   │                            │    │    │
 *                          │   └─────────┬──────────────────┘    │    │
 *                          │             │                       │    │
 *                          │   runInstruction()    startRecording()   │
 *                          │             │                       │    │
 *                          │             ▼                       │    │
 *                          │   ┌────────────────┐               │    │
 *                          │   │   executing    │───────────────┼────┘
 *                          │   │                │  instruction  │ (completes)
 *                          │   │ (one at a time │    done       │
 *                          │   │  per session)  │               │
 *                          │   └────────────────┘               │
 *                          │                                     │
 *                          │   ┌────────────────┐               │
 *              ┌───────────┼──▶│   recording    │───────────────┼────┐
 *              │           │   │                │  stopRecording()   │
 *  startRecording()        │   │ (captures user │               │    │
 *              │           │   │    actions)    │               │    │
 *              │           │   └────────────────┘               │    │
 *              │           │                                     │    │
 *              │           └─────────────────────────────────────┘    │
 *              │                                                      │
 *              └──────────────────────────────────────────────────────┘
 *
 *    resetSession()               closeSession()
 *         │                            │
 *         ▼                            ▼
 *  ┌──────────────┐            ┌──────────────┐
 *  │  resetting   │            │   closing    │
 *  │              │            │              │
 *  │ (clears      │            │ (frees       │
 *  │  state)      │────────▶   │  resources)  │
 *  └──────────────┘  done      └──────────────┘
 *                       │
 *                       ▼
 *                     ready
 * ```
 *
 * ## TRANSITION RULES
 *
 * | From         | To          | Trigger                    | Concurrency      |
 * |--------------|-------------|----------------------------|------------------|
 * | initializing | ready       | Browser context created    | N/A              |
 * | ready        | executing   | POST /session/:id/run      | One at a time    |
 * | executing    | ready       | Instruction completes      | Automatic        |
 * | ready        | recording   | POST /record/start         | -                |
 * | recording    | ready       | POST /record/stop          | -                |
 * | executing    | recording   | If recording was active    | Restores state   |
 * | any          | resetting   | POST /session/:id/reset    | -                |
 * | resetting    | ready       | Reset completes            | Automatic        |
 * | any          | closing     | POST /session/:id/close    | Terminal         |
 *
 * ## CONCURRENCY RULES
 *
 * - Only ONE instruction can execute at a time per session
 * - Concurrent instruction attempts return 409 Conflict
 * - Recording mode allows streaming but blocks instruction execution
 *
 * ## RELATIONSHIP TO RECORDING PIPELINE STATE
 *
 * The `recording` phase here is an **API-level** indicator that the session is in record mode.
 * Internally, recording uses a separate state machine (`RecordingPipelinePhase`) with 8 phases
 * for tracking infrastructure health (initializing, verifying, ready, starting, capturing, etc.).
 *
 * **Why two state machines?**
 * - SessionPhase is for external consumers (UI, API clients) - simplified 6-phase view
 * - RecordingPipelinePhase is for internal infrastructure - detailed 8-phase view
 * - This separation allows recording infrastructure to fail/recover without affecting API state
 *
 * **Synchronization pattern** (query at boundaries):
 * When determining session phase, code queries the pipeline manager:
 * ```typescript
 * const phase = session.pipelineManager?.isRecording() ? 'recording' : 'ready';
 * ```
 *
 * @see RecordingPipelinePhase in recording/state-machine.ts for infrastructure-level phases
 * @see RecordingPipelineManager in recording/pipeline-manager.ts for recording orchestration
 *
 * Phase meanings:
 * - 'initializing': Session is being created, browser context not yet ready
 * - 'ready': Session is ready to accept instructions
 * - 'executing': An instruction is currently running
 * - 'recording': Record mode is active, capturing user actions
 * - 'resetting': Session state is being reset (clearing cookies, storage, etc.)
 * - 'closing': Session is being torn down, resources being freed
 */
export type SessionPhase = 'initializing' | 'ready' | 'executing' | 'recording' | 'resetting' | 'closing';

export interface SessionState {
  id: string;
  browser: Browser;
  context: BrowserContext;
  page: Page;
  spec: SessionSpec;
  createdAt: Date;
  lastUsedAt: Date;
  tracing: boolean;
  video: boolean;
  harPath?: string;
  tracePath?: string;
  videoDir?: string;

  /**
   * Current session lifecycle phase.
   * Used for observability and debugging - exposed in status endpoints and logs.
   */
  phase: SessionPhase;

  /**
   * Count of instructions executed in this session.
   * Useful for tracking session activity and debugging.
   */
  instructionCount: number;

  // Frame navigation stack (for frame-switch)
  frameStack: Frame[];

  // Tab/page stack (for multi-tab support)
  pages: Page[];
  currentPageIndex: number;

  /**
   * Map from driver page ID (UUID) to Playwright Page object.
   * Enables efficient lookup when switching pages by ID.
   */
  pageIdMap: Map<string, Page>;

  /**
   * Reverse map from Playwright Page object to driver page ID.
   * Used to find the ID when a page emits events.
   */
  pageToIdMap: WeakMap<Page, string>;

  // Network mocking state
  activeMocks: Map<string, MockRoute>;

  /**
   * Recording pipeline manager for this session.
   * Manages recording state via unified state machine.
   * Single source of truth for all recording operations.
   */
  pipelineManager?: RecordingPipelineManager;

  /**
   * Cleanup function for page lifecycle listeners.
   * Called when recording stops to remove event handlers.
   */
  pageLifecycleCleanup?: () => void;

  /**
   * Instruction idempotency tracking.
   * Maps instruction key (node_id:index) to last execution result.
   * Enables replay-safe instruction execution.
   */
  executedInstructions?: Map<string, ExecutedInstructionRecord>;

  /**
   * Service worker controller for this session.
   * Manages CDP-based service worker monitoring and control.
   */
  serviceWorkerController?: ServiceWorkerController;

  /**
   * Recording context initializer for this session.
   * Handles context-level binding and init script for recording.
   * Shared across all recording sessions in this browser context.
   */
  recordingInitializer?: RecordingContextInitializer;
}

export interface SessionCloseResult {
  videoPaths: string[];
  tracePath?: string;
  harPath?: string;
}

/**
 * Record of an executed instruction for idempotency tracking.
 * Stores enough information to return the same result on replay.
 */
export interface ExecutedInstructionRecord {
  /** Composite key: node_id:index */
  key: string;
  /** When the instruction was executed */
  executedAt: Date;
  /** Whether execution succeeded */
  success: boolean;
  /** Cached outcome for replay (optional, may be large) */
  cachedOutcome?: unknown;
}

export interface MockRoute {
  urlPattern: string | RegExp;
  method?: string;
  handler: (route: unknown) => Promise<void>;
}

export interface SessionMetrics {
  totalSessions: number;
  activeSessions: number;
  idleSessions: number;
  avgSessionDuration: number;
  peakSessions: number;
}

export interface StartSessionRequest {
  execution_id: string;
  workflow_id: string;
  viewport: {
    width: number;
    height: number;
  };
  reuse_mode: string;
  base_url?: string;
  labels?: Record<string, string>;
  required_capabilities?: {
    tabs?: boolean;
    iframes?: boolean;
    uploads?: boolean;
    downloads?: boolean;
    har?: boolean;
    video?: boolean;
    tracing?: boolean;
    viewport_width?: number;
    viewport_height?: number;
  };
  artifact_paths?: {
    root?: string;
    video_dir?: string;
    har_path?: string;
    trace_path?: string;
  };
  storage_state?: SessionSpec['storage_state'];
  /**
   * Optional: Enable live frame streaming immediately when session is created.
   * This allows viewing the browser before starting action recording.
   */
  frame_streaming?: {
    /** Callback URL for frame delivery (API constructs WebSocket URL from this) */
    callback_url: string;
    /** JPEG quality 1-100 (default: 55 from API config) */
    quality?: number;
    /** Target FPS 1-60 (default: 30 from API config) */
    /** Note: For CDP screencast, Chrome controls actual FPS. This is a target/hint. */
    fps?: number;
    /** Screenshot scale: 'css' for 1x scale (default), 'device' for device pixel ratio */
    scale?: 'css' | 'device';
  };
  /**
   * Optional: Browser profile for anti-detection and human-like behavior.
   */
  browser_profile?: BrowserProfile;
}

/**
 * Describes what determined the actual viewport dimensions.
 * This attribution helps users understand why dimensions may differ from requested.
 */
export type ViewportSource =
  | 'requested'           // Used the UI-requested dimensions
  | 'fingerprint'         // Browser profile fingerprint override
  | 'fingerprint_partial' // Fingerprint set one dimension, requested used for other
  | 'default';            // Fallback defaults used

/**
 * Actual viewport with source attribution.
 * Includes the dimensions and explanation of what determined them.
 */
export interface ActualViewportResponse {
  width: number;
  height: number;
  /** What determined these dimensions */
  source: ViewportSource;
  /** Human-readable explanation of why this source was used */
  reason: string;
}

export interface StartSessionResponse {
  session_id: string;
  /** Session phase after creation (always 'ready' for new sessions) */
  phase: SessionPhase;
  /** ISO 8601 timestamp when session was created */
  created_at: string;
  /** Whether this was a reused session (only true if reuse_mode != 'fresh' and match found) */
  reused?: boolean;
  /**
   * Actual viewport dimensions applied by Playwright with source attribution.
   * May differ from requested dimensions due to browser profile fingerprint overrides.
   */
  actual_viewport?: ActualViewportResponse;
}

/**
 * Response from updating session viewport dimensions.
 */
export interface UpdateViewportResponse {
  /** Success status */
  success: boolean;
  /**
   * Actual viewport dimensions after update with source attribution.
   * This is what Playwright is actually using, which may differ from requested
   * dimensions due to constraints or browser profile settings.
   */
  actual_viewport: ActualViewportResponse;
}

/**
 * Health check component status
 */
export interface HealthCheck {
  /** pass: healthy, fail: unhealthy */
  status: 'pass' | 'fail';
  /** Human-readable status message */
  message: string;
  /** Actionable hint for operators when status is 'fail' */
  hint?: string;
}

/**
 * Health endpoint response
 *
 * Semantic meaning:
 * - status='ok' + ready=true: Accept traffic, fully operational
 * - status='degraded' + ready=false: Functional but with issues
 * - status='error': Critical failure, do not route traffic
 */
export interface HealthResponse {
  /** Overall health: ok (healthy), degraded (issues), error (critical) */
  status: 'ok' | 'degraded' | 'error';
  /** True when driver is ready to accept new sessions */
  ready: boolean;
  /** ISO 8601 timestamp of health check */
  timestamp: string;
  /** Number of active sessions */
  sessions: number;
  /** Number of sessions with active recording */
  active_recordings?: number;
  /** Driver version */
  version?: string;
  /** Browser subsystem health */
  browser?: {
    healthy: boolean;
    version?: string;
    error?: string;
  };
  /** Server uptime in milliseconds */
  uptime_ms?: number;
  /** Individual component checks for debugging */
  checks?: {
    browser: HealthCheck;
    sessions: HealthCheck;
    recordings: HealthCheck;
  };
}
