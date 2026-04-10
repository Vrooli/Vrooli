import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import "@testing-library/jest-dom/vitest";
import { KeyboardShortcutHelp } from "./KeyboardShortcutHelp";

describe("KeyboardShortcutHelp", () => {
  it("renders nothing when closed", () => {
    const { container } = render(<KeyboardShortcutHelp open={false} onClose={vi.fn()} />);
    expect(container.innerHTML).toBe("");
  });

  it("renders shortcut list when open", () => {
    render(<KeyboardShortcutHelp open={true} onClose={vi.fn()} />);
    expect(screen.getByTestId("keyboard-shortcut-help")).toBeInTheDocument();
    expect(screen.getByText("Keyboard Shortcuts")).toBeInTheDocument();
  });

  it("displays all canvas shortcuts", () => {
    render(<KeyboardShortcutHelp open={true} onClose={vi.fn()} />);
    expect(screen.getByText("Pan the canvas")).toBeInTheDocument();
    expect(screen.getByText("Zoom in")).toBeInTheDocument();
    expect(screen.getByText("Zoom out")).toBeInTheDocument();
    expect(screen.getByText("Toggle this help")).toBeInTheDocument();
  });

  it("calls onClose when close button is clicked", () => {
    const onClose = vi.fn();
    render(<KeyboardShortcutHelp open={true} onClose={onClose} />);
    fireEvent.click(screen.getByLabelText("Close shortcuts help"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("calls onClose when backdrop is clicked", () => {
    const onClose = vi.fn();
    render(<KeyboardShortcutHelp open={true} onClose={onClose} />);
    fireEvent.click(screen.getByTestId("keyboard-shortcut-help"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("does not close when dialog content is clicked", () => {
    const onClose = vi.fn();
    render(<KeyboardShortcutHelp open={true} onClose={onClose} />);
    fireEvent.click(screen.getByText("Keyboard Shortcuts"));
    expect(onClose).not.toHaveBeenCalled();
  });

  it("has dialog role with accessible label", () => {
    render(<KeyboardShortcutHelp open={true} onClose={vi.fn()} />);
    expect(screen.getByRole("dialog")).toHaveAttribute("aria-label", "Keyboard shortcuts");
  });
});
