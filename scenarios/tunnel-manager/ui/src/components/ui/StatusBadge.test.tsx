/**
 * StatusBadge tests — tone → token class mapping and content rendering.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";

import { StatusBadge, type BadgeTone } from "./StatusBadge";

// provider-free-exception: this pure presentational leaf has no provider dependency.

describe("StatusBadge", () => {
  afterEach(() => cleanup());

  it("renders its children as the visible label", () => {
    render(
      <StatusBadge tone="success" data-testid="badge">
        Healthy
      </StatusBadge>,
    );
    expect(screen.getByTestId("badge")).toHaveTextContent("Healthy");
  });

  it.each<[BadgeTone, string]>([
    ["success", "text-app-success"],
    ["danger", "text-app-danger"],
    ["warning", "text-app-warning"],
    ["info", "text-app-info"],
    ["neutral", "text-app-muted-foreground"],
  ])("maps the %s tone to its token class", (tone, expected) => {
    render(
      <StatusBadge tone={tone} data-testid="badge">
        x
      </StatusBadge>,
    );
    expect(screen.getByTestId("badge").className).toContain(expected);
  });
});
