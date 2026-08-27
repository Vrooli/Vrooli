import { renderWithProviders as render } from "../test-utils";
/**
 * Regression tests for stop semantics. Stop is now a playback-intent action
 * owned by the controller; TerminalPane must stop the provider without marking
 * assistant messages as listened.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { createTerminalSessionStub } from "../test-utils";
import { act, cleanup } from "@testing-library/react";
import { createRef } from "react";
import { apiBaseMock } from "../test-utils";
import type { TerminalPaneHandle } from "../components/TerminalPane";

vi.mock("@vrooli/api-base", () => apiBaseMock());

Object.defineProperty(window, "speechSynthesis", {
  value: { speak: vi.fn(), cancel: vi.fn(), getVoices: vi.fn(() => []), speaking: false, paused: false, onvoiceschanged: null },
  writable: true,
  configurable: true,
});
vi.stubGlobal("SpeechSynthesisUtterance", class { text: string; rate = 1; pitch = 1; voice = null; onend = null; onerror = null; constructor(t: string) { this.text = t; } });

// ── TTS mock: speakParagraphs returns a controllable promise ──
const mockSpeakParagraphs = vi.fn<() => Promise<string | undefined>>();
const mockStop = vi.fn();
const mockSetMuted = vi.fn();
const mockResume = vi.fn();
const mockGetServerSize = vi.fn(() => null);
const mockSendResize = vi.fn();
let mockIsSpeaking = false;

vi.mock("../hooks/useTextToSpeech", () => ({
  useTextToSpeech: () => ({
    supported: true,
    isSpeaking: mockIsSpeaking,
    isPaused: false,
    backend: "browser",
    voices: [],
    error: null,
    speak: vi.fn(),
    speakParagraphs: mockSpeakParagraphs,
    stop: mockStop,
    pause: vi.fn(),
    resume: mockResume,
    seek: vi.fn(),
    setPlaybackRate: vi.fn(),
    setVolume: vi.fn(),
    setMuted: mockSetMuted,
    getPlaybackState: vi.fn().mockReturnValue(null),
  }),
}));

vi.mock("../hooks/terminal/useTerminalSession", () => {
  // One stable session object: TerminalPane keys effects on these references.
  const session = createTerminalSessionStub({ sendResize: mockSendResize, getServerSize: mockGetServerSize });
  return { useTerminalSession: () => session };
});

vi.mock("../hooks/useTerminalTouch", () => ({
  useTerminalTouch: () => ({ hasSelection: false, copySelection: vi.fn(), clearSelection: vi.fn() }),
}));

vi.mock("../hooks/useMobileBackspaceRepeat", () => ({
  useMobileBackspaceRepeat: vi.fn(),
}));

vi.mock("@xterm/xterm", () => ({
  Terminal: vi.fn().mockImplementation(() => ({
    open: vi.fn(), dispose: vi.fn(), focus: vi.fn(),
    onTitleChange: vi.fn().mockReturnValue({ dispose: vi.fn() }),
    onData: vi.fn().mockReturnValue({ dispose: vi.fn() }),
    write: vi.fn(), cols: 80, rows: 24, options: {},
    buffer: { active: { viewportY: 0, baseY: 0, length: 0, getLine: () => ({ translateToString: () => "" }) } },
    loadAddon: vi.fn(), selectAll: vi.fn(), clear: vi.fn(),
    getSelection: vi.fn().mockReturnValue(""), getSelectionPosition: vi.fn().mockReturnValue(undefined),
    clearSelection: vi.fn(), select: vi.fn(), scrollLines: vi.fn(), textarea: null, reset: vi.fn(),
  })),
}));

vi.mock("@xterm/addon-fit", () => ({ FitAddon: vi.fn().mockImplementation(() => ({ fit: vi.fn(), dispose: vi.fn() })) }));
vi.mock("@xterm/addon-web-links", () => ({ WebLinksAddon: vi.fn().mockImplementation(() => ({ dispose: vi.fn() })) }));

const SESSION_ID = "stop-test";
const storeState: Record<string, unknown> = {
  autoTtsEnabled: true,
  ttsVoice: "", ttsRate: 1.0, ttsPitch: 1.0,
  kokoroVoice: "af_heart", kokoroSpeed: 1.0,
  ttsBackendPreference: "auto", voiceShortcut: "",
  deviceFontSize: {},
  panes: [{ sessionId: SESSION_ID, name: "test", headerColor: "transparent", themeId: "default", fontSize: 14, groupId: null }],
  activePane: SESSION_ID,
  renamePaneById: vi.fn(),
  setPendingInputBuffer: vi.fn(),
  consumePendingInputBuffer: vi.fn(() => undefined),
  setPendingInputDraft: vi.fn(),
  consumePendingInputDraft: vi.fn(() => undefined),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector?: (s: Record<string, unknown>) => unknown) =>
    selector ? selector(storeState) : storeState,
  useEffectiveFontSize: () => 14,
}));

const { default: TerminalPane } = await import("../components/TerminalPane");
const { useConversationStore } = await import("../stores/useConversationStore");
type StoreConversationEvent = Parameters<ReturnType<typeof useConversationStore.getState>["appendEvent"]>[0];

function makeEvent(id: string, sequence: number, text: string): StoreConversationEvent {
  return {
    id,
    sessionId: SESSION_ID,
    source: "claude_hook",
    role: "assistant",
    text,
    speechParagraphs: [text],
    summarized: false,
    createdAt: new Date().toISOString(),
    sequence,
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "unseen",
  };
}

describe("TerminalPane TTS stop prevents retry loop", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockIsSpeaking = false;
    storeState.autoTtsEnabled = true;
    // Reset conversation store
    useConversationStore.setState({ sessions: {}, viewModes: {} });

    if (typeof globalThis.ResizeObserver === "undefined") {
      globalThis.ResizeObserver = class {
        observe() {}
        unobserve() {}
        disconnect() {}
      } as unknown as typeof ResizeObserver;
    }
  });

  afterEach(() => {
    cleanup();
  });

  it("stopTts stops provider playback without advancing lastListenedSequence", async () => {
    const ref = createRef<TerminalPaneHandle>();
    render(<TerminalPane ref={ref} sessionId={SESSION_ID} />);

    await act(async () => { await new Promise((r) => setTimeout(r, 10)); });

    act(() => {
      useConversationStore.getState().appendEvent(makeEvent("evt-stop-1", 42, "Stop me"));
    });

    await act(async () => {
      ref.current?.playback.stop();
    });

    const session = useConversationStore.getState().sessions[SESSION_ID];
    expect(session).toBeDefined();
    expect(session?.cursor.lastListenedSequence).toBe(0);
    expect(mockStop).toHaveBeenCalledTimes(1);
    expect(mockSpeakParagraphs).not.toHaveBeenCalled();
  });

  it("stopTts leaves multiple assistant events unlistened instead of skipping them", async () => {
    const ref = createRef<TerminalPaneHandle>();
    render(<TerminalPane ref={ref} sessionId={SESSION_ID} />);
    await act(async () => { await new Promise((r) => setTimeout(r, 10)); });

    for (let i = 1; i <= 3; i++) {
      act(() => {
        useConversationStore.getState().appendEvent(makeEvent(`evt-multi-${i}`, 100 + i, `Message ${i}`));
      });
    }

    await act(async () => {
      ref.current?.playback.stop();
    });

    const session = useConversationStore.getState().sessions[SESSION_ID];
    expect(session?.cursor.lastSeenSequence).toBeGreaterThanOrEqual(103);
    expect(session?.cursor.lastListenedSequence).toBe(0);
    expect(mockSpeakParagraphs).not.toHaveBeenCalled();
  });

  it("manual playback paths auto-unmute before speaking or resuming", async () => {
    mockSpeakParagraphs.mockResolvedValue("browser");

    const ref = createRef<TerminalPaneHandle>();
    render(<TerminalPane ref={ref} sessionId={SESSION_ID} />);
    await act(async () => { await new Promise((r) => setTimeout(r, 10)); });

    act(() => {
      ref.current?.playback.speak("Manual", ["Manual"], { eventId: "evt-manual", version: "active" });
    });
    expect(mockSetMuted).toHaveBeenCalledWith(false);

    act(() => {
      ref.current?.playback.resume();
    });
    expect(mockSetMuted).toHaveBeenLastCalledWith(false);
    expect(mockResume).toHaveBeenCalledTimes(1);
  });

  it("auto playback preserves the current muted state", async () => {
    mockSpeakParagraphs.mockResolvedValue("browser");

    const ref = createRef<TerminalPaneHandle>();
    render(<TerminalPane ref={ref} sessionId={SESSION_ID} />);
    await act(async () => { await new Promise((r) => setTimeout(r, 10)); });

    act(() => {
      void ref.current?.playback.speak("Auto", ["Auto"], { eventId: "evt-auto", version: "active", initiatedBy: "auto" });
    });

    expect(mockSetMuted).not.toHaveBeenCalledWith(false);
    expect(mockSpeakParagraphs).toHaveBeenCalledWith(["Auto"], { eventId: "evt-auto", version: "active", initiatedBy: "auto" });
  });
});
