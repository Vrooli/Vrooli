import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { forwardRef } from "react";
import Workspace from "../components/Workspace";
import { strings } from "../consts/strings";
import type { SessionInfo } from "../api/sessions";
import type { ConversationEvent } from "../api/conversation";
import { useConversationStore } from "../stores/useConversationStore";

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
  backend: "standard",
  survives_restart: false,
  policy: { mode: "never" },
  busy: false,
  origin: "ui",
  owner: "",
  display_label: "",
};

const conversationEvent = (
  sessionId: string,
  sequence: number,
  role: "assistant" | "user" = "assistant",
): ConversationEvent => ({
  id: `${sessionId}-${sequence}`,
  sessionId,
  source: "test",
  role,
  text: "Test event",
  speechParagraphs: [],
  summarized: false,
  createdAt: "2026-05-14T12:00:00Z",
  sequence,
  deliveryState: "delivered",
  ttsState: "idle",
  consumptionState: "new",
});

// Track hook return values so tests can control pane state
let mockLaunchSession: ReturnType<typeof vi.fn>;
let mockRemovePane: ReturnType<typeof vi.fn>;
let mockClearError: ReturnType<typeof vi.fn>;
let mockSendToActiveTerminal: ReturnType<typeof vi.fn>;
let mockFocusActiveTerminal: ReturnType<typeof vi.fn>;
let mockHandleExit: ReturnType<typeof vi.fn>;
let mockRegisterTerminalRef: ReturnType<typeof vi.fn>;
const { mockSyncPaneUpdate, mockSyncPaneOrder } = vi.hoisted(() => ({
  mockSyncPaneUpdate: vi.fn(),
  mockSyncPaneOrder: vi.fn(),
}));
const { mockVoiceInputState } = vi.hoisted(() => ({
  mockVoiceInputState: {
    supported: true,
    backend: "whisper",
    voiceState: "idle",
    error: null,
    audioLevel: 0,
    voiceActivity: undefined,
    fallbackNotice: null as string | null,
    partialTranscript: "",
    voiceMode: "one-shot",
    segments: [],
    commandSuggestion: null,
    wakeWordConfigured: false,
    passiveListeningActive: false,
    staleLiveMicLease: false,
    rejectedAudio: null,
    speakerVerificationEnabled: false,
    speakerProfileConfigured: false,
    isRecording: false,
    isListening: false,
    isTranscribing: false,
    isPreparing: false,
    isPassive: false,
    isActive: false,
    prepareRecording: vi.fn(),
    startRecording: vi.fn(),
    stopRecording: vi.fn(),
    cancelTranscription: vi.fn(),
    dismissCommandSuggestion: vi.fn(),
    dismissFallbackNotice: vi.fn(),
    dismissRejection: vi.fn(),
    retryWithoutFilter: vi.fn(),
    enterPassiveMode: vi.fn(),
    exitPassiveMode: vi.fn(),
    releaseMicrophone: vi.fn(),
  },
}));

// Shared mutable state to let tests control the hook's return
const hookState = {
  panes: [] as Array<{ session: SessionInfo }>,
  isHydrated: true,
  isCreating: false,
  createError: null as null | { message: string; recovery?: string; retry?: boolean },
};

const touchControlsState = {
  needsTouchControls: false,
};

vi.mock("../hooks/useSessionManager", () => ({
  useSessionManager: () => ({
    panes: hookState.panes,
    isHydrated: hookState.isHydrated,
    isCreating: hookState.isCreating,
    createError: hookState.createError,
    clearError: mockClearError,
    launchSession: mockLaunchSession,
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
    syncPaneOrder: mockSyncPaneOrder,
    syncPaneUpdate: mockSyncPaneUpdate,
    syncCreateGroup: vi.fn(),
    syncUpdateGroup: vi.fn(),
    syncDeleteGroup: vi.fn(),
  }),
}));

vi.mock("../hooks/useTouchControls", () => ({
  useTouchControls: () => touchControlsState.needsTouchControls,
}));

