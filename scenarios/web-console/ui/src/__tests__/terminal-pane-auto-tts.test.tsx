import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, act, cleanup } from "@testing-library/react";
import { apiBaseMock } from "../test-utils";

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

let capturedCandidateHandler:
  | ((candidate: { id: string; source: string; role: "assistant"; text: string; speechParagraphs?: string[]; sequence: number; createdAt?: string }, sendAck: (stage: string, message?: string, backend?: string) => void) => void | Promise<void>)
  | undefined;
vi.mock("../hooks/useTerminalSocket", () => ({
  useTerminalSocket: (opts: { onConversationEvent?: typeof capturedCandidateHandler }) => {
    capturedCandidateHandler = opts.onConversationEvent;
    return {
      sendInput: vi.fn().mockReturnValue(true),
      sendResize: vi.fn(),
      totalBytesRef: { current: 0 },
    };
  },
}));

vi.mock("../hooks/useTerminalTouch", () => ({
  useTerminalTouch: () => ({
    hasSelection: false,
    copySelection: vi.fn(),
    clearSelection: vi.fn(),
  }),
}));

vi.mock("../hooks/useMobileBackspaceRepeat", () => ({
  useMobileBackspaceRepeat: vi.fn(),
}));

const makeLine = (text: string) => ({ translateToString: vi.fn(() => text) });
const terminalBufferLines = [
  "Hello world",
  "First paragraph",
  "Second paragraph",
  "Third paragraph",
  "New message replaces old",
  "Only real content",
  "Second part",
];

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
    buffer: {
      active: {
        viewportY: 0,
        baseY: 0,
        length: terminalBufferLines.length,
        getLine: (index: number) => makeLine(terminalBufferLines[index] ?? ""),
      },
    },
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
  FitAddon: vi.fn().mockImplementation(() => ({
    fit: vi.fn(),
    dispose: vi.fn(),
  })),
}));

vi.mock("@xterm/addon-serialize", () => ({
  SerializeAddon: vi.fn().mockImplementation(() => ({
    serialize: vi.fn(() => ""),
    dispose: vi.fn(),
  })),
}));

vi.mock("@xterm/addon-web-links", () => ({
  WebLinksAddon: vi.fn().mockImplementation(() => ({
    dispose: vi.fn(),
  })),
}));

vi.mock("../lib/terminalCache", () => ({
  loadTerminalCache: vi.fn(() => null),
  saveTerminalCache: vi.fn(),
}));

const storeState: Record<string, unknown> = {
  autoTtsEnabled: true,
  ttsVoice: "",
  ttsRate: 1.0,
  ttsPitch: 1.0,
  kokoroVoice: "af_heart",
  kokoroSpeed: 1.0,
  ttsBackendPreference: "auto",
  voiceShortcut: "",
  panes: [{ sessionId: "tts-test", name: "test", headerColor: "transparent", themeId: "default", fontSize: 14, groupId: null }],
  activePane: "tts-test",
  renamePaneById: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector?: (s: Record<string, unknown>) => unknown) =>
    selector ? selector(storeState) : storeState,
}));

const { default: TerminalPane } = await import("../components/TerminalPane");

