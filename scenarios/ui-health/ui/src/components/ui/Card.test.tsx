/**
 * Card composition test.
 *
 * Primitive content here is intentionally synthetic (no i18n) — we are
 * exercising the structural slot contract, not user-visible copy. We
 * query by testid + role to keep the lint rule against copy-driven
 * queries happy.
 */
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Card, CardBody, CardDescription, CardFooter, CardHeader, CardTitle } from "./Card";

describe("Card", () => {
  it("renders header, title, description, body, and footer slots", () => {
    render(
      <Card data-testid="c">
        <CardHeader data-testid="c-header">
          <CardTitle data-testid="c-title">title-x</CardTitle>
          <CardDescription data-testid="c-desc">desc-x</CardDescription>
        </CardHeader>
        <CardBody data-testid="c-body">body-x</CardBody>
        <CardFooter data-testid="c-footer">foot-x</CardFooter>
      </Card>,
    );
    expect(screen.getByTestId("c")).toBeInTheDocument();
    expect(screen.getByTestId("c-header")).toBeInTheDocument();
    const title = screen.getByTestId("c-title");
    expect(title.tagName).toBe("H3");
    expect(title).toHaveTextContent("title-x");
    expect(screen.getByTestId("c-desc")).toHaveTextContent("desc-x");
    expect(screen.getByTestId("c-body")).toHaveTextContent("body-x");
    expect(screen.getByTestId("c-footer")).toHaveTextContent("foot-x");
  });

  it("applies token-based surface class", () => {
    render(<Card data-testid="c">x</Card>);
    expect(screen.getByTestId("c").className).toMatch(/bg-app-surface/);
  });
});
