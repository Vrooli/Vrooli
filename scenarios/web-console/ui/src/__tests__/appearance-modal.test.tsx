import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import AppearanceModal from "../components/AppearanceModal";
import { HEADER_COLORS } from "../consts/config";

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
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector(mockStoreState),
}));

// Mock draggable position hook
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
    resetPosition: vi.fn(),
    isDragging: false,
    position: { x: 100, y: 100 },
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
    fireEvent.click(screen.getByTestId("appearance-backdrop"));
    expect(mockStoreState.setAppearanceModalPane).toHaveBeenCalledWith(null);
  });

  it("close button sets appearanceModalPane to null", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-close"));
    expect(mockStoreState.setAppearanceModalPane).toHaveBeenCalledWith(null);
  });

  it("clicking a header color swatch calls setPaneColor", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    const firstColor = HEADER_COLORS[0]!;
    fireEvent.click(screen.getByTestId(`appearance-header-color-${firstColor}`));
    expect(mockStoreState.setPaneColor).toHaveBeenCalledWith("sess-1", firstColor);
  });

  it("clicking transparent swatch calls setPaneColor with transparent", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-header-color-transparent"));
    expect(mockStoreState.setPaneColor).toHaveBeenCalledWith("sess-1", "transparent");
  });

  it("clicking a theme card calls setPaneTheme", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-theme-dracula"));
    expect(mockStoreState.setPaneTheme).toHaveBeenCalledWith("sess-1", "dracula");
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
  });

  it("font decrease button calls setPaneFontSize", () => {
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    fireEvent.click(screen.getByTestId("appearance-font-decrease"));
    expect(mockStoreState.setPaneFontSize).toHaveBeenCalledWith("sess-1", 13);
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

  it("displays current font size value", () => {
    mockStoreState.panes = [
      { sessionId: "sess-1", name: "bash", headerColor: "transparent", themeId: "slate-ocean", fontSize: 16 },
    ];
    mockStoreState.appearanceModalPane = "sess-1";
    render(<AppearanceModal />);
    expect(screen.getByTestId("appearance-font-value").textContent).toBe("16");
  });
});
