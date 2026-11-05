import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Skeleton } from "./Skeleton";

describe("Skeleton", () => {
  it("renders an aria-hidden pulsing block", () => {
    render(<Skeleton data-testid="s" />);
    const el = screen.getByTestId("s");
    expect(el).toHaveAttribute("aria-hidden");
    expect(el.className).toMatch(/animate-pulse/);
  });
});
