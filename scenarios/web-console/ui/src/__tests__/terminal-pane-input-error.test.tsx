import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { render, screen, act, cleanup } from "@testing-library/react";
import { apiBaseMock } from "../test-utils";
import type { InputFailureReason } from "../hooks/terminal/useStdinAck";

/**
 * The pane must tell the user when input the backend refused is gone —
 * and must stay quiet when the client is going to retry it.
 *
 * Both halves matter. Before this, a rejected payload was silent for
 * everything typed or pasted through xterm: the server sent ok=false
 * with a typed reason and nothing rendered it. But announcing *every*
 * failure would be worse than useless, because ack timeouts and
 * connection closes re-enqueue the payload and send it again — the user
 * would see "it never reached the terminal" about bytes that arrive a
 * second later.
 */

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
  FitAddon: vi.fn().mockImplementation(() => ({ fit: vi.fn(), dispose: vi.fn() })),
}));
vi.mock("@xterm/addon-web-links", () => ({
  WebLinksAddon: vi.fn().mockImplementation(() => ({ dispose: vi.fn() })),
}));

// Captured settlement listeners, so a test can settle a seq the way the
// server (or the client's own timeout) would.
const settledListeners = new Set<
  (seq: number, ok: boolean, reason?: InputFailureReason) => void
>();

function settle(seq: number, ok: boolean, reason?: InputFailureReason): void {
  for (const cb of settledListeners) cb(seq, ok, reason);
}

vi.mock("../hooks/terminal/useTerminalSession", () => {
  // Every one of these must keep a stable identity across renders.
  // TerminalPane feeds them into effect dep arrays, so a fresh vi.fn()
  // per render re-runs those effects forever and the test OOMs rather
  // than failing with anything readable.
  const gate = {
    submit: vi.fn(() => ({ status: "sent" as const, seq: 1 })),
    dispose: vi.fn(),
    canAcceptPaste: () => true,
  };
  const submitInput = vi.fn(() => ({ status: "sent" as const, seq: 1 }));
  const sendResize = vi.fn();
  const getServerSize = vi.fn(() => null);
  const subscribePendingInput = vi.fn(() => () => {});
  const getPendingInputSnapshot = vi.fn(() => []);
  const subscribeInputSettled = vi.fn(
    (cb: (seq: number, ok: boolean, reason?: InputFailureReason) => void) => {
      settledListeners.add(cb);
      return () => settledListeners.delete(cb);
    },
  );
  return {
    useTerminalSession: () => ({
      submitInput,
      gate,
      sendResize,
      getServerSize,
      serverSize: null,
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
    deviceFontSize: {},
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

const { default: TerminalPane } = await import("../components/TerminalPane");

describe("TerminalPane rejected-input reporting", () => {
  beforeEach(() => {
    settledListeners.clear();
    if (typeof globalThis.ResizeObserver === "undefined") {
      globalThis.ResizeObserver = class {
        observe() {}
        unobserve() {}
        disconnect() {}
      } as unknown as typeof ResizeObserver;
    }
  });

  afterEach(cleanup);

  it("surfaces a backend rejection so dropped input is not silent", async () => {
    render(<TerminalPane sessionId="s1" />);
    expect(screen.queryByTestId("input-error")).toBeNull();

    await act(async () => {
      settle(1, false, "tmux_write_failed");
    });

    expect(screen.getByTestId("input-error")).toBeTruthy();
  });

  it("stays silent for failures the client retries", async () => {
    render(<TerminalPane sessionId="s1" />);

    await act(async () => {
      settle(1, false, "ack-timeout");
      settle(2, false, "connection-closed");
    });

    expect(screen.queryByTestId("input-error")).toBeNull();
  });

  it("stays silent when input succeeds", async () => {
    render(<TerminalPane sessionId="s1" />);

    await act(async () => {
      settle(1, true);
    });

    expect(screen.queryByTestId("input-error")).toBeNull();
  });
});
