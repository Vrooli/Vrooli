import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders as render } from "../test-utils/index.js";
import { describe, expect, it, vi } from "vitest";

import { TimeMentionProvider } from "@vrooli/react-component-library/TimeMention/1";
import { MarkdownRenderer } from "./MarkdownRenderer.js";
import { NO_CAPABILITIES, type PreviewModel, type PreviewSource, type ResolveOutcome } from "../source/types.js";

// A source whose resolve is a spy: each test scripts what a local image path resolves to.
const resolveFilePreview = vi.fn<(path: string, context: string | undefined) => Promise<ResolveOutcome>>();
const imageSource: PreviewSource = {
  scope: "session",
  capabilities: NO_CAPABILITIES,
  resolve: (path, options) => resolveFilePreview(path, options?.context),
  readText: () => Promise.reject(new Error("unused")),
  listDirectory: () => Promise.reject(new Error("unused")),
};
const resolvedTo = (partial: Pick<PreviewModel, "kind" | "blobHref"> & Partial<PreviewModel>): ResolveOutcome => ({
  outcome: "resolved",
  model: { canPreview: true, ...partial } as PreviewModel,
});

describe("MarkdownRenderer local images", () => {
  it("resolves a local path through the image source before loading its image", async () => {
    resolveFilePreview.mockResolvedValue(resolvedTo({ kind: "image", blobHref: "/api/v1/sessions/session/file-previews/id/blob" }));
    render(<MarkdownRenderer content="![Project browser](/home/operator/mockups/project.png)" imageSource={imageSource} />);
    expect(screen.queryByRole("img")).toBeNull();
    await waitFor(() => { expect(screen.getByRole("img", { name: "Project browser" })).toHaveAttribute("src", "/api/v1/sessions/session/file-previews/id/blob"); });
    expect(resolveFilePreview).toHaveBeenCalledWith("/home/operator/mockups/project.png", "message_link");
  });

  it("offers an accessible file reference when the image is unavailable", async () => {
    resolveFilePreview.mockRejectedValue(new Error("missing"));
    const open = vi.fn();
    render(<MarkdownRenderer content="![Shot editor](/home/operator/mockups/shot.png)" imageSource={imageSource} onFileReferenceClick={open} />);
    const link = await screen.findByRole("button", { name: /Shot editor/ });
    fireEvent.click(link);
    expect(open).toHaveBeenCalledWith("/home/operator/mockups/shot.png");
  });

  it("does not render a resolved non-image file as an image", async () => {
    resolveFilePreview.mockResolvedValue(resolvedTo({ kind: "code", blobHref: "/api/v1/private/blob" }));
    render(<MarkdownRenderer content="![Not an image](/home/operator/note.txt)" imageSource={imageSource} />);
    expect(await screen.findByRole("button", { name: /Not an image/ })).toBeInTheDocument();
    expect(screen.queryByRole("img")).toBeNull();
  });

  it("resolves a relative image path against the document's own directory", async () => {
    resolveFilePreview.mockResolvedValue(resolvedTo({ kind: "image", blobHref: "/api/v1/sessions/session/file-previews/id/blob" }));
    render(<MarkdownRenderer content="![Diagram](./assets/diagram.png)" imageSource={imageSource} imageBasePath="/home/operator/docs/" />);
    await waitFor(() => { expect(screen.getByRole("img", { name: "Diagram" })).toHaveAttribute("src", "/api/v1/sessions/session/file-previews/id/blob"); });
    expect(resolveFilePreview).toHaveBeenCalledWith("/home/operator/docs/./assets/diagram.png", "message_link");
  });

  it("leaves an absolute image path unchanged even when a base path is present", async () => {
    resolveFilePreview.mockResolvedValue(resolvedTo({ kind: "image", blobHref: "/api/v1/sessions/session/file-previews/id/blob" }));
    render(<MarkdownRenderer content="![Shot](/home/operator/mockups/shot.png)" imageSource={imageSource} imageBasePath="/home/operator/docs/" />);
    await waitFor(() => { expect(screen.getByRole("img", { name: "Shot" })).toBeInTheDocument(); });
    expect(resolveFilePreview).toHaveBeenCalledWith("/home/operator/mockups/shot.png", "message_link");
  });

  it("opens a resolved local image in the full file preview when pressed", async () => {
    resolveFilePreview.mockResolvedValue(resolvedTo({ kind: "image", blobHref: "/api/v1/sessions/session/file-previews/id/blob" }));
    const open = vi.fn();
    render(<MarkdownRenderer content="![Project browser](/home/operator/mockups/project.png)" imageSource={imageSource} onFileReferenceClick={open} />);
    const trigger = await screen.findByRole("button", { name: /Project browser/ });
    fireEvent.click(trigger);
    expect(open).toHaveBeenCalledWith("/home/operator/mockups/project.png");
  });

  it("keeps external HTTPS images as external images", () => {
    resolveFilePreview.mockClear();
    render(<MarkdownRenderer content="![Remote](https://example.com/image.png)" imageSource={imageSource} />);
    expect(screen.getByRole("img", { name: "Remote" })).toHaveAttribute("src", "https://example.com/image.png");
    expect(resolveFilePreview).not.toHaveBeenCalled();
  });
});

