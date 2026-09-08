import { getSpatialNav, initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import { Drawer } from "./drawer";

// jsdom doesn't provide matchMedia (needed by useIsMobile).
beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

describe("Drawer", () => {
  it("does not render when isOpen is false", () => {
    render(
      <Drawer isOpen={false} onClose={vi.fn()} title="Test">
        <p>content</p>
      </Drawer>,
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("renders when isOpen is true", () => {
    render(
      <Drawer isOpen={true} onClose={vi.fn()} title="Test Title">
        <p>drawer content</p>
      </Drawer>,
    );
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("Test Title")).toBeInTheDocument();
    expect(screen.getByText("drawer content")).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    render(
      <Drawer isOpen={true} onClose={vi.fn()} title="T" description="A description">
        <p>body</p>
      </Drawer>,
    );
    expect(screen.getByText("A description")).toBeInTheDocument();
  });

  it("renders footer when provided", () => {
    render(
      <Drawer isOpen={true} onClose={vi.fn()} title="T" footer={<button>Save</button>}>
        <p>body</p>
      </Drawer>,
    );
    expect(screen.getByText("Save")).toBeInTheDocument();
  });

  it("calls onClose when X button is clicked", () => {
    const onClose = vi.fn();
    render(
      <Drawer isOpen={true} onClose={onClose} title="T">
        <p>body</p>
      </Drawer>,
    );
    fireEvent.click(screen.getByLabelText("Close drawer"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("calls onClose on Escape key press", () => {
    const onClose = vi.fn();
    render(
      <Drawer isOpen={true} onClose={onClose} title="T">
        <p>body</p>
      </Drawer>,
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("selects drawer controls and consumes gamepad Back without navigating history", () => {
    getSpatialNav()?.dispose();
    vi.useFakeTimers();
    let buttonIndex = -1;
    const controller = initSpatialNav({
      isVisible: () => true,
      getGamepads: () => [{ id: 'test-controller', timestamp: 0, vibrationActuator: { type: "dual-rumble", effects: [],
        playEffect: async () => "complete", reset: async () => "complete" },
        index: 0, mapping: 'standard', connected: true, axes: [0, 0],
        buttons: Array.from({ length: 17 }, (_, index) => ({ pressed: index === buttonIndex, touched: false, value: index === buttonIndex ? 1 : 0 })),
      } as Gamepad],
    });
    const onClose = vi.fn();
    const onSave = vi.fn();
    const historyBack = vi.spyOn(window.history, 'back').mockImplementation(() => {});
    const view = render(<Drawer isOpen onClose={onClose} title="Gamepad drawer">
      <button onClick={onSave}>Save with gamepad</button>
    </Drawer>);
    try {
      controller.enterSpatialMode();
      screen.getByText('Save with gamepad').focus();
      window.dispatchEvent(new Event('gamepadconnected'));
      act(() => { buttonIndex = 0; vi.advanceTimersByTime(16); });
      expect(onSave).toHaveBeenCalledOnce();
      act(() => { buttonIndex = -1; vi.advanceTimersByTime(16); });
      act(() => { buttonIndex = 1; vi.advanceTimersByTime(16); });
      expect(onClose).toHaveBeenCalledOnce();
      expect(historyBack).not.toHaveBeenCalled();
    } finally {
      view.unmount();
      controller.dispose();
      historyBack.mockRestore();
      vi.useRealTimers();
    }
  });

  it("has correct ARIA attributes", () => {
    render(
      <Drawer isOpen={true} onClose={vi.fn()} title="Accessible Title" testId="my-drawer">
        <p>body</p>
      </Drawer>,
    );
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(dialog).toHaveAttribute("data-testid", "my-drawer");
    // aria-labelledby should point to the title element
    const labelledBy = dialog.getAttribute("aria-labelledby");
    expect(labelledBy).toBeTruthy();
    const titleEl = labelledBy ? document.getElementById(labelledBy) : null;
    expect(titleEl?.textContent).toBe("Accessible Title");
  });
});
