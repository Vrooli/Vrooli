import { describe, expect, it } from "vitest";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { FreshnessTimestamp } from "./FreshnessTimestamp";

describe("FreshnessTimestamp", () => {
  it("renders an owner-provided observation time as a semantic time element", () => {
    renderWithProviders(<FreshnessTimestamp label="Board generated" value="2026-11-14T23:29:17.000Z" />);

    expect(screen.getByText("Board generated:")).toBeInTheDocument();
    expect(screen.getByTestId("freshness-timestamp").querySelector("time")).toHaveAttribute("datetime", "2026-11-14T23:29:17.000Z");
  });

  it("distinguishes a pending read from an absent or invalid observation time", () => {
    const { rerender } = renderWithProviders(<FreshnessTimestamp label="Source checked" loading />);
    expect(screen.getByTestId("freshness-timestamp")).toHaveTextContent("Source checked: checking");

    rerender(<FreshnessTimestamp label="Source checked" />);
    expect(screen.getByTestId("freshness-timestamp")).toHaveTextContent("Source checked: unknown");

    rerender(<FreshnessTimestamp label="Source checked" value="not-a-date" />);
    expect(screen.getByTestId("freshness-timestamp")).toHaveTextContent("Source checked: unknown");
  });
});
