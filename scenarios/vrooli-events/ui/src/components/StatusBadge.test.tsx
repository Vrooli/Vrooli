import { renderWithProviders } from "../test-utils";
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";

describe("shared enabled status badge", () => {
  it("shows the enabled label with the success tone", () => {
    renderWithProviders(<StatusBadge tone="success">On</StatusBadge>);
    expect(screen.getByText("On").closest("[data-rcl-status-badge]")).toHaveAttribute("data-tone", "success");
  });

  it("shows the disabled label with a neutral tone", () => {
    renderWithProviders(<StatusBadge tone="neutral">Off</StatusBadge>);
    expect(screen.getByText("Off").closest("[data-rcl-status-badge]")).toHaveAttribute("data-tone", "neutral");
  });
});
