/**
 * Tests for the persistent TTS replay bar in Workspace.
 *
 * When auto-TTS is enabled and there is a previous response available,
 * the AudioPlayerBar should remain visible after playback ends so the
 * user can replay the last response with one tap instead of navigating
 * to the messages view.
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import { forwardRef, useImperativeHandle } from "react";
import type { TerminalPaneHandle } from "../components/TerminalPane";
import type { ConversationEvent } from "../lib/api";
import { apiBaseMock } from "../test-utils";
import Workspace from "../components/Workspace";

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
    },
    hookState: {
      panes: [{ session: { id: SESSION_ID, shell: "/bin/bash", created_at: "2026-01-01T00:00:00Z", cols: 80, rows: 24, policy: { mode: "never" as const }, busy: false } }],
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
    handleTerminalReady: vi.fn(),
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
    getTtsStateOnPane: vi.fn().mockReturnValue(null),
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
}));

vi.mock("../stores/useConversationStore", () => {
  const store = {
    sessions: mockConversationSessions,
    viewModes: {} as Record<string, string>,
    setViewMode: vi.fn(),
    clearSession: vi.fn(),
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
      sendInput: vi.fn().mockReturnValue(true),
      focus: vi.fn(),
      stopTts: vi.fn(),
      speakText: vi.fn(),
      speakSequence: vi.fn().mockResolvedValue(undefined),
      pauseTts: vi.fn(),
      resumeTts: vi.fn(),
      seekTts: vi.fn(),
      setTtsPlaybackRate: vi.fn(),
      setTtsVolume: vi.fn(),
      getTtsState: vi.fn().mockReturnValue(null),
      subscribeInputSettled: vi.fn(() => () => {}),
      subscribePendingInput: vi.fn(() => () => {}),
      getPendingInputSnapshot: vi.fn(() => []),
    }));
    return <div data-testid={`mock-terminal-${sessionId}`}>Terminal {sessionId}</div>;
  }),
}));

vi.mock("../components/TerminalHeader", () => ({
  default: vi.fn(({ sessionId }: { sessionId: string }) => (
    <div data-testid={`mock-header-${sessionId}`}>Header</div>
  )),
}));

vi.mock("../components/AudioPlayerBar", () => ({
  default: vi.fn(({ onResume, onStop }: { onResume: () => void; onStop: () => void }) => (
    <div data-testid="audio-player-bar">
      <button data-testid="replay-resume" onClick={onResume}>Resume</button>
      <button data-testid="replay-stop" onClick={onStop}>Stop</button>
    </div>
  )),
}));

vi.mock("../components/TabBar", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/GridSplitter", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/SettingsModal", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/WorkspaceMinimap", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/TerminalLauncher", () => ({ default: vi.fn(() => null) }));
vi.mock("../components/MobileToolbar", () => ({
  default: forwardRef(function MockMobileToolbar(_, _ref) { return null; }),
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
vi.mock("../hooks/useVoiceInput", () => ({
  useVoiceInput: () => ({ supported: false, isRecording: false, startRecording: vi.fn(), stopRecording: vi.fn() }),
}));
vi.mock("../hooks/useConversationSession", () => ({
  useConversationSession: () => ({ events: [], cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, persistCursor: vi.fn() }),
}));
vi.mock("../hooks/useImageUpload", () => ({
  useImageUpload: () => ({ uploadImage: vi.fn() }),
}));
vi.mock("../lib/api", () => ({
  getSession: vi.fn(),
  uploadFile: vi.fn(),
  summarizeEvent: vi.fn().mockResolvedValue({}),
  fetchCapabilities: vi.fn().mockResolvedValue({ capabilities: [], timestamp: "" }),
  getSessionDefaults: vi.fn().mockResolvedValue({ default_backend: "standard", default_policy: { mode: "never" } }),
}));

// ── Test constants ──
const SESSION_ID = "sess-replay-001";

const testEvent: ConversationEvent = {
  id: "evt-001",
  sessionId: SESSION_ID,
  source: "claude",
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

function setupPaneState() {
  mockStoreState.panes = [{ sessionId: SESSION_ID, name: "/bin/bash", headerColor: "transparent" }];
  mockStoreState.activePane = SESSION_ID;
  mockConversationSessions[SESSION_ID] = { events: [testEvent] };
}

describe("Workspace TTS replay bar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockStoreState.autoTtsEnabled = false;
    mockStoreState.activePane = null;
    mockStoreState.panes = [];
    captured.onSpeakingEventChange = undefined;
    captured.onTtsSpeakingChange = undefined;
    delete mockConversationSessions[SESSION_ID];
  });

  it("does not show audio player bar when auto-TTS is off and not speaking", () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = false;

    render(<Workspace />);
    expect(screen.queryByTestId("audio-player-bar")).toBeNull();
  });

  it("shows audio player bar while TTS is actively speaking", () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = true;

    render(<Workspace />);

    // Simulate TTS starting
    act(() => {
      captured.onTtsSpeakingChange?.(true);
    });
    act(() => {
      captured.onSpeakingEventChange?.(testEvent.id);
    });

    expect(screen.getByTestId("audio-player-bar")).toBeInTheDocument();
  });

  it("keeps audio player bar visible after TTS stops when auto-TTS is enabled", () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = true;

    render(<Workspace />);

    // Simulate TTS starting then stopping
    act(() => {
      captured.onTtsSpeakingChange?.(true);
      captured.onSpeakingEventChange?.(testEvent.id);
    });
    act(() => {
      captured.onTtsSpeakingChange?.(false);
    });

    // Bar should still be visible in replay mode
    expect(screen.getByTestId("audio-player-bar")).toBeInTheDocument();
  });

  it("hides audio player bar after TTS stops when auto-TTS is disabled", () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = false;

    render(<Workspace />);

    // Simulate TTS starting then stopping
    act(() => {
      captured.onTtsSpeakingChange?.(true);
      captured.onSpeakingEventChange?.(testEvent.id);
    });
    act(() => {
      captured.onTtsSpeakingChange?.(false);
    });

    // Bar should be hidden
    expect(screen.queryByTestId("audio-player-bar")).toBeNull();
  });

  it("replay resume button triggers speakTextOnPane with last event", () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = true;

    render(<Workspace />);

    // Simulate TTS starting then stopping (enter replay mode)
    act(() => {
      captured.onTtsSpeakingChange?.(true);
      captured.onSpeakingEventChange?.(testEvent.id);
    });
    act(() => {
      captured.onTtsSpeakingChange?.(false);
    });

    // Press the resume/play button in replay mode
    fireEvent.click(screen.getByTestId("replay-resume"));
    expect(mockSpeakTextOnPane).toHaveBeenCalledWith(
      SESSION_ID,
      testEvent.text,
      testEvent.speechParagraphs,
      { eventId: testEvent.id },
    );
  });

  it("stop button in replay mode dismisses the bar", () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = true;

    render(<Workspace />);

    // Simulate TTS starting then stopping (enter replay mode)
    act(() => {
      captured.onTtsSpeakingChange?.(true);
      captured.onSpeakingEventChange?.(testEvent.id);
    });
    act(() => {
      captured.onTtsSpeakingChange?.(false);
    });

    expect(screen.getByTestId("audio-player-bar")).toBeInTheDocument();

    // Press stop in replay mode → dismisses bar
    fireEvent.click(screen.getByTestId("replay-stop"));
    expect(screen.queryByTestId("audio-player-bar")).toBeNull();
  });
});
