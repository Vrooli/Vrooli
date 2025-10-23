// @ts-check

import { describe, expect, it, beforeEach, afterEach, vi } from "vitest";

vi.mock("../../tab-manager.js", () => {
  return {
    createTerminalTab: vi.fn(),
    findTab: vi.fn(),
    setActiveTab: vi.fn(),
    destroyTerminalTab: vi.fn(),
    renderTabs: vi.fn(),
    applyTabAppearance: vi.fn(),
    getActiveTab: vi.fn(() => null),
    refreshTabButton: vi.fn(),
    getDetachedTabs: vi.fn(() => []),
    getTabs: vi.fn(() => []),
  };
});

vi.mock("../../event-feed.js", () => {
  return {
    appendEvent: vi.fn(),
  };
});

vi.mock("../../utils.js", () => {
  return {
    proxyToApi: vi.fn(() => Promise.resolve({ ok: true })),
    scheduleMicrotask: /** @type {(cb: () => void) => void} */ ((cb) => cb()),
  };
});

vi.mock("../../notifications.js", () => {
  return {
    showToast: vi.fn(() => Promise.resolve()),
  };
});

vi.mock("../../session-service.js", () => {
  return {
    reconnectSession: vi.fn(() => Promise.resolve()),
    updateSessionActions: vi.fn(),
    setTabPhase: vi.fn(),
    setTabSocketState: vi.fn(),
  };
});

vi.mock("../constants.js", () => {
  return {
    workspaceAnomalyState: { toastShown: false },
    debugWorkspace: false,
  };
});

vi.mock("../callbacks.js", () => {
  return {
    queueSessionOverviewRefresh: vi.fn(),
  };
});

const tabManager = await import("../../tab-manager.js");
const createTerminalTabMock = /** @type {import("vitest").Mock} */ (
  tabManager.createTerminalTab
);
const findTabMock = /** @type {import("vitest").Mock} */ (tabManager.findTab);
const setActiveTabMock = /** @type {import("vitest").Mock} */ (tabManager.setActiveTab);
const destroyTerminalTabMock = /** @type {import("vitest").Mock} */ (
  tabManager.destroyTerminalTab
);
const renderTabsMock = /** @type {import("vitest").Mock} */ (tabManager.renderTabs);
const applyTabAppearanceMock = /** @type {import("vitest").Mock} */ (
  tabManager.applyTabAppearance
);
const getActiveTabMock = /** @type {import("vitest").Mock} */ (tabManager.getActiveTab);
const refreshTabButtonMock = /** @type {import("vitest").Mock} */ (
  tabManager.refreshTabButton
);
const utilsModule = await import("../../utils.js");
const proxyToApiMock = /** @type {import("vitest").Mock} */ (
  utilsModule.proxyToApi
);
const { state } = await import("../../state.js");
const {
  sanitizeWorkspaceTabsFromServer,
  pruneDuplicateLocalTabs,
  applyWorkspaceSnapshot,
  syncTabToWorkspace,
  syncActiveTabState,
} = await import("../tabs.js");

/**
 * @param {{ id: string; label?: string; colorId?: string }} meta
 */
function buildStubTab(meta) {
  return /** @type {import("../../types.d.ts").TerminalTab} */ ({
    id: meta.id,
    label: meta.label ?? meta.id,
    defaultLabel: meta.label ?? meta.id,
    colorId: meta.colorId ?? "sky",
    errorMessage: "",
    term: /** @type {any} */ ({
      reset: vi.fn(),
      write: vi.fn(),
      focus: vi.fn(),
      scrollToBottom: vi.fn(),
      refresh: vi.fn(),
      cols: 80,
      rows: 24,
    }),
    fitAddon: /** @type {any} */ ({ fit: vi.fn() }),
    container: /** @type {any} */ ({}),
    transcript: [],
    transcriptByteSize: 0,
    transcriptHydrated: false,
    transcriptHydrating: false,
    events: [],
    suppressed: {},
    telemetry: { typed: 0, queued: 0, sent: 0, batches: 0, lastBatchSize: 0 },
    lastSentSize: { cols: 0, rows: 0 },
    phase: "idle",
    socketState: "disconnected",
    wasDetached: false,
    inputSeq: 0,
    inputBatch: null,
    inputBatchScheduled: false,
    replayPending: false,
    replayComplete: true,
    lastReplayCount: 0,
    lastReplayTruncated: false,
    hasReceivedLiveOutput: false,
    hasEverConnected: false,
    reconnecting: false,
    pendingWrites: [],
    localEchoBuffer: [],
    session: null,
    socket: null,
    heartbeatInterval: null,
    domItem: null,
    domButton: null,
    domClose: null,
    domLabel: null,
    domStatus: null,
  });
}

