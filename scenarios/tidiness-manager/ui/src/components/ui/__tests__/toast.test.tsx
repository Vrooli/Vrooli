import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ToastProvider, useToast } from "../toast";

// Test component to trigger toasts
function TestComponent() {
  const { showToast } = useToast();

  return (
    <div>
      <button onClick={() => showToast("Test message", "info")}>Show Info</button>
      <button onClick={() => showToast("Success message", "success")}>Show Success</button>
      <button onClick={() => showToast("Error message", "error")}>Show Error</button>
      <button onClick={() => showToast("Warning message", "warning")}>Show Warning</button>
      <button onClick={() => showToast("No auto-dismiss", "info", 0)}>Show Persistent</button>
    </div>
  );
}

describe("ToastProvider", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders children correctly", () => {
    render(
      <ToastProvider>
        <div>Test Child</div>
      </ToastProvider>
    );
    expect(screen.getByText("Test Child")).toBeInTheDocument();
  });

  it("renders notification region with proper aria-label", () => {
    render(
      <ToastProvider>
        <div>Test</div>
      </ToastProvider>
    );
    const region = screen.getByRole("region", { name: "Notifications" });
    expect(region).toBeInTheDocument();
  });

  describe("useToast hook", () => {
    it("throws error when used outside ToastProvider", () => {
      // Suppress console.error for this test
      const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

      expect(() => {
        const Component = () => {
          useToast();
          return null;
        };
        render(<Component />);
      }).toThrow("useToast must be used within ToastProvider");

      consoleError.mockRestore();
    });

    it("provides showToast function inside ToastProvider", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );
      expect(screen.getByText("Show Info")).toBeInTheDocument();
    });
  });

  describe("toast variants", () => {
    it("displays info toast", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Info"));
      expect(screen.getByText("Test message")).toBeInTheDocument();
    });

    it("displays success toast", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Success"));
      expect(screen.getByText("Success message")).toBeInTheDocument();
    });

    it("displays error toast", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Error"));
      expect(screen.getByText("Error message")).toBeInTheDocument();
    });

    it("displays warning toast", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Warning"));
      expect(screen.getByText("Warning message")).toBeInTheDocument();
    });
  });

  describe("toast accessibility", () => {
    it("has role='alert' on toast items", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Info"));
      const alert = screen.getByRole("alert");
      expect(alert).toBeInTheDocument();
    });

    it("has aria-live='polite' on toast items", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Info"));
      const alert = screen.getByRole("alert");
      expect(alert).toHaveAttribute("aria-live", "polite");
    });

    it("dismiss button has proper aria-label", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Info"));
      const dismissButton = screen.getByLabelText("Dismiss notification");
      expect(dismissButton).toBeInTheDocument();
    });

    it("icons have aria-hidden=true", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Info"));
      const alert = screen.getByRole("alert");
      const icon = alert.querySelector('svg[aria-hidden="true"]');
      expect(icon).toBeInTheDocument();
    });
  });

  describe("toast lifecycle", () => {
    it("automatically dismisses toast after default duration (5000ms)", async () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Info"));
      expect(screen.getByText("Test message")).toBeInTheDocument();

      // Advance timers and run all pending timers
      vi.advanceTimersByTime(5000);
      await vi.runAllTimersAsync();

      await waitFor(() => {
        expect(screen.queryByText("Test message")).not.toBeInTheDocument();
      }, { timeout: 500 });
    });

    it("does not auto-dismiss when duration is 0", async () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Persistent"));
      expect(screen.getByText("No auto-dismiss")).toBeInTheDocument();

      vi.advanceTimersByTime(10000);

      expect(screen.getByText("No auto-dismiss")).toBeInTheDocument();
    });

    it("dismisses toast when dismiss button is clicked", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Info"));
      expect(screen.getByText("Test message")).toBeInTheDocument();

      const dismissButton = screen.getByLabelText("Dismiss notification");
      fireEvent.click(dismissButton);

      expect(screen.queryByText("Test message")).not.toBeInTheDocument();
    });

    it("supports multiple toasts at once", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Info"));
      fireEvent.click(screen.getByText("Show Success"));
      fireEvent.click(screen.getByText("Show Error"));

      expect(screen.getByText("Test message")).toBeInTheDocument();
      expect(screen.getByText("Success message")).toBeInTheDocument();
      expect(screen.getByText("Error message")).toBeInTheDocument();
    });
  });

  describe("toast styling", () => {
    it("has slide-in animation", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Info"));
      const alert = screen.getByRole("alert");
      expect(alert).toHaveClass("animate-slide-in-right");
    });

    it("applies correct styling for info variant", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Info"));
      const alert = screen.getByRole("alert");
      expect(alert).toHaveClass("bg-blue-500/10", "border-blue-500/20", "text-blue-200");
    });

    it("applies correct styling for success variant", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Success"));
      const alert = screen.getByRole("alert");
      expect(alert).toHaveClass("bg-green-500/10", "border-green-500/20", "text-green-200");
    });

    it("applies correct styling for error variant", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Error"));
      const alert = screen.getByRole("alert");
      expect(alert).toHaveClass("bg-red-500/10", "border-red-500/20", "text-red-200");
    });

    it("applies correct styling for warning variant", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Warning"));
      const alert = screen.getByRole("alert");
      expect(alert).toHaveClass("bg-yellow-500/10", "border-yellow-500/20", "text-yellow-200");
    });

    it("has proper layout classes", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Info"));
      const alert = screen.getByRole("alert");
      expect(alert).toHaveClass("flex", "items-start", "gap-3", "p-4", "rounded-lg", "border", "shadow-lg");
    });

    it("notification region is positioned fixed bottom-right", () => {
      render(
        <ToastProvider>
          <div>Test</div>
        </ToastProvider>
      );

      const region = screen.getByRole("region", { name: "Notifications" });
      expect(region).toHaveClass("fixed", "bottom-4", "right-4", "z-50");
    });
  });

  describe("toast content", () => {
    it("renders message in monospace font", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Info"));
      const message = screen.getByText("Test message");
      expect(message).toHaveClass("font-mono");
    });

    it("supports multiline messages with whitespace-pre-line", () => {
      render(
        <ToastProvider>
          <TestComponent />
        </ToastProvider>
      );

      fireEvent.click(screen.getByText("Show Info"));
      const message = screen.getByText("Test message");
      expect(message).toHaveClass("whitespace-pre-line");
    });
  });
});
