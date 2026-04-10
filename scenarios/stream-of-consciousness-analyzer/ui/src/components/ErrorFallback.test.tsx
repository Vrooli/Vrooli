import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ErrorFallback } from "./ErrorFallback";

// [REQ:P0-001] Shared error fallback UI

describe("ErrorFallback", () => {
  it("renders the message and retry button", () => {
    const onRetry = vi.fn();
    render(<ErrorFallback message="Something broke" onRetry={onRetry} />);

    expect(screen.getByText("Something broke")).toBeDefined();
    expect(screen.getByText("Try again")).toBeDefined();
  });

  it("calls onRetry when retry button is clicked", () => {
    const onRetry = vi.fn();
    render(<ErrorFallback message="Error" onRetry={onRetry} />);

    fireEvent.click(screen.getByText("Try again"));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it("renders optional detail text", () => {
    render(<ErrorFallback message="Error" detail="Extra info" onRetry={vi.fn()} />);
    expect(screen.getByText("Extra info")).toBeDefined();
  });

  it("renders custom retry label", () => {
    render(<ErrorFallback message="Error" retryLabel="Retry now" onRetry={vi.fn()} />);
    expect(screen.getByText("Retry now")).toBeDefined();
  });

  it("renders secondary action when provided", () => {
    const onClick = vi.fn();
    render(
      <ErrorFallback
        message="Error"
        onRetry={vi.fn()}
        secondaryAction={{ label: "Reload", onClick }}
      />,
    );

    const btn = screen.getByText("Reload");
    expect(btn).toBeDefined();
    fireEvent.click(btn);
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("does not render secondary action when not provided", () => {
    render(<ErrorFallback message="Error" onRetry={vi.fn()} />);
    expect(screen.queryByText("Reload page")).toBeNull();
  });

  it("has role=alert for accessibility", () => {
    render(<ErrorFallback message="Error" onRetry={vi.fn()} />);
    expect(screen.getByRole("alert")).toBeDefined();
  });
});
