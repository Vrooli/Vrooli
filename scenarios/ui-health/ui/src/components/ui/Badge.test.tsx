import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { describe, expect, it } from "vitest";

import { Badge } from "./Badge";

describe("Badge", () => {
  it("renders neutral tone by default", () => {
    renderWithProviders(<Badge data-testid="b">x</Badge>);
    expect(screen.getByTestId("b").className).toMatch(/bg-app-surface-muted/);
  });
  it("applies semantic tones", () => {
    renderWithProviders(<Badge data-testid="b" tone="error">err</Badge>);
    expect(screen.getByTestId("b").className).toMatch(/text-app-danger/);
  });
});
