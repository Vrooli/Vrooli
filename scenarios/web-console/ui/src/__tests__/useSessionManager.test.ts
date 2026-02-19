import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useSessionManager } from "../hooks/useSessionManager";

// Mock the API module
vi.mock("../lib/api", () => ({
  createSession: vi.fn(),
  deleteSession: vi.fn(),
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
  let mockDeleteSession: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    vi.clearAllMocks();
    const api = await import("../lib/api");
    mockCreateSession = api.createSession as ReturnType<typeof vi.fn>;
    mockDeleteSession = api.deleteSession as ReturnType<typeof vi.fn>;
  });

  it("starts with empty panes", () => {
    const { result } = renderHook(() => useSessionManager());
    expect(result.current.panes).toEqual([]);
    expect(result.current.isCreating).toBe(false);
    expect(result.current.createError).toBeNull();
    expect(result.current.activePane).toBeNull();
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
    expect(result.current.activePane).toBe("sess-1");
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

  it("sets activePane on launch", async () => {
    const sess1 = { id: "s1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    const sess2 = { id: "s2", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(sess1).mockResolvedValueOnce(sess2);

    const { result } = renderHook(() => useSessionManager());

    await act(async () => { await result.current.launchSession(); });
    expect(result.current.activePane).toBe("s1");

    await act(async () => { await result.current.launchSession(); });
    expect(result.current.activePane).toBe("s2");
  });

  it("allows manual activePane switching", async () => {
    const sess1 = { id: "s1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    const sess2 = { id: "s2", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(sess1).mockResolvedValueOnce(sess2);

    const { result } = renderHook(() => useSessionManager());

    await act(async () => { await result.current.launchSession(); });
    await act(async () => { await result.current.launchSession(); });

    act(() => { result.current.setActivePane("s1"); });
    expect(result.current.activePane).toBe("s1");
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

  it("removePane clears activePane if removing active session", async () => {
    const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(mockSession);
    mockDeleteSession.mockResolvedValueOnce(undefined);

    const { result } = renderHook(() => useSessionManager());

    await act(async () => { await result.current.launchSession(); });
    expect(result.current.activePane).toBe("sess-1");

    await act(async () => { await result.current.removePane("sess-1"); });
    expect(result.current.activePane).toBeNull();
  });
});
