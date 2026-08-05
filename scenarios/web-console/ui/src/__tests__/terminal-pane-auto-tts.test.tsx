import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, act, cleanup } from "@testing-library/react";
import { apiBaseMock } from "../test-utils";
import { useConversationStore } from "../stores/useConversationStore";
import type { ConversationEvent } from "../api/conversation";

vi.mock("@vrooli/api-base", () => apiBaseMock());

Object.defineProperty(window, "speechSynthesis", {
  value: { speak: vi.fn(), cancel: vi.fn(), getVoices: vi.fn(() => []), speaking: false, paused: false, onvoiceschanged: null },
  writable: true,
  configurable: true,
});

class MockUtterance {
  text: string;
  rate = 1;
  pitch = 1;
  voice: SpeechSynthesisVoice | null = null;
  onend: (() => void) | null = null;
  onerror: (() => void) | null = null;
  constructor(text: string) {
    this.text = text;
  }
}
vi.stubGlobal("SpeechSynthesisUtterance", MockUtterance);

const mockSpeak = vi.fn();
const mockSpeakParagraphs = vi.fn().mockResolvedValue("browser");
const mockStop = vi.fn();
const mockIncoming = vi.fn();
const mockSendAck = vi.fn();
const mockGetServerSize = vi.fn(() => null);
const mockSendResize = vi.fn();

vi.mock("../hooks/useTextToSpeech", () => ({
  useTextToSpeech: () => ({
    supported: true,
    isSpeaking: false,
    backend: "browser",
    voices: [],
    error: null,
    speak: mockSpeak,
    speakParagraphs: mockSpeakParagraphs,
    stop: mockStop,
  }),
}));

// Conversation events now arrive via the global SSE channel into the store;
// TerminalPane drives auto-TTS by observing the store tail for the active pane.
// So we mock the session hook (no WS conversation callbacks anymore) and drive
// tests by appending events to the real conversation store.
vi.mock("../hooks/terminal/useTerminalSession", () => {
  const gate = { submit: vi.fn(() => ({ status: "sent" as const, seq: 1 })), dispose: vi.fn(), canAcceptPaste: () => true };
  const submitInput = vi.fn(() => ({ status: "sent" as const, seq: 1 }));
  return {
    useTerminalSession: () => ({
      submitInput,
      gate,
      sendResize: mockSendResize,
      getServerSize: mockGetServerSize,
      serverSize: null,
      subscribeInputSettled: vi.fn(() => () => {}),
      subscribePendingInput: vi.fn(() => () => {}),
      getPendingInputSnapshot: vi.fn(() => []),
      sendConversationAck: mockSendAck,
    }),
  };
});

// Avoid the async mount-hydrate fetch; tests seed the store explicitly.
vi.mock("../hooks/useConversationSession", () => ({
  useConversationSession: () => ({
    persistCursor: vi.fn().mockResolvedValue(undefined),
    appendConversationEvent: vi.fn(),
    refresh: vi.fn().mockResolvedValue(true),
  }),
  refreshConversationSession: vi.fn().mockResolvedValue(true),
}));

vi.mock("../hooks/useTerminalTouch", () => ({
  useTerminalTouch: () => ({ hasSelection: false, copySelection: vi.fn(), clearSelection: vi.fn() }),
}));

vi.mock("../hooks/useMobileBackspaceRepeat", () => ({ useMobileBackspaceRepeat: vi.fn() }));

vi.mock("@xterm/xterm", () => ({
  Terminal: vi.fn().mockImplementation(() => ({
    open: vi.fn(),
    dispose: vi.fn(),
    focus: vi.fn(),
    onTitleChange: vi.fn().mockReturnValue({ dispose: vi.fn() }),
    onData: vi.fn().mockReturnValue({ dispose: vi.fn() }),
    write: vi.fn(),
    cols: 80,
    rows: 24,
    options: {},
    buffer: { active: { viewportY: 0, baseY: 0, length: 0, getLine: () => ({ translateToString: () => "" }) } },
    loadAddon: vi.fn(),
    selectAll: vi.fn(),
    clear: vi.fn(),
    getSelection: vi.fn().mockReturnValue(""),
    getSelectionPosition: vi.fn().mockReturnValue(undefined),
    clearSelection: vi.fn(),
    select: vi.fn(),
    scrollLines: vi.fn(),
    textarea: null,
    reset: vi.fn(),
  })),
}));