describe("workspace tabs helpers", () => {
  beforeEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();

    state.tabs = [];
    state.activeTabId = null;

    createTerminalTabMock.mockImplementation(({ id, label, colorId }) => {
      const tab = buildStubTab({ id, label, colorId });
      state.tabs.push(tab);
      return tab;
    });
    findTabMock.mockImplementation((id) => state.tabs.find((tab) => tab.id === id) || null);
    setActiveTabMock.mockImplementation((id) => {
      state.activeTabId = id;
    });
    getActiveTabMock.mockImplementation(
      () => state.tabs.find((tab) => tab.id === state.activeTabId) || null,
    );
    destroyTerminalTabMock.mockImplementation((tab) => {
      state.tabs = state.tabs.filter((entry) => entry !== tab);
    });
    renderTabsMock.mockImplementation(() => {});
    applyTabAppearanceMock.mockImplementation(() => {});
    refreshTabButtonMock.mockImplementation(() => {});
    proxyToApiMock.mockResolvedValue({ ok: true });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("sanitizes raw entries and reports duplicates and invalid ids", () => {
    const rawTabs = [
      { id: " primary ", label: "Primary", colorId: "sky", sessionId: "s1" },
      { id: "primary", label: "Duplicate by id", colorId: "violet" },
      { id: "secondary", label: "", colorId: "emerald", sessionId: "" },
      { id: "", label: "missing id" },
      { label: "no id at all" },
    ];

    const result = sanitizeWorkspaceTabsFromServer(rawTabs);

    expect(result.tabs).toEqual([
      {
        id: "primary",
        label: "Primary",
        colorId: "sky",
        order: 0,
        sessionId: "s1",
      },
      {
        id: "secondary",
        label: "Terminal 2",
        colorId: "emerald",
        order: 1,
        sessionId: null,
      },
    ]);
    expect(result.duplicateIds).toEqual(["primary"]);
    expect(result.invalidIds).toEqual(["(missing)", "(missing)"]);
  });

  it("prunes duplicate local tabs while keeping first instance", () => {
    const duplicate = buildStubTab({ id: "dup", label: "First", colorId: "sky" });
    const laterDuplicate = buildStubTab({ id: "dup", label: "Second", colorId: "sky" });
    const unique = buildStubTab({ id: "unique", label: "U", colorId: "sky" });

    state.tabs = [duplicate, laterDuplicate, unique];

    const { tabs, removedIds } = pruneDuplicateLocalTabs();

    expect(tabs).toHaveLength(2);
    expect(tabs[0]).toBe(duplicate);
    expect(removedIds).toEqual(["dup"]);
    expect(state.tabs).toEqual(tabs);
    expect(destroyTerminalTabMock).toHaveBeenCalledWith(laterDuplicate);
    expect(renderTabsMock).toHaveBeenCalled();
  });

  it("applies workspace snapshot, removes stale tabs, and sets active tab", () => {
    const stale = buildStubTab({ id: "stale", label: "Stale" });
    state.tabs.push(stale);
    state.activeTabId = "stale";

    const resolved = applyWorkspaceSnapshot(
      [
        {
          id: "tab-1",
          label: "One",
          colorId: "emerald",
          order: 0,
          sessionId: null,
        },
        {
          id: "tab-2",
          label: "Two",
          colorId: "sky",
          order: 1,
          sessionId: null,
        },
      ],
      { activeTabId: "tab-2" },
    );

    expect(destroyTerminalTabMock).toHaveBeenCalledWith(stale);
    expect(state.tabs.map((tab) => tab.id)).toEqual(["tab-1", "tab-2"]);
    expect(state.activeTabId).toBe("tab-2");
    expect(resolved.get("tab-1")).toBe(state.tabs[0]);
    expect(createTerminalTabMock).toHaveBeenCalledTimes(2);
    expect(renderTabsMock).toHaveBeenCalled();
  });

  it("updates existing tabs when applying workspace snapshot", () => {
    const existing = buildStubTab({ id: "tab-1", label: "Old", colorId: "sky" });
    state.tabs.push(existing);

    const resolved = applyWorkspaceSnapshot(
      [
        {
          id: "tab-1",
          label: "New",
          colorId: "emerald",
          order: 0,
          sessionId: null,
        },
        {
          id: "tab-2",
          label: "Added",
          colorId: "violet",
          order: 1,
          sessionId: null,
        },
      ],
      { activeTabId: "tab-1" },
    );

    expect(resolved.get("tab-1")).toBe(existing);
    expect(existing.label).toBe("New");
    expect(createTerminalTabMock).toHaveBeenCalledTimes(1);
    expect(state.tabs).toHaveLength(2);
    expect(state.activeTabId).toBe("tab-1");
  });

  it("debounces tab sync requests and sends latest payload", async () => {
    vi.useFakeTimers();
    const tab = buildStubTab({ id: "sync", label: "Initial" });
    state.tabs.push(tab);

    const firstPromise = syncTabToWorkspace(tab);
    tab.label = "Updated";
    const secondPromise = syncTabToWorkspace(tab);

    expect(proxyToApiMock).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(200);
    await firstPromise;
    await secondPromise;

    expect(proxyToApiMock).toHaveBeenCalledTimes(1);
    const [path, options] = proxyToApiMock.mock.calls[0];
    expect(path).toBe("/api/v1/workspace/tabs");
    expect(options?.json.label).toBe("Updated");
  });

  it("debounces active tab state sync", async () => {
    vi.useFakeTimers();
    const tabA = buildStubTab({ id: "a", label: "A" });
    const tabB = buildStubTab({ id: "b", label: "B" });
    state.tabs.push(tabA, tabB);
    state.activeTabId = "a";

    const promiseOne = syncActiveTabState();
    const promiseTwo = syncActiveTabState();

    await vi.advanceTimersByTimeAsync(220);
    await promiseOne;
    await promiseTwo;

    expect(proxyToApiMock).toHaveBeenCalledTimes(1);
    const [path, options] = proxyToApiMock.mock.calls[0];
    expect(path).toBe("/api/v1/workspace");
    expect(options?.json.activeTabId).toBe("a");
    expect(options?.json.tabs).toHaveLength(2);
  });
});
