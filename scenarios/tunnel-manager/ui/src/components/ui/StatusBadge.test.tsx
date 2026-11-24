/**
 * StatusBadge consumer contract — tone and visible status remain explicit.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";

import { StatusBadge, type StatusTone as BadgeTone } from "@vrooli/react-component-library/StatusBadge/1";

// provider-free-exception: this pure presentational leaf has no provider dependency.

describe("StatusBadge", () => {
  afterEach(() => cleanup());

  it("renders its children as the visible label", () => {
    render(
      <StatusBadge tone="success" data-testid="badge">
        Healthy
      </StatusBadge>,
    );
    expect(screen.getByTestId("badge")).toHaveTextContent("Healthy");
  });

  it.each<BadgeTone>(["success", "danger", "warning", "info", "neutral"])("exposes the %s tone and shared indicator", (tone) => {
    render(
      <StatusBadge tone={tone} data-testid="badge">
        x
      </StatusBadge>,
    );
    const badge = screen.getByTestId("badge");
    expect(badge).toHaveAttribute("data-rcl-status-badge");
    expect(badge).toHaveAttribute("data-tone", tone);
    expect(badge.querySelector("[data-rcl-status-badge-indicator]")).toBeInTheDocument();
  });
});
