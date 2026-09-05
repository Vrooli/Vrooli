import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useSessionManager } from "../hooks/useSessionManager";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import type { SessionInfo } from "../api/sessions";

// Minimal SessionInfo builder for the external-merge tests.
function extSession(id: string, over: Partial<SessionInfo> = {}): SessionInfo {
  return {
    id,
    shell: "/bin/bash",
    created_at: "2026-07-12T00:00:00Z",
    cols: 80,
    rows: 24,
    backend: "standard",
    survives_restart: false,
    policy: { mode: "never" },
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
  archiveSession: vi.fn(),
  unarchiveSession: vi.fn(),
  deleteSession: vi.fn(),
}));

vi.mock("../api/workspace", () => ({
  getWorkspaceLayout: vi.fn().mockResolvedValue({ active_pane: "", panes: [], groups: [], roles: [] }),
  updateWorkspacePane: vi.fn().mockResolvedValue({}),
}));

vi.mock("../lib/errors", () => ({
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
  let mockArchiveSession: ReturnType<typeof vi.fn>;
  let mockUnarchiveSession: ReturnType<typeof vi.fn>;
  let consoleErrorSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(async () => {
    vi.clearAllMocks();
    consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const api = await import("../api/sessions");
    mockCreateSession = api.createSession as ReturnType<typeof vi.fn>;
    mockListSessions = api.listSessions as ReturnType<typeof vi.fn>;
    mockDeleteSession = api.deleteSession as ReturnType<typeof vi.fn>;
    mockArchiveSession = api.archiveSession as ReturnType<typeof vi.fn>;
    mockUnarchiveSession = api.unarchiveSession as ReturnType<typeof vi.fn>;
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

  it("does not publish hydration results after the hook unmounts", async () => {
    let resolveList: (value: SessionInfo[]) => void = () => {};
    mockListSessions.mockImplementationOnce(() => new Promise<SessionInfo[]>((resolve) => { resolveList = resolve; }));
    const { result, unmount } = renderHook(() => useSessionManager());

    unmount();
    await act(async () => {
      resolveList([extSession("late")]);
      await Promise.resolve();
    });

    expect(result.current.isHydrated).toBe(false);
  });

  it("falls back to session-derived pane metadata when layout is unavailable", async () => {
    const api = await import("../api/workspace");
    const mockGetWorkspaceLayout = api.getWorkspaceLayout as ReturnType<typeof vi.fn>;
    mockGetWorkspaceLayout.mockResolvedValueOnce(null);
    mockListSessions.mockResolvedValueOnce([extSession("fallback", { shell: "/bin/zsh" })]);
    useWorkspaceStore.setState({ panes: [], activePane: null });

    const { result } = renderHook(() => useSessionManager());
    await flushHydration();

    expect(result.current.panes[0]?.session.id).toBe("fallback");
    expect(useWorkspaceStore.getState().panes[0]?.name).toBe("zsh");
  });

  it("maps layout metadata and supplies defaults for sessions missing a pane", async () => {
    const api = await import("../api/workspace");
    const mockGetWorkspaceLayout = api.getWorkspaceLayout as ReturnType<typeof vi.fn>;
    mockListSessions.mockResolvedValueOnce([
      extSession("known"),
      extSession("missing", { shell: "/bin/fish" }),
    ]);
    mockGetWorkspaceLayout.mockResolvedValueOnce({
      active_pane: "",
      panes: [{
        session_id: "known",
        name: "Known pane",
        header_color: "#7aa0ff",
        theme_id: "dracula",
        font_size: 16,
        group_id: null,
        supports_messages_view: undefined,
        manually_unread: undefined,
      }],
      groups: [],
      roles: [],
    });
    useWorkspaceStore.setState({ panes: [], activePane: null });

    const { result } = renderHook(() => useSessionManager());
    await flushHydration();

    expect(result.current.panes.map((pane) => pane.session.id).sort()).toEqual(["known", "missing"]);
    expect(useWorkspaceStore.getState().panes.map((pane) => pane.sessionId).sort()).toEqual(["known", "missing"]);
  });

  it("surfaces hydrationError when both hydration calls fail", async () => {
    // Simulate the exact regression that prompted this safety net:
    // both calls reject (e.g. proxy misroute, network drop). The hook
    // must log each failure and set a user-visible hydrationError with
    // retry=true so the empty catch can't hide it again.
    const api = await import("../api/workspace");
    const mockGetWorkspaceLayout = api.getWorkspaceLayout as ReturnType<typeof vi.fn>;
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
    expect(consoleErrorSpy).toHaveBeenCalledWith(
      "hydratePanes: listSessions failed",
      expect.any(Error),
    );
    expect(consoleErrorSpy).toHaveBeenCalledWith(
      "hydratePanes: getWorkspaceLayout failed",
      expect.any(Error),
    );
  });

  it("logs a single-call hydration failure without setting hydrationError", async () => {
    // Layout fetch fails, sessions fetch succeeds — the fallback path can
    // still render. We log but do not surface a banner.
    const api = await import("../api/workspace");
    const mockGetWorkspaceLayout = api.getWorkspaceLayout as ReturnType<typeof vi.fn>;
    mockListSessions.mockResolvedValueOnce([]);
    mockGetWorkspaceLayout.mockRejectedValueOnce(new Error("layout 503"));

    const { result } = renderHook(() => useSessionManager());

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    expect(result.current.isHydrated).toBe(true);
    expect(result.current.hydrationError).toBeNull();
    expect(consoleErrorSpy).toHaveBeenCalledWith(
      "hydratePanes: getWorkspaceLayout failed",
      expect.any(Error),
    );
  });

  it("clearHydrationError dismisses the error", async () => {
    const api = await import("../api/workspace");
    const mockGetWorkspaceLayout = api.getWorkspaceLayout as ReturnType<typeof vi.fn>;
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

  it("archives a pane without calling permanent delete", async () => {
    const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(mockSession);
    mockArchiveSession.mockResolvedValueOnce(undefined);

    const { result } = renderHook(() => useSessionManager());

    await act(async () => {
      await result.current.launchSession();
    });
    expect(result.current.panes).toHaveLength(1);

    await act(async () => {
      await result.current.removePane("sess-1");
    });

    expect(result.current.panes).toHaveLength(0);
    expect(mockArchiveSession).toHaveBeenCalledWith("sess-1");
    expect(mockDeleteSession).not.toHaveBeenCalled();
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

  it("keeps the pane visible when archive fails", async () => {
    const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(mockSession);
    mockArchiveSession.mockRejectedValueOnce(new Error("session already dead"));

    const { result } = renderHook(() => useSessionManager());

    await act(async () => { await result.current.launchSession(); });

    let outcome: string | undefined;
    await act(async () => {
      outcome = await result.current.removePane("sess-1");
    });
    expect(outcome).toBe("failed");
    expect(result.current.panes).toHaveLength(1);
  });

  it("removePane removes pane from list", async () => {
    const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(mockSession);
    mockArchiveSession.mockResolvedValueOnce(undefined);

    const { result } = renderHook(() => useSessionManager());

    await act(async () => { await result.current.launchSession(); });
    expect(result.current.panes).toHaveLength(1);

    await act(async () => { await result.current.removePane("sess-1"); });
    expect(result.current.panes).toHaveLength(0);
  });

  it("undoes an archive without creating or deleting a session", async () => {
    const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(mockSession);
    mockArchiveSession.mockResolvedValueOnce(undefined);
    mockUnarchiveSession.mockResolvedValueOnce(undefined);
    const { result } = renderHook(() => useSessionManager());

    await act(async () => { await result.current.launchSession(); });
    await act(async () => { await result.current.removePane("sess-1"); });
    await act(async () => { expect(await result.current.undoArchive("sess-1")).toBe(true); });

    expect(result.current.panes).toHaveLength(1);
    expect(mockUnarchiveSession).toHaveBeenCalledWith("sess-1");
    expect(mockDeleteSession).not.toHaveBeenCalled();
  });

  it("launches with execute_launch_command so the server runs the command once", async () => {
    const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(mockSession);

    const { result } = renderHook(() => useSessionManager());

    await act(async () => {
      await result.current.launchSession({ command: "codex --yolo" });
    });

    expect(mockCreateSession).toHaveBeenCalledWith(
      expect.objectContaining({
        launch_command: "codex --yolo",
        execute_launch_command: true,
        agent_type: "codex",
        idempotency_key: expect.stringMatching(/^ui-session-/),
      }),
    );
  });

  it("passes remote target, working directory, and replay key to the typed create call", async () => {
    const mockSession = { id: "remote-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(mockSession);

    const { result } = renderHook(() => useSessionManager());

    await act(async () => {
      await result.current.launchSession({
        command: "codex login --device-auth",
        target: {
          id: "bridge-node:build-a",
          kind: "bridge-node",
          label: "Build node A",
          available: true,
          state: "dispatchable",
        },
        workingDir: "/workspaces/project",
      });
    });

    expect(mockCreateSession).toHaveBeenCalledWith(expect.objectContaining({
      target_id: "bridge-node:build-a",
      working_dir: "/workspaces/project",
      execute_launch_command: true,
      idempotency_key: expect.stringMatching(/^ui-session-/),
    }));
  });

  it("does not request server execution when launching without a command", async () => {
    const mockSession = { id: "sess-1", shell: "/bin/bash", cols: 80, rows: 24, created_at: "2026-01-01T00:00:00Z", policy: {} };
    mockCreateSession.mockResolvedValueOnce(mockSession);

    const { result } = renderHook(() => useSessionManager());

    await act(async () => {
      await result.current.launchSession();
    });

    expect(mockCreateSession).toHaveBeenCalledWith(
      expect.objectContaining({ execute_launch_command: false }),
    );
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
    let resolveList: (v: SessionInfo[]) => void = () => {};
    mockListSessions.mockImplementationOnce(() => new Promise<SessionInfo[]>((res) => { resolveList = res; }));

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

const handle = {
  input: { submit: vi.fn(() => ({ status: "sent" as const, offset: 1 })), subscribeSettled: vi.fn(() => () => {}), awaitOffset: vi.fn(() => () => {}) },
  control: { send: vi.fn(() => true), scroll: vi.fn(), focus: vi.fn() },
  selection: { copy: vi.fn(async () => true), paste: vi.fn(async () => true) },
  pendingInput: { subscribe: vi.fn(() => () => {}), snapshot: vi.fn(() => []), discard: vi.fn(), discardAll: vi.fn(), flushNow: vi.fn() },
  playback: { stop: vi.fn(), speak: vi.fn(), pause: vi.fn(), resume: vi.fn(), seek: vi.fn(), setPlaybackRate: vi.fn(), setVolume: vi.fn(), setMuted: vi.fn(), getState: vi.fn() },
};
    act(() => {
      result.current.registerTerminalRef("sess-1", handle);
    });

    // The server pastes the launch command into the PTY; the client must not
    // also type it, or the command would execute twice.
    expect(handle.input.submit).not.toHaveBeenCalled();
  });

  it("handles missing panes, failed undo/delete, and the terminal facade branches", async () => {
    const { result } = renderHook(() => useSessionManager());
    await flushHydration();

    expect(await result.current.removePane("missing")).toBe("failed");
    expect(await result.current.undoArchive("missing")).toBe(false);
    act(() => result.current.endExternalSession("missing"));
    expect(result.current.submitToActiveTerminal("x", "typing")).toEqual({ status: "rejected", reason: "disposed" });
    result.current.subscribeActiveInputSettled(undefined, vi.fn())();
    result.current.awaitActiveInputOffset(undefined, 1, vi.fn())();
    result.current.subscribeActivePendingInput(undefined, vi.fn())();
    expect(result.current.getActivePendingInputSnapshot()).toEqual([]);
    await expect(result.current.copySelectionOnPane()).resolves.toBe(false);
    await expect(result.current.pasteFromClipboardOnPane()).resolves.toBe(false);
    result.current.scrollTerminalOnPane(1);
    result.current.focusActiveTerminal();
    result.current.stopActiveTts();
    await expect(result.current.speakTextOnPane("missing", "hello")).resolves.toBeUndefined();
    result.current.pauseTtsOnPane("missing");
    result.current.resumeTtsOnPane("missing");
    result.current.seekTtsOnPane("missing", 1);
    result.current.setTtsPlaybackRateOnPane("missing", 1.25);
    result.current.setTtsVolumeOnPane("missing", 0.5);
    result.current.setTtsMutedOnPane("missing", true);
    expect(result.current.getTtsStateOnPane("missing")).toBeNull();

    // An explicit but stale pane id must be a safe no-op for every facade. This
    // is the path used after a pane closes while an async toolbar action is
    // still resolving.
    result.current.subscribeActiveInputSettled("missing", vi.fn())();
    result.current.awaitActiveInputOffset("missing", 1, vi.fn())();
    result.current.subscribeActivePendingInput("missing", vi.fn())();
    expect(result.current.getActivePendingInputSnapshot("missing")).toEqual([]);
    await expect(result.current.copySelectionOnPane("missing")).resolves.toBe(false);
    await expect(result.current.pasteFromClipboardOnPane("missing")).resolves.toBe(false);
    result.current.scrollTerminalOnPane(1, "missing");
    result.current.focusActiveTerminal("missing");
    result.current.stopActiveTts("missing");
    act(() => result.current.registerTerminalRef("missing", null));

    const session = extSession("facade");
    mockCreateSession.mockResolvedValueOnce(session);
    await act(async () => { await result.current.launchSession(); });
    const handle = {
      input: {
        submit: vi.fn(() => ({ status: "sent" as const, offset: 4 })),
        subscribeSettled: vi.fn(() => () => {}),
        awaitOffset: vi.fn(() => () => {}),
      },
      control: { send: vi.fn(() => true), scroll: vi.fn(), focus: vi.fn() },
      selection: { copy: vi.fn().mockResolvedValue(true), paste: vi.fn().mockResolvedValue(true) },
      pendingInput: { subscribe: vi.fn(() => () => {}), snapshot: vi.fn(() => [{ data: "x", addedAt: 1, intent: "typing" as const }]), discard: vi.fn(), discardAll: vi.fn(), flushNow: vi.fn() },
      playback: {
        stop: vi.fn(), speak: vi.fn().mockResolvedValue("ok"), pause: vi.fn(), resume: vi.fn(), seek: vi.fn(),
        setPlaybackRate: vi.fn(), setVolume: vi.fn(), setMuted: vi.fn(), getState: vi.fn().mockReturnValue({ playing: true }),
      },
    };
    act(() => result.current.registerTerminalRef("facade", handle));
    expect(result.current.submitToActiveTerminal("x", "typing")).toEqual({ status: "sent", offset: 4 });
    expect(result.current.getActivePendingInputSnapshot()).toEqual([{ data: "x", addedAt: 1, intent: "typing" }]);
    await expect(result.current.copySelectionOnPane()).resolves.toBe(true);
    await expect(result.current.pasteFromClipboardOnPane()).resolves.toBe(true);
    result.current.scrollTerminalOnPane(2);
    result.current.focusActiveTerminal();
    result.current.stopActiveTts();
    await expect(result.current.speakTextOnPane("facade", "hello")).resolves.toBe("ok");
    result.current.pauseTtsOnPane("facade");
    result.current.resumeTtsOnPane("facade");
    result.current.seekTtsOnPane("facade", 2);
    result.current.setTtsPlaybackRateOnPane("facade", 1.5);
    result.current.setTtsVolumeOnPane("facade", 0.75);
    result.current.setTtsMutedOnPane("facade", false);
    expect(result.current.getTtsStateOnPane("facade")).toEqual({ playing: true });
    expect(handle.control.scroll).toHaveBeenCalledWith(2);
    expect(handle.control.focus).toHaveBeenCalled();

    mockArchiveSession.mockResolvedValueOnce(undefined);
    mockUnarchiveSession.mockRejectedValueOnce(new Error("archive race"));
    await act(async () => { await result.current.removePane("facade"); });
    await expect(result.current.undoArchive("facade")).resolves.toBe(false);

    mockCreateSession.mockResolvedValueOnce(extSession("delete-me"));
    await act(async () => { await result.current.launchSession(); });
    mockDeleteSession.mockRejectedValueOnce(new Error("already deleted"));
    await act(async () => {
      await expect(result.current.deletePanePermanently("delete-me")).rejects.toThrow("already deleted");
    });
    expect(result.current.panes.some((pane) => pane.session.id === "delete-me")).toBe(false);
  });

  it("rejects a concurrent launch while the first create is in flight", async () => {
    let resolveCreate: (session: SessionInfo) => void = () => {};
    mockCreateSession.mockImplementationOnce(() => new Promise<SessionInfo>((resolve) => { resolveCreate = resolve; }));
    const { result } = renderHook(() => useSessionManager());
    let first: Promise<SessionInfo | null> | undefined;
    await act(async () => {
      first = result.current.launchSession();
      expect(await result.current.launchSession()).toBeNull();
    });
    resolveCreate(extSession("first"));
    await act(async () => { await first; });
    expect(result.current.panes.map((pane) => pane.session.id)).toEqual(["first"]);
  });

  // Roles ride along in the layout response and nothing else in the client
  // populates them. Before this, a waiting role was created server-side,
  // stored, and then silently dropped on every reload — it read as a feature
  // that had never been persisted at all.
  it("hydrates waiting roles from the layout", async () => {
    const api = await import("../api/workspace");
    const mockGetWorkspaceLayout = api.getWorkspaceLayout as ReturnType<typeof vi.fn>;
    mockListSessions.mockResolvedValueOnce([extSession("running")]);
    mockGetWorkspaceLayout.mockResolvedValueOnce({
      active_pane: "running",
      panes: [],
      groups: [{ id: "g1", name: "Test", color: "#22d3ee", is_collapsed: false }],
      roles: [
        { id: "r1", group_id: "g1", label: "Planner", command: "claude", working_dir: "", incoming_prompt: "", backend: "", target_id: "", session_id: "running", sort_order: 0 },
        { id: "r2", group_id: "g1", label: "Implementer", command: "codex --yolo", working_dir: "", incoming_prompt: "Implement {{payload}}", backend: "", target_id: "", session_id: null, sort_order: 1 },
      ],
    });
    useWorkspaceStore.setState({ panes: [], activePane: null, roles: [] });

    renderHook(() => useSessionManager());
    await flushHydration();

    const roles = useWorkspaceStore.getState().roles;
    expect(roles.map((role) => role.label)).toEqual(["Planner", "Implementer"]);
    expect(roles[1]).toMatchObject({ sessionId: null, incomingPrompt: "Implement {{payload}}" });
  });

  // A role attached to a session that no longer exists is a placeholder, not
  // a running position: a handoff aimed at that session id would go nowhere.
  it("returns a role whose session is gone to waiting", async () => {
    const api = await import("../api/workspace");
    const mockGetWorkspaceLayout = api.getWorkspaceLayout as ReturnType<typeof vi.fn>;
    mockListSessions.mockResolvedValueOnce([extSession("alive")]);
    mockGetWorkspaceLayout.mockResolvedValueOnce({
      active_pane: "alive",
      panes: [],
      groups: [{ id: "g1", name: "Test", color: "#22d3ee", is_collapsed: false }],
      roles: [
        { id: "r1", group_id: "g1", label: "Ghost", command: "claude", working_dir: "", incoming_prompt: "", backend: "", target_id: "", session_id: "reaped", sort_order: 0 },
      ],
    });
    useWorkspaceStore.setState({ panes: [], activePane: null, roles: [] });

    renderHook(() => useSessionManager());
    await flushHydration();

    expect(useWorkspaceStore.getState().roles[0]).toMatchObject({ label: "Ghost", sessionId: null });
  });

  it("keeps a role created while hydration was in flight", async () => {
    const api = await import("../api/workspace");
    const mockGetWorkspaceLayout = api.getWorkspaceLayout as ReturnType<typeof vi.fn>;
    mockListSessions.mockResolvedValueOnce([]);
    mockGetWorkspaceLayout.mockResolvedValueOnce({
      active_pane: "",
      panes: [],
      groups: [{ id: "g1", name: "Test", color: "#22d3ee", is_collapsed: false }],
      roles: [
        { id: "r1", group_id: "g1", label: "Stored", command: "claude", working_dir: "", incoming_prompt: "", backend: "", target_id: "", session_id: null, sort_order: 0 },
      ],
    });
    useWorkspaceStore.setState({
      panes: [],
      activePane: null,
      roles: [{ id: "local", groupId: "g1", label: "Local", command: "codex", workingDir: "", incomingPrompt: "", backend: "", targetId: "", sessionId: null, sortOrder: 1 }],
    });

    renderHook(() => useSessionManager());
    await flushHydration();

    expect(useWorkspaceStore.getState().roles.map((role) => role.label)).toEqual(["Stored", "Local"]);
  });
});
