import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders as renderWithCanonicalProviders } from "../../test-utils/renderWithProviders";
import userEvent from "@testing-library/user-event";
import { StatusChipPopover } from "./status-chip-popover";
import { USER_SETTABLE_STATUSES } from "../../types";

describe("StatusChipPopover", () => {
  it("renders current status with dot and label", () => {
    renderWithCanonicalProviders(<StatusChipPopover currentStatus="ready" onStatusChange={vi.fn()} />);
    expect(screen.getByText(/ready/i)).toBeInTheDocument();
    expect(screen.getByTestId("status-chip-trigger")).toBeInTheDocument();
  });

  it("opens popover on click with all settable statuses", async () => {
    const user = userEvent.setup();
    renderWithCanonicalProviders(<StatusChipPopover currentStatus="ready" onStatusChange={vi.fn()} />);

    await user.click(screen.getByTestId("status-chip-trigger"));
    const popover = screen.getByTestId("status-chip-popover");
    expect(popover).toBeInTheDocument();

    for (const status of USER_SETTABLE_STATUSES) {
      expect(screen.getByTestId(`status-option-${status}`)).toBeInTheDocument();
    }
  });

  it("calls onStatusChange when a different status is selected", async () => {
    const user = userEvent.setup();
    const onStatusChange = vi.fn();
    renderWithCanonicalProviders(<StatusChipPopover currentStatus="ready" onStatusChange={onStatusChange} />);

    await user.click(screen.getByTestId("status-chip-trigger"));
    await user.click(screen.getByTestId("status-option-researching"));

    expect(onStatusChange).toHaveBeenCalledWith("researching");
  });

  it("does not call onStatusChange when clicking the current status", async () => {
    const user = userEvent.setup();
    const onStatusChange = vi.fn();
    renderWithCanonicalProviders(<StatusChipPopover currentStatus="ready" onStatusChange={onStatusChange} />);

    await user.click(screen.getByTestId("status-chip-trigger"));
    await user.click(screen.getByTestId("status-option-ready"));

    expect(onStatusChange).not.toHaveBeenCalled();
  });

  it("does not open popover when pending", async () => {
    const user = userEvent.setup();
    renderWithCanonicalProviders(<StatusChipPopover currentStatus="ready" onStatusChange={vi.fn()} pending />);

    await user.click(screen.getByTestId("status-chip-trigger"));
    expect(screen.queryByTestId("status-chip-popover")).not.toBeInTheDocument();
  });

  it("renders a pulsing dot when status is in_review", () => {
    const { container } = renderWithCanonicalProviders(
      <StatusChipPopover currentStatus="in_review" onStatusChange={vi.fn()} />,
    );
    expect(container.querySelector(".animate-ping")).toBeTruthy();
  });

  it("does not pulse for review_pending (user decision needed, not busy)", () => {
    const { container } = renderWithCanonicalProviders(
      <StatusChipPopover currentStatus="review_pending" onStatusChange={vi.fn()} />,
    );
    expect(container.querySelector(".animate-ping")).toBeNull();
  });
});
