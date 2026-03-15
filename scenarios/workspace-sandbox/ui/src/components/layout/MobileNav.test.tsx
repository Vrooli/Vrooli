import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MobileNav } from "./MobileNav";

describe("MobileNav", () => {
  it("renders all 3 tabs", () => {
    render(<MobileNav activePanel="sandboxes" onPanelChange={vi.fn()} />);
    expect(screen.getByTestId("mobile-nav-sandboxes")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-nav-details")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-nav-changes")).toBeInTheDocument();
  });

  it("marks the active tab with aria-current", () => {
    render(<MobileNav activePanel="details" onPanelChange={vi.fn()} />);
    expect(screen.getByTestId("mobile-nav-details")).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId("mobile-nav-sandboxes")).not.toHaveAttribute("aria-current");
  });

  it("calls onPanelChange when a tab is clicked", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<MobileNav activePanel="sandboxes" onPanelChange={onChange} />);

    await user.click(screen.getByTestId("mobile-nav-details"));
    expect(onChange).toHaveBeenCalledWith("details");

    await user.click(screen.getByTestId("mobile-nav-changes"));
    expect(onChange).toHaveBeenCalledWith("changes");
  });

  it("shows change count badge when changeCount > 0", () => {
    render(<MobileNav activePanel="sandboxes" onPanelChange={vi.fn()} changeCount={5} />);
    expect(screen.getByText("5")).toBeInTheDocument();
  });

  it("does not show badge when changeCount is 0", () => {
    render(<MobileNav activePanel="sandboxes" onPanelChange={vi.fn()} changeCount={0} />);
    expect(screen.queryByText("0")).not.toBeInTheDocument();
  });

  it("caps badge display at 99+", () => {
    render(<MobileNav activePanel="sandboxes" onPanelChange={vi.fn()} changeCount={150} />);
    expect(screen.getByText("99+")).toBeInTheDocument();
  });

  it("has data-testid on the nav element", () => {
    render(<MobileNav activePanel="sandboxes" onPanelChange={vi.fn()} />);
    expect(screen.getByTestId("mobile-nav")).toBeInTheDocument();
  });
});
