import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import SettingsModal from "../components/SettingsModal";

const mockStoreState = {
  settingsModalOpen: true,
  setSettingsModalOpen: vi.fn(),
};

const mediaQueryState = {
  isMobile: false,
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) => selector(mockStoreState),
}));

vi.mock("../hooks/useMediaQuery", () => ({
  useMediaQuery: () => mediaQueryState.isMobile,
}));

vi.mock("../hooks/useDraggablePosition", () => ({
  useDraggablePosition: () => ({
    elementRef: { current: null },
    floatingStyle: { transform: "translate3d(100px, 100px, 0)" },
    pointerHandlers: {
      onPointerDown: vi.fn(),
      onPointerMove: vi.fn(),
      onPointerUp: vi.fn(),
      onPointerCancel: vi.fn(),
    },
    handleClickCapture: vi.fn(),
  }),
}));

vi.mock("../components/settings/SessionManagementSection", () => ({
  default: () => <div data-testid="sessions-section">Sessions section</div>,
}));
vi.mock("../components/settings/WorkspaceSection", () => ({
  default: () => <div data-testid="workspace-section">Workspace section</div>,
}));
vi.mock("../components/settings/VoiceInputSection", () => ({
  default: () => <div data-testid="voice-input-section">Voice input section</div>,
}));
vi.mock("../components/settings/TtsSettingsSection", () => ({
  default: () => <div data-testid="voice-output-section">Voice output section</div>,
}));
vi.mock("../components/settings/ShortcutProfilesSection", () => ({
  default: () => <div data-testid="shortcuts-section">Shortcuts section</div>,
}));
vi.mock("../components/settings/NewPaneDefaultsSection", () => ({
  default: () => <div data-testid="defaults-section">Defaults section</div>,
}));
vi.mock("../components/settings/IntegrationsSection", () => ({
  default: () => <div data-testid="integrations-section">Integrations section</div>,
}));

describe("SettingsModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockStoreState.settingsModalOpen = true;
    mediaQueryState.isMobile = false;
  });

  it("does not render when closed", () => {
    mockStoreState.settingsModalOpen = false;
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    expect(screen.queryByTestId("settings-modal")).toBeNull();
  });

  it("renders desktop shell with sidebar by default", () => {
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    expect(screen.getByTestId("settings-modal")).toBeTruthy();
    expect(screen.getByTestId("settings-sidebar")).toBeTruthy();
    expect(screen.getByTestId("workspace-section")).toBeTruthy();
  });

  it("switches sections when a desktop tab is clicked", () => {
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    fireEvent.click(screen.getByTestId("settings-tab-sessions"));
    expect(screen.getByTestId("sessions-section")).toBeTruthy();
  });

  it("closes on backdrop click", () => {
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    fireEvent.click(screen.getByTestId("settings-backdrop"));
    expect(mockStoreState.setSettingsModalOpen).toHaveBeenCalledWith(false);
  });

  it("renders mobile tabs row on mobile", () => {
    mediaQueryState.isMobile = true;
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    expect(screen.getByTestId("settings-tabs-row")).toBeTruthy();
    expect(screen.queryByTestId("settings-sidebar")).toBeNull();
  });
});
