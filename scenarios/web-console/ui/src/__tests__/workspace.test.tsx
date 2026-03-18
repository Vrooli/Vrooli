import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { forwardRef } from "react";
import Workspace from "../components/Workspace";
import type { SessionInfo } from "../lib/api";

// [REQ:P0-001a] Responsive Pane Grid Layout — layout rendering
// [REQ:P0-001b] Independent Pane Session Lifecycle — pane lifecycle
// [REQ:P0-001c] Pane Management Controls — header controls
// [REQ:P0-006a] Terminal Launch Flow UI — launcher integration

const mockSession: SessionInfo = {
  id: "sess-test-001",
  shell: "/bin/bash",
  created_at: "2026-01-15T14:30:00Z",
  cols: 80,
  rows: 24,
  policy: { mode: "never" },
  busy: false,
};

// Track hook return values so tests can control pane state
let mockLaunchSession: ReturnType<typeof vi.fn>;
let mockRemovePane: ReturnType<typeof vi.fn>;
let mockClearError: ReturnType<typeof vi.fn>;
let mockSendToActiveTerminal: ReturnType<typeof vi.fn>;
let mockFocusActiveTerminal: ReturnType<typeof vi.fn>;
let mockHandleTerminalReady: ReturnType<typeof vi.fn>;
let mockHandleExit: ReturnType<typeof vi.fn>;
let mockRegisterTerminalRef: ReturnType<typeof vi.fn>;

// Shared mutable state to let tests control the hook's return
const hookState = {
  panes: [] as Array<{ session: SessionInfo }>,
  isHydrated: true,
  isCreating: false,
  createError: null as null | { message: string; recovery?: string; retry?: boolean },
};

vi.mock("../hooks/useSessionManager", () => ({
  useSessionManager: () => ({
    panes: hookState.panes,
    isHydrated: hookState.isHydrated,
    isCreating: hookState.isCreating,
    createError: hookState.createError,
    clearError: mockClearError,
    launchSession: mockLaunchSession,
    handleTerminalReady: mockHandleTerminalReady,
    removePane: mockRemovePane,
    handleExit: mockHandleExit,
    sendToActiveTerminal: mockSendToActiveTerminal,
    focusActiveTerminal: mockFocusActiveTerminal,
    registerTerminalRef: mockRegisterTerminalRef,
  }),
}));

vi.mock("../hooks/useWorkspaceSync", () => ({
  useWorkspaceSync: () => ({
    syncActivePane: vi.fn(),
    syncPaneOrder: vi.fn(),
    syncPaneUpdate: vi.fn(),
    syncCreateGroup: vi.fn(),
    syncUpdateGroup: vi.fn(),
    syncDeleteGroup: vi.fn(),
  }),
}));

// Mock workspace store
const mockStoreState = {
  panes: [] as Array<{ sessionId: string; name: string; headerColor: string }>,
  columnFractions: [] as number[],
  rowFractions: [] as number[],
  activePane: null as string | null,
  appearanceModalPane: null as string | null,
  isMinimapVisible: true,
  displayMode: "grid",
  settingsModalOpen: false,
  aiModalOpen: false,
};

const mockStoreActions = {
  addPane: vi.fn(),
  removePane: vi.fn(),
  renamePaneById: vi.fn(),
  setPaneColor: vi.fn(),
  setPaneTheme: vi.fn(),
  setPaneFontSize: vi.fn(),
  movePaneToIndex: vi.fn(),
  setColumnFractions: vi.fn(),
  setRowFractions: vi.fn(),
  setActivePane: vi.fn(),
  setAppearanceModalPane: vi.fn(),
  setMinimapVisible: vi.fn(),
  setDisplayMode: vi.fn(),
  setSettingsModalOpen: vi.fn(),
  setAiModalOpen: vi.fn(),
  resetLayout: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector?: (state: Record<string, unknown>) => unknown) => {
    const fullState = { ...mockStoreState, ...mockStoreActions };
    return selector ? selector(fullState) : fullState;
  },
}));