vi.mock("@xterm/addon-fit", () => ({
  FitAddon: vi.fn().mockImplementation(() => ({ fit: vi.fn(), dispose: vi.fn() })),
}));

vi.mock("@xterm/addon-web-links", () => ({
  WebLinksAddon: vi.fn().mockImplementation(() => ({ dispose: vi.fn() })),
}));

const storeState: Record<string, unknown> = {
  autoTtsEnabled: true,
  ttsVoice: "",
  ttsRate: 1.0,
  ttsPitch: 1.0,
  kokoroVoice: "af_heart",
  kokoroSpeed: 1.0,
  ttsBackendPreference: "auto",
  deviceFontSize: {},
  voiceShortcut: "",
  panes: [{ sessionId: "tts-test", name: "test", headerColor: "transparent", themeId: "default", fontSize: 14, groupId: null }],
  activePane: "tts-test",
  renamePaneById: vi.fn(),
  setPendingInputDraft: vi.fn(),
  consumePendingInputDraft: vi.fn(() => undefined),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector?: (s: Record<string, unknown>) => unknown) =>
    selector ? selector(storeState) : storeState,
}));

const { default: TerminalPane } = await import("../components/TerminalPane");

function makeEvent(id: string, sequence: number, text: string, speechParagraphs?: string[]): ConversationEvent {
  return {
    id,
    sessionId: "tts-test",
    source: "claude_hook",
    role: "assistant",
    text,
    speechParagraphs: speechParagraphs ?? [text],
    summarized: false,
    createdAt: new Date().toISOString(),
    sequence,
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "unseen",
  };
}

describe("TerminalPane auto-TTS (store-driven)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    storeState.autoTtsEnabled = true;
    storeState.activePane = "tts-test";
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

  function renderPane() {
    const r = render(<TerminalPane sessionId="tts-test" onConversationEventReceived={mockIncoming} />);
    // Establish the auto-TTS baseline from an (empty) hydrate so the first
    // live event that follows is treated as fresh, not as the baseline.
    act(() => {
      useConversationStore.getState().hydrateSession("tts-test", [], { lastSeenSequence: 0, lastListenedSequence: 0 });
    });
    return r;
  }

  it("forwards a fresh assistant event for the active pane to the controller", () => {
    renderPane();
    act(() => useConversationStore.getState().appendEvent(makeEvent("evt-1", 1, "Hello world", ["Hello world"])));

    expect(mockIncoming).toHaveBeenCalledWith(
      "tts-test",
      expect.objectContaining({ id: "evt-1", speechParagraphs: ["Hello world"] }),
      expect.any(Function),
    );
  });

  it("forwards the event's own speechParagraphs", () => {
    renderPane();
    act(() => useConversationStore.getState().appendEvent(
      makeEvent("evt-2", 2, "a\n\nb\n\nc", ["First paragraph", "Second paragraph", "Third paragraph"]),
    ));

    expect(mockIncoming).toHaveBeenCalledWith(
      "tts-test",
      expect.objectContaining({ id: "evt-2", speechParagraphs: ["First paragraph", "Second paragraph", "Third paragraph"] }),
      expect.any(Function),
    );
  });

  it("leaves auto-TTS policy to the controller (forwards even with autoTtsEnabled off)", () => {
    storeState.autoTtsEnabled = false;
    renderPane();
    act(() => useConversationStore.getState().appendEvent(makeEvent("evt-3", 3, "Hello", ["Hello"])));

    // TerminalPane forwards; the controller (mocked here) is what gates on policy.
    expect(mockIncoming).toHaveBeenCalled();
  });

  it("does not forward events for an inactive pane", () => {
    storeState.activePane = "other";
    renderPane();
    act(() => useConversationStore.getState().appendEvent(makeEvent("evt-4", 4, "Background", ["Background"])));

    expect(mockIncoming).not.toHaveBeenCalled();
  });

  it("does not auto-read a stale (old) event — guards SSE reconnect backlog", () => {
    renderPane();
    const stale = makeEvent("evt-5", 5, "Old message", ["Old message"]);
    stale.createdAt = new Date(Date.now() - 5 * 60_000).toISOString();
    act(() => useConversationStore.getState().appendEvent(stale));

    expect(mockIncoming).not.toHaveBeenCalled();
  });
});
