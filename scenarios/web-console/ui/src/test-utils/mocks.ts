/**
 * Shared test doubles and mock factories.
 *
 * Centralizes mocks that were previously duplicated across test files,
 * providing a single source of truth for test infrastructure.
 */
import { vi } from "vitest";
import type { TerminalMessage, SocketFactory } from "../hooks/useTerminalSocket";

// ---------------------------------------------------------------------------
// @vrooli/api-base mock factory
// ---------------------------------------------------------------------------

/**
 * Returns the vi.mock factory object for @vrooli/api-base.
 *
 * Usage at the top of a test file (must be hoisted):
 *   vi.mock("@vrooli/api-base", () => apiBaseMock());
 *
 * Uses a deterministic fake URL so fetch assertions are stable.
 * The URL is intentionally non-routable (RFC 2606 .invalid TLD)
 * to avoid hardcoded localhost:PORT audit violations.
 */
export function apiBaseMock() {
  const apiBase = "http://test-api.invalid/api/v1";
  const wsBase = "ws://test-api.invalid/ws";
  return {
    resolveApiBase: () => apiBase,
    buildApiUrl: (path: string, opts: { baseUrl: string }) =>
      `${opts.baseUrl}${path}`,
    resolveWsBase: () => wsBase,
    buildWsUrl: (path: string, opts: { baseUrl: string }) =>
      `${opts.baseUrl}${path}`,
  };
}

// ---------------------------------------------------------------------------
// FakeWebSocket — lightweight WebSocket test double
// ---------------------------------------------------------------------------

/**
 * Minimal fake WebSocket that mirrors the subset of the real API
 * used by useTerminalSocket. The test controls the lifecycle via
 * triggerOpen / triggerMessage / triggerClose.
 */
export class FakeWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSED = 3;
  readyState = FakeWebSocket.CONNECTING;
  onopen: ((ev: Event) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;

  sent: string[] = [];
  closed = false;

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.closed = true;
  }

  triggerOpen() {
    this.readyState = FakeWebSocket.OPEN;
    this.onopen?.(new Event("open"));
  }

  triggerMessage(msg: TerminalMessage) {
    this.onmessage?.(
      new MessageEvent("message", { data: JSON.stringify(msg) }),
    );
  }

  triggerClose(code: number) {
    this.readyState = FakeWebSocket.CLOSED;
    this.onclose?.(new CloseEvent("close", { code }));
  }
}

/**
 * Creates a FakeWebSocket and a SocketFactory that returns it.
 * The factory is a vi.fn() so callers can assert on creation URL.
 */
export function createFakeSocketPair(): {
  fakeWs: FakeWebSocket;
  createSocket: SocketFactory;
} {
  const fakeWs = new FakeWebSocket();
  const createSocket = vi.fn(
    () => fakeWs as unknown as WebSocket,
  ) as SocketFactory;
  return { fakeWs, createSocket };
}

// ---------------------------------------------------------------------------
// Mock xterm Terminal
// ---------------------------------------------------------------------------

export interface MockTerminalLine {
  translateToString: ReturnType<typeof vi.fn>;
}

export interface MockTerminal {
  cols: number;
  rows: number;
  write: ReturnType<typeof vi.fn>;
  onData: ReturnType<typeof vi.fn>;
  written: string[];
  /** Simulate user typing in the terminal. */
  simulateInput(data: string): void;
  // Scroll APIs
  scrollLines: ReturnType<typeof vi.fn>;
  scrollToBottom: ReturnType<typeof vi.fn>;
  // Terminal control
  clear: ReturnType<typeof vi.fn>;
  reset: ReturnType<typeof vi.fn>;
  // Selection APIs
  select: ReturnType<typeof vi.fn>;
  selectAll: ReturnType<typeof vi.fn>;
  getSelection: ReturnType<typeof vi.fn>;
  getSelectionPosition: ReturnType<typeof vi.fn>;
  clearSelection: ReturnType<typeof vi.fn>;
  // Focus
  focus: ReturnType<typeof vi.fn>;
  // Buffer
  buffer: {
    active: {
      viewportY: number;
      baseY: number;
      length: number;
      getLine: ReturnType<typeof vi.fn>;
    };
  };
}

/**
 * Creates a minimal xterm Terminal mock with controllable I/O.
 * Captures all write() calls and exposes simulateInput() to fire
 * onData callbacks. Includes scroll, selection, and buffer APIs
 * needed by useTerminalTouch.
 */
export function createMockTerminal(): MockTerminal {
  const written: string[] = [];
  const dataCallbacks: ((data: string) => void)[] = [];
  const mockLine: MockTerminalLine = {
    translateToString: vi.fn().mockReturnValue("hello world test line"),
  };
  return {
    cols: 80,
    rows: 24,
    write: vi.fn((data: string) => written.push(data)),
    onData: vi.fn((cb: (data: string) => void) => {
      dataCallbacks.push(cb);
      return { dispose: vi.fn() };
    }),
    written,
    simulateInput(data: string) {
      for (const cb of dataCallbacks) cb(data);
    },
    scrollLines: vi.fn(),
    scrollToBottom: vi.fn(),
    clear: vi.fn(),
    reset: vi.fn(),
    select: vi.fn(),
    selectAll: vi.fn(),
    getSelection: vi.fn().mockReturnValue(""),
    getSelectionPosition: vi.fn().mockReturnValue(undefined),
    clearSelection: vi.fn(),
    focus: vi.fn(),
    buffer: {
      active: {
        viewportY: 0,
        baseY: 0,
        length: 24,
        getLine: vi.fn().mockReturnValue(mockLine),
      },
    },
  };
}

/**
 * Find a write() call containing `needle` in the first argument.
 * Returns the full string or undefined if not found.
 */
export function findWriteCall(
  mock: ReturnType<typeof vi.fn>,
  needle: string,
): string | undefined {
  const calls = mock.mock.calls as string[][];
  const match = calls.find(
    (c) => typeof c[0] === "string" && c[0].includes(needle),
  );
  return match ? match[0] : undefined;
}

// ---------------------------------------------------------------------------
// Session data factories
// ---------------------------------------------------------------------------

import type { SessionInfo } from "../lib/api";

/**
 * Creates an array of session entries suitable for component props.
 * Each session gets a bash shell and default policy.
 */
export function makeSessions(
  ...ids: string[]
): Array<{ session: SessionInfo }> {
  return ids.map((id) => ({
    session: {
      id,
      shell: "/bin/bash",
      created_at: "2026-01-15T14:30:00Z",
      cols: 80,
      rows: 24,
      policy: { mode: "never" as const },
      busy: false,
    },
  }));
}

/**
 * Creates a single SessionInfo object with optional overrides.
 */
export function createMockSession(
  overrides: Partial<SessionInfo> = {},
): SessionInfo {
  return {
    id: "test-session-id",
    shell: "/bin/bash",
    created_at: "2026-01-15T14:30:00Z",
    cols: 80,
    rows: 24,
    policy: { mode: "never" as const },
    busy: false,
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// Fetch mock helpers
// ---------------------------------------------------------------------------

/**
 * Installs a successful fetch mock that returns the given JSON body.
 */
export function mockFetchSuccess(body: unknown) {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(body),
  }) as typeof fetch;
}

/**
 * Installs a failing fetch mock that returns the given status and JSON body.
 */
export function mockFetchError(status: number, body?: unknown) {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: body !== undefined
      ? () => Promise.resolve(body)
      : () => Promise.reject(new Error("not json")),
  }) as typeof fetch;
}
