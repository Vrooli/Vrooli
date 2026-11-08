import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { describe, expect, it } from "vitest";

import { ProgressBar } from "./ProgressBar";

describe("ProgressBar", () => {
  it("renders progressbar role with aria-valuenow/min/max and label", () => {
    renderWithProviders(<ProgressBar value={42} label="Indexing" />);
    const bar = screen.getByRole("progressbar", { name: "Indexing" });
    expect(bar).toHaveAttribute("aria-valuenow", "42");
    expect(bar).toHaveAttribute("aria-valuemin", "0");
    expect(bar).toHaveAttribute("aria-valuemax", "100");
  });

  it("clamps out-of-range values", () => {
    renderWithProviders(<ProgressBar value={150} label="x" />);
    const bar = screen.getByRole("progressbar");
    expect(bar.getAttribute("aria-valuenow")).toBe("100");
  });
});
