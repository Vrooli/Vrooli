import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useSessionManager } from "../hooks/useSessionManager";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
// Minimal SessionInfo builder for the external-merge tests.
function extSession(id, over = {}) {
    return {
        id,
        shell: "/bin/bash",
        created_at: "2026-07-12T00:00:00Z",
        cols: 80,
        rows: 24,
        backend: "standard",
        survives_restart: false,
        policy: { mode: "never" },
        busy: false,
        origin: "programmatic",
        owner: "",
        display_label: "",
        ...over,
    };
}
// Settle the mount-time hydration (listSessions + getWorkspaceLayout).
async function flushHydration() {
    await act(async () => {
        await new Promise((r) => setTimeout(r, 0));
    });
}
// Mock the API modules
vi.mock("../api/sessions", () => ({
    createSession: vi.fn(),
    listSessions: vi.fn(),
    deleteSession: vi.fn(),
}));
vi.mock("../api/workspace", () => ({
    getWorkspaceLayout: vi.fn().mockResolvedValue({ active_pane: "", panes: [], groups: [] }),
    updateWorkspacePane: vi.fn().mockResolvedValue({}),
}));
vi.mock("../lib/errors", () => ({
    toErrorInfo: vi.fn((err) => ({
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
    let mockCreateSession;
    let mockListSessions;
    let mockDeleteSession;
    let consoleErrorSpy;
    beforeEach(async () => {
        vi.clearAllMocks();
        consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => { });
        const api = await import("../api/sessions");
        mockCreateSession = api.createSession;
        mockListSessions = api.listSessions;
        mockDeleteSession = api.deleteSession;
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
    it("surfaces hydrationError when both hydration calls fail", async () => {
        // Simulate the exact regression that prompted this safety net:
        // both calls reject (e.g. proxy misroute, network drop). The hook
        // must log each failure and set a user-visible hydrationError with
        // retry=true so the empty catch can't hide it again.
        const api = await import("../api/workspace");
        const mockGetWorkspaceLayout = api.getWorkspaceLayout;
        mockListSessions.mockRejectedValueOnce(new Error("sessions 404"));
        mockGetWorkspaceLayout.mockRejectedValueOnce(new Error("layout 404"));
        const { result } = renderHook(() => useSessionManager());
        await act(async () => {
            await new Promise((r) => setTimeout(r, 0));
        });
        expect(result.current.isHydrated).toBe(true);
        expect(result.current.panes).toHaveLength(0);
        expect(result.current.hydrationError).not.toBeNull();
        expect(result.current.hydrationError?.retry).toBe(true);
        expect(consoleErrorSpy).toHaveBeenCalledWith("hydratePanes: listSessions failed", expect.any(Error));
        expect(consoleErrorSpy).toHaveBeenCalledWith("hydratePanes: getWorkspaceLayout failed", expect.any(Error));
    });
    it("logs a single-call hydration failure without setting hydrationError", async () => {
        // Layout fetch fails, sessions fetch succeeds — the fallback path can
        // still render. We log but do not surface a banner.
        const api = await import("../api/workspace");
        const mockGetWorkspaceLayout = api.getWorkspaceLayout;
        mockListSessions.mockResolvedValueOnce([]);
        mockGetWorkspaceLayout.mockRejectedValueOnce(new Error("layout 503"));
        const { result } = renderHook(() => useSessionManager());
        await act(async () => {
            await new Promise((r) => setTimeout(r, 0));
        });
        expect(result.current.isHydrated).toBe(true);
        expect(result.current.hydrationError).toBeNull();
        expect(consoleErrorSpy).toHaveBeenCalledWith("hydratePanes: getWorkspaceLayout failed", expect.any(Error));
    });
    it("clearHydrationError dismisses the error", async () => {
        const api = await import("../api/workspace");
        const mockGetWorkspaceLayout = api.getWorkspaceLayout;
        mockListSessions.mockRejectedValueOnce(new Error("sessions fail"));
        mockGetWorkspaceLayout.mockRejectedValueOnce(new Error("layout fail"));
        const { result } = renderHook(() => useSessionManager());
        await act(async () => {
            await new Promise((r) => setTimeout(r, 0));
        });
        expect(result.current.hydrationError).not.toBeNull();
        act(() => {
            result.current.clearHydrationError();
        });
        expect(result.current.hydrationError).toBeNull();
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
    it("launches with execute_launch_command so the server runs the command once", async () => {
        const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
        mockCreateSession.mockResolvedValueOnce(mockSession);
        const { result } = renderHook(() => useSessionManager());
        await act(async () => {
            await result.current.launchSession({ command: "codex --yolo" });
        });
        expect(mockCreateSession).toHaveBeenCalledWith(expect.objectContaining({
            launch_command: "codex --yolo",
            execute_launch_command: true,
            agent_type: "codex",
        }));
    });
    it("does not request server execution when launching without a command", async () => {
        const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
        mockCreateSession.mockResolvedValueOnce(mockSession);
        const { result } = renderHook(() => useSessionManager());
        await act(async () => {
            await result.current.launchSession();
        });
        expect(mockCreateSession).toHaveBeenCalledWith(expect.objectContaining({ execute_launch_command: false }));
    });
    // --- External session lifecycle merge (SSE-driven, Phase 3) ---
    it("mergeExternalSession adds an externally created session into the pane list", async () => {
        const { result } = renderHook(() => useSessionManager());
        await flushHydration();
        expect(result.current.isHydrated).toBe(true);
        act(() => {
            result.current.mergeExternalSession(extSession("ext-1", { origin: "programmatic" }), true);
        });
        expect(result.current.panes).toHaveLength(1);
        expect(result.current.panes[0]?.session.id).toBe("ext-1");
        expect(result.current.panes[0]?.session.origin).toBe("programmatic");
        expect(result.current.panes[0]?.supportsMessagesView).toBe(true);
    });
    it("mergeExternalSession is a no-op for a session already present (self-origination dedupe)", async () => {
        const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
        mockCreateSession.mockResolvedValueOnce(mockSession);
        const { result } = renderHook(() => useSessionManager());
        await flushHydration();
        // This client creates the session (optimistic append).
        await act(async () => { await result.current.launchSession(); });
        expect(result.current.panes).toHaveLength(1);
        // The session.created echo for our own create arrives over SSE — must not
        // double-add.
        act(() => {
            result.current.mergeExternalSession(extSession("sess-1"), false);
        });
        expect(result.current.panes).toHaveLength(1);
    });
    it("endExternalSession removes an externally deleted session from the pane list", async () => {
        const { result } = renderHook(() => useSessionManager());
        await flushHydration();
        act(() => { useWorkspaceStore.setState({ activePane: null }); });
        act(() => { result.current.mergeExternalSession(extSession("ext-1"), false); });
        expect(result.current.panes).toHaveLength(1);
        act(() => { result.current.endExternalSession("ext-1"); });
        expect(result.current.panes).toHaveLength(0);
    });
    it("endExternalSession spares the focused pane rather than yanking the user's view", async () => {
        const { result } = renderHook(() => useSessionManager());
        await flushHydration();
        act(() => { result.current.mergeExternalSession(extSession("ext-focus"), false); });
        act(() => { useWorkspaceStore.setState({ activePane: "ext-focus" }); });
        act(() => { result.current.endExternalSession("ext-focus"); });
        // Still present: the terminal WS shows disconnected; the user closes it.
        expect(result.current.panes).toHaveLength(1);
        expect(result.current.panes[0]?.session.id).toBe("ext-focus");
        act(() => { useWorkspaceStore.setState({ activePane: null }); });
    });
    it("preserves hydrate-once and buffers a pre-hydration external create until flush", async () => {
        // Hydration is in-flight: hold listSessions unresolved so isHydrated is false.
        let resolveList = () => { };
        mockListSessions.mockImplementationOnce(() => new Promise((res) => { resolveList = res; }));
        const { result } = renderHook(() => useSessionManager());
        expect(result.current.isHydrated).toBe(false);
        // External create lands BEFORE hydration completes — must buffer, not apply.
        act(() => { result.current.mergeExternalSession(extSession("ext-early"), false); });
        expect(result.current.panes).toHaveLength(0);
        // Hydration returns the authoritative list; it must NOT be clobbered, and the
        // buffered external create flushes on top of it.
        await act(async () => {
            resolveList([extSession("hydrated-a")]);
            await new Promise((r) => setTimeout(r, 0));
        });
        const ids = result.current.panes.map((p) => p.session.id).sort();
        expect(ids).toEqual(["ext-early", "hydrated-a"]);
        expect(result.current.isHydrated).toBe(true);
    });
    it("a merge leaves existing panes' identity and order untouched (live state survives)", async () => {
        const { result } = renderHook(() => useSessionManager());
        await flushHydration();
        act(() => { result.current.mergeExternalSession(extSession("a"), false); });
        act(() => { result.current.mergeExternalSession(extSession("b"), false); });
        const before = result.current.panes;
        const paneA = before[0];
        const paneB = before[1];
        act(() => { result.current.mergeExternalSession(extSession("c"), false); });
        const after = result.current.panes;
        // Prior panes keep object identity (no re-creation) and position; only the
        // new one is appended.
        expect(after[0]).toBe(paneA);
        expect(after[1]).toBe(paneB);
        expect(after.map((p) => p.session.id)).toEqual(["a", "b", "c"]);
    });
    it("never types the launch command client-side (no double execution on ref register)", async () => {
        const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
        mockCreateSession.mockResolvedValueOnce(mockSession);
        const { result } = renderHook(() => useSessionManager());
        await act(async () => {
            await result.current.launchSession({ command: "echo hi" });
        });
        const handle = { submitInput: vi.fn(() => ({ status: "sent", seq: 1 })), focus: vi.fn(), stopTts: vi.fn(), speakText: vi.fn(), speakSequence: vi.fn(), pauseTts: vi.fn(), resumeTts: vi.fn(), seekTts: vi.fn(), setTtsPlaybackRate: vi.fn(), setTtsVolume: vi.fn(), setTtsMuted: vi.fn(), getTtsState: vi.fn(), subscribeInputSettled: vi.fn(() => () => { }), subscribePendingInput: vi.fn(() => () => { }), getPendingInputSnapshot: vi.fn(() => []) };
        act(() => {
            result.current.registerTerminalRef("sess-1", handle);
        });
        // The server pastes the launch command into the PTY; the client must not
        // also type it, or the command would execute twice.
        expect(handle.submitInput).not.toHaveBeenCalled();
    });
});
