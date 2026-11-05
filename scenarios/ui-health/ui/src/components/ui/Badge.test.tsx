import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Badge } from "./Badge";

describe("Badge", () => {
  it("renders neutral tone by default", () => {
    render(<Badge data-testid="b">x</Badge>);
    expect(screen.getByTestId("b").className).toMatch(/bg-app-surface-muted/);
  });
  it("applies semantic tones", () => {
    render(<Badge data-testid="b" tone="error">err</Badge>);
    expect(screen.getByTestId("b").className).toMatch(/text-app-danger/);
  });
});
