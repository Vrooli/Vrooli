import { screen, cleanup, fireEvent } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { ScrollHistoryHint } from "./ScrollHistoryHint";

afterEach(cleanup);

describe("ScrollHistoryHint", () => {
  it("renders nothing when not visible", () => {
    renderWithProviders(
      <ScrollHistoryHint visible={false} canViewMessages onViewMessages={() => {}} />,
    );
    expect(screen.queryByTestId("scroll-history-hint")).not.toBeInTheDocument();
  });

  it("shows the hint and a Messages shortcut when the pane supports it", () => {
    const onViewMessages = vi.fn();
    renderWithProviders(
      <ScrollHistoryHint visible canViewMessages onViewMessages={onViewMessages} />,
    );
    expect(screen.getByTestId("scroll-history-hint")).toBeInTheDocument();
    const button = screen.getByRole("button");
    fireEvent.click(button);
    expect(onViewMessages).toHaveBeenCalledTimes(1);
  });

  it("omits the Messages shortcut when the pane has no Messages view", () => {
    renderWithProviders(
      <ScrollHistoryHint visible canViewMessages={false} onViewMessages={() => {}} />,
    );
    expect(screen.getByTestId("scroll-history-hint")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
