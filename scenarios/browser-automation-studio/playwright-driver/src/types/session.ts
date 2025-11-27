import { Page, BrowserContext, Browser, Frame } from 'playwright';

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
}

export interface StartSessionResponse {
  session_id: string;
}

export interface HealthResponse {
  status: 'ok' | 'degraded' | 'error';
  timestamp: string;
  sessions: number;
  version?: string;
}