// Mock child components to isolate Workspace layout logic
vi.mock("../components/TerminalPane", () => ({
  default: forwardRef<HTMLDivElement, { sessionId: string }>(function MockTerminalPane({ sessionId }, ref) {
    return (
      <div ref={ref} data-testid={`mock-terminal-${sessionId}`}>
        Terminal {sessionId}
      </div>
    );
  }),
}));

vi.mock("../components/TerminalHeader", () => ({
  default: vi.fn(({ sessionId, name }: { sessionId: string; name: string }) => (
    <div data-testid={`mock-header-${sessionId}`}>{name}</div>
  )),
}));

vi.mock("../components/GridSplitter", () => ({
  default: vi.fn(({ axis }: { axis: string }) => (
    <div data-testid={`mock-splitter-${axis}`} />
  )),
}));

vi.mock("../components/SettingsModal", () => ({
  default: vi.fn(() => null),
}));

vi.mock("../components/WorkspaceMinimap", () => ({
  default: vi.fn(() => <div data-testid="mock-minimap" />),
}));

vi.mock("../components/TerminalLauncher", () => ({
  default: vi.fn(({ open, onClose, onLaunch, isCreating: creating }: {
    open: boolean; onClose: () => void; onLaunch: (cmd?: string) => void; isCreating: boolean;
  }) =>
    open ? (
      <div data-testid="mock-launcher">
        <button data-testid="mock-launcher-launch" onClick={() => onLaunch()}>Launch</button>
        <button data-testid="mock-launcher-close" onClick={onClose}>Close</button>
        {creating && <span>creating...</span>}
      </div>
    ) : null,
  ),
}));

vi.mock("../components/MobileToolbar", () => ({
  default: forwardRef<HTMLDivElement>(function MockMobileToolbar(_, ref) {
    return <div ref={ref} data-testid="mock-mobile-toolbar" />;
  }),
}));

vi.mock("../components/AiInput", () => ({
  default: vi.fn(() => <div data-testid="mock-ai-input" />),
}));

vi.mock("../components/FloatingToolbar", () => ({
  default: vi.fn(({ onOpenSettings, onNewTerminal, onOpenLauncher, isCreating: creating }: {
    onOpenSettings: () => void;
    onNewTerminal: () => void; onOpenLauncher: () => void; isCreating: boolean;
  }) => (
    <div data-testid="floating-toolbar">
      <button data-testid="toolbar-settings" onClick={onOpenSettings}>Settings</button>
      <button data-testid="toolbar-new" onClick={onNewTerminal} disabled={creating}>New</button>
      <button data-testid="toolbar-launcher" onClick={onOpenLauncher}>Launcher</button>
    </div>
  )),
}));

vi.mock("../components/ErrorBanner", () => ({
  default: vi.fn(({ error, onRetry }: { error: { message: string }; onRetry?: () => void }) => (
    <div data-testid="mock-error-banner">
      {error.message}
      {onRetry && <button data-testid="mock-retry" onClick={onRetry}>Retry</button>}
    </div>
  )),
}));

