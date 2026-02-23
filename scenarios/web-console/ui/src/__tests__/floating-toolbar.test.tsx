import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
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
    isDragging: false,
    position: { x: 100, y: 12 },
  }),
}));

vi.mock("../hooks/useLongPress", () => ({
  useLongPress: ({ onPress, onLongPress }: { onPress: () => void; onLongPress: () => void }) => ({
    onPointerDown: vi.fn(),
    onPointerUp: () => onPress(),
    onPointerCancel: vi.fn(),
    onContextMenu: (e: { preventDefault: () => void }) => {
      e.preventDefault();
      onLongPress();
    },
  }),
}));

describe("FloatingToolbar", () => {
  const onOpenSessions = vi.fn();
  const onOpenSettings = vi.fn();
  const onOpenAi = vi.fn();
  const onNewTerminal = vi.fn();
  const onOpenLauncher = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders all three buttons", () => {
    render(
      <FloatingToolbar
        onOpenSessions={onOpenSessions}
        onOpenSettings={onOpenSettings}
        onOpenAi={onOpenAi}
        onNewTerminal={onNewTerminal}
        onOpenLauncher={onOpenLauncher}
        isCreating={false}
      />,
    );
    expect(screen.getByTestId("floating-toolbar")).toBeTruthy();
    expect(screen.getByTestId("toolbar-sessions")).toBeTruthy();
    expect(screen.getByTestId("toolbar-settings")).toBeTruthy();
    expect(screen.getByTestId("toolbar-new")).toBeTruthy();
  });

  it("calls onOpenSessions when sessions button is clicked", () => {
    render(
      <FloatingToolbar
        onOpenSessions={onOpenSessions}
        onOpenSettings={onOpenSettings}
        onOpenAi={onOpenAi}
        onNewTerminal={onNewTerminal}
        onOpenLauncher={onOpenLauncher}
        isCreating={false}
      />,
    );
    fireEvent.click(screen.getByTestId("toolbar-sessions"));
    expect(onOpenSessions).toHaveBeenCalledOnce();
  });

  it("calls onOpenSettings when settings button is clicked", () => {
    render(
      <FloatingToolbar
        onOpenSessions={onOpenSessions}
        onOpenSettings={onOpenSettings}
        onOpenAi={onOpenAi}
        onNewTerminal={onNewTerminal}
        onOpenLauncher={onOpenLauncher}
        isCreating={false}
      />,
    );
    fireEvent.click(screen.getByTestId("toolbar-settings"));
    expect(onOpenSettings).toHaveBeenCalledOnce();
  });

  it("calls onNewTerminal on short press (pointerUp) of plus button", () => {
    render(
      <FloatingToolbar
        onOpenSessions={onOpenSessions}
        onOpenSettings={onOpenSettings}
        onOpenAi={onOpenAi}
        onNewTerminal={onNewTerminal}
        onOpenLauncher={onOpenLauncher}
        isCreating={false}
      />,
    );
    // The mock useLongPress fires onPress on pointerUp
    fireEvent.pointerUp(screen.getByTestId("toolbar-new"));
    expect(onNewTerminal).toHaveBeenCalledOnce();
  });

  it("calls onOpenLauncher on right-click of plus button", () => {
    render(
      <FloatingToolbar
        onOpenSessions={onOpenSessions}
        onOpenSettings={onOpenSettings}
        onOpenAi={onOpenAi}
        onNewTerminal={onNewTerminal}
        onOpenLauncher={onOpenLauncher}
        isCreating={false}
      />,
    );
    fireEvent.contextMenu(screen.getByTestId("toolbar-new"));
    expect(onOpenLauncher).toHaveBeenCalledOnce();
  });

  it("disables plus button when isCreating", () => {
    render(
      <FloatingToolbar
        onOpenSessions={onOpenSessions}
        onOpenSettings={onOpenSettings}
        onOpenAi={onOpenAi}
        onNewTerminal={onNewTerminal}
        onOpenLauncher={onOpenLauncher}
        isCreating={true}
      />,
    );
    expect(screen.getByTestId("toolbar-new")).toHaveProperty("disabled", true);
  });
});
