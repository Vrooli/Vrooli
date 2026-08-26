import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach, beforeAll } from "vitest";
import { screen } from "@testing-library/react";

// jsdom doesn't provide ResizeObserver
beforeAll(() => {
  globalThis.ResizeObserver = vi.fn().mockImplementation(() => ({
    observe: vi.fn(),
    unobserve: vi.fn(),
    disconnect: vi.fn(),
  }));
});

// Mock xterm and addons so TerminalPane can mount without real canvas
vi.mock("@xterm/xterm", () => ({
  Terminal: vi.fn().mockImplementation(() => ({
    cols: 80,
    rows: 24,
    open: vi.fn(),
    write: vi.fn(),
    dispose: vi.fn(),
    focus: vi.fn(),
    loadAddon: vi.fn(),
    onData: vi.fn(() => ({ dispose: vi.fn() })),
    onTitleChange: vi.fn(() => ({ dispose: vi.fn() })),
    onSelectionChange: vi.fn(() => ({ dispose: vi.fn() })),
    scrollLines: vi.fn(),
    scrollToBottom: vi.fn(),
    select: vi.fn(),
    selectAll: vi.fn(),
    getSelection: vi.fn().mockReturnValue(""),
    getSelectionPosition: vi.fn(),
    clearSelection: vi.fn(),
    options: { fontSize: 14 },
    buffer: {
      active: {
        viewportY: 0,
        baseY: 0,
        length: 24,
        getLine: vi.fn().mockReturnValue({
          translateToString: vi.fn().mockReturnValue(""),
        }),
      },
    },
  })),
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

  const gate = { submit: vi.fn(() => ({ status: "sent" as const, offset: 1 })), dispose: vi.fn() };
  const submitInput = vi.fn(() => ({ status: "sent" as const, offset: 1 }));
  const sendResize = vi.fn();
  const getServerSize = vi.fn(() => null);
  const subscribeInputSettled = vi.fn(() => () => {});
  const subscribePendingInput = vi.fn(() => () => {});
  const getPendingInputSnapshot = vi.fn(() => []);
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

vi.mock("../stores/useWorkspaceStore", () => {
  const baseState = {
    panes: [{ sessionId: "test-session", fontSize: 14, themeId: "slate-ocean" }],
    renamePaneById: vi.fn(),
    startMutedOnLoad: true,
    deviceFontSize: {},
    setPendingInputBuffer: vi.fn(),
    consumePendingInputBuffer: vi.fn(() => undefined),
    setPendingInputDraft: vi.fn(),
    consumePendingInputDraft: vi.fn(() => undefined),
  };
  const useWorkspaceStore = (selector: (s: Record<string, unknown>) => unknown) =>
    selector(baseState);
  useWorkspaceStore.getState = () => baseState;
  return { useWorkspaceStore, useEffectiveFontSize: () => 14 };
});

// We need to mock useTerminalTouch to control hasSelection
let mockHasSelection = false;
const mockCopySelection = vi.fn().mockResolvedValue(true);
const mockClearSelection = vi.fn();

vi.mock("../hooks/useTerminalTouch", () => ({
  useTerminalTouch: () => ({
    hasSelection: mockHasSelection,
    copySelection: mockCopySelection,
    clearSelection: mockClearSelection,
  }),
}));

import TerminalPane from "../components/TerminalPane";

describe("TerminalPane touch integration", () => {
  beforeEach(() => {
    mockHasSelection = false;
    mockCopySelection.mockClear();
    mockClearSelection.mockClear();
  });

  it("does not render floating copy button (removed in favor of context menu)", () => {
    mockHasSelection = true;
    render(<TerminalPane sessionId="test-session" />);
    // The standalone copy button was removed — selection now triggers the
    // context menu via useTerminalTouch, which provides Copy/Speak actions.
    expect(screen.queryByTestId("touch-copy-btn")).toBeNull();
  });

  it("does not render context menu when no selection and menu not triggered", () => {
    mockHasSelection = false;
    render(<TerminalPane sessionId="test-session" />);
    expect(screen.queryByTestId("terminal-context-menu")).toBeNull();
  });
});
