import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ClarifyButton } from "./clarify-button";

describe("ClarifyButton", () => {
  it("renders without dot badge when hasClarification is false", () => {
    render(<ClarifyButton onClick={vi.fn()} />);
    expect(screen.queryByTestId("clarification-badge")).toBeNull();
  });

  it("renders dot badge when hasClarification is true", () => {
    render(<ClarifyButton hasClarification onClick={vi.fn()} />);
    expect(screen.getByTestId("clarification-badge")).toBeInTheDocument();
  });

  it("does not render dot badge when isActive is true", () => {
    render(<ClarifyButton hasClarification isActive onClick={vi.fn()} />);
    expect(screen.queryByTestId("clarification-badge")).toBeNull();
  });

  it("calls onClick when clicked", () => {
    const onClick = vi.fn();
    render(<ClarifyButton onClick={onClick} />);
    screen.getByRole("button").click();
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("does not call onClick when disabled", () => {
    const onClick = vi.fn();
    render(<ClarifyButton disabled onClick={onClick} />);
    screen.getByRole("button").click();
    expect(onClick).not.toHaveBeenCalled();
  });
});
