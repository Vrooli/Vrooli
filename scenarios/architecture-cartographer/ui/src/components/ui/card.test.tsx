import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { Card, CardBody, CardFooter, CardHeader, CardTitle } from "./card";

describe("Card", () => {
  it("renders the surface token classes", () => {
    render(<Card data-testid="c">body</Card>);
    expect(screen.getByTestId("c").className).toMatch(/bg-app-surface/);
  });

  it("composes header, title, body, and footer", () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle data-testid="t">title</CardTitle>
        </CardHeader>
        <CardBody data-testid="b">body</CardBody>
        <CardFooter data-testid="f">footer</CardFooter>
      </Card>,
    );
    expect(screen.getByTestId("t").tagName).toBe("H3");
    expect(screen.getByTestId("b")).toHaveTextContent("body");
    expect(screen.getByTestId("f")).toHaveTextContent("footer");
  });
});
