import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import { describe, expect, it, vi } from "vitest";

import { TimeMentionProvider } from "@vrooli/react-component-library/TimeMention/1";
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

  it("resolves a relative image path against the document's own directory", async () => {
    vi.mocked(resolveFilePreview).mockResolvedValue({ kind: "image", canPreview: true, blobHref: "/api/v1/sessions/session/file-previews/id/blob" } as Awaited<ReturnType<typeof resolveFilePreview>>);
    render(<MarkdownRenderer content="![Diagram](./assets/diagram.png)" imageSessionId="session" imageBasePath="/home/operator/docs/" />);
    await waitFor(() => { expect(screen.getByRole("img", { name: "Diagram" })).toHaveAttribute("src", "/api/v1/sessions/session/file-previews/id/blob"); });
    expect(resolveFilePreview).toHaveBeenCalledWith("session", "/home/operator/docs/./assets/diagram.png", "message_link");
  });

  it("leaves an absolute image path unchanged even when a base path is present", async () => {
    vi.mocked(resolveFilePreview).mockResolvedValue({ kind: "image", canPreview: true, blobHref: "/api/v1/sessions/session/file-previews/id/blob" } as Awaited<ReturnType<typeof resolveFilePreview>>);
    render(<MarkdownRenderer content="![Shot](/home/operator/mockups/shot.png)" imageSessionId="session" imageBasePath="/home/operator/docs/" />);
    await waitFor(() => { expect(screen.getByRole("img", { name: "Shot" })).toBeInTheDocument(); });
    expect(resolveFilePreview).toHaveBeenCalledWith("session", "/home/operator/mockups/shot.png", "message_link");
  });

  it("opens a resolved local image in the full file preview when pressed", async () => {
    vi.mocked(resolveFilePreview).mockResolvedValue({ kind: "image", canPreview: true, blobHref: "/api/v1/sessions/session/file-previews/id/blob" } as Awaited<ReturnType<typeof resolveFilePreview>>);
    const open = vi.fn();
    render(<MarkdownRenderer content="![Project browser](/home/operator/mockups/project.png)" imageSessionId="session" onFileReferenceClick={open} />);
    const trigger = await screen.findByRole("button", { name: /Project browser/ });
    fireEvent.click(trigger);
    expect(open).toHaveBeenCalledWith("/home/operator/mockups/project.png");
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
    vi.mocked(resolveFilePreview).mockResolvedValue({ kind: "image", canPreview: true, blobHref: "/api/v1/sessions/session/file-previews/id/blob" } as Awaited<ReturnType<typeof resolveFilePreview>>);
    const open = vi.fn();
    render(<MarkdownRenderer content="![Wiring](/home/operator/wiring.png)" imageSessionId="session" onFileReferenceClick={open} />);
    const image = await screen.findByRole("img", { name: "Wiring" });
    fireEvent.error(image);
    const fallback = await screen.findByRole("button", { name: /Wiring/ });
    expect(screen.queryByRole("img")).toBeNull();
    fireEvent.click(fallback);
    expect(open).toHaveBeenCalledWith("/home/operator/wiring.png");
  });
});
