import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { describe, expect, it } from "vitest";

import { Skeleton } from "./Skeleton";

describe("Skeleton", () => {
  it("renders an aria-hidden pulsing block", () => {
    renderWithProviders(<Skeleton data-testid="s" />);
    const el = screen.getByTestId("s");
    expect(el).toHaveAttribute("aria-hidden");
    expect(el.className).toMatch(/animate-pulse/);
  });
});
