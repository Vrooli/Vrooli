import { screen } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import { describe, expect, it } from "vitest";

import { MarkdownRenderer } from "./MarkdownRenderer";

describe("MarkdownRenderer link scheme handling", () => {
  it("preserves a webapp:// href (installed-PWA scheme) instead of stripping it", () => {
    render(
      <MarkdownRenderer content="[open hub](webapp://notification-hub.itsagitime.com/settings)" />,
    );
    const link = screen.getByRole("link", { name: "open hub" });
    expect(link).toHaveAttribute(
      "href",
      "webapp://notification-hub.itsagitime.com/settings",
    );
  });

  it("still renders a normal https link alongside a custom-scheme link", () => {
    render(
      <MarkdownRenderer content="[hub](webapp://notification-hub.itsagitime.com) and [site](https://notification-hub.itsagitime.com)" />,
    );
    expect(screen.getByRole("link", { name: "hub" })).toHaveAttribute(
      "href",
      "webapp://notification-hub.itsagitime.com",
    );
    expect(screen.getByRole("link", { name: "site" })).toHaveAttribute(
      "href",
      "https://notification-hub.itsagitime.com",
    );
  });

  it("still neutralizes a dangerous javascript: href", () => {
    const { container } = render(
      <MarkdownRenderer content="[x](javascript:alert(1))" />,
    );
    const link = container.querySelector("a");
    expect(link).not.toBeNull();
    // defaultUrlTransform strips the unsafe scheme, leaving no usable href.
    expect(link?.getAttribute("href") ?? "").toBe("");
  });
});
