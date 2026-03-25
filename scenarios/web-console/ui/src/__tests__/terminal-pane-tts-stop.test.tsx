/**
 * Regression test for the TTS stop-retry-loop bug.
 *
 * Root cause: when the user stops TTS mid-playback, `speakParagraphs`
 * resolves without advancing `lastListenedSequence`. The pending-events
 * effect re-fires (because `isSpeaking` flipped to false) and finds the
 * same "unlistened" event, replaying it in an infinite loop.
 *
 * Fix: `stopTts()` now advances the cursor past all current assistant
 * events so the pending-events effect finds nothing to replay.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, act, cleanup } from "@testing-library/react";
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
    resume: vi.fn(),
    seek: vi.fn(),
    setPlaybackRate: vi.fn(),
    setVolume: vi.fn(),
    getPlaybackState: vi.fn().mockReturnValue(null),
  }),
}));

let capturedHandler:
  | ((event: { id: string; source: string; role: "assistant" | "user"; text: string; speechParagraphs?: string[]; sequence: number; createdAt?: string }, sendAck: (stage: string, message?: string, backend?: string) => void) => void | Promise<void>)
  | undefined;
vi.mock("../hooks/useTerminalSocket", () => ({
  useTerminalSocket: (opts: { onConversationEvent?: typeof capturedHandler }) => {
    capturedHandler = opts.onConversationEvent;
    return { sendInput: vi.fn().mockReturnValue(true), sendResize: vi.fn(), totalBytesRef: { current: 0 } };
  },
}));

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
vi.mock("@xterm/addon-serialize", () => ({ SerializeAddon: vi.fn().mockImplementation(() => ({ serialize: vi.fn(() => ""), dispose: vi.fn() })) }));
vi.mock("@xterm/addon-web-links", () => ({ WebLinksAddon: vi.fn().mockImplementation(() => ({ dispose: vi.fn() })) }));
vi.mock("../lib/terminalCache", () => ({ loadTerminalCache: vi.fn(() => null), saveTerminalCache: vi.fn() }));

const SESSION_ID = "stop-test";
const storeState: Record<string, unknown> = {
  autoTtsEnabled: true,
  ttsVoice: "", ttsRate: 1.0, ttsPitch: 1.0,
  kokoroVoice: "af_heart", kokoroSpeed: 1.0,
  ttsBackendPreference: "auto", voiceShortcut: "",
  panes: [{ sessionId: SESSION_ID, name: "test", headerColor: "transparent", themeId: "default", fontSize: 14, groupId: null }],
  activePane: SESSION_ID,
  renamePaneById: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector?: (s: Record<string, unknown>) => unknown) =>
    selector ? selector(storeState) : storeState,
}));

const { default: TerminalPane } = await import("../components/TerminalPane");
const { useConversationStore } = await import("../stores/useConversationStore");

describe("TerminalPane TTS stop prevents retry loop", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    capturedHandler = undefined;
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

  it("stopTts advances lastListenedSequence so the pending-events effect does not retry", async () => {
    // speakParagraphs returns a promise that never resolves (simulates ongoing playback)
    let rejectSpeak: ((reason?: unknown) => void) | undefined;
    mockSpeakParagraphs.mockImplementation(
      () => new Promise<string | undefined>((_, reject) => { rejectSpeak = reject; }),
    );

    const ref = createRef<TerminalPaneHandle>();
    render(<TerminalPane ref={ref} sessionId={SESSION_ID} />);

    // Wait for conversation session hydration (catches API error → empty state)
    await act(async () => { await new Promise((r) => setTimeout(r, 10)); });

    // Fire a conversation event — this starts TTS playback via handleConversationEvent.
    // Because speakParagraphs never resolves, the cursor won't be advanced by normal flow.
    const ack = vi.fn();
    // Don't await — the handler is blocked on speakParagraphs
    let handlerDone = false;
    void (async () => {
      await capturedHandler?.({
        id: "evt-stop-1", source: "claude_hook", role: "assistant",
        sequence: 42, text: "Stop me", speechParagraphs: ["Stop me"],
      }, ack);
      handlerDone = true;
    })();

    // Let the handler start executing up to the await speakParagraphs
    await act(async () => { await new Promise((r) => setTimeout(r, 10)); });
    expect(mockSpeakParagraphs).toHaveBeenCalledTimes(1);
    expect(handlerDone).toBe(false); // Still awaiting speakParagraphs

    // User clicks Stop — this should advance cursor past sequence 42
    await act(async () => {
      ref.current?.stopTts();
    });

    // Verify cursor was advanced
    const session = useConversationStore.getState().sessions[SESSION_ID];
    expect(session).toBeDefined();
    expect(session!.cursor.lastListenedSequence).toBeGreaterThanOrEqual(42);

    // Reject the pending speakParagraphs to unblock handleConversationEvent
    rejectSpeak?.(new DOMException("The operation was aborted.", "AbortError"));
    await act(async () => { await new Promise((r) => setTimeout(r, 10)); });

    // speakParagraphs should NOT have been called again (no retry loop)
    expect(mockSpeakParagraphs).toHaveBeenCalledTimes(1);
  });

  it("stopTts skips multiple pending events, not just the current one", async () => {
    // speakParagraphs resolves immediately for initial events, then blocks
    let callCount = 0;
    mockSpeakParagraphs.mockImplementation(() => {
      callCount++;
      if (callCount <= 2) return Promise.resolve("browser");
      return new Promise<string | undefined>(() => {}); // block on 3rd
    });

    const ref = createRef<TerminalPaneHandle>();
    render(<TerminalPane ref={ref} sessionId={SESSION_ID} />);
    await act(async () => { await new Promise((r) => setTimeout(r, 10)); });

    // Send 3 events — first 2 resolve, 3rd blocks
    const ack = vi.fn();
    for (let i = 1; i <= 3; i++) {
      void capturedHandler?.({
        id: `evt-multi-${i}`, source: "claude_hook", role: "assistant",
        sequence: 100 + i, text: `Message ${i}`, speechParagraphs: [`Message ${i}`],
      }, ack);
      await act(async () => { await new Promise((r) => setTimeout(r, 10)); });
    }

    // Stop all TTS
    await act(async () => {
      ref.current?.stopTts();
    });

    // Cursor should be at sequence 103 (the highest assistant event)
    const session = useConversationStore.getState().sessions[SESSION_ID];
    expect(session!.cursor.lastListenedSequence).toBeGreaterThanOrEqual(103);
  });
});