vi.mock("../hooks/useVoiceInput", () => ({
  useVoiceInput: () => mockVoiceInputState,
}));

// Mock workspace store
const mockStoreState = {
  panes: [] as Array<{
    sessionId: string;
    name: string;
    headerColor: string;
    themeId?: string;
    fontSize?: number;
    groupId?: string | null;
    supportsMessagesView?: boolean;
  }>,
  columnFractions: [] as number[],
  rowFractions: [] as number[],
  activePane: null as string | null,
  appearanceModalPane: null as string | null,
  isMinimapVisible: true,
  displayMode: "grid",
  settingsModalOpen: false,
  aiModalOpen: false,
  groups: [],
  plusButtonBehavior: "launcher",
  defaultHeaderColor: "transparent",
  defaultThemeId: "default",
  defaultFontSize: 14,
  tabContextMenu: null as null | { sessionId: string; position: { x: number; y: number } },
};

const mockStoreActions = {
  addPane: vi.fn(),
  addPaneToGroup: vi.fn(),
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
  setTabContextMenu: vi.fn(),
  resetLayout: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => {
  const useWorkspaceStoreMock = (selector?: (state: Record<string, unknown>) => unknown) => {
    const fullState = { ...mockStoreState, ...mockStoreActions };
    return selector ? selector(fullState) : fullState;
  };
  useWorkspaceStoreMock.getState = () => ({ ...mockStoreState, ...mockStoreActions });
  return { useWorkspaceStore: useWorkspaceStoreMock };
});

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
    open: boolean; onClose: () => void; onLaunch: (opts?: Record<string, unknown>) => void; isCreating: boolean;
  }) =>
    open ? (
      <div data-testid="mock-launcher">
        <button data-testid="mock-launcher-launch" onClick={() => onLaunch({})}>Launch</button>
        <button data-testid="mock-launcher-close" onClick={onClose}>Close</button>
        {creating && <span>creating...</span>}
      </div>
    ) : null,
  ),
}));

