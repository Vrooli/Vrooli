import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import SettingsModal from "../components/SettingsModal";
import { setDesktopViewport, setMobileViewport } from "../test-utils/viewport";

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
    setDesktopViewport();
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

  it("closes on backdrop press, not on panel press", () => {
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    const panel = screen.getByTestId("settings-modal");
    fireEvent.pointerDown(panel);
    expect(mockStoreState.setSettingsModalOpen).not.toHaveBeenCalled();
    // The backdrop dismisses on press, not on click: a click only lands after
    // the pointer is released over the same element, which a drag that starts
    // on the backdrop never satisfies.
    fireEvent.pointerDown(screen.getByTestId("settings-modal.backdrop"));
    expect(mockStoreState.setSettingsModalOpen).toHaveBeenCalledWith(false);
  });

  it("closes on Escape and renders dialog semantics", () => {
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    const panel = screen.getByTestId("settings-modal");
    expect(panel.getAttribute("role")).toBe("dialog");
    expect(panel.getAttribute("aria-modal")).toBe("true");
    fireEvent.keyDown(window, { key: "Escape" });
    expect(mockStoreState.setSettingsModalOpen).toHaveBeenCalledWith(false);
  });

  it("traps focus inside the settings dialog", () => {
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    const panel = screen.getByTestId("settings-modal");
    const tab = screen.getByTestId("settings-tab-sessions");
    tab.focus();
    fireEvent.keyDown(panel, { key: "Tab" });
    expect(panel.contains(document.activeElement)).toBe(true);
    fireEvent.keyDown(panel, { key: "Tab", shiftKey: true });
    expect(panel.contains(document.activeElement)).toBe(true);
  });

  it("renders mobile tabs row on mobile", () => {
    mediaQueryState.isMobile = true;
    setMobileViewport();
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    expect(screen.getByTestId("settings-tabs-row")).toBeTruthy();
    expect(screen.queryByTestId("settings-sidebar")).toBeNull();
  });

  it("offers a drag handle on mobile and a close button on desktop", () => {
    mediaQueryState.isMobile = true;
    setMobileViewport();
    const { unmount } = render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    expect(screen.getByTestId("settings-modal.grabber")).toBeTruthy();
    expect(screen.queryByTestId("settings-modal.close")).toBeNull();
    unmount();

    mediaQueryState.isMobile = false;
    setDesktopViewport();
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    expect(screen.getByTestId("settings-modal.close")).toBeTruthy();
    expect(screen.queryByTestId("settings-modal.grabber")).toBeNull();
  });

  it("dismisses when the mobile handle is dragged down", () => {
    mediaQueryState.isMobile = true;
    setMobileViewport();
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    const grabber = screen.getByTestId("settings-modal.grabber");
    fireEvent.pointerDown(grabber, { pointerId: 1, clientX: 100, clientY: 100 });
    fireEvent.pointerMove(grabber, { pointerId: 1, clientX: 100, clientY: 260 });
    fireEvent.pointerUp(grabber, { pointerId: 1, clientX: 100, clientY: 260 });
    expect(mockStoreState.setSettingsModalOpen).toHaveBeenCalledWith(false);
  });

  it("moves the sheet exactly as far as the finger", () => {
    mediaQueryState.isMobile = true;
    setMobileViewport();
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    const panel = screen.getByTestId("settings-modal");
    // jsdom lays nothing out, so the surface has to be given a height for the
    // drag to have something to be a fraction of.
    panel.getBoundingClientRect = () => ({ height: 600 }) as DOMRect;
    const grabber = screen.getByTestId("settings-modal.grabber");
    fireEvent.pointerDown(grabber, { pointerId: 1, clientX: 0, clientY: 100 });
    fireEvent.pointerMove(grabber, { pointerId: 1, clientX: 0, clientY: 160 });
    // The surface moves the same 60px the finger did. Two earlier shapes got
    // this wrong: translating by a fraction of the 96px commit threshold moved
    // it 375px, and routing the offset through an inherited custom property
    // fixed the distance but invalidated style for the whole subtree each
    // frame. The offset is a plain non-inheriting transform, in pixels.
    expect(panel.style.transform).toBe("translateY(60px)");
  });

  it("keeps the mobile handle reachable without a pointer", () => {
    mediaQueryState.isMobile = true;
    setMobileViewport();
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    fireEvent.keyDown(screen.getByTestId("settings-modal.grabber"), { key: "Enter" });
    expect(mockStoreState.setSettingsModalOpen).toHaveBeenCalledWith(false);
  });

  it("seats the mobile sheet flush with the bottom edge below a top gap", () => {
    mediaQueryState.isMobile = true;
    setMobileViewport();
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    // Addressed by prefix: the sheet key carries the stylesheet revision, and
    // pinning the whole key here would turn every unrelated drawer release
    // into a failure in this file.
    const sheet = document.head.querySelector(
      '[data-rcl-sheet^="full-page-drawer-"]',
    )?.textContent;
    expect(sheet).toBeTruthy();
    // The bug this guards: the panel used to inset itself from the bottom by
    // env(safe-area-inset-bottom), resolved against a viewport this app had
    // already narrowed, which left the sheet floating above the edge it slid
    // in from. The surface reaches the edge; the safe area is padding inside.
    expect(sheet).toContain("inset-block-end: 0");
    expect(sheet).toContain(
      "inset-block-start: calc(var(--rcl-safe-top, 0px) + var(--overlay-drawer-top-gap, 32px))",
    );
    expect(sheet).toContain("padding-block-end: var(--rcl-safe-bottom, 0px)");
  });

  it("gives the mobile tab strip the full width instead of the content gutter", () => {
    mediaQueryState.isMobile = true;
    setMobileViewport();
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    const subheader = screen.getByTestId("settings-modal.subheader");
    expect(subheader.contains(screen.getByTestId("settings-tabs-row"))).toBe(true);
    // The drawer pads its scroll region by default, which would indent the
    // strip and then indent web-console's own gutter on top of it.
    const drawer = screen.getByTestId("settings-modal").parentElement;
    expect(drawer?.getAttribute("data-content-padding")).toBe("none");
  });

  it("moves between mobile tabs with the arrow keys", () => {
    mediaQueryState.isMobile = true;
    setMobileViewport();
    render(<SettingsModal sessions={[]} onDeleteSession={vi.fn()} />);
    const sessions = screen.getByTestId("settings-tab-sessions");
    sessions.focus();
    fireEvent.keyDown(sessions, { key: "ArrowRight" });
    expect(screen.getByTestId("settings-tab-workspace").getAttribute("aria-selected")).toBe("true");
  });
});