describe("Workspace", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    hookState.panes = [];
    hookState.isHydrated = true;
    hookState.isCreating = false;
    hookState.createError = null;
    mockStoreState.panes = [];
    mockStoreState.columnFractions = [];
    mockStoreState.rowFractions = [];
    mockStoreState.activePane = null;
    mockStoreState.isMinimapVisible = true;
    mockStoreState.settingsModalOpen = false;
    mockLaunchSession = vi.fn().mockResolvedValue(mockSession);
    mockRemovePane = vi.fn();
    mockClearError = vi.fn();
    mockSendToActiveTerminal = vi.fn();
    mockFocusActiveTerminal = vi.fn();
    mockHandleTerminalReady = vi.fn();
    mockHandleExit = vi.fn();
    mockRegisterTerminalRef = vi.fn();
  });

  // --- Empty state ---

  it("renders empty state with 'New Terminal' button when no panes", () => {
    render(<Workspace />);
    expect(screen.getByText("Web Console")).toBeTruthy();
    expect(screen.getByText("Browser terminal with PTY-backed sessions")).toBeTruthy();
    expect(screen.getByTestId("new-terminal-button")).toBeTruthy();
  });

  it("opens launcher when 'New Terminal' button is clicked in empty state", () => {
    render(<Workspace />);
    fireEvent.click(screen.getByTestId("new-terminal-button"));
    expect(screen.getByTestId("mock-launcher")).toBeTruthy();
  });

  it("shows 'Creating...' text when isCreating in empty state", () => {
    hookState.isCreating = true;
    render(<Workspace />);
    expect(screen.getByText("Creating...")).toBeTruthy();
  });

  it("shows error banner in empty state when createError exists", () => {
    hookState.createError = { message: "Connection refused", retry: true };
    render(<Workspace />);
    expect(screen.getByTestId("mock-error-banner")).toBeTruthy();
    expect(screen.getByText("Connection refused")).toBeTruthy();
  });

  it("provides retry handler in empty state when error has retry flag", async () => {
    hookState.createError = { message: "Connection refused", retry: true };
    render(<Workspace />);
    fireEvent.click(screen.getByTestId("mock-retry"));
    expect(mockClearError).toHaveBeenCalledOnce();
    await waitFor(() => {
      expect(mockLaunchSession).toHaveBeenCalledOnce();
    });
  });

  // --- Pane grid state ---

  it("renders pane grid with terminal when panes exist", () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.panes = [{ sessionId: mockSession.id, name: "/bin/bash", headerColor: "transparent" }];
    mockStoreState.activePane = mockSession.id;
    render(<Workspace />);
    expect(screen.getByTestId("pane-grid")).toBeTruthy();
    expect(screen.getByTestId(`mock-terminal-${mockSession.id}`)).toBeTruthy();
  });

  it("renders floating toolbar when panes exist", () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.panes = [{ sessionId: mockSession.id, name: "/bin/bash", headerColor: "transparent" }];
    render(<Workspace />);
    expect(screen.getByTestId("floating-toolbar")).toBeTruthy();
  });

  it("does not render floating toolbar in empty state", () => {
    render(<Workspace />);
    expect(screen.queryByTestId("floating-toolbar")).toBeNull();
  });

  it("toolbar new button launches empty shell", async () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.panes = [{ sessionId: mockSession.id, name: "/bin/bash", headerColor: "transparent" }];
    render(<Workspace />);
    fireEvent.click(screen.getByTestId("toolbar-new"));
    await waitFor(() => {
      expect(mockLaunchSession).toHaveBeenCalledOnce();
    });
  });

  it("toolbar launcher button opens launcher dialog", () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.panes = [{ sessionId: mockSession.id, name: "/bin/bash", headerColor: "transparent" }];
    render(<Workspace />);
    fireEvent.click(screen.getByTestId("toolbar-launcher"));
    expect(screen.getByTestId("mock-launcher")).toBeTruthy();
  });

  it("shows error banner in pane view when createError exists", () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.panes = [{ sessionId: mockSession.id, name: "/bin/bash", headerColor: "transparent" }];
    hookState.createError = { message: "Server error", retry: true };
    render(<Workspace />);
    expect(screen.getByTestId("mock-error-banner")).toBeTruthy();
    expect(screen.getByText("Server error")).toBeTruthy();
  });

  it("renders AiInput and MobileToolbar in pane view", () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.panes = [{ sessionId: mockSession.id, name: "/bin/bash", headerColor: "transparent" }];
    render(<Workspace />);
    expect(screen.getByTestId("mock-ai-input")).toBeTruthy();
    expect(screen.getByTestId("mock-mobile-toolbar")).toBeTruthy();
  });

  it("renders terminal headers for panes", () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.panes = [{ sessionId: mockSession.id, name: "/bin/bash", headerColor: "transparent" }];
    render(<Workspace />);
    expect(screen.getByTestId(`mock-header-${mockSession.id}`)).toBeTruthy();
  });

  it("renders multiple panes", () => {
    const session2: SessionInfo = { ...mockSession, id: "sess-test-002" };
    hookState.panes = [{ session: mockSession }, { session: session2 }];
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "/bin/bash", headerColor: "transparent" },
      { sessionId: session2.id, name: "/bin/bash", headerColor: "transparent" },
    ];
    render(<Workspace />);
    expect(screen.getByTestId("mock-terminal-sess-test-001")).toBeTruthy();
    expect(screen.getByTestId("mock-terminal-sess-test-002")).toBeTruthy();
  });
});
