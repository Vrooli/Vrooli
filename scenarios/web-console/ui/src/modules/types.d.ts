import type { Terminal } from "xterm";
import type { FitAddon } from "xterm-addon-fit";

declare global {
  interface Window {
    html2canvas?: (element: HTMLElement, options?: Record<string, unknown>) => Promise<HTMLCanvasElement>;
    lucide?: {
      createIcons: () => void;
    };
    __WEB_CONSOLE_DEBUG__?: Record<string, unknown>;
    __WEB_CONSOLE_DIAGNOSTICS__?: Record<string, unknown>;
    __keyboardToolbarMode?: string;
  }

  interface HTMLElement {
    value?: string;
    checked?: boolean;
    disabled?: boolean;
    select?: () => void;
  }

  interface Performance {
    memory?: {
      usedJSHeapSize: number;
      totalJSHeapSize: number;
      jsHeapSizeLimit: number;
    };
  }
}

export type TabPhase = "idle" | "creating" | "running" | "closing" | "closed";
export type TabSocketState = "disconnected" | "connecting" | "open" | "closing" | "error";

export interface SessionDetails {
  id: string;
  command?: string;
  commandLine?: string;
  args?: string[];
  [key: string]: unknown;
}

export interface TabEvent {
  type: string;
  payload: unknown;
  timestamp: number;
}

export interface TranscriptEntry {
  timestamp: number;
  direction: string;
  encoding?: string;
  data?: string;
  message?: string;
  [key: string]: unknown;
}

export interface PendingWrite {
  value: string;
  meta: Record<string, unknown>;
}

export interface TerminalTab {
  id: string;
  label: string;
  defaultLabel: string;
  colorId: string;
  term: Terminal;
  fitAddon: FitAddon | null;
  container: HTMLDivElement;
  phase: string;
  socketState: string;
  session: SessionDetails | null;
  socket: WebSocket | null;
  reconnecting: boolean;
  hasEverConnected: boolean;
  hasReceivedLiveOutput: boolean;
  heartbeatInterval: ReturnType<typeof setInterval> | null;
  inputSeq: number;
  inputBatch: PendingWrite | null;
  inputBatchScheduled: boolean;
  replayPending: boolean;
  replayComplete: boolean;
  lastReplayCount: number;
  lastReplayTruncated: boolean;
  transcriptHydrated: boolean;
  transcriptHydrating: boolean;
  transcript: TranscriptEntry[];
  transcriptByteSize: number;
  events: TabEvent[];
  suppressed: Record<string, number>;
  pendingWrites: PendingWrite[];
  localEchoBuffer: string[];
  errorMessage: string;
  lastSentSize: { cols: number; rows: number };
  telemetry: {
    typed: number;
    queued: number;
    sent: number;
    batches: number;
    lastBatchSize: number;
  };
  domItem: HTMLElement | null;
  domButton: HTMLButtonElement | null;
  domClose: HTMLButtonElement | null;
  domLabel: HTMLSpanElement | null;
  domStatus: HTMLElement | null;
  wasDetached: boolean;
}

export interface WorkspaceTabSnapshot {
  id: string;
  label: string;
  colorId: string;
  order: number;
  sessionId: string | null;
}

export interface WorkspaceFullUpdatePayload {
  activeTabId: string | null;
  tabs: WorkspaceTabSnapshot[];
  timestamp?: string;
}

export interface IframeBridge {
  emit: (type: string, payload: unknown) => void;
}

export interface DrawerState {
  open: boolean;
  unreadCount: number;
  previousFocus: HTMLElement | null;
}

export interface TabMenuState {
  open: boolean;
  tabId: string | null;
  selectedColor: string;
  anchor: { x: number; y: number };
}

export interface ComposerState {
  open: boolean;
  previousFocus: HTMLElement | null;
  appendNewline: boolean;
}

export interface SessionOverviewItem {
  id: string;
  command?: string;
  args?: string[];
  [key: string]: unknown;
}

export interface SessionsState {
  items: SessionOverviewItem[];
  loading: boolean;
  lastFetched: number;
  error: Error | null;
  pollHandle: ReturnType<typeof setInterval> | null;
  refreshTimer: ReturnType<typeof setTimeout> | null;
  needsRefresh: boolean;
  capacity: number | null;
}

export interface WorkspaceUiState {
  loading: boolean;
  idleTimeoutSeconds: number;
  updatingIdleTimeout: boolean;
  idleControlsInitialized: boolean;
}

export interface AppState {
  tabs: TerminalTab[];
  activeTabId: string | null;
  bridge: IframeBridge | null;
  workspaceSocket: WebSocket | null;
  workspaceReconnectTimer: ReturnType<typeof setTimeout> | null;
  drawer: DrawerState;
  tabMenu: TabMenuState;
  composer: ComposerState;
  sessions: SessionsState;
  workspace: WorkspaceUiState;
}

export interface WorkspaceAnomalySummary {
  tabs: WorkspaceTabSnapshot[];
  duplicateIds: string[];
  invalidIds: string[];
}

export interface WorkspaceAnomalyReport {
  duplicateIds?: string[];
  invalidIds?: string[];
  localDuplicates?: string[];
}

export interface WorkspaceEventEnvelope {
  type: string;
  payload?: unknown;
  activeTabId?: string | null;
  tabs?: WorkspaceTabSnapshot[];
}
