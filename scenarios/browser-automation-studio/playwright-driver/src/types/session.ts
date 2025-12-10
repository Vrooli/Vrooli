import { Page, BrowserContext, Browser, Frame } from 'playwright';
import type { RecordModeController } from '../recording/controller';

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
}

/**
 * Session lifecycle phases.
 * These phases represent the high-level state of a session and are exposed
 * in API responses and logs for observability.
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

  // Network mocking state
  activeMocks: Map<string, MockRoute>;

  // Recording state (Record Mode)
  recordingController?: RecordModeController;
  recordingId?: string;

  /**
   * Instruction idempotency tracking.
   * Maps instruction key (node_id:index) to last execution result.
   * Enables replay-safe instruction execution.
   */
  executedInstructions?: Map<string, ExecutedInstructionRecord>;
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
  storage_state?: SessionSpec['storage_state'];
}

export interface StartSessionResponse {
  session_id: string;
  /** Session phase after creation (always 'ready' for new sessions) */
  phase: SessionPhase;
  /** ISO 8601 timestamp when session was created */
  created_at: string;
  /** Whether this was a reused session (only true if reuse_mode != 'fresh' and match found) */
  reused?: boolean;
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
