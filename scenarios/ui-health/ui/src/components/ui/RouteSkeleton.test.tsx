import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { describe, expect, it } from "vitest";

import { RouteSkeleton } from "./RouteSkeleton";

describe("RouteSkeleton", () => {
  it("renders a live status region with the supplied label", () => {
    renderWithProviders(<RouteSkeleton label="loading-x" />);
    const el = screen.getByTestId("route-skeleton");
    expect(el).toHaveAttribute("role", "status");
    expect(el).toHaveAttribute("aria-live", "polite");
    expect(el).toHaveAttribute("aria-label", "loading-x");
  });
});
