import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import { describe, expect, it, vi } from "vitest";

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


describe("MarkdownRenderer interaction continuity", () => {
  it("keeps a pressed link connected and focused when callbacks change", () => {
    const previous = vi.fn((_, event) => event.preventDefault());
    const current = vi.fn((_, event) => event.preventDefault());
    const content = "[open document](/docs/README.md)";
    const { rerender } = render(<MarkdownRenderer content={content} onLinkClick={previous} />, { withoutRouter: true });
    const pressed = screen.getByRole("link", { name: "open document" });
    pressed.focus();
    fireEvent.pointerDown(pressed, { pointerId: 1, pointerType: "touch" });
    rerender(<MarkdownRenderer content={content} onLinkClick={current} onFileReferenceClick={() => {}} onMermaidOpen={() => {}} />);
    expect(screen.getByRole("link", { name: "open document" })).toBe(pressed);
    expect(pressed).toHaveFocus();
    fireEvent.pointerUp(pressed, { pointerId: 1, pointerType: "touch" });
    fireEvent.click(pressed);
    expect(current).toHaveBeenCalledOnce();
    expect(previous).not.toHaveBeenCalled();
  });

  it("keeps existing links connected when a reply streams more text", () => {
    const onLinkClick = vi.fn((_, event) => event.preventDefault());
    const { rerender } = render(<MarkdownRenderer content="[open document](/docs/README.md)" onLinkClick={onLinkClick} />, { withoutRouter: true });
    const pressed = screen.getByRole("link", { name: "open document" });
    fireEvent.pointerDown(pressed, { pointerId: 2, pointerType: "touch" });
    rerender(<MarkdownRenderer content="[open document](/docs/README.md)\n\nMore streamed text." onLinkClick={onLinkClick} />);
    expect(screen.getByRole("link", { name: "open document" })).toBe(pressed);
    fireEvent.click(pressed);
    expect(onLinkClick).toHaveBeenCalledOnce();
  });
});
