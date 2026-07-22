import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import AppearanceModal from "../components/AppearanceModal";
import { HEADER_COLORS } from "../consts/config";
import { strings } from "../consts/strings";

const { mockSyncPaneUpdate, mockSyncPaneUpdates } = vi.hoisted(() => ({
  mockSyncPaneUpdate: vi.fn(),
  mockSyncPaneUpdates: vi.fn(async () => [] as string[]),
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
  applyAppearance: vi.fn(),
  defaultHeaderColor: "transparent",
  defaultThemeId: "slate-ocean",
  defaultFontSize: 14,
  setSettingsModalOpen: vi.fn(),
  setSettingsInitialTab: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector(mockStoreState),
}));

vi.mock("../hooks/useWorkspaceSync", () => ({
  useWorkspaceSync: () => ({
    syncPaneUpdate: mockSyncPaneUpdate,
    syncPaneUpdates: mockSyncPaneUpdates,
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
    mockStoreState.applyAppearance = vi.fn();
    mockStoreState.setSettingsModalOpen = vi.fn();
    mockStoreState.setSettingsInitialTab = vi.fn();
    mockSyncPaneUpdate.mockClear();
    mockSyncPaneUpdates.mockClear();
    mockSyncPaneUpdates.mockImplementation(async () => []);
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

  it("renders the composite preview", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    expect(screen.getByTestId("appearance-preview")).toBeTruthy();
  });

  it("does not show apply-to-open button with only one pane, but keeps set-default", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    expect(screen.queryByTestId("appearance-apply-open")).toBeNull();
    expect(screen.getByTestId("appearance-set-default")).toBeTruthy();
  });

  it("shows apply-to-open button with multiple panes", () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 14 },
      { sessionId: "sess-2", name: "zsh", headerColor: "#ff7a7a", themeId: "dracula", fontSize: 16 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    expect(screen.getByTestId("appearance-apply-open")).toBeTruthy();
  });

  it("apply-to-open requires confirmation, then applies and syncs other panes", async () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 14 },
      { sessionId: "sess-2", name: "zsh", headerColor: "#ff7a7a", themeId: "dracula", fontSize: 16 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);

    fireEvent.click(screen.getByTestId("appearance-apply-open"));
    // Nothing applied before the user confirms.
    expect(mockStoreState.applyAppearance).not.toHaveBeenCalled();
    expect(screen.getByTestId("appearance-apply-dialog")).toBeTruthy();

    fireEvent.click(screen.getByTestId("appearance-apply-confirm"));
    expect(mockStoreState.applyAppearance).toHaveBeenCalledWith("sess-1", {
      properties: ["headerColor", "themeId", "fontSize"],
      toExistingPanes: true,
      asNewPaneDefault: false,
    });
    expect(mockSyncPaneUpdates).toHaveBeenCalledWith(["sess-2"], {
      header_color: "transparent",
      theme_id: "slate-ocean",
      font_size: 14,
    });
    await screen.findByTestId("appearance-apply-feedback");
  });

  it("cancelling the confirmation applies nothing", () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 14 },
      { sessionId: "sess-2", name: "zsh", headerColor: "#ff7a7a", themeId: "dracula", fontSize: 16 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-apply-open"));
    fireEvent.click(screen.getByTestId("appearance-apply-cancel"));
    expect(mockStoreState.applyAppearance).not.toHaveBeenCalled();
    expect(mockSyncPaneUpdates).not.toHaveBeenCalled();
  });

  it("deselected properties are excluded from the bulk apply", async () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 14 },
      { sessionId: "sess-2", name: "zsh", headerColor: "#ff7a7a", themeId: "dracula", fontSize: 16 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);

    fireEvent.click(screen.getByTestId("appearance-prop-headerColor"));
    fireEvent.click(screen.getByTestId("appearance-prop-themeId"));
    fireEvent.click(screen.getByTestId("appearance-apply-open"));
    fireEvent.click(screen.getByTestId("appearance-apply-confirm"));

    expect(mockStoreState.applyAppearance).toHaveBeenCalledWith("sess-1", {
      properties: ["fontSize"],
      toExistingPanes: true,
      asNewPaneDefault: false,
    });
    expect(mockSyncPaneUpdates).toHaveBeenCalledWith(["sess-2"], { font_size: 14 });
    await screen.findByTestId("appearance-apply-feedback");
  });

  it("disables both apply buttons when no properties are selected", () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 14 },
      { sessionId: "sess-2", name: "zsh", headerColor: "#ff7a7a", themeId: "dracula", fontSize: 16 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-prop-headerColor"));
    fireEvent.click(screen.getByTestId("appearance-prop-themeId"));
    fireEvent.click(screen.getByTestId("appearance-prop-fontSize"));
    expect(screen.getByTestId("appearance-apply-open")).toBeDisabled();
    expect(screen.getByTestId("appearance-set-default")).toBeDisabled();
  });

  it("set-default saves selected properties as defaults without touching panes", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-set-default"));
    expect(mockStoreState.applyAppearance).toHaveBeenCalledWith("sess-1", {
      properties: ["headerColor", "themeId", "fontSize"],
      toExistingPanes: false,
      asNewPaneDefault: true,
    });
    expect(mockSyncPaneUpdates).not.toHaveBeenCalled();
    expect(screen.getByTestId("appearance-apply-feedback")).toBeTruthy();
  });

  it("shows an error message when some bulk syncs fail", async () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 14 },
      { sessionId: "sess-2", name: "zsh", headerColor: "#ff7a7a", themeId: "dracula", fontSize: 16 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    mockSyncPaneUpdates.mockImplementation(async () => ["sess-2"]);
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-apply-open"));
    fireEvent.click(screen.getByTestId("appearance-apply-confirm"));
    const feedback = await screen.findByTestId("appearance-apply-feedback");
    expect(feedback.textContent).toBe(strings.appearance.applySection.applyError);
  });

  it("reset-to-defaults restores this pane to the stored defaults and syncs", () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "#ff7a7a", themeId: "dracula", fontSize: 18 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-reset-defaults"));
    expect(mockStoreState.setPaneColor).toHaveBeenCalledWith("sess-1", "transparent");
    expect(mockStoreState.setPaneTheme).toHaveBeenCalledWith("sess-1", "slate-ocean");
    expect(mockStoreState.setPaneFontSize).toHaveBeenCalledWith("sess-1", 14);
    expect(mockSyncPaneUpdate).toHaveBeenCalledWith("sess-1", {
      header_color: "transparent",
      theme_id: "slate-ocean",
      font_size: 14,
    });
  });

  it("manage-defaults link closes the modal and deep-links settings", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-manage-defaults"));
    expect(mockStoreState.setAppearanceModalPane).toHaveBeenCalledWith(null);
    expect(mockStoreState.setSettingsInitialTab).toHaveBeenCalledWith("new-pane-defaults");
    expect(mockStoreState.setSettingsModalOpen).toHaveBeenCalledWith(true);
  });

  it("displays current font size value in the editable input", () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 16 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    expect((screen.getByTestId("appearance-font-value") as HTMLInputElement).value).toBe("16");
  });

  it("committing a typed font size clamps and applies it", () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 16 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    const input = screen.getByTestId("appearance-font-value") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "99" } });
    fireEvent.blur(input);
    expect(mockStoreState.setPaneFontSize).toHaveBeenCalledWith("sess-1", 24);
    expect(mockSyncPaneUpdate).toHaveBeenCalledWith("sess-1", { font_size: 24 });
  });
});
