import { renderWithProviders as render } from "../test-utils";
/**
 * The playback pill in Workspace: when it shows, how it replays the last
 * response, how dismissing stops playback, and how the toolbar restores it.
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, act, waitFor } from "@testing-library/react";
import { forwardRef, useImperativeHandle } from "react";
import type { TerminalPaneHandle } from "../components/TerminalPane";
import type { ConversationEvent } from "../api/conversation";
import { apiBaseMock } from "../test-utils";
import Workspace from "../components/Workspace";
import { useTtsPlaybackIntentStore } from "../domains/tts-playback/store";

// ── Hoisted shared state (accessible inside vi.mock factories) ──
const {
  mockSpeakTextOnPane,
  mockStopActiveTts,
  mockStoreState,
  mockConversationSessions,
  captured,
  hookState,
} = vi.hoisted(() => {
  const SESSION_ID = "sess-replay-001";
  return {
    mockSpeakTextOnPane: vi.fn(),
    mockStopActiveTts: vi.fn(),
    mockStoreState: {
      panes: [] as Array<{ sessionId: string; name: string; headerColor: string }>,
      columnFractions: [] as number[],
      rowFractions: [] as number[],
      activePane: null as string | null,
      autoTtsEnabled: false,
      appearanceModalPane: null,
      isMinimapVisible: false,
      displayMode: "tabs",
      settingsModalOpen: false,
      aiModalOpen: false,
      aiSuggestActive: false,
      keepScreenAwake: false,
      pasteMode: "normal",
      ttsVoice: "",
      ttsRate: 1,
      ttsPitch: 1,
      ttsBackendPreference: "auto",
      kokoroVoice: "af_heart",
      vadAutoStop: false,
      sections: [],
      sectionOrder: [],
      paneGroups: [],
      groups: [],
      // Roles are additive: this fixture uses none, which is the case every
      // assertion in this file describes.
      roles: [],
      closedGroupUndo: null,
      autoCloseEmptyGroups: true,
      manageGroupsOpen: false,
      setRoles: vi.fn(),
      addRole: vi.fn(),
      updateRole: vi.fn(),
      removeRole: vi.fn(),
      setRoleSession: vi.fn(),
      setClosedGroupUndo: vi.fn(),
      setAutoCloseEmptyGroups: vi.fn(),
      setManageGroupsOpen: vi.fn(),
      addPane: vi.fn(),
      removePane: vi.fn(),
      renamePaneById: vi.fn(),
      setPaneColor: vi.fn(),
      setPaneTheme: vi.fn(),
      setPaneFontSize: vi.fn(),
      movePaneToIndex: vi.fn(),
      setColumnFractions: vi.fn(),
      setRowFractions: vi.fn(),
      setActivePane: vi.fn(),
      setAppearanceModalPane: vi.fn(),
      setMinimapVisible: vi.fn(),
      setDisplayMode: vi.fn(),
      setSettingsModalOpen: vi.fn(),
      setAiModalOpen: vi.fn(),
      setAiSuggestActive: vi.fn(),
      resetLayout: vi.fn(),
    } as Record<string, unknown>,
    mockConversationSessions: {} as Record<string, { events: ConversationEvent[] }>,
    captured: {
      onSpeakingEventChange: undefined as ((eventId: string | null) => void) | undefined,
      onTtsSpeakingChange: undefined as ((speaking: boolean) => void) | undefined,
      toolbar: { ttsDismissed: false as boolean | undefined, onTtsRestore: undefined as (() => void) | undefined },
    },
    hookState: {
      panes: [{ session: { id: SESSION_ID, shell: "/bin/bash", created_at: "2026-01-01T00:00:00Z", cols: 80, rows: 24, policy: { mode: "never" as const } } }],
    },
  };
});

vi.mock("@vrooli/api-base", () => apiBaseMock());

vi.mock("../hooks/useSessionManager", () => ({
  useSessionManager: () => ({
    panes: hookState.panes,
    isHydrated: true,
    isCreating: false,
    createError: null,
    clearError: vi.fn(),
    launchSession: vi.fn().mockResolvedValue(hookState.panes[0]?.session),
    removePane: vi.fn(),
    handleExit: vi.fn(),
    sendToActiveTerminal: vi.fn(),
    focusActiveTerminal: vi.fn(),
    registerTerminalRef: vi.fn(),
    stopActiveTts: mockStopActiveTts,
    speakTextOnPane: mockSpeakTextOnPane,
    speakSequenceOnPane: vi.fn(),
    pauseTtsOnPane: vi.fn(),
    resumeTtsOnPane: vi.fn(),
    seekTtsOnPane: vi.fn(),
    setTtsPlaybackRateOnPane: vi.fn(),
    setTtsVolumeOnPane: vi.fn(),
    setTtsMutedOnPane: vi.fn(),
  }),
}));

vi.mock("../hooks/useWorkspaceSync", () => ({
  useWorkspaceSync: () => ({
    syncActivePane: vi.fn(),
    syncPaneOrder: vi.fn(),
    syncPaneUpdate: vi.fn(),
    syncCreateGroup: vi.fn(),
    syncUpdateGroup: vi.fn(),
    syncDeleteGroup: vi.fn(),
  }),
}));

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector?: (state: Record<string, unknown>) => unknown) => {
    return selector ? selector(mockStoreState) : mockStoreState;
  },
  useEffectiveFontSize: () => 14,
}));

vi.mock("../stores/useConversationStore", () => {
  const store = {
    sessions: mockConversationSessions,
    clearSession: vi.fn(),
    hydrateSession: vi.fn(),
    appendEvent: vi.fn(),
    updateEvent: vi.fn(),
  };
  const useConversationStore = (selector?: (state: typeof store) => unknown) => {
    return selector ? selector(store) : store;
  };
  useConversationStore.getState = () => store;
  useConversationStore.setState = (partial: Partial<typeof store>) => Object.assign(store, partial);
  return { useConversationStore };
});

vi.mock("../components/TerminalPane", () => ({
  default: forwardRef<TerminalPaneHandle, {
    sessionId: string;
    onSpeakingEventChange?: (eventId: string | null) => void;
    onTtsSpeakingChange?: (speaking: boolean) => void;
  }>(function MockTerminalPane({ sessionId, onSpeakingEventChange, onTtsSpeakingChange }, ref) {
    captured.onSpeakingEventChange = onSpeakingEventChange;
    captured.onTtsSpeakingChange = onTtsSpeakingChange;
    useImperativeHandle(ref, () => ({
      input: { submit: vi.fn().mockReturnValue({ status: "sent", offset: 1 }), subscribeSettled: vi.fn(() => () => {}), awaitOffset: vi.fn(() => () => {}) },
      control: { send: vi.fn().mockReturnValue(true), scroll: vi.fn(), focus: vi.fn() },
      selection: { copy: vi.fn().mockResolvedValue(true), paste: vi.fn().mockResolvedValue(true) },
      pendingInput: { subscribe: vi.fn(() => () => {}), snapshot: vi.fn(() => []), discard: vi.fn(), discardAll: vi.fn(), flushNow: vi.fn() },
      playback: { stop: vi.fn(), speak: vi.fn(), pause: vi.fn(), resume: vi.fn(), seek: vi.fn(), setPlaybackRate: vi.fn(), setVolume: vi.fn(), setMuted: vi.fn() },
    }));
    return <div data-testid={`mock-terminal-${sessionId}`}>Terminal {sessionId}</div>;
  }),
}));

vi.mock("../components/TerminalHeader", () => ({
  default: vi.fn(({ sessionId }: { sessionId: string }) => (
    <div data-testid={`mock-header-${sessionId}`}>Header</div>
  )),
}));

vi.mock("../components/TabBar", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/GridSplitter", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/SettingsModal", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/WorkspaceMinimap", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/TerminalLauncher", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/MobileToolbar", () => ({
  default: forwardRef(function MockMobileToolbar(props: { ttsDismissed?: boolean; onTtsRestore?: () => void }, _ref) {
    captured.toolbar = { ttsDismissed: props.ttsDismissed, onTtsRestore: props.onTtsRestore };
    return null;
  }),
}));
vi.mock("../components/AiInput", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/FloatingToolbar", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/ErrorBanner", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/AiSuggestBar", () => ({ default: vi.fn(() => null) }));

vi.mock("../hooks/useAppViewport", () => ({ useAppViewport: vi.fn() }));
vi.mock("../hooks/useWakeLock", () => ({
  useWakeLock: vi.fn().mockReturnValue("released"),
  useWakeLockStatus: () => ({ setStatus: vi.fn() }),
}));
vi.mock("../hooks/useConversationSession", () => ({
  useConversationSession: () => ({ events: [], cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, persistCursor: vi.fn() }),
}));
vi.mock("../hooks/useImageUpload", () => ({
  useImageUpload: () => ({ uploadImage: vi.fn() }),
}));
vi.mock("../api/uploads", () => ({
  uploadFile: vi.fn(),
}));
vi.mock("../audio-integration", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../audio-integration")>();
  return {
    ...actual,
    getTTSSummarizeConfig: vi.fn().mockResolvedValue({ enabled: false, charThreshold: 500, level: "moderate", model: "qwen3:1.7b", timeoutSeconds: 30 }),
    updateTTSSummarizeConfig: vi.fn().mockResolvedValue({ enabled: false, charThreshold: 500, level: "moderate", model: "qwen3:1.7b", timeoutSeconds: 30 }),
  };
});
vi.mock("../api/sessions", async () => {
  const actual = await vi.importActual<typeof import("../api/sessions")>("../api/sessions");
  return { ...actual, getSession: vi.fn() };
});
vi.mock("../api/conversation", async () => {
  const actual = await vi.importActual<typeof import("../api/conversation")>("../api/conversation");
  return { ...actual, summarizeEvent: vi.fn().mockResolvedValue({}) };
});
vi.mock("../api/capabilities", () => ({
  fetchCapabilities: vi.fn(() => new Promise(() => {})),
}));
vi.mock("../api/settings", () => ({
  getSessionDefaults: vi.fn(() => new Promise(() => {})),
}));

// ── Test constants ──
const SESSION_ID = "sess-replay-001";

const testEvent: ConversationEvent = {
  id: "evt-001",
  sessionId: SESSION_ID,
  source: "claude_hook",
  role: "assistant",
  text: "Hello, I can help you with that.",
  speechParagraphs: ["Hello, I can help you with that."],
  summarized: false,
  sequence: 1,
  createdAt: "2026-01-01T00:01:00Z",
  deliveryState: "delivered",
  ttsState: "idle",
  consumptionState: "unseen",
};

function setupPaneState(events: ConversationEvent[] = [testEvent]) {
  mockStoreState.panes = [{ sessionId: SESSION_ID, name: "/bin/bash", headerColor: "transparent" }];
  mockStoreState.activePane = SESSION_ID;
  mockConversationSessions[SESSION_ID] = { events };
}

async function renderWorkspace() {
  await act(async () => {
    render(<Workspace />);
    await Promise.resolve();
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
}

function startSpeaking(eventId: string) {
  act(() => {
    captured.onTtsSpeakingChange?.(true);
    captured.onSpeakingEventChange?.(eventId);
  });
}

function stopSpeaking() {
  act(() => {
    captured.onTtsSpeakingChange?.(false);
  });
}

describe("Workspace playback pill", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    useTtsPlaybackIntentStore.setState({
      playbackIntent: "continuous",
      selectedTarget: null,
    });
    mockStoreState.autoTtsEnabled = false;
    mockStoreState.activePane = null;
    mockStoreState.panes = [];
    captured.onSpeakingEventChange = undefined;
    captured.onTtsSpeakingChange = undefined;
    captured.toolbar = { ttsDismissed: false, onTtsRestore: undefined };
    delete mockConversationSessions[SESSION_ID];
  });

  it("[REQ:P0-017h] shows no pill and offers no restore when nothing is queued", async () => {
    setupPaneState();
    await renderWorkspace();
    expect(screen.queryByTestId("playback-pill")).toBeNull();
    expect(captured.toolbar.ttsDismissed).toBe(false);
  });

  it("[REQ:P0-017h] shows the pill while a message is speaking, collapsed", async () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = true;
    await renderWorkspace();
    startSpeaking(testEvent.id);
    expect(screen.getByTestId("playback-pill")).toHaveAttribute("data-state", "collapsed");
    expect(screen.queryByTestId("audio-player-bar")).toBeNull();
  });

  it("[REQ:P0-017h] tapping the pill expands it and tapping again collapses it", async () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = true;
    await renderWorkspace();
    startSpeaking(testEvent.id);
    fireEvent.click(screen.getByTestId("pill-summary"));
    expect(screen.getByTestId("playback-pill")).toHaveAttribute("data-state", "expanded");
    fireEvent.click(screen.getByTestId("pill-summary"));
    expect(screen.getByTestId("playback-pill")).toHaveAttribute("data-state", "collapsed");
  });

  it("[REQ:P0-017h] closing the pill stops playback and hides it; the toolbar's restore brings it back", async () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = true;
    await renderWorkspace();
    startSpeaking(testEvent.id);
    fireEvent.click(screen.getByTestId("pill-close"));
    expect(mockStopActiveTts).toHaveBeenCalled();
    stopSpeaking();
    expect(screen.queryByTestId("playback-pill")).toBeNull();
    expect(captured.toolbar.ttsDismissed).toBe(true);
    act(() => { captured.toolbar.onTtsRestore?.(); });
    expect(screen.getByTestId("playback-pill")).toBeInTheDocument();
    expect(captured.toolbar.ttsDismissed).toBe(false);
  });

  it("a new message starting to speak brings a dismissed pill back", async () => {
    const second = { ...testEvent, id: "evt-002", sequence: 2, text: "Second reply.", speechParagraphs: ["Second reply."] };
    setupPaneState([testEvent, second]);
    mockStoreState.autoTtsEnabled = true;
    await renderWorkspace();
    startSpeaking(testEvent.id);
    fireEvent.click(screen.getByTestId("pill-close"));
    stopSpeaking();
    expect(screen.queryByTestId("playback-pill")).toBeNull();
    startSpeaking(second.id);
    expect(screen.getByTestId("playback-pill")).toBeInTheDocument();
  });

  it("keeps the pill for replay after TTS stops when auto-TTS is enabled", async () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = true;
    await renderWorkspace();
    startSpeaking(testEvent.id);
    stopSpeaking();
    expect(screen.getByTestId("playback-pill")).toBeInTheDocument();
    expect(screen.queryByTestId("pill-equalizer")).toBeNull();
  });

  it("shows no pill after TTS stops when auto-TTS is disabled", async () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = false;
    await renderWorkspace();
    startSpeaking(testEvent.id);
    expect(screen.getByTestId("playback-pill")).toBeInTheDocument();
    stopSpeaking();
    expect(screen.queryByTestId("playback-pill")).toBeNull();
  });

  it("the replay play button speaks the last event again", async () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = true;
    await renderWorkspace();
    startSpeaking(testEvent.id);
    stopSpeaking();
    fireEvent.click(screen.getByTestId("pill-play-pause"));
    await waitFor(() => {
      expect(mockSpeakTextOnPane).toHaveBeenCalledWith(
        SESSION_ID,
        testEvent.text,
        testEvent.speechParagraphs,
        { eventId: testEvent.id, version: "active", initiatedBy: "manual" },
      );
    });
  });

  it("choosing Original in the expanded pill re-speaks the original text", async () => {
    const summarizedEvent: ConversationEvent = {
      ...testEvent,
      id: "evt-002",
      summarized: true,
      speechParagraphs: ["Short summary."],
      originalSpeechParagraphs: ["Original paragraph one.", "Original paragraph two."],
    };
    setupPaneState([summarizedEvent]);
    mockStoreState.autoTtsEnabled = true;
    await renderWorkspace();
    startSpeaking(summarizedEvent.id);
    fireEvent.click(screen.getByTestId("pill-summary"));
    fireEvent.click(screen.getByTestId("pill-mode-control"));
    fireEvent.click(screen.getByTestId("pill-mode-option-original"));
    expect(mockSpeakTextOnPane).toHaveBeenCalledWith(
      SESSION_ID,
      summarizedEvent.text,
      summarizedEvent.originalSpeechParagraphs,
      { eventId: summarizedEvent.id, version: "original", initiatedBy: "manual" },
    );
  });
});
