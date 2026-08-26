import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import FloatingToolbar from "../components/FloatingToolbar";

vi.mock("../hooks/useDraggablePosition", () => ({
  useDraggablePosition: () => ({
    elementRef: { current: null },
    floatingStyle: { transform: "translate3d(100px, 12px, 0)" },
    pointerHandlers: {
      onPointerDown: vi.fn(),
      onPointerMove: vi.fn(),
      onPointerUp: vi.fn(),
      onPointerCancel: vi.fn(),
    },
    handleClickCapture: vi.fn(),
    resetPosition: vi.fn(),
    moveTo: vi.fn(),
    isDragging: false,
    position: { x: 100, y: 12 },
  }),
}));

vi.mock("../hooks/useLongPress", () => ({
  useLongPress: ({ onPress, onLongPress }: { onPress: () => void; onLongPress: () => void }) => ({
    onPointerDown: vi.fn(),
    onPointerUp: () => onPress(),
    onPointerCancel: vi.fn(),
    onContextMenu: (event: { preventDefault: () => void }) => {
      event.preventDefault();
      onLongPress();
    },
  }),
}));

describe("FloatingToolbar", () => {
  const onOpenSettings = vi.fn();
  const onOpenAi = vi.fn();
  const onNewTerminal = vi.fn();
  const onOpenLauncher = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.removeItem("wc-toolbar-dock");
  });

  const onExpandComposer = vi.fn();

  function renderToolbar(isCreating = false) {
    return render(
      <FloatingToolbar
        onOpenSettings={onOpenSettings}
        onOpenAi={onOpenAi}
        onNewTerminal={onNewTerminal}
        onOpenLauncher={onOpenLauncher}
        onExpandComposer={onExpandComposer}
        isCreating={isCreating}
      />,
    );
  }

  it("renders settings, AI, and new terminal buttons", () => {
    renderToolbar();
    expect(screen.getByTestId("floating-toolbar")).toBeTruthy();
    expect(screen.getByTestId("toolbar-settings")).toBeTruthy();
    expect(screen.getByTestId("toolbar-ai")).toBeTruthy();
    expect(screen.getByTestId("toolbar-new")).toBeTruthy();
    expect(screen.getByTestId("toolbar-settings")).toHaveClass("h-8", "w-8");
    expect(screen.getByTestId("toolbar-new")).toHaveClass("h-8", "w-8");
  });

  it("calls onOpenSettings when settings button is clicked", () => {
    renderToolbar();
    fireEvent.click(screen.getByTestId("toolbar-settings"));
    expect(onOpenSettings).toHaveBeenCalledOnce();
  });

  it("opens the composer when the expand-composer button is clicked", () => {
    renderToolbar();
    const btn = screen.getByTestId("toolbar-expand-composer");
    expect(btn).toBeTruthy();
    fireEvent.click(btn);
    expect(onExpandComposer).toHaveBeenCalledOnce();
  });

  it("calls onOpenLauncher on short press of the plus button (default launcher behavior)", () => {
    renderToolbar();
    fireEvent.pointerUp(screen.getByTestId("toolbar-new"));
    expect(onOpenLauncher).toHaveBeenCalledOnce();
  });

  it("calls onNewTerminal on right-click of the plus button (default launcher behavior)", () => {
    renderToolbar();
    fireEvent.contextMenu(screen.getByTestId("toolbar-new"));
    expect(onNewTerminal).toHaveBeenCalledOnce();
  });

  it("disables the plus button when creating", () => {
    renderToolbar(true);
    expect(screen.getByTestId("toolbar-new")).toHaveProperty("disabled", true);
  });

  it("does not show dock tab when not docked", () => {
    renderToolbar();
    expect(screen.queryByTestId("dock-tab")).toBeNull();
  });

  it("shows dock tab when docked via localStorage", () => {
    localStorage.setItem("wc-toolbar-dock", "right");
    renderToolbar();
    expect(screen.getByTestId("dock-tab")).toBeTruthy();
  });

  it("marks toolbar buttons aria-hidden when docked", () => {
    localStorage.setItem("wc-toolbar-dock", "left");
    renderToolbar();
    const buttonsContainer = screen.getByTestId("toolbar-settings").parentElement;
    expect(buttonsContainer?.getAttribute("aria-hidden")).toBe("true");
  });
});