vi.mock("../components/MobileToolbar", () => ({
  default: forwardRef<HTMLDivElement, { visible?: boolean }>(function MockMobileToolbar({ visible = true }, ref) {
    if (!visible) return null;
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
    mockVoiceInputState.fallbackNotice = null;
    mockVoiceInputState.dismissFallbackNotice.mockClear();
    touchControlsState.needsTouchControls = false;
    mockStoreState.panes = [];
    mockStoreState.columnFractions = [];
    mockStoreState.rowFractions = [];
    mockStoreState.activePane = null;
    mockStoreState.isMinimapVisible = true;
    mockStoreState.displayMode = "grid";
    mockStoreState.settingsModalOpen = false;
    mockStoreState.groups = [];
    mockStoreState.plusButtonBehavior = "launcher";
    mockStoreState.defaultHeaderColor = "transparent";
    mockStoreState.defaultThemeId = "default";
    mockStoreState.defaultFontSize = 14;
    mockStoreState.tabContextMenu = null;
    useConversationStore.setState({ sessions: {}, viewModes: {} });
    mockLaunchSession = vi.fn().mockResolvedValue(mockSession);
    mockRemovePane = vi.fn();
    mockClearError = vi.fn();
    mockSendToActiveTerminal = vi.fn();
    mockFocusActiveTerminal = vi.fn();
    mockHandleExit = vi.fn();
    mockRegisterTerminalRef = vi.fn();
    mockStoreActions.setTabContextMenu = vi.fn((next: typeof mockStoreState.tabContextMenu) => {
      mockStoreState.tabContextMenu = next;
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  // --- Empty state ---

  it("renders empty state with 'New Terminal' button when no panes", () => {
    render(<Workspace />);
    expect(screen.getByText(strings.app.title)).toBeTruthy();
    expect(screen.getByText(strings.workspace.tagline)).toBeTruthy();
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
    expect(screen.getByText(strings.workspace.creating)).toBeTruthy();
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

  it("persists selected appearance defaults when adding a new workspace pane", async () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.defaultHeaderColor = "#ff7a7a";
    mockStoreState.defaultThemeId = "dracula";
    mockStoreState.defaultFontSize = 18;

    render(<Workspace />);

    await waitFor(() => {
      expect(mockSyncPaneUpdate).toHaveBeenCalledWith(mockSession.id, {
        name: "bash",
        header_color: "#ff7a7a",
        theme_id: "dracula",
        font_size: 18,
        supports_messages_view: undefined,
      });
    });
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
    touchControlsState.needsTouchControls = true;
    render(<Workspace />);
    expect(screen.getByTestId("mock-ai-input")).toBeTruthy();
    expect(screen.getByTestId("mock-mobile-toolbar")).toBeTruthy();
  });

  it("hides the mobile toolbar on non-touch desktop viewports", () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.panes = [{ sessionId: mockSession.id, name: "/bin/bash", headerColor: "transparent" }];
    touchControlsState.needsTouchControls = false;

    render(<Workspace />);

    expect(screen.queryByTestId("mock-mobile-toolbar")).toBeNull();
  });

  it("keeps touch controls available on wide phone landscape viewports", () => {
    const originalWidth = window.innerWidth;
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 844 });
    hookState.panes = [{ session: mockSession }];
    mockStoreState.panes = [{ sessionId: mockSession.id, name: "/bin/bash", headerColor: "transparent" }];
    touchControlsState.needsTouchControls = true;

    render(<Workspace />);

    expect(screen.getByTestId("mock-mobile-toolbar")).toBeTruthy();
    Object.defineProperty(window, "innerWidth", { configurable: true, value: originalWidth });
  });

  it("keeps the terminal/messages toggle aligned to the pane control row", () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.activePane = mockSession.id;
    mockStoreState.displayMode = "tabs";
    mockStoreState.panes = [{
      sessionId: mockSession.id,
      name: "/bin/bash",
      headerColor: "transparent",
      supportsMessagesView: true,
    }];

    render(<Workspace />);

    const toggleShell = screen.getByTitle(strings.workspace.switchToMessagesTitle).parentElement;
    expect(toggleShell?.className).toContain("end-2");
    expect(toggleShell?.className).toContain("top-2.5");
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

  it("reserves top safe area once for tabs mode", () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.displayMode = "tabs";
    mockStoreState.activePane = mockSession.id;
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "Primary", headerColor: "transparent", supportsMessagesView: true },
    ];

    render(<Workspace />);

    expect(screen.getByTestId("workspace-top-edge-fill").className).toContain("--wc-safe-top");
    expect(screen.getByTestId("workspace-top-edge-fill").className).toContain("wc-chrome-surface");
    expect(screen.getByTestId("tab-bar").className).not.toContain("--wc-safe-top");
  });

  it("renders the voice status notice as a dismissible top-edge banner", () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.displayMode = "tabs";
    mockStoreState.activePane = mockSession.id;
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "Primary", headerColor: "transparent", supportsMessagesView: true },
    ];
    mockVoiceInputState.fallbackNotice = "Waiting for the speech backend to acknowledge the stream.";

    render(<Workspace />);

    expect(screen.getByTestId("voice-status-banner").textContent).toContain("Waiting for the speech backend");
    expect(screen.getByTestId("workspace-top-edge-fill").className).toContain("bg-amber-500/10");

    fireEvent.click(screen.getByTestId("voice-status-dismiss"));
    expect(mockVoiceInputState.dismissFallbackNotice).toHaveBeenCalledTimes(1);
  });

  it("does not double-reserve top safe area when parent already owns it", () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.displayMode = "tabs";
    mockStoreState.activePane = mockSession.id;
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "Primary", headerColor: "transparent", supportsMessagesView: true },
    ];

    render(<Workspace topSafeAreaReserved />);

    expect(screen.queryByTestId("workspace-top-edge-fill")).toBeNull();
    expect(screen.getByTestId("tab-bar")).toBeTruthy();
  });

  it("renders sidebar mode as tab-like stacked panes with desktop sidebar", () => {
    const session2: SessionInfo = { ...mockSession, id: "sess-test-002" };
    hookState.panes = [{ session: mockSession }, { session: session2 }];
    mockStoreState.displayMode = "sidebar";
    mockStoreState.activePane = mockSession.id;
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "Primary", headerColor: "transparent", supportsMessagesView: true },
      { sessionId: session2.id, name: "Secondary", headerColor: "transparent", supportsMessagesView: false },
    ];

    render(<Workspace />);

    expect(screen.getByTestId("workspace-sidebar-shell")).toBeTruthy();
    expect(screen.queryByTestId("tab-bar")).toBeNull();
    expect(screen.queryByTestId("pane-grid")).toBeNull();
    expect(screen.getByTestId(`tab-pane-${mockSession.id}`)).toBeTruthy();
    // Offscreen panes are now UNMOUNTED (warm-set scaling fix), not just hidden,
    // so the inactive, non-warm pane is absent from the DOM.
    expect(screen.queryByTestId(`tab-pane-${session2.id}`)).toBeNull();
  });

  it("opens and closes the mobile sidebar drawer", () => {
    const originalWidth = window.innerWidth;
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 500 });
    hookState.panes = [{ session: mockSession }];
    mockStoreState.displayMode = "sidebar";
    mockStoreState.activePane = mockSession.id;
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "Primary", headerColor: "transparent", supportsMessagesView: true },
    ];

    render(<Workspace />);
    expect(screen.getByTestId("workspace-top-edge-fill").className).toContain("--wc-safe-top");
    expect(screen.getByTestId("workspace-top-edge-fill").className).toContain("wc-chrome-surface");
    expect(screen.getByTestId("workspace-sidebar-topbar").className).toContain("h-10");
    expect(screen.getByTestId("workspace-sidebar-topbar").className).not.toContain("--wc-safe-top");
    fireEvent.click(screen.getByTestId("workspace-sidebar-toggle"));
    expect(screen.getByTestId("workspace-sidebar-backdrop")).toBeTruthy();
    expect(screen.getByTestId("workspace-sidebar-shell").className).toContain("--wc-safe-top");
    expect(screen.getByTestId("workspace-sidebar-new").parentElement?.className).toContain("border-b");
    expect(screen.getByTestId("workspace-sidebar-settings").parentElement?.className).toContain("border-b");

    fireEvent.click(screen.getByTestId("workspace-sidebar-backdrop"));
    expect(screen.queryByTestId("workspace-sidebar-backdrop")).toBeNull();
    Object.defineProperty(window, "innerWidth", { configurable: true, value: originalWidth });
  });

  it("activates a mobile sidebar session with one tap", () => {
    const originalWidth = window.innerWidth;
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 500 });
    const session2: SessionInfo = { ...mockSession, id: "sess-test-002" };
    hookState.panes = [{ session: mockSession }, { session: session2 }];
    mockStoreState.displayMode = "sidebar";
    mockStoreState.activePane = mockSession.id;
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "Primary", headerColor: "transparent", supportsMessagesView: true },
      { sessionId: session2.id, name: "Secondary", headerColor: "transparent", supportsMessagesView: false },
    ];

    render(<Workspace />);
    fireEvent.click(screen.getByTestId("workspace-sidebar-toggle"));
    fireEvent.click(screen.getByTestId(`sidebar-session-${session2.id}`));

    expect(mockStoreActions.setActivePane).toHaveBeenCalledTimes(1);
    expect(mockStoreActions.setActivePane).toHaveBeenCalledWith(session2.id);
    expect(screen.queryByTestId("workspace-sidebar-backdrop")).toBeNull();
    Object.defineProperty(window, "innerWidth", { configurable: true, value: originalWidth });
  });

  it("activates a mobile sidebar session on the first touch release", () => {
    const originalWidth = window.innerWidth;
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 500 });
    const session2: SessionInfo = { ...mockSession, id: "sess-test-002" };
    hookState.panes = [{ session: mockSession }, { session: session2 }];
    mockStoreState.displayMode = "sidebar";
    mockStoreState.activePane = mockSession.id;
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "Primary", headerColor: "transparent", supportsMessagesView: true },
      { sessionId: session2.id, name: "Secondary", headerColor: "transparent", supportsMessagesView: false },
    ];

    render(<Workspace />);
    fireEvent.click(screen.getByTestId("workspace-sidebar-toggle"));
    const row = screen.getByTestId(`sidebar-session-${session2.id}`);
    fireEvent.pointerDown(row, { pointerType: "touch", pointerId: 1, button: 0, clientX: 24, clientY: 120 });
    fireEvent.pointerUp(window, { pointerType: "touch", pointerId: 1, clientX: 24, clientY: 120 });

    expect(mockStoreActions.setActivePane).toHaveBeenCalledTimes(1);
    expect(mockStoreActions.setActivePane).toHaveBeenCalledWith(session2.id);
    expect(screen.queryByTestId("workspace-sidebar-backdrop")).toBeNull();
    Object.defineProperty(window, "innerWidth", { configurable: true, value: originalWidth });
  });

  it("does not activate or close the mobile sidebar when a touch moves like a swipe", () => {
    const originalWidth = window.innerWidth;
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 500 });
    const session2: SessionInfo = { ...mockSession, id: "sess-test-002" };
    hookState.panes = [{ session: mockSession }, { session: session2 }];
    mockStoreState.displayMode = "sidebar";
    mockStoreState.activePane = mockSession.id;
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "Primary", headerColor: "transparent", supportsMessagesView: true },
      { sessionId: session2.id, name: "Secondary", headerColor: "transparent", supportsMessagesView: false },
    ];

    render(<Workspace />);
    fireEvent.click(screen.getByTestId("workspace-sidebar-toggle"));
    const row = screen.getByTestId(`sidebar-session-${session2.id}`);
    fireEvent.pointerDown(row, { pointerType: "touch", pointerId: 1, button: 0, clientX: 24, clientY: 120 });
    fireEvent.pointerMove(window, { pointerType: "touch", pointerId: 1, clientX: 24, clientY: 180 });
    fireEvent.pointerUp(window, { pointerType: "touch", pointerId: 1, clientX: 24, clientY: 180 });
    fireEvent.click(row);

    expect(mockStoreActions.setActivePane).not.toHaveBeenCalled();
    expect(screen.getByTestId("workspace-sidebar-backdrop")).toBeTruthy();
    Object.defineProperty(window, "innerWidth", { configurable: true, value: originalWidth });
  });

  it("opens the tab context menu on mobile sidebar long press without activating", () => {
    vi.useFakeTimers();
    const originalWidth = window.innerWidth;
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 500 });
    const session2: SessionInfo = { ...mockSession, id: "sess-test-002" };
    hookState.panes = [{ session: mockSession }, { session: session2 }];
    mockStoreState.displayMode = "sidebar";
    mockStoreState.activePane = mockSession.id;
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "Primary", headerColor: "transparent", supportsMessagesView: true },
      { sessionId: session2.id, name: "Secondary", headerColor: "transparent", supportsMessagesView: false },
    ];

    render(<Workspace />);
    fireEvent.click(screen.getByTestId("workspace-sidebar-toggle"));
    const row = screen.getByTestId(`sidebar-session-${session2.id}`);
    fireEvent.pointerDown(row, { pointerType: "touch", pointerId: 1, button: 0, clientX: 24, clientY: 120 });
    vi.advanceTimersByTime(500);
    fireEvent.pointerUp(window, { pointerType: "touch", pointerId: 1, clientX: 24, clientY: 120 });

    expect(mockStoreActions.setActivePane).not.toHaveBeenCalled();
    expect(mockStoreActions.setTabContextMenu).toHaveBeenCalledWith({
      sessionId: session2.id,
      position: { x: 24, y: 120 },
    });
    expect(screen.getByTestId("workspace-sidebar-backdrop")).toBeTruthy();
    Object.defineProperty(window, "innerWidth", { configurable: true, value: originalWidth });
    vi.useRealTimers();
  });

  it("opens the tab context menu on mobile sidebar header title long press", () => {
    vi.useFakeTimers();
    const originalWidth = window.innerWidth;
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 500 });
    const session2: SessionInfo = { ...mockSession, id: "sess-test-002" };
    hookState.panes = [{ session: mockSession }, { session: session2 }];
    mockStoreState.displayMode = "sidebar";
    mockStoreState.activePane = session2.id;
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "Primary", headerColor: "transparent", supportsMessagesView: true },
      { sessionId: session2.id, name: "Secondary", headerColor: "transparent", supportsMessagesView: false },
    ];

    render(<Workspace />);
    const title = screen.getByTestId("workspace-sidebar-active-title");
    expect(title.className).toContain("select-none");
    fireEvent.pointerDown(title, { pointerType: "touch", pointerId: 1, button: 0, clientX: 120, clientY: 24 });
    vi.advanceTimersByTime(500);
    fireEvent.pointerUp(window, { pointerType: "touch", pointerId: 1, clientX: 120, clientY: 24 });

    expect(mockStoreActions.setTabContextMenu).toHaveBeenCalledWith({
      sessionId: session2.id,
      position: { x: 120, y: 24 },
    });
    expect(mockStoreActions.setActivePane).not.toHaveBeenCalled();
    Object.defineProperty(window, "innerWidth", { configurable: true, value: originalWidth });
    vi.useRealTimers();
  });

  it("opens the tab context menu on sidebar right click", () => {
    const session2: SessionInfo = { ...mockSession, id: "sess-test-002" };
    hookState.panes = [{ session: mockSession }, { session: session2 }];
    mockStoreState.displayMode = "sidebar";
    mockStoreState.activePane = mockSession.id;
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "Primary", headerColor: "transparent", supportsMessagesView: true },
      { sessionId: session2.id, name: "Secondary", headerColor: "transparent", supportsMessagesView: false },
    ];

    render(<Workspace />);
    fireEvent.contextMenu(screen.getByTestId(`sidebar-session-${session2.id}`), { clientX: 80, clientY: 40 });

    expect(mockStoreActions.setTabContextMenu).toHaveBeenCalledWith({
      sessionId: session2.id,
      position: { x: 80, y: 40 },
    });
    expect(mockStoreActions.setActivePane).not.toHaveBeenCalled();
  });

  it("shows total unread messages on the mobile sidebar toggle", () => {
    const originalWidth = window.innerWidth;
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 500 });
    const session2: SessionInfo = { ...mockSession, id: "sess-test-002" };
    hookState.panes = [{ session: mockSession }, { session: session2 }];
    mockStoreState.displayMode = "sidebar";
    mockStoreState.activePane = mockSession.id;
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "Primary", headerColor: "transparent", supportsMessagesView: true },
      { sessionId: session2.id, name: "Secondary", headerColor: "transparent", supportsMessagesView: true },
    ];
    useConversationStore.setState({
      sessions: {
        [mockSession.id]: {
          events: [
            conversationEvent(mockSession.id, 1, "user"),
            conversationEvent(mockSession.id, 2),
            conversationEvent(mockSession.id, 3),
          ],
          cursor: { lastSeenSequence: 1, lastListenedSequence: 0 },
          hydrated: true,
        },
        [session2.id]: {
          events: [conversationEvent(session2.id, 4)],
          cursor: { lastSeenSequence: 3, lastListenedSequence: 0 },
          hydrated: true,
        },
      },
      viewModes: {},
    });

    render(<Workspace />);

    expect(screen.getByTestId("workspace-sidebar-toggle-unread")).toHaveTextContent("3");
    Object.defineProperty(window, "innerWidth", { configurable: true, value: originalWidth });
  });
});
