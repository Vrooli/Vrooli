import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, act, cleanup } from "@testing-library/react";
import { apiBaseMock } from "../test-utils";

// --- Mocks (must be hoisted before component import) ---

vi.mock("@vrooli/api-base", () => apiBaseMock());

// Mock speechSynthesis on window (needed for browser support detection)
Object.defineProperty(window, "speechSynthesis", {
  value: { speak: vi.fn(), cancel: vi.fn(), getVoices: vi.fn(() => []), speaking: false, paused: false, onvoiceschanged: null },
  writable: true,
  configurable: true,
});

// Mock SpeechSynthesisUtterance
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

// Track calls to useTextToSpeech return values
const mockSpeak = vi.fn();
const mockSpeakParagraphs = vi.fn();
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

// Mock useTerminalSocket to capture the onTTS handler
let capturedOnTTS: ((text: string) => void) | undefined;
vi.mock("../hooks/useTerminalSocket", () => ({
  useTerminalSocket: (opts: { onTTS?: (text: string) => void }) => {
    capturedOnTTS = opts.onTTS;
    return {
      sendInput: vi.fn().mockReturnValue(true),
      sendResize: vi.fn(),
      totalBytesRef: { current: 0 },
    };
  },
}));

// Mock useTerminalTouch
vi.mock("../hooks/useTerminalTouch", () => ({
  useTerminalTouch: () => ({
    hasSelection: false,
    copySelection: vi.fn(),
    clearSelection: vi.fn(),
  }),
}));

// Mock useMobileBackspaceRepeat
vi.mock("../hooks/useMobileBackspaceRepeat", () => ({
  useMobileBackspaceRepeat: vi.fn(),
}));

// Mock xterm Terminal
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
    buffer: { active: { viewportY: 0, baseY: 0 } },
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

// Mock workspace store
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

// Dynamic import after mocks are set up
const { default: TerminalPane } = await import("../components/TerminalPane");

describe("TerminalPane auto-TTS via useTextToSpeech hook", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    capturedOnTTS = undefined;
    storeState.autoTtsEnabled = true;

    // Polyfill ResizeObserver for jsdom
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

  it("calls speakParagraphs when autoTtsEnabled and TTS message arrives", () => {
    render(<TerminalPane sessionId="tts-test" />);

    expect(capturedOnTTS).toBeDefined();
    act(() => {
      capturedOnTTS?.("Hello world");
    });

    expect(mockStop).toHaveBeenCalled();
    expect(mockSpeakParagraphs).toHaveBeenCalledWith(["Hello world"]);
  });

  it("splits text on paragraph boundaries", () => {
    render(<TerminalPane sessionId="tts-test" />);

    expect(capturedOnTTS).toBeDefined();
    act(() => {
      capturedOnTTS?.("First paragraph\n\nSecond paragraph\n\nThird paragraph");
    });

    expect(mockSpeakParagraphs).toHaveBeenCalledWith([
      "First paragraph",
      "Second paragraph",
      "Third paragraph",
    ]);
  });

  it("stops previous speech when new TTS message arrives", () => {
    render(<TerminalPane sessionId="tts-test" />);

    expect(capturedOnTTS).toBeDefined();

    act(() => {
      capturedOnTTS?.("First message");
    });

    mockStop.mockClear();
    mockSpeakParagraphs.mockClear();

    act(() => {
      capturedOnTTS?.("New message replaces old");
    });

    expect(mockStop).toHaveBeenCalled();
    expect(mockSpeakParagraphs).toHaveBeenCalledWith(["New message replaces old"]);
  });

  it("does not speak when autoTtsEnabled is false", () => {
    storeState.autoTtsEnabled = false;

    render(<TerminalPane sessionId="tts-test" />);

    expect(capturedOnTTS).toBeDefined();
    act(() => {
      capturedOnTTS?.("Should not be spoken");
    });

    expect(mockSpeakParagraphs).not.toHaveBeenCalled();
  });

  it("sub-splits long blocks on single newlines", () => {
    render(<TerminalPane sessionId="tts-test" />);

    expect(capturedOnTTS).toBeDefined();
    // Build a single block >500 chars with single-newline separators
    const lineA = "A".repeat(300);
    const lineB = "B".repeat(300);
    const text = `${lineA}\n${lineB}`;
    act(() => {
      capturedOnTTS?.(text);
    });

    // Because the block exceeds 500 chars, it should be sub-split on \n
    expect(mockSpeakParagraphs).toHaveBeenCalledWith([lineA, lineB]);
  });

  it("does not sub-split short blocks on single newlines", () => {
    render(<TerminalPane sessionId="tts-test" />);

    expect(capturedOnTTS).toBeDefined();
    const text = "Short line A\nShort line B";
    act(() => {
      capturedOnTTS?.(text);
    });

    // Block is <500 chars, so it stays as one paragraph
    expect(mockSpeakParagraphs).toHaveBeenCalledWith(["Short line A\nShort line B"]);
  });

  it("stops previous speech before starting new TTS (concurrent messages)", () => {
    render(<TerminalPane sessionId="tts-test" />);
    expect(capturedOnTTS).toBeDefined();

    // Fire two messages in quick succession
    act(() => {
      capturedOnTTS?.("First message");
      capturedOnTTS?.("Second message");
    });

    // stop should be called for each message
    expect(mockStop).toHaveBeenCalledTimes(2);
    // speakParagraphs called twice; the last call is the one that matters
    expect(mockSpeakParagraphs).toHaveBeenLastCalledWith(["Second message"]);
  });

  it("filters out empty paragraphs from split", () => {
    render(<TerminalPane sessionId="tts-test" />);

    expect(capturedOnTTS).toBeDefined();
    act(() => {
      capturedOnTTS?.("Only real content\n\n\n\n\n\nSecond part");
    });

    expect(mockSpeakParagraphs).toHaveBeenCalledWith([
      "Only real content",
      "Second part",
    ]);
  });
});
