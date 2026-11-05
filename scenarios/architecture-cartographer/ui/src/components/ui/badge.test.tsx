import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { Badge } from "./badge";

describe("Badge", () => {
  it("renders the default variant with base classes", () => {
    render(<Badge data-testid="b">x</Badge>);
    const el = screen.getByTestId("b");
    expect(el.className).toMatch(/rounded-pill/);
    expect(el.className).toMatch(/bg-app-surface-muted/);
  });

  it("applies a danger variant token", () => {
    render(
      <Badge data-testid="b" variant="danger">
        x
      </Badge>,
    );
    expect(screen.getByTestId("b").className).toMatch(/text-app-danger/);
  });

  it("merges custom className with base classes", () => {
    render(
      <Badge data-testid="b" className="custom">
        x
      </Badge>,
    );
    const el = screen.getByTestId("b");
    expect(el.className).toMatch(/custom/);
    expect(el.className).toMatch(/rounded-pill/);
  });
});