describe("MarkdownRenderer link scheme handling", () => {
  it("renders a link at the target the host's rewriteHref chooses", () => {
    render(<MarkdownRenderer content="[open app](http://localhost:21238/tasks?state=active#latest)" />, {
      host: { rewriteHref: (href) => href.replace("http://localhost:21238", "https://tunnel.example/_local/21238") },
    });
    expect(screen.getByRole("link", { name: "open app" })).toHaveAttribute("href", "https://tunnel.example/_local/21238/tasks?state=active#latest");
  });

  it("keeps the authored target when the host does not rewrite it", () => {
    render(<MarkdownRenderer content="[docs](http://localhost:21238/docs)" />, { host: { rewriteHref: () => null } });
    expect(screen.getByRole("link", { name: "docs" })).toHaveAttribute("href", "http://localhost:21238/docs");
  });

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
    const { rerender } = render(<MarkdownRenderer content={content} onLinkClick={previous} />);
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
    const { rerender } = render(<MarkdownRenderer content="[open document](/docs/README.md)" onLinkClick={onLinkClick} />);
    const pressed = screen.getByRole("link", { name: "open document" });
    fireEvent.pointerDown(pressed, { pointerId: 2, pointerType: "touch" });
    rerender(<MarkdownRenderer content="[open document](/docs/README.md)\n\nMore streamed text." onLinkClick={onLinkClick} />);
    expect(screen.getByRole("link", { name: "open document" })).toBe(pressed);
    fireEvent.click(pressed);
    expect(onLinkClick).toHaveBeenCalledOnce();
  });
});

describe("MarkdownRenderer time mentions", () => {
  // A message sent at 9:27 PM New York time on Oct 28, 2026, read four minutes later.
  const sent = "2026-10-29T01:27:00Z";
  const reader = { now: Date.parse(sent) + 4 * 60_000, timeZone: "America/New_York", locale: "en-US" };
  const squash = (value: string | null | undefined) => (value ?? "").replace(/\s+/g, " ");

  it("dates a bare UTC time by the message's send time and shows the reader's time beside it", () => {
    const { container } = render(
      <TimeMentionProvider {...reader}>
        <MarkdownRenderer content="Keep the original **08:53:33 UTC** trial cutoff; `retry_at=08:53:33Z` stays." timeAnchor={sent} />
      </TimeMentionProvider>,
    );
    const mentions = container.querySelectorAll("[data-rcl-time-mention]");
    expect(mentions).toHaveLength(1);
    expect(mentions[0]?.querySelector("time")?.textContent).toBe("08:53:33 UTC");
    expect(squash(mentions[0]?.querySelector("[data-rcl-time-mention-tag]")?.textContent)).toBe("4:53 AM tomorrow");
    expect(mentions[0]?.querySelector("[data-rcl-time-mention-badge]")?.textContent).toBe("in 7h 22m");
    expect(container.querySelector("code")?.textContent).toBe("retry_at=08:53:33Z");
  });

  it("without a send time leaves words that need one as written", () => {
    const { container } = render(
      <TimeMentionProvider {...reader}>
        <MarkdownRenderer content="I'll check back in 20 minutes, before 08:53:33 UTC." />
      </TimeMentionProvider>,
    );
    const mentions = [...container.querySelectorAll("[data-rcl-time-mention]")];
    expect(mentions.map((m) => m.querySelector("time")?.textContent)).toEqual(["08:53:33 UTC"]);
    expect(mentions[0]?.querySelector("[data-rcl-time-mention-badge]")).toBeNull();
  });
});

describe("MarkdownRenderer document structure", () => {
  it("keeps third- and fourth-level headings at their levels", () => {
    render(<MarkdownRenderer content={"### Findings\n\n#### Evidence"} />);
    expect(screen.getByRole("heading", { level: 3, name: "Findings" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 4, name: "Evidence" })).toBeInTheDocument();
  });

  it("renders a large document progressively until every chunk is on screen", async () => {
    const paragraphs = Array.from({ length: 40 }, (_, index) => `Paragraph ${index + 1} ${"lorem ".repeat(60)}`);
    const { container } = render(<MarkdownRenderer content={paragraphs.join("\n\n")} progressive />);
    expect(container.querySelectorAll("p").length).toBeLessThan(40);
    await waitFor(() => { expect(container.querySelectorAll("p")).toHaveLength(40); });
    expect(screen.getByText(/^Paragraph 40 /)).toBeInTheDocument();
  });

  it("falls back to a file reference when a resolved image fails to load", async () => {
    resolveFilePreview.mockResolvedValue(resolvedTo({ kind: "image", blobHref: "/api/v1/sessions/session/file-previews/id/blob" }));
    const open = vi.fn();
    render(<MarkdownRenderer content="![Wiring](/home/operator/wiring.png)" imageSource={imageSource} onFileReferenceClick={open} />);
    const image = await screen.findByRole("img", { name: "Wiring" });
    fireEvent.error(image);
    const fallback = await screen.findByRole("button", { name: /Wiring/ });
    expect(screen.queryByRole("img")).toBeNull();
    fireEvent.click(fallback);
    expect(open).toHaveBeenCalledWith("/home/operator/wiring.png");
  });
});
