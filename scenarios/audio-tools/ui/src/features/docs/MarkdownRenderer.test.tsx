import { describe, it, expect, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders as render } from "../../test-utils/renderWithProviders";
import { MarkdownRenderer } from "./MarkdownRenderer";

afterEach(() => {
  cleanup();
});

describe("MarkdownRenderer", () => {
  it("renders headings, lists, and inline code from a markdown sample", () => {
    render(
      <MarkdownRenderer content={"# Title\n\n- one\n- two\n\nuse `audio-tools` here"} />,
    );
    expect(screen.getByRole("heading", { name: "Title", level: 1 })).toBeInTheDocument();
    expect(screen.getByText(/^one$/)).toBeInTheDocument();
    expect(screen.getByText(/^two$/)).toBeInTheDocument();
    expect(screen.getByText(/^audio-tools$/)).toBeInTheDocument();
  });

  it("renders external links with noopener noreferrer", () => {
    render(<MarkdownRenderer content="[ext](https://example.com)" />);
    const link = screen.getByRole("link", { name: "ext" });
    expect(link).toHaveAttribute("href", "https://example.com");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("does NOT execute embedded script tags (react-markdown does not render raw HTML)", () => {
    const dangerous = '<script>window.__pwned = 1</script>\n\nhello';
    render(<MarkdownRenderer content={dangerous} />);
    expect((window as unknown as { __pwned?: number }).__pwned).toBeUndefined();
    // The literal text survives but as text content, not as an executed tag.
    expect(screen.getByText(/hello/)).toBeInTheDocument();
    expect(document.querySelector("script")).toBeNull();
  });

  it("renders GFM table syntax", () => {
    render(
      <MarkdownRenderer
        content={"| a | b |\n|---|---|\n| 1 | 2 |"}
      />,
    );
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(screen.getByRole("columnheader", { name: "a" })).toBeInTheDocument();
    expect(screen.getByRole("cell", { name: "1" })).toBeInTheDocument();
  });
});
