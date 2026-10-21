import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import { describe, expect, it, vi } from "vitest";

import { MarkdownRenderer } from "./MarkdownRenderer";
import { resolveFilePreview } from "../../api/filePreview";

vi.mock("../../api/filePreview", async (original) => ({
  ...await original<typeof import("../../api/filePreview")>(),
  resolveFilePreview: vi.fn(),
}));

describe("MarkdownRenderer local images", () => {
  it("resolves a host path through the session preview before loading its image", async () => {
    vi.mocked(resolveFilePreview).mockResolvedValue({ kind: "image", canPreview: true, blobHref: "/api/v1/sessions/session/file-previews/id/blob" } as Awaited<ReturnType<typeof resolveFilePreview>>);
    render(<MarkdownRenderer content="![Project browser](/home/operator/mockups/project.png)" imageSessionId="session" />);
    expect(screen.queryByRole("img")).toBeNull();
    await waitFor(() => { expect(screen.getByRole("img", { name: "Project browser" })).toHaveAttribute("src", "/api/v1/sessions/session/file-previews/id/blob"); });
    expect(resolveFilePreview).toHaveBeenCalledWith("session", "/home/operator/mockups/project.png", "message_link");
  });

  it("offers an accessible file reference when the image is unavailable", async () => {
    vi.mocked(resolveFilePreview).mockRejectedValue(new Error("missing"));
    const open = vi.fn();
    render(<MarkdownRenderer content="![Shot editor](/home/operator/mockups/shot.png)" imageSessionId="session" onFileReferenceClick={open} />);
    const link = await screen.findByRole("button", { name: /Shot editor/ });
    fireEvent.click(link);
    expect(open).toHaveBeenCalledWith("/home/operator/mockups/shot.png");
  });

  it("does not render a resolved non-image file as an image", async () => {
    vi.mocked(resolveFilePreview).mockResolvedValue({ kind: "code", canPreview: true, blobHref: "/api/v1/private/blob" } as Awaited<ReturnType<typeof resolveFilePreview>>);
    render(<MarkdownRenderer content="![Not an image](/home/operator/note.txt)" imageSessionId="session" />);
    expect(await screen.findByRole("button", { name: /Not an image/ })).toBeInTheDocument();
    expect(screen.queryByRole("img")).toBeNull();
  });

  it("keeps external HTTPS images as external images", () => {
    vi.mocked(resolveFilePreview).mockClear();
    render(<MarkdownRenderer content="![Remote](https://example.com/image.png)" imageSessionId="session" />);
    expect(screen.getByRole("img", { name: "Remote" })).toHaveAttribute("src", "https://example.com/image.png");
    expect(resolveFilePreview).not.toHaveBeenCalled();
  });
});

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
