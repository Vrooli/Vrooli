import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { forwardRef } from "react";
import Workspace from "../components/Workspace";
import type { SessionInfo } from "../lib/api";

// [REQ:P0-001a] Responsive Pane Grid Layout — layout rendering
// [REQ:P0-001b] Independent Pane Session Lifecycle — pane lifecycle
// [REQ:P0-001c] Pane Management Controls — header controls
// [REQ:P0-006a] Terminal Launch Flow UI — launcher integration
// [REQ:P0-008a] Drawer Layout Component — drawer toggle

const mockSession: SessionInfo = {
  id: "sess-test-001",
  shell: "/bin/bash",
  created_at: "2026-01-15T14:30:00Z",
  cols: 80,
  rows: 24,
  policy: { mode: "never" },
};

// Track hook return values so tests can control pane state
let mockLaunchSession: ReturnType<typeof vi.fn>;
let mockRemovePane: ReturnType<typeof vi.fn>;
let mockClearError: ReturnType<typeof vi.fn>;
let mockSetActivePane: ReturnType<typeof vi.fn>;
let mockSendToActiveTerminal: ReturnType<typeof vi.fn>;
let mockHandleTerminalReady: ReturnType<typeof vi.fn>;
let mockHandleExit: ReturnType<typeof vi.fn>;
let mockRegisterTerminalRef: ReturnType<typeof vi.fn>;

// Shared mutable state to let tests control the hook's return
const hookState = {
  panes: [] as Array<{ session: SessionInfo }>,
  isCreating: false,
  createError: null as null | { message: string; recovery?: string; retry?: boolean },
  activePane: null as string | null,
};

vi.mock("../hooks/useSessionManager", () => ({
  useSessionManager: () => ({
    panes: hookState.panes,
    isCreating: hookState.isCreating,
    createError: hookState.createError,
    activePane: hookState.activePane,
    setActivePane: mockSetActivePane,
    clearError: mockClearError,
    launchSession: mockLaunchSession,
    handleTerminalReady: mockHandleTerminalReady,
    removePane: mockRemovePane,
    handleExit: mockHandleExit,
    sendToActiveTerminal: mockSendToActiveTerminal,
    registerTerminalRef: mockRegisterTerminalRef,
  }),
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
  default: vi.fn(() => <div data-testid="mock-mobile-toolbar" />),
}));

vi.mock("../components/AiInput", () => ({
  default: vi.fn(() => <div data-testid="mock-ai-input" />),
}));

vi.mock("../components/SessionDrawer", () => ({
  default: vi.fn(({ open }: { open: boolean }) =>
    open ? <div data-testid="mock-session-drawer">Drawer Open</div> : null,
  ),
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
    hookState.isCreating = false;
    hookState.createError = null;
    hookState.activePane = null;
    mockLaunchSession = vi.fn().mockResolvedValue(mockSession);
    mockRemovePane = vi.fn();
    mockClearError = vi.fn();
    mockSetActivePane = vi.fn();
    mockSendToActiveTerminal = vi.fn();
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

  it("provides retry handler in empty state when error has retry flag", () => {
    hookState.createError = { message: "Connection refused", retry: true };
    render(<Workspace />);
    fireEvent.click(screen.getByTestId("mock-retry"));
    expect(mockClearError).toHaveBeenCalledOnce();
    expect(mockLaunchSession).toHaveBeenCalledOnce();
  });

  // --- Pane grid state ---

  it("renders pane grid with terminal when panes exist", () => {
    hookState.panes = [{ session: mockSession }];
    hookState.activePane = mockSession.id;
    render(<Workspace />);
    expect(screen.getByTestId("pane-grid")).toBeTruthy();
    expect(screen.getByTestId(`mock-terminal-${mockSession.id}`)).toBeTruthy();
    expect(screen.getByText("1 terminal")).toBeTruthy();
  });

  it("renders multiple panes with correct pluralization", () => {
    const session2: SessionInfo = { ...mockSession, id: "sess-test-002" };
    hookState.panes = [{ session: mockSession }, { session: session2 }];
    render(<Workspace />);
    expect(screen.getByText("2 terminals")).toBeTruthy();
    expect(screen.getByTestId("mock-terminal-sess-test-001")).toBeTruthy();
    expect(screen.getByTestId("mock-terminal-sess-test-002")).toBeTruthy();
  });

  it("has drawer toggle button in header", () => {
    hookState.panes = [{ session: mockSession }];
    render(<Workspace />);
    expect(screen.getByTestId("drawer-toggle")).toBeTruthy();
  });

  it("toggles session drawer when drawer button is clicked", () => {
    hookState.panes = [{ session: mockSession }];
    render(<Workspace />);

    // Initially drawer not visible
    expect(screen.queryByTestId("mock-session-drawer")).toBeNull();

    // Click drawer toggle
    fireEvent.click(screen.getByTestId("drawer-toggle"));
    expect(screen.getByTestId("mock-session-drawer")).toBeTruthy();

    // Click again to close
    fireEvent.click(screen.getByTestId("drawer-toggle"));
    expect(screen.queryByTestId("mock-session-drawer")).toBeNull();
  });

  it("renders navigation buttons when onNavigate is provided", () => {
    hookState.panes = [{ session: mockSession }];
    const onNavigate = vi.fn();
    render(<Workspace onNavigate={onNavigate} />);
    expect(screen.getByTestId("nav-sessions")).toBeTruthy();
    expect(screen.getByTestId("nav-settings")).toBeTruthy();
  });

  it("calls onNavigate with 'sessions' when sessions nav is clicked", () => {
    hookState.panes = [{ session: mockSession }];
    const onNavigate = vi.fn();
    render(<Workspace onNavigate={onNavigate} />);
    fireEvent.click(screen.getByTestId("nav-sessions"));
    expect(onNavigate).toHaveBeenCalledWith("sessions");
  });

  it("calls onNavigate with 'settings' when settings nav is clicked", () => {
    hookState.panes = [{ session: mockSession }];
    const onNavigate = vi.fn();
    render(<Workspace onNavigate={onNavigate} />);
    fireEvent.click(screen.getByTestId("nav-settings"));
    expect(onNavigate).toHaveBeenCalledWith("settings");
  });

  it("shows error banner in pane view when createError exists", () => {
    hookState.panes = [{ session: mockSession }];
    hookState.createError = { message: "Server error", retry: true };
    render(<Workspace />);
    expect(screen.getByTestId("mock-error-banner")).toBeTruthy();
    expect(screen.getByText("Server error")).toBeTruthy();
  });

  it("renders AiInput and MobileToolbar in pane view", () => {
    hookState.panes = [{ session: mockSession }];
    render(<Workspace />);
    expect(screen.getByTestId("mock-ai-input")).toBeTruthy();
    expect(screen.getByTestId("mock-mobile-toolbar")).toBeTruthy();
  });

  it("opens launcher from header 'New' button in pane view", () => {
    hookState.panes = [{ session: mockSession }];
    render(<Workspace />);
    fireEvent.click(screen.getByTestId("new-terminal-button"));
    expect(screen.getByTestId("mock-launcher")).toBeTruthy();
  });
});
