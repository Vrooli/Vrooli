/**
 * Tests for the persistent TTS replay bar in Workspace.
 *
 * When auto-TTS is enabled and there is a previous response available,
 * the AudioPlayerBar should remain visible after playback ends so the
 * user can replay the last response with one tap instead of navigating
 * to the messages view.
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, act, waitFor } from "@testing-library/react";
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
    setTtsMutedOnPane: vi.fn(),
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
      submitInput: vi.fn().mockReturnValue({ status: "sent", seq: 1 }),
      focus: vi.fn(),
      stopTts: vi.fn(),
      speakText: vi.fn(),
      speakSequence: vi.fn().mockResolvedValue(undefined),
      pauseTts: vi.fn(),
      resumeTts: vi.fn(),
      seekTts: vi.fn(),
      setTtsPlaybackRate: vi.fn(),
      setTtsVolume: vi.fn(),
      setTtsMuted: vi.fn(),
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
  default: vi.fn(({
    onResume,
    onDismiss,
    isSummarized,
    onToggleSummarized,
    currentLevel,
    currentMessageLabel,
    hasQueuedNext,
  }: {
    onResume: () => void;
    onDismiss?: () => void;
    isSummarized?: boolean;
    onToggleSummarized?: (useSummarized: boolean) => void;
    currentLevel?: "light" | "moderate" | "heavy";
    currentMessageLabel?: string | null;
    hasQueuedNext?: boolean;
  }) => (
    <div data-testid="audio-player-bar">
      <button data-testid="replay-resume" onClick={onResume}>Resume</button>
      <span data-testid="tts-mode-control">
        {isSummarized
          ? (currentLevel === "light" ? "Light" : currentLevel === "heavy" ? "Heavy" : "Moderate")
          : "Original"}
      </span>
      <button data-testid="tts-mode-option-original" onClick={() => onToggleSummarized?.(false)}>Original</button>
      <button data-testid="tts-mode-option-active" onClick={() => onToggleSummarized?.(true)}>
        {currentLevel === "light" ? "Light" : currentLevel === "heavy" ? "Heavy" : "Moderate"}
      </button>
      {onDismiss && <button data-testid="tts-dismiss" onClick={onDismiss}>Close</button>}
      <span data-testid="tts-current-message">{currentMessageLabel ?? ""}</span>
      <span data-testid="tts-has-next">{String(hasQueuedNext ?? false)}</span>
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

async function renderWorkspace() {
  await act(async () => {
    render(<Workspace />);
    await Promise.resolve();
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
}

describe("Workspace TTS replay bar", () => {
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
    delete mockConversationSessions[SESSION_ID];
  });

  it("does not show audio player bar when auto-TTS is off and not speaking", async () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = false;

    await renderWorkspace();
    expect(screen.queryByTestId("audio-player-bar")).toBeNull();
  });

  it("shows audio player bar while TTS is actively speaking", async () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = true;

    await renderWorkspace();

    // Simulate TTS starting
    act(() => {
      captured.onTtsSpeakingChange?.(true);
    });
    act(() => {
      captured.onSpeakingEventChange?.(testEvent.id);
    });

    expect(screen.getByTestId("audio-player-bar")).toBeInTheDocument();
  });

  it("shows a dismissible audio player bar during manual playback when auto-TTS is disabled", async () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = false;

    await renderWorkspace();

    act(() => {
      captured.onTtsSpeakingChange?.(true);
      captured.onSpeakingEventChange?.(testEvent.id);
    });

    expect(screen.getByTestId("audio-player-bar")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("tts-dismiss"));

    expect(mockStopActiveTts).toHaveBeenCalledWith(SESSION_ID);
  });

  it("keeps audio player bar visible after TTS stops when auto-TTS is enabled", async () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = true;

    await renderWorkspace();

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

  it("shows queue context for the current replay target", async () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = true;

    await renderWorkspace();

    act(() => {
      captured.onTtsSpeakingChange?.(true);
      captured.onSpeakingEventChange?.(testEvent.id);
    });
    act(() => {
      captured.onTtsSpeakingChange?.(false);
    });

    expect(screen.getByTestId("tts-current-message").textContent).toContain("#1");
    expect(screen.getByTestId("tts-has-next").textContent).toBe("false");
  });

  it("does not show the replay bar after TTS stops when auto-TTS is disabled", async () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = false;

    await renderWorkspace();

    // Simulate TTS starting then stopping
    act(() => {
      captured.onTtsSpeakingChange?.(true);
      captured.onSpeakingEventChange?.(testEvent.id);
    });
    act(() => {
      captured.onTtsSpeakingChange?.(false);
    });

    expect(screen.queryByTestId("audio-player-bar")).toBeNull();
  });

  it("replay resume button triggers speakTextOnPane with last event", async () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = true;

    await renderWorkspace();

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
    await waitFor(() => {
      expect(mockSpeakTextOnPane).toHaveBeenCalledWith(
        SESSION_ID,
        testEvent.text,
        testEvent.speechParagraphs,
        { eventId: testEvent.id, version: "active", initiatedBy: "manual" },
      );
    });
  });

  it("toggling to Original updates the bar label immediately and re-speaks the original text", async () => {
    // Event that has been summarized and still has the original available.
    const summarizedEvent: ConversationEvent = {
      ...testEvent,
      id: "evt-002",
      summarized: true,
      speechParagraphs: ["Short summary."],
      originalSpeechParagraphs: ["Original paragraph one.", "Original paragraph two."],
    };
    mockStoreState.panes = [{ sessionId: SESSION_ID, name: "/bin/bash", headerColor: "transparent" }];
    mockStoreState.activePane = SESSION_ID;
    mockStoreState.autoTtsEnabled = true;
    mockConversationSessions[SESSION_ID] = { events: [summarizedEvent] };

    await renderWorkspace();

    act(() => {
      captured.onTtsSpeakingChange?.(true);
      captured.onSpeakingEventChange?.(summarizedEvent.id);
    });

    // The bar initially labels playback as the active summary level because a summary
    // exists and the default playback version is active.
    const modeBtn = screen.getByTestId("tts-mode-control");
    expect(modeBtn.textContent).toMatch(/Moderate/);

    // Open the dropdown and pick Original.
    fireEvent.click(modeBtn);
    fireEvent.click(screen.getByTestId("tts-mode-option-original"));

    // Label must flip immediately instead of staying on the active summary level.
    expect(screen.getByTestId("tts-mode-control").textContent).toMatch(/Original/);

    // Re-speak must have been called with the original paragraphs + version.
    expect(mockSpeakTextOnPane).toHaveBeenCalledWith(
      SESSION_ID,
      summarizedEvent.text,
      summarizedEvent.originalSpeechParagraphs,
      { eventId: summarizedEvent.id, version: "original", initiatedBy: "manual" },
    );
  });

  it("does not render dismiss or compact playback controls", async () => {
    setupPaneState();
    mockStoreState.autoTtsEnabled = true;

    await renderWorkspace();

    // Simulate TTS starting then stopping (enter replay mode)
    act(() => {
      captured.onTtsSpeakingChange?.(true);
      captured.onSpeakingEventChange?.(testEvent.id);
    });
    act(() => {
      captured.onTtsSpeakingChange?.(false);
    });

    expect(screen.getByTestId("audio-player-bar")).toBeInTheDocument();
    expect(screen.queryByTestId("tts-dismiss")).toBeNull();
    expect(screen.queryByTestId("tts-compact-control")).toBeNull();
  });
});
