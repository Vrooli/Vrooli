import { describe, it, expect, vi, beforeEach, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";

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

vi.mock("../hooks/useTerminalSocket", () => ({
  useTerminalSocket: () => ({
    sendInput: vi.fn(),
    sendResize: vi.fn(),
  }),
}));

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({
      terminalFontSize: 14,
      renamePaneById: vi.fn(),
    }),
}));

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

  it("does not render copy button when no selection", () => {
    mockHasSelection = false;
    render(<TerminalPane sessionId="test-session" />);
    expect(screen.queryByTestId("touch-copy-btn")).toBeNull();
  });

  it("renders copy button when hasSelection is true", () => {
    mockHasSelection = true;
    render(<TerminalPane sessionId="test-session" />);
    expect(screen.getByTestId("touch-copy-btn")).toBeTruthy();
  });

  it("copy button calls copySelection and clearSelection on click", async () => {
    mockHasSelection = true;
    render(<TerminalPane sessionId="test-session" />);

    const btn = screen.getByTestId("touch-copy-btn");
    btn.click();

    expect(mockCopySelection).toHaveBeenCalled();
    expect(mockClearSelection).toHaveBeenCalled();
  });
});
