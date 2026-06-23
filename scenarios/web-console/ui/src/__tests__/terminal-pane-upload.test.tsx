import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { render, screen, fireEvent, act, cleanup } from "@testing-library/react";
import { apiBaseMock, mockFetchSuccess } from "../test-utils";

// Mock external dependencies
vi.mock("@vrooli/api-base", () => apiBaseMock());

function makeTerminalMock() {
  return {
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
  };
}

vi.mock("@xterm/xterm", () => ({
  Terminal: vi.fn().mockImplementation(() => makeTerminalMock()),
}));
vi.mock("@xterm/addon-fit", () => ({
  FitAddon: vi.fn().mockImplementation(() => ({
    fit: vi.fn(),
    dispose: vi.fn(),
  })),
}));
vi.mock("@xterm/addon-web-links", () => ({
  WebLinksAddon: vi.fn().mockImplementation(() => ({
    dispose: vi.fn(),
  })),
}));
vi.mock("../hooks/terminal/useTerminalSession", () => {
  // Stable references so dep arrays in TerminalPane do not re-run
  // effects every render and trigger the unmount-save path.

  const gate = { submit: vi.fn(() => ({ status: "sent" as const, seq: 1 })), dispose: vi.fn(), canAcceptPaste: () => true };
  const submitInput = vi.fn(() => ({ status: "sent" as const, seq: 1 }));
  const sendResize = vi.fn();
  const subscribeInputSettled = vi.fn(() => () => {});
  const subscribePendingInput = vi.fn(() => () => {});
  const getPendingInputSnapshot = vi.fn(() => []);
  return {
    useTerminalSession: () => ({
      submitInput,
      gate,
      sendResize,

      subscribeInputSettled,
      subscribePendingInput,
      getPendingInputSnapshot,
    }),
  };
});
vi.mock("../hooks/useTerminalTouch", () => ({
  useTerminalTouch: () => ({
    hasSelection: false,
    copySelection: vi.fn(),
    clearSelection: vi.fn(),
  }),
}));
vi.mock("../stores/useWorkspaceStore", () => {
  const store = {
    panes: [],
    voiceShortcut: "",
    renamePaneById: vi.fn(),
    ttsVoice: "",
    ttsRate: 1,
    ttsPitch: 1,
    kokoroVoice: "af_heart",
    kokoroSpeed: 1,
    ttsBackendPreference: "auto" as const,
    autoTtsEnabled: false,
    setPendingInputDraft: vi.fn(),
    consumePendingInputDraft: vi.fn(() => undefined),
  };
  return {
    useWorkspaceStore: (selector?: (s: typeof store) => unknown) =>
      selector ? selector(store) : store,
  };
});
vi.mock("../hooks/useTextToSpeech", () => ({
  useTextToSpeech: () => ({
    supported: false,
    isSpeaking: false,
    backend: "none",
    voices: [],
    error: null,
    speak: vi.fn(),
    speakParagraphs: vi.fn(),
    stop: vi.fn(),
  }),
}));

// Use dynamic import to get the component after mocks are in place
const { default: TerminalPane } = await import("../components/TerminalPane");

describe("TerminalPane upload integration", () => {
  const savedFetch = globalThis.fetch;

  beforeEach(() => {
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
    globalThis.fetch = savedFetch;
  });

  it("paste event with image blob triggers upload", async () => {
    mockFetchSuccess({ path: "/tmp/web-console-uploads/s1/pasted.png" });

    render(<TerminalPane sessionId="s1" />);
    const pane = screen.getByTestId("terminal-pane");

    const file = new File(["png data"], "pasted.png", { type: "image/png" });
    const clipboardData = {
      items: [
        {
          type: "image/png",
          getAsFile: () => file,
        },
      ],
    };

    await act(async () => {
      fireEvent.paste(pane, { clipboardData });
    });

    expect(globalThis.fetch).toHaveBeenCalled();
  });

  it("paste event with text only does not trigger upload", async () => {
    mockFetchSuccess({ path: "/tmp/test.png" });

    await act(async () => {
      render(<TerminalPane sessionId="s2" />);
    });
    const pane = screen.getByTestId("terminal-pane");

    const clipboardData = {
      items: [
        {
          type: "text/plain",
          getAsFile: () => null,
        },
      ],
    };

    fireEvent.paste(pane, { clipboardData });

    expect(globalThis.fetch).not.toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/upload/image"),
      expect.anything(),
    );
  });

  it("drop event with image file triggers upload", async () => {
    mockFetchSuccess({ path: "/tmp/web-console-uploads/s1/dropped.png" });

    await act(async () => {
      render(<TerminalPane sessionId="s3" />);
    });
    const pane = screen.getByTestId("terminal-pane");

    const file = new File(["png data"], "dropped.png", { type: "image/png" });
    const dataTransfer = {
      files: [file],
    };

    await act(async () => {
      fireEvent.drop(pane, { dataTransfer });
    });

    expect(globalThis.fetch).toHaveBeenCalled();
  });

  it("drop event with non-image file is ignored", async () => {
    mockFetchSuccess({ path: "/tmp/test.txt" });

    await act(async () => {
      render(<TerminalPane sessionId="s4" />);
    });
    const pane = screen.getByTestId("terminal-pane");

    const file = new File(["text data"], "readme.txt", { type: "text/plain" });
    const dataTransfer = {
      files: [file],
    };

    fireEvent.drop(pane, { dataTransfer });

    expect(globalThis.fetch).not.toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/upload/image"),
      expect.anything(),
    );
  });
});
