import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { KeyboardShortcuts } from "../keyboard-shortcuts";

describe("KeyboardShortcuts", () => {
  it("renders the help button by default", () => {
    render(<KeyboardShortcuts />);
    const button = screen.getByRole("button", { name: /show keyboard shortcuts/i });
    expect(button).toBeInTheDocument();
  });

  it("has proper aria-label on help button", () => {
    render(<KeyboardShortcuts />);
    const button = screen.getByLabelText("Show keyboard shortcuts");
    expect(button).toBeInTheDocument();
  });

  it("has Keyboard icon on help button", () => {
    render(<KeyboardShortcuts />);
    const button = screen.getByRole("button", { name: /show keyboard shortcuts/i });
    expect(button.querySelector('svg')).toBeInTheDocument();
  });

  describe("modal behavior", () => {
    it("opens modal when help button is clicked", () => {
      render(<KeyboardShortcuts />);
      const button = screen.getByRole("button", { name: /show keyboard shortcuts/i });
      fireEvent.click(button);

      expect(screen.getByRole("dialog")).toBeInTheDocument();
      expect(screen.getByText("Keyboard Shortcuts")).toBeInTheDocument();
    });

    it("modal has proper ARIA attributes", () => {
      render(<KeyboardShortcuts />);
      const button = screen.getByRole("button", { name: /show keyboard shortcuts/i });
      fireEvent.click(button);

      const dialog = screen.getByRole("dialog");
      expect(dialog).toHaveAttribute("aria-modal", "true");
      expect(dialog).toHaveAttribute("aria-labelledby", "keyboard-shortcuts-title");
    });

    it("shows backdrop when modal is open", () => {
      render(<KeyboardShortcuts />);
      const button = screen.getByRole("button", { name: /show keyboard shortcuts/i });
      fireEvent.click(button);

      const backdrop = document.querySelector('.fixed.inset-0.bg-black\\/50');
      expect(backdrop).toBeInTheDocument();
    });

    it("closes modal when close button is clicked", () => {
      render(<KeyboardShortcuts />);
      const openButton = screen.getByRole("button", { name: /show keyboard shortcuts/i });
      fireEvent.click(openButton);

      const closeButton = screen.getByLabelText("Close keyboard shortcuts");
      fireEvent.click(closeButton);

      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });

    it("closes modal when backdrop is clicked", () => {
      render(<KeyboardShortcuts />);
      const openButton = screen.getByRole("button", { name: /show keyboard shortcuts/i });
      fireEvent.click(openButton);

      const backdrop = document.querySelector('.fixed.inset-0.bg-black\\/50');
      fireEvent.click(backdrop!);

      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });

    it("closes modal when Escape key is pressed", () => {
      render(<KeyboardShortcuts />);
      const button = screen.getByRole("button", { name: /show keyboard shortcuts/i });
      fireEvent.click(button);

      fireEvent.keyDown(window, { key: "Escape" });

      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });
  });

  describe("keyboard shortcuts display", () => {
    it("displays General section shortcuts", () => {
      render(<KeyboardShortcuts />);
      const button = screen.getByRole("button", { name: /show keyboard shortcuts/i });
      fireEvent.click(button);

      expect(screen.getByText("General")).toBeInTheDocument();
      expect(screen.getByText("Show this help")).toBeInTheDocument();
      expect(screen.getByText("Focus search (when available)")).toBeInTheDocument();
      expect(screen.getByText("Close dialogs / Clear focus")).toBeInTheDocument();
    });

    it("displays Navigation section shortcuts", () => {
      render(<KeyboardShortcuts />);
      const button = screen.getByRole("button", { name: /show keyboard shortcuts/i });
      fireEvent.click(button);

      expect(screen.getByText("Navigation")).toBeInTheDocument();
      expect(screen.getByText("Go to Dashboard")).toBeInTheDocument();
      expect(screen.getByText("Go to Campaigns")).toBeInTheDocument();
      expect(screen.getByText("Go to Settings")).toBeInTheDocument();
    });

    it("displays keyboard key badges", () => {
      render(<KeyboardShortcuts />);
      const button = screen.getByRole("button", { name: /show keyboard shortcuts/i });
      fireEvent.click(button);

      const kbdElements = document.querySelectorAll("kbd");
      expect(kbdElements.length).toBeGreaterThan(0);
    });
  });

  describe("keyboard navigation functionality", () => {
    it("calls onNavigate when g+d is pressed", () => {
      const mockNavigate = vi.fn();
      render(<KeyboardShortcuts onNavigate={mockNavigate} />);

      fireEvent.keyDown(window, { key: "g" });
      fireEvent.keyDown(window, { key: "d" });

      expect(mockNavigate).toHaveBeenCalledWith("/dashboard");
    });

    it("calls onNavigate when g+c is pressed", () => {
      const mockNavigate = vi.fn();
      render(<KeyboardShortcuts onNavigate={mockNavigate} />);

      fireEvent.keyDown(window, { key: "g" });
      fireEvent.keyDown(window, { key: "c" });

      expect(mockNavigate).toHaveBeenCalledWith("/campaigns");
    });

    it("calls onNavigate when g+s is pressed", () => {
      const mockNavigate = vi.fn();
      render(<KeyboardShortcuts onNavigate={mockNavigate} />);

      fireEvent.keyDown(window, { key: "g" });
      fireEvent.keyDown(window, { key: "s" });

      expect(mockNavigate).toHaveBeenCalledWith("/settings");
    });

    it("does not trigger shortcuts when typing in input", () => {
      const mockNavigate = vi.fn();
      render(
        <>
          <input data-testid="test-input" />
          <KeyboardShortcuts onNavigate={mockNavigate} />
        </>
      );

      const input = screen.getByTestId("test-input");
      input.focus();

      fireEvent.keyDown(input, { key: "g" });
      fireEvent.keyDown(input, { key: "d" });

      expect(mockNavigate).not.toHaveBeenCalled();
    });

    it("does not trigger shortcuts when typing in textarea", () => {
      const mockNavigate = vi.fn();
      render(
        <>
          <textarea data-testid="test-textarea" />
          <KeyboardShortcuts onNavigate={mockNavigate} />
        </>
      );

      const textarea = screen.getByTestId("test-textarea");
      textarea.focus();

      fireEvent.keyDown(textarea, { key: "g" });
      fireEvent.keyDown(textarea, { key: "d" });

      expect(mockNavigate).not.toHaveBeenCalled();
    });

    it("toggles modal when ? is pressed", () => {
      render(<KeyboardShortcuts />);

      fireEvent.keyDown(window, { key: "?", shiftKey: false });
      expect(screen.getByRole("dialog")).toBeInTheDocument();

      fireEvent.keyDown(window, { key: "?", shiftKey: false });
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });
  });

  describe("responsive design", () => {
    it("has responsive padding on mobile (p-4) and desktop (sm:p-6)", () => {
      render(<KeyboardShortcuts />);
      const button = screen.getByRole("button", { name: /show keyboard shortcuts/i });
      fireEvent.click(button);

      // Find the header container (parent of the title)
      const header = screen.getByText("Keyboard Shortcuts").closest('div')?.parentElement;
      expect(header).toHaveClass("p-4", "sm:p-6");
    });

    it("has max-height constraint for scrolling", () => {
      render(<KeyboardShortcuts />);
      const button = screen.getByRole("button", { name: /show keyboard shortcuts/i });
      fireEvent.click(button);

      const dialog = screen.getByRole("dialog");
      expect(dialog).toHaveClass("max-h-[80vh]", "overflow-y-auto");
    });
  });
});
