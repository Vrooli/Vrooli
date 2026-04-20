import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useSessionManager } from "../hooks/useSessionManager";

// Mock the API module
vi.mock("../lib/api", () => ({
  createSession: vi.fn(),
  listSessions: vi.fn(),
  deleteSession: vi.fn(),
  getWorkspaceLayout: vi.fn().mockResolvedValue({ active_pane: "", panes: [], groups: [] }),
  updateWorkspacePane: vi.fn().mockResolvedValue({}),
  toErrorInfo: vi.fn((err: unknown) => ({
    code: "test_error",
    message: String(err),
    category: "test",
    recovery: "retry",
    retry: false,
  })),
}));

// [REQ:P0-001b] Independent Pane Session Lifecycle
// [REQ:P0-006a] Terminal Launch Flow UI
describe("useSessionManager", () => {
  let mockCreateSession: ReturnType<typeof vi.fn>;
  let mockListSessions: ReturnType<typeof vi.fn>;
  let mockDeleteSession: ReturnType<typeof vi.fn>;
  let consoleErrorSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(async () => {
    vi.clearAllMocks();
    consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const api = await import("../lib/api");
    mockCreateSession = api.createSession as ReturnType<typeof vi.fn>;
    mockListSessions = api.listSessions as ReturnType<typeof vi.fn>;
    mockDeleteSession = api.deleteSession as ReturnType<typeof vi.fn>;
    mockListSessions.mockResolvedValue([]);
  });

  afterEach(() => {
    consoleErrorSpy.mockRestore();
  });

  it("starts with empty panes and not hydrated", () => {
    const { result } = renderHook(() => useSessionManager());
    expect(result.current.panes).toEqual([]);
    expect(result.current.isCreating).toBe(false);
    expect(result.current.createError).toBeNull();
    expect(result.current.isHydrated).toBe(false);
  });

  it("hydrates panes from existing sessions on mount", async () => {
    const existing = [
      { id: "sess-a", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} },
      { id: "sess-b", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} },
    ];
    mockListSessions.mockResolvedValueOnce(existing);

    const { result } = renderHook(() => useSessionManager());

    // Wait for both Promise.allSettled (listSessions + getWorkspaceLayout) to settle
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    expect(result.current.panes).toHaveLength(2);
    expect(result.current.panes[0]?.session.id).toBe("sess-a");
    expect(result.current.isHydrated).toBe(true);
  });

  it("sets isHydrated even when no sessions exist", async () => {
    mockListSessions.mockResolvedValueOnce([]);

    const { result } = renderHook(() => useSessionManager());

    await act(async () => {
      await Promise.resolve();
    });

    expect(result.current.panes).toHaveLength(0);
    expect(result.current.isHydrated).toBe(true);
  });

  it("launches a session and adds a pane", async () => {
    const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(mockSession);

    const { result } = renderHook(() => useSessionManager());

    await act(async () => {
      const session = await result.current.launchSession();
      expect(session).toEqual(mockSession);
    });

    expect(result.current.panes).toHaveLength(1);
    expect(result.current.panes[0]?.session.id).toBe("sess-1");
    expect(result.current.isCreating).toBe(false);
  });

  it("handles launch failure gracefully", async () => {
    mockCreateSession.mockRejectedValueOnce(new Error("connection refused"));

    const { result } = renderHook(() => useSessionManager());

    await act(async () => {
      const session = await result.current.launchSession();
      expect(session).toBeNull();
    });

    expect(result.current.panes).toHaveLength(0);
    expect(result.current.createError).not.toBeNull();
    expect(result.current.isCreating).toBe(false);
  });

  it("removes a pane and calls deleteSession", async () => {
    const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(mockSession);
    mockDeleteSession.mockResolvedValueOnce(undefined);

    const { result } = renderHook(() => useSessionManager());

    await act(async () => {
      await result.current.launchSession();
    });
    expect(result.current.panes).toHaveLength(1);

    await act(async () => {
      await result.current.removePane("sess-1");
    });

    expect(result.current.panes).toHaveLength(0);
    expect(mockDeleteSession).toHaveBeenCalledWith("sess-1");
  });

  it("clears error on demand", async () => {
    mockCreateSession.mockRejectedValueOnce(new Error("fail"));

    const { result } = renderHook(() => useSessionManager());

    await act(async () => {
      await result.current.launchSession();
    });
    expect(result.current.createError).not.toBeNull();

    act(() => {
      result.current.clearError();
    });
    expect(result.current.createError).toBeNull();
  });

  it("adds multiple panes on successive launches", async () => {
    const sess1 = { id: "s1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    const sess2 = { id: "s2", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(sess1).mockResolvedValueOnce(sess2);

    const { result } = renderHook(() => useSessionManager());

    await act(async () => { await result.current.launchSession(); });
    expect(result.current.panes).toHaveLength(1);

    await act(async () => { await result.current.launchSession(); });
    expect(result.current.panes).toHaveLength(2);
  });

  it("removePane handles already-dead sessions", async () => {
    const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(mockSession);
    mockDeleteSession.mockRejectedValueOnce(new Error("session already dead"));

    const { result } = renderHook(() => useSessionManager());

    await act(async () => { await result.current.launchSession(); });

    // Should not throw even if deleteSession fails
    await act(async () => {
      await result.current.removePane("sess-1");
    });
    expect(result.current.panes).toHaveLength(0);
  });

  it("removePane removes pane from list", async () => {
    const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(mockSession);
    mockDeleteSession.mockResolvedValueOnce(undefined);

    const { result } = renderHook(() => useSessionManager());

    await act(async () => { await result.current.launchSession(); });
    expect(result.current.panes).toHaveLength(1);

    await act(async () => { await result.current.removePane("sess-1"); });
    expect(result.current.panes).toHaveLength(0);
  });

  it("flushes queued launch command when ref registers after onReady", async () => {
    const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(mockSession);

    const { result } = renderHook(() => useSessionManager());

    await act(async () => {
      await result.current.launchSession({ command: "echo from-queued-command" });
    });

    // Simulate race: onReady fires before ref registration
    act(() => {
      result.current.handleTerminalReady("sess-1");
    });

    const handle = { sendInput: vi.fn(), focus: vi.fn(), stopTts: vi.fn(), speakText: vi.fn(), speakSequence: vi.fn(), pauseTts: vi.fn(), resumeTts: vi.fn(), seekTts: vi.fn(), setTtsPlaybackRate: vi.fn(), setTtsVolume: vi.fn(), getTtsState: vi.fn(), subscribeInputSettled: vi.fn(() => () => {}), subscribePendingInput: vi.fn(() => () => {}), getPendingInputSnapshot: vi.fn(() => []) };
    act(() => {
      result.current.registerTerminalRef("sess-1", handle);
    });

    expect(handle.sendInput).toHaveBeenCalledWith("echo from-queued-command\n");
  });
});
