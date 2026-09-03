import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { renderWithProviders } from "../test-utils";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "./Card";

describe("Card primitives", () => {
  afterEach(() => cleanup());

  it("renders each part with its base classes merged with a custom className", () => {
    renderWithProviders(
      <Card data-testid="card" className="custom-card">
        <CardHeader data-testid="header" className="custom-header">
          <CardTitle data-testid="title" className="custom-title">
            Title
          </CardTitle>
          <CardDescription data-testid="description" className="custom-description">
            Description
          </CardDescription>
        </CardHeader>
        <CardContent data-testid="content" className="custom-content">
          Body
        </CardContent>
      </Card>,
    );

    // Base chunk (cn/twMerge output) plus the caller's custom class both survive.
    const card = screen.getByTestId("card");
    expect(card.className).toContain("rounded-panel");
    expect(card.className).toContain("custom-card");

    expect(screen.getByTestId("header").className).toContain("custom-header");
    expect(screen.getByTestId("title").tagName).toBe("H3");
    expect(screen.getByTestId("title").className).toContain("custom-title");
    expect(screen.getByTestId("description").tagName).toBe("P");
    expect(screen.getByTestId("description").className).toContain("custom-description");
    expect(screen.getByTestId("content").className).toContain("custom-content");

    // Children render through every wrapper.
    expect(screen.getByTestId("title")).toHaveTextContent("Title");
    expect(screen.getByTestId("description")).toHaveTextContent("Description");
    expect(screen.getByTestId("content")).toHaveTextContent("Body");
  });

  it("passes arbitrary DOM props through to the underlying element", () => {
    renderWithProviders(
      <Card aria-label="panel" role="group">
        <span>content</span>
      </Card>,
    );
    expect(screen.getByRole("group", { name: "panel" })).toBeInTheDocument();
  });
});
