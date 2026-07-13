import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import AppearanceModal from "../components/AppearanceModal";
import { HEADER_COLORS } from "../consts/config";
import { strings } from "../consts/strings";

const { mockSyncPaneUpdate } = vi.hoisted(() => ({
  mockSyncPaneUpdate: vi.fn(),
}));

// Mock workspace store
const mockStoreState: Record<string, unknown> = {
  appearanceModalPane: null,
  setAppearanceModalPane: vi.fn(),
  panes: [
    { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 14 },
  ],
  setPaneColor: vi.fn(),
  setPaneTheme: vi.fn(),
  setPaneFontSize: vi.fn(),
  applyAppearanceToAll: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector(mockStoreState),
}));

vi.mock("../hooks/useWorkspaceSync", () => ({
  useWorkspaceSync: () => ({
    syncPaneUpdate: mockSyncPaneUpdate,
  }),
}));

describe("AppearanceModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockStoreState.appearanceModalPane = null;
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 14 },
    ];
    mockStoreState.setPaneColor = vi.fn();
    mockStoreState.setPaneTheme = vi.fn();
    mockStoreState.setPaneFontSize = vi.fn();
    mockStoreState.setAppearanceModalPane = vi.fn();
    mockStoreState.applyAppearanceToAll = vi.fn();
    mockSyncPaneUpdate.mockClear();
  });

  it("does not render when appearanceModalPane is null", () => {
    render(<AppearanceModal />);
    expect(screen.queryByTestId("appearance-modal")).toBeNull();
  });

  it("renders when appearanceModalPane matches a pane sessionId", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    expect(screen.getByTestId("appearance-modal")).toBeTruthy();
  });

  it("backdrop click sets appearanceModalPane to null", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    const panel = screen.getByTestId("appearance-modal");
    const backdrop = panel.parentElement?.firstElementChild as HTMLElement;
    fireEvent.click(backdrop);
    expect(mockStoreState.setAppearanceModalPane).toHaveBeenCalledWith(null);
  });

  it("close button sets appearanceModalPane to null", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByLabelText(strings.appearance.closeAriaLabel));
    expect(mockStoreState.setAppearanceModalPane).toHaveBeenCalledWith(null);
  });

  it("closes on Escape and renders dialog semantics via DrawerShell compact", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    const panel = screen.getByTestId("appearance-modal");
    expect(panel.getAttribute("role")).toBe("dialog");
    expect(panel.getAttribute("aria-modal")).toBe("true");
    expect(panel.className).toContain("md:max-w-md");
    fireEvent.keyDown(window, { key: "Escape" });
    expect(mockStoreState.setAppearanceModalPane).toHaveBeenCalledWith(null);
  });

  it("clicking a header color swatch calls setPaneColor", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    const firstColor = HEADER_COLORS[0] ?? "transparent";
    fireEvent.click(screen.getByTestId(`appearance-header-color-${firstColor}`));
    expect(mockStoreState.setPaneColor).toHaveBeenCalledWith("sess-1", firstColor);
    expect(mockSyncPaneUpdate).toHaveBeenCalledWith("sess-1", { header_color: firstColor });
  });

  it("clicking transparent swatch calls setPaneColor with transparent", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-header-color-transparent"));
    expect(mockStoreState.setPaneColor).toHaveBeenCalledWith("sess-1", "transparent");
    expect(mockSyncPaneUpdate).toHaveBeenCalledWith("sess-1", { header_color: "transparent" });
  });

  it("clicking a theme card calls setPaneTheme", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-theme-dracula"));
    expect(mockStoreState.setPaneTheme).toHaveBeenCalledWith("sess-1", "dracula");
    expect(mockSyncPaneUpdate).toHaveBeenCalledWith("sess-1", { theme_id: "dracula" });
  });

  it("selected theme card has accent styling", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    const selected = screen.getByTestId("appearance-theme-slate-ocean");
    expect(selected.className).toContain("border-wc-accent");
    const other = screen.getByTestId("appearance-theme-dracula");
    expect(other.className).not.toContain("border-wc-accent");
  });

  it("font increase button calls setPaneFontSize", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-font-increase"));
    expect(mockStoreState.setPaneFontSize).toHaveBeenCalledWith("sess-1", 15);
    expect(mockSyncPaneUpdate).toHaveBeenCalledWith("sess-1", { font_size: 15 });
  });

  it("font decrease button calls setPaneFontSize", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-font-decrease"));
    expect(mockStoreState.setPaneFontSize).toHaveBeenCalledWith("sess-1", 13);
    expect(mockSyncPaneUpdate).toHaveBeenCalledWith("sess-1", { font_size: 13 });
  });

  it("font decrease disabled at FONT_SIZE_MIN", () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 8 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    expect(screen.getByTestId("appearance-font-decrease")).toBeDisabled();
  });

  it("font increase disabled at FONT_SIZE_MAX", () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 24 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    expect(screen.getByTestId("appearance-font-increase")).toBeDisabled();
  });

  it("does not show apply-all button with only one pane", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    expect(screen.queryByTestId("appearance-apply-all")).toBeNull();
  });

  it("shows apply-all button with multiple panes", () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 14 },
      { sessionId: "sess-2", name: "zsh", headerColor: "#ff7a7a", themeId: "dracula", fontSize: 16 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    expect(screen.getByTestId("appearance-apply-all")).toBeTruthy();
  });

  it("apply-all button calls applyAppearanceToAll with session id", () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 14 },
      { sessionId: "sess-2", name: "zsh", headerColor: "#ff7a7a", themeId: "dracula", fontSize: 16 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-apply-all"));
    expect(mockStoreState.applyAppearanceToAll).toHaveBeenCalledWith("sess-1");
    expect(mockSyncPaneUpdate).toHaveBeenCalledWith("sess-1", {
      header_color: "transparent",
      theme_id: "slate-ocean",
      font_size: 14,
    });
    expect(mockSyncPaneUpdate).toHaveBeenCalledWith("sess-2", {
      header_color: "transparent",
      theme_id: "slate-ocean",
      font_size: 14,
    });
  });

  it("displays current font size value", () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 16 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    expect(screen.getByTestId("appearance-font-value").textContent).toBe("16");
  });
});
