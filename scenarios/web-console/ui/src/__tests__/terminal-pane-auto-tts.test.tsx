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
  | ((candidate: { eventId: string; source: string; text: string }, sendAck: (stage: string, message?: string, backend?: string) => void) => void | Promise<void>)
  | undefined;
vi.mock("../hooks/useTerminalSocket", () => ({
  useTerminalSocket: (opts: { onTTSCandidate?: typeof capturedCandidateHandler }) => {
    capturedCandidateHandler = opts.onTTSCandidate;
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
      await capturedCandidateHandler?.({ eventId: "evt-1", source: "claude_hook", text: "Hello world" }, ack);
    });

    expect(mockStop).toHaveBeenCalled();
    expect(mockSpeakParagraphs).toHaveBeenCalledWith(["Hello world"]);
    expect(ack).toHaveBeenCalledWith("received");
    expect(ack).toHaveBeenCalledWith("correlated");
    expect(ack).toHaveBeenCalledWith("playback_started", undefined, "browser");
    expect(ack).toHaveBeenCalledWith("playback_succeeded", undefined, "browser");
  });

  it("splits text on paragraph boundaries", async () => {
    render(<TerminalPane sessionId="tts-test" />);

    await act(async () => {
      await capturedCandidateHandler?.({ eventId: "evt-2", source: "claude_hook", text: "First paragraph\n\nSecond paragraph\n\nThird paragraph" }, vi.fn());
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
      await capturedCandidateHandler?.({ eventId: "evt-3", source: "claude_hook", text: "Hello world" }, ack);
    });

    expect(mockSpeakParagraphs).not.toHaveBeenCalled();
    expect(ack).toHaveBeenCalledWith("rejected", "Auto-TTS is disabled in this tab");
  });

  it("rejects candidates that do not match the rendered terminal buffer", async () => {
    const ack = vi.fn();
    render(<TerminalPane sessionId="tts-test" />);

    await act(async () => {
      await capturedCandidateHandler?.({ eventId: "evt-4", source: "claude_hook", text: "This text is nowhere in the visible terminal" }, ack);
    });

    expect(mockSpeakParagraphs).not.toHaveBeenCalled();
    expect(ack).toHaveBeenCalledWith("rejected", "Assistant text did not match the rendered terminal buffer");
  });

  it("sub-splits long blocks on single newlines", async () => {
    render(<TerminalPane sessionId="tts-test" />);

    const lineA = "A".repeat(300);
    const lineB = "B".repeat(300);
    const originalA = terminalBufferLines[0] ?? "";
    const originalB = terminalBufferLines[1] ?? "";
    terminalBufferLines[0] = lineA;
    terminalBufferLines[1] = lineB;
    await act(async () => {
      await capturedCandidateHandler?.({ eventId: "evt-5", source: "claude_hook", text: `${lineA}\n${lineB}` }, vi.fn());
    });

    expect(mockSpeakParagraphs).toHaveBeenCalledWith([lineA, lineB]);
    terminalBufferLines[0] = originalA;
    terminalBufferLines[1] = originalB;
  });

  it("filters out empty paragraphs from split", async () => {
    render(<TerminalPane sessionId="tts-test" />);

    await act(async () => {
      await capturedCandidateHandler?.({ eventId: "evt-6", source: "claude_hook", text: "Only real content\n\n\n\n\n\nSecond part" }, vi.fn());
    });

    expect(mockSpeakParagraphs).toHaveBeenCalledWith([
      "Only real content",
      "Second part",
    ]);
  });
});
