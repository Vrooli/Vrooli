import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { StatusBadge } from "./StatusBadge";

describe("StatusBadge", () => {
  it("renders status text as a span when onStatusChange is not provided", () => {
    render(<StatusBadge status="ready" />);
    const badge = screen.getByTestId("status-badge");
    expect(badge.tagName).toBe("SPAN");
    expect(badge).toHaveTextContent("ready");
  });

  it("renders as a button when onStatusChange is provided", () => {
    render(<StatusBadge status="ready" onStatusChange={vi.fn()} />);
    const badge = screen.getByTestId("status-badge");
    expect(badge.tagName).toBe("BUTTON");
  });

  it("opens popover on click and calls onStatusChange", async () => {
    const user = userEvent.setup();
    const onStatusChange = vi.fn();
    render(<StatusBadge status="ready" onStatusChange={onStatusChange} />);

    await user.click(screen.getByTestId("status-badge"));
    expect(screen.getByTestId("status-badge-popover")).toBeInTheDocument();

    await user.click(screen.getByTestId("status-badge-option-researching"));
    expect(onStatusChange).toHaveBeenCalledWith("researching");
  });

  it("does not call onStatusChange when clicking the current status", async () => {
    const user = userEvent.setup();
    const onStatusChange = vi.fn();
    render(<StatusBadge status="ready" onStatusChange={onStatusChange} />);

    await user.click(screen.getByTestId("status-badge"));
    await user.click(screen.getByTestId("status-badge-option-ready"));
    expect(onStatusChange).not.toHaveBeenCalled();
  });

  it("does not open popover when statusChangePending is true", async () => {
    const user = userEvent.setup();
    render(<StatusBadge status="ready" onStatusChange={vi.fn()} statusChangePending />);

    await user.click(screen.getByTestId("status-badge"));
    expect(screen.queryByTestId("status-badge-popover")).not.toBeInTheDocument();
  });
});
