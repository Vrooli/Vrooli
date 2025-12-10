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
}

export interface HealthResponse {
  status: 'ok' | 'degraded' | 'error';
  timestamp: string;
  sessions: number;
  /** Number of sessions with active recording */
  active_recordings?: number;
  version?: string;
  browser?: {
    healthy: boolean;
    version?: string;
    error?: string;
  };
  /** Server uptime in milliseconds */
  uptime_ms?: number;
}
