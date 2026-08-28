import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { forwardRef } from "react";
import Workspace from "../components/Workspace";
import { strings } from "../consts/strings";
import { BANNER_CHROME } from "../components/banners/arbitrate";

/**
 * The colour actually painted into the iOS safe-area strip. The library's
 * `ChromeTheme` service writes it to the document element, which is the one
 * place both status-bar channels are resolved together, so asserting here is
 * asserting what the reader sees rather than which class happened to be
 * threaded through props.
 */
/*
 * Note the value asserted here is the library's per-tone FALLBACK. In a real
 * browser the region measures the banner's rendered background and publishes
 * that instead — which is the whole point, since the palette is `color-mix`
 * over tokens. jsdom computes no such colour, so the fallback is what this
 * environment can see; the measured path is asserted in the library's own
 * browser story.
 */
function statusFill(): string {
  return document.documentElement.style.getPropertyValue("--rcl-status-fill");
}
import type { SessionInfo } from "../api/sessions";
import type { ConversationEvent } from "../api/conversation";
import { useConversationStore, createConversationSessionState } from "../stores/useConversationStore";

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
    submitToActiveTerminal: mockSendToActiveTerminal,
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

vi.mock("../audio-integration", () => ({
  useScenarioVoiceInput: () => mockVoiceInputState,
  getTTSSummarizeConfig: vi.fn().mockResolvedValue({ level: "moderate" }),
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
  aiSuggestActive: false,
  groups: [],
  // Roles are additive, so this fixture keeps them empty: every assertion in
  // this file describes behaviour that must hold for a workspace that uses
  // no roles at all.
  roles: [] as Array<{ id: string; groupId: string; sessionId: string | null }>,
  closedGroupUndo: null,
  autoCloseEmptyGroups: true,
  manageGroupsOpen: false,
  sidebarView: "list" as const,
  sidebarSortMode: "manual" as const,
  sidebarOriginTab: "ui" as const,
  plusButtonBehavior: "launcher",
  defaultHeaderColor: "transparent",
  defaultThemeId: "default",
  defaultFontSize: 14,
  tabContextMenu: null as null | { sessionId: string; position: { x: number; y: number } },
  viewerCounts: {} as Record<string, number>,
};

const mockStoreActions = {
  addPane: vi.fn(),
  setRoles: vi.fn(),
  addRole: vi.fn(),
  updateRole: vi.fn(),
  removeRole: vi.fn(),
  setRoleSession: vi.fn(),
  setClosedGroupUndo: vi.fn(),
  setAutoCloseEmptyGroups: vi.fn(),
  setManageGroupsOpen: vi.fn(),
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
  setAiSuggestActive: vi.fn(),
  setTabContextMenu: vi.fn(),
  setSidebarView: vi.fn(),
  setSidebarSortMode: vi.fn(),
  setSidebarOriginTab: vi.fn(),
  resetLayout: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => {
  const useWorkspaceStoreMock = (selector?: (state: Record<string, unknown>) => unknown) => {
    const fullState = { ...mockStoreState, ...mockStoreActions };
    return selector ? selector(fullState) : fullState;
  };
  useWorkspaceStoreMock.getState = () => ({ ...mockStoreState, ...mockStoreActions });
  return { useWorkspaceStore: useWorkspaceStoreMock, useEffectiveFontSize: () => 14 };
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
  default: forwardRef<HTMLDivElement, { visible?: boolean; onOpenAi?: () => void; onAiSuggestExecute?: (cmd: string) => void }>(function MockMobileToolbar({ visible = true, onOpenAi, onAiSuggestExecute }, ref) {
    if (!visible) return null;
    return <div ref={ref} data-testid="mock-mobile-toolbar">
      {onOpenAi && <button data-testid="mock-mobile-ai" onClick={onOpenAi}>AI</button>}
      {onAiSuggestExecute && <button data-testid="mock-mobile-ai-execute" onClick={() => onAiSuggestExecute("pwd")}>Execute</button>}
    </div>;
  }),
}));

vi.mock("../components/AiInput", () => ({
  default: vi.fn(({ onExecute }: { onExecute?: (command: string) => void }) => <div data-testid="mock-ai-input">
    {onExecute && <button data-testid="mock-ai-execute" onClick={() => onExecute("echo ai")}>Execute</button>}
  </div>),
}));

vi.mock("../components/FloatingToolbar", () => ({
  default: vi.fn(({ onOpenSettings, onOpenAi, onNewTerminal, onOpenLauncher, onExpandComposer, isCreating: creating }: {
    onOpenSettings: () => void;
    onOpenAi?: () => void;
    onNewTerminal: () => void; onOpenLauncher: () => void; isCreating: boolean;
    onExpandComposer?: () => void;
  }) => (
    <div data-testid="floating-toolbar">
      <button data-testid="toolbar-settings" onClick={onOpenSettings}>Settings</button>
      {onOpenAi && <button data-testid="toolbar-ai" onClick={onOpenAi}>AI</button>}
      <button data-testid="toolbar-new" onClick={onNewTerminal} disabled={creating}>New</button>
      <button data-testid="toolbar-launcher" onClick={onOpenLauncher}>Launcher</button>
      {onExpandComposer && <button data-testid="toolbar-expand" onClick={onExpandComposer}>Expand</button>}
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
    mockStoreState.aiSuggestActive = false;
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
        // Creation must carry a position. Omitting sort_order left the pane at
        // the server's zero value, which sorted it to the top of the list on
        // the next reload and split whatever group was sitting there.
        sort_order: 0,
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

  it("routes floating settings and launcher lifecycle callbacks", async () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.panes = [{ sessionId: mockSession.id, name: "/bin/bash", headerColor: "transparent" }];
    touchControlsState.needsTouchControls = true;
    render(<Workspace />);
    fireEvent.click(screen.getByTestId("toolbar-settings"));
    expect(mockStoreActions.setSettingsModalOpen).toHaveBeenCalledWith(true);
    fireEvent.click(screen.getByTestId("toolbar-ai"));
    expect(mockStoreActions.setAiModalOpen).toHaveBeenCalledWith(true);
    fireEvent.click(screen.getByTestId("toolbar-expand"));
    fireEvent.click(screen.getByTestId("toolbar-launcher"));
    fireEvent.click(screen.getByTestId("mock-launcher-launch"));
    await waitFor(() => expect(mockLaunchSession).toHaveBeenCalled());
    const aiExecuteButtons = screen.getAllByTestId("mock-ai-execute");
    expect(aiExecuteButtons.length).toBeGreaterThan(0);
    fireEvent.click(aiExecuteButtons[0]!);
    fireEvent.click(screen.getByTestId("mock-mobile-ai"));
  });

  it("shows error banner in pane view when createError exists", () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.panes = [{ sessionId: mockSession.id, name: "/bin/bash", headerColor: "transparent" }];
    hookState.createError = { message: "Server error", retry: true };
    render(<Workspace />);
    // In pane view the failure is top chrome, arbitrated with every other
    // notice. In the empty state it stays inline beside the button that
    // produced it, so the two never say the same thing twice.
    expect(screen.getByTestId("create-error-banner")).toBeTruthy();
    expect(screen.queryByTestId("mock-error-banner")).toBeNull();
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

    fireEvent.click(screen.getByTestId("workspace-toggle-view"));
    expect(useConversationStore.getState().viewModes[mockSession.id]).toBe("messages");
  });

  it("responds to a viewport resize while panes are mounted", () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.panes = [{ sessionId: mockSession.id, name: "/bin/bash", headerColor: "transparent" }];
    const originalWidth = window.innerWidth;
    render(<Workspace />);

    Object.defineProperty(window, "innerWidth", { configurable: true, value: 500 });
    fireEvent(window, new Event("resize"));
    expect(screen.getByTestId("pane-grid")).toBeTruthy();
    Object.defineProperty(window, "innerWidth", { configurable: true, value: originalWidth });
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

    expect(screen.getByTestId("workspace-top-edge-fill")).toHaveAttribute(
      "data-rcl-status-fill-strip",
    );
    // Reserved exactly once: the strip owns the inset, and no surface below it
    // may claim the same space again.
    expect(screen.getByTestId("tab-bar").className).not.toContain("--wc-safe-top");
  });

  // Workspace ships the real damping policy, so a warning notice paints only
  // after it has outlasted the enter delay. That wait is the feature.
  it("renders the voice status notice as a dismissible top-edge banner", async () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.displayMode = "tabs";
    mockStoreState.activePane = mockSession.id;
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "Primary", headerColor: "transparent", supportsMessagesView: true },
    ];
    mockVoiceInputState.fallbackNotice = "Waiting for the speech backend to acknowledge the stream.";

    render(<Workspace />);

    const notice = await screen.findByTestId("voice-status-banner");
    expect(notice.textContent).toContain("Waiting for the speech backend");
    // The safe-area strip is tinted by whatever banner is on top, so the notch
    // matches the notice instead of the surface beneath it. On iOS this strip
    // *is* the status bar, so it is the notch, not a decoration near it.
    expect(statusFill()).toBe(BANNER_CHROME.warning.statusColor);

    fireEvent.click(screen.getByTestId("voice-status-banner-dismiss"));
    expect(mockVoiceInputState.dismissFallbackNotice).toHaveBeenCalledTimes(1);

    // Regression: dismissing must clear the notch even though the *condition*
    // still holds — the mocked notice is still set, exactly as a real one would
    // be until its source clears. Deriving the tint from the raw conditions
    // instead of the presented set left the status bar tinted for a banner the
    // reader had already closed.
    await waitFor(() => {
      expect(screen.queryByTestId("voice-status-banner")).toBeNull();
    });
    // Back to the resting chrome, not stuck on the notice's tone. The strip is
    // still painted — the base contribution always holds it — which is why the
    // assertion is "not the warning colour" rather than "unset".
    expect(statusFill()).not.toBe(BANNER_CHROME.warning.statusColor);
    expect(document.documentElement.dataset.rclStatusFill).toBe("base");
  });

  it("arbitrates App's notices into its own region rather than stacking a second one", async () => {
    hookState.panes = [{ session: mockSession }];
    mockStoreState.displayMode = "tabs";
    mockStoreState.activePane = mockSession.id;
    mockStoreState.panes = [
      { sessionId: mockSession.id, name: "Primary", headerColor: "transparent", supportsMessagesView: true },
    ];
    mockVoiceInputState.fallbackNotice = "Waiting for the speech backend to acknowledge the stream.";

    render(
      <Workspace
        appBanners={[
          {
            id: "connection-lost",
            testId: "connection-banner",
            tone: "danger",
            priority: 90,
            title: "Unable to reach the API",
          },
        ]}
      />,
    );

    // App no longer owns a second TopSafeArea; Workspace always owns the edge.
    expect(screen.getByTestId("workspace-top-edge-fill")).toBeTruthy();
    // A danger notice paints at once — something is broken and waiting would
    // be the wrong trade. The voice warning arrives after its enter delay and
    // collapses behind the summary rather than stacking below.
    expect(screen.getByTestId("connection-banner")).toBeTruthy();
    expect(screen.queryByTestId("voice-status-banner")).toBeNull();
    expect(await screen.findByTestId("banner-region-overflow-toggle")).toBeTruthy();
    expect(statusFill()).toBe(BANNER_CHROME.danger.statusColor);
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

    expect(screen.getByTestId("workspace-sidebar")).toBeTruthy();
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
    expect(screen.getByTestId("workspace-top-edge-fill")).toHaveAttribute(
      "data-rcl-status-fill-strip",
    );
    expect(screen.getByTestId("workspace-sidebar-topbar").className).toContain("h-10");
    expect(screen.getByTestId("workspace-sidebar-topbar").className).not.toContain("--wc-safe-top");
    fireEvent.click(screen.getByTestId("workspace-sidebar-toggle"));
    expect(screen.getByTestId("workspace-sidebar-backdrop")).toBeTruthy();
    expect(screen.getByTestId("workspace-sidebar").className).toContain("--wc-safe-top");
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
        [mockSession.id]: createConversationSessionState({
          events: [
            conversationEvent(mockSession.id, 1, "user"),
            conversationEvent(mockSession.id, 2),
            conversationEvent(mockSession.id, 3),
          ],
          cursor: { lastSeenSequence: 1, lastListenedSequence: 0 },
          hydrated: true,
        }),
        [session2.id]: createConversationSessionState({
          events: [conversationEvent(session2.id, 4)],
          cursor: { lastSeenSequence: 3, lastListenedSequence: 0 },
          hydrated: true,
        }),
      },
      viewModes: {},
    });

    render(<Workspace />);

    expect(screen.getByTestId("workspace-sidebar-toggle-unread")).toHaveTextContent("3");
    Object.defineProperty(window, "innerWidth", { configurable: true, value: originalWidth });
  });
});