describe("TerminalPane auto-TTS via useTextToSpeech hook", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    capturedCandidateHandler = undefined;
    storeState.autoTtsEnabled = true;

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

  it("calls speakParagraphs when autoTtsEnabled and a matching candidate arrives", async () => {
    const ack = vi.fn();
    render(<TerminalPane sessionId="tts-test" />);

    await act(async () => {
      await capturedCandidateHandler?.({ id: "evt-1", source: "claude_hook", role: "assistant", sequence: 1, text: "Hello world", speechParagraphs: ["Hello world"] }, ack);
    });

    expect(mockStop).toHaveBeenCalled();
    expect(mockSpeakParagraphs).toHaveBeenCalledWith(["Hello world"]);
    expect(ack).toHaveBeenCalledWith("received");
    expect(ack).toHaveBeenCalledWith("seen");
    expect(ack).toHaveBeenCalledWith("playback_started", undefined, "browser");
    expect(ack).toHaveBeenCalledWith("playback_succeeded", undefined, "browser");
  });

  it("uses backend-provided speechParagraphs for TTS playback", async () => {
    render(<TerminalPane sessionId="tts-test" />);

    await act(async () => {
      await capturedCandidateHandler?.({
        id: "evt-2", source: "claude_hook", role: "assistant", sequence: 2,
        text: "First paragraph\n\nSecond paragraph\n\nThird paragraph",
        speechParagraphs: ["First paragraph", "Second paragraph", "Third paragraph"],
      }, vi.fn());
    });

    expect(mockSpeakParagraphs).toHaveBeenCalledWith([
      "First paragraph",
      "Second paragraph",
      "Third paragraph",
    ]);
  });

  it("does not speak when autoTtsEnabled is false", async () => {
    storeState.autoTtsEnabled = false;
    const ack = vi.fn();
    render(<TerminalPane sessionId="tts-test" />);

    await act(async () => {
      await capturedCandidateHandler?.({ id: "evt-3", source: "claude_hook", role: "assistant", sequence: 3, text: "Hello world", speechParagraphs: ["Hello world"] }, ack);
    });

    expect(mockSpeakParagraphs).not.toHaveBeenCalled();
    expect(ack).toHaveBeenCalledWith("received");
  });

  it("does not reject based on terminal text matching anymore", async () => {
    const ack = vi.fn();
    render(<TerminalPane sessionId="tts-test" />);

    await act(async () => {
      await capturedCandidateHandler?.({ id: "evt-4", source: "claude_hook", role: "assistant", sequence: 4, text: "This text is nowhere in the visible terminal", speechParagraphs: ["This text is nowhere in the visible terminal"] }, ack);
    });

    expect(mockSpeakParagraphs).toHaveBeenCalledWith(["This text is nowhere in the visible terminal"]);
    expect(ack).not.toHaveBeenCalledWith("rejected", expect.anything());
  });

  it("matches Codex markdown-formatted candidate text against rendered terminal output", async () => {
    const ack = vi.fn();
    render(<TerminalPane sessionId="tts-test" />);

    const original = terminalBufferLines[0] ?? "";
    terminalBufferLines[0] = "Hi. What do you need help with in /home/matthalloran8/Vrooli?";

    await act(async () => {
      await capturedCandidateHandler?.({
        id: "evt-codex-markdown",
        source: "codex_tailer",
        role: "assistant",
        sequence: 5,
        text: "Hi. What do you need help with in `/home/matthalloran8/Vrooli`?",
        speechParagraphs: ["Hi. What do you need help with in Vrooli?"],
      }, ack);
    });

    expect(mockSpeakParagraphs).toHaveBeenCalledWith([
      "Hi. What do you need help with in Vrooli?",
    ]);
    expect(ack).toHaveBeenCalledWith("seen");
    terminalBufferLines[0] = original;
  });

  it("speaks delayed Codex events without terminal correlation retries", async () => {
    const ack = vi.fn();
    render(<TerminalPane sessionId="tts-test" />);

    await act(async () => {
      await capturedCandidateHandler?.({
        id: "evt-retry",
        source: "codex_tailer",
        role: "assistant",
        sequence: 6,
        text: "Rendered a moment later",
        speechParagraphs: ["Rendered a moment later"],
      }, ack);
    });

    expect(mockSpeakParagraphs).toHaveBeenCalledWith(["Rendered a moment later"]);
    expect(ack).toHaveBeenCalledWith("seen");
  });

  it("passes backend-provided speechParagraphs directly to speakParagraphs", async () => {
    render(<TerminalPane sessionId="tts-test" />);

    const lineA = "A".repeat(300);
    const lineB = "B".repeat(300);
    await act(async () => {
      await capturedCandidateHandler?.({
        id: "evt-5", source: "claude_hook", role: "assistant", sequence: 7,
        text: `${lineA}\n${lineB}`,
        speechParagraphs: [lineA, lineB],
      }, vi.fn());
    });

    expect(mockSpeakParagraphs).toHaveBeenCalledWith([lineA, lineB]);
  });

  it("falls back to [text] when speechParagraphs is missing", async () => {
    render(<TerminalPane sessionId="tts-test" />);

    await act(async () => {
      await capturedCandidateHandler?.({ id: "evt-6", source: "claude_hook", role: "assistant", sequence: 8, text: "Only real content" }, vi.fn());
    });

    expect(mockSpeakParagraphs).toHaveBeenCalledWith([
      "Only real content",
    ]);
  });
});
