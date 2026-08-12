import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { MouseEvent } from "react";
import { InlineCode } from "./InlineCode";
import { MarkdownRenderer } from "./MarkdownRenderer";
import { renderWithProviders } from "../../test-utils";

vi.mock("mermaid", () => ({
  default: {
    initialize: vi.fn(),
    render: vi.fn().mockResolvedValue({ svg: "<svg aria-label='diagram'></svg>" }),
  },
}));

afterEach(cleanup);

describe("MarkdownRenderer", () => {
  it("renders GFM tables and blockquotes", () => {
    renderWithProviders(
      <MarkdownRenderer
        content={"| Name | Value |\n| --- | --- |\n| state | ready |\n\n> Operator guidance"}
      />,
    );
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(screen.getByText("Operator guidance")).toBeInTheDocument();
  });

  it("delegates entity and file inline-code seams", () => {
    const onLinkClick = vi.fn();
    const onFileReferenceClick = vi.fn();
    renderWithProviders(
      <MarkdownRenderer
        content={"`initiative:ship` `docs/plan.md`"}
        resolveInlineToken={(text) =>
          text === "initiative:ship" ? { href: "/initiatives/ship", kind: "entity" } : null
        }
        looksLikeFileReference={(text) => text.endsWith(".md")}
        onLinkClick={(href, event) => {
          event.preventDefault();
          onLinkClick(href, event);
        }}
        onFileReferenceClick={onFileReferenceClick}
      />,
    );
    const entity = screen.getByRole("link", { name: "initiative:ship" });
    expect(entity).toHaveAttribute("data-entity-ref", "true");
    fireEvent.click(entity);
    expect(onLinkClick).toHaveBeenCalledWith("/initiatives/ship", expect.anything());
    fireEvent.click(screen.getByRole("button", { name: "docs/plan.md" }));
    expect(onFileReferenceClick).toHaveBeenCalledWith("docs/plan.md");
  });

  it("keeps an unresolved token semantic inline code", () => {
    renderWithProviders(<InlineCode>unknown:token</InlineCode>);
    expect(screen.getByText("unknown:token").tagName).toBe("CODE");
  });

  it("supports inline rendering and ordinary markdown links", () => {
    const onLinkClick = vi.fn((_: string, event: MouseEvent) => event.preventDefault());
    const { container } = renderWithProviders(
      <MarkdownRenderer
        inline
        className="inline-copy"
        content="[Guide](/docs/guide)"
        onLinkClick={onLinkClick}
      />,
    );
    expect(container.firstElementChild?.tagName).toBe("SPAN");
    expect(container.firstElementChild).toHaveClass("inline-copy");
    fireEvent.click(screen.getByRole("link", { name: "Guide" }));
    expect(onLinkClick).toHaveBeenCalledWith("/docs/guide", expect.anything());
  });

  it("does not render an empty markdown value", () => {
    const { container } = renderWithProviders(<MarkdownRenderer content="" />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders fenced code and Mermaid blocks through their specialized surfaces", async () => {
    renderWithProviders(
      <MarkdownRenderer
        content={"```ts\nconst story = true\n```\n\n```mermaid\ngraph TD; A-->B\n```"}
      />,
    );
    expect(await screen.findByText("TYPESCRIPT")).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: "Source" })).toBeInTheDocument();
  });

  it("keeps ordinary links usable when no navigation callback is supplied", () => {
    renderWithProviders(<MarkdownRenderer content="[Guide](/docs/guide)" />);
    expect(screen.getByRole("link", { name: "Guide" })).toHaveAttribute("href", "/docs/guide");
  });

  it("keeps inline-code content bounded to text and numeric children", () => {
    const { rerender, container } = renderWithProviders(<InlineCode>{42}</InlineCode>);
    expect(screen.getByText("42")).toBeInTheDocument();
    rerender(
      <InlineCode>
        <span>nested</span>
      </InlineCode>,
    );
    expect(container.querySelector("code")).toHaveTextContent("");
  });

  it("renders resolved and file-reference inline tokens without optional callbacks", () => {
    const { rerender } = renderWithProviders(
      <InlineCode resolveInlineToken={() => ({ href: "/asset" })}>asset:Button</InlineCode>,
    );
    expect(screen.getByRole("link", { name: "asset:Button" })).toHaveAttribute("href", "/asset");
    rerender(<InlineCode looksLikeFileReference={() => true}>docs/story.md</InlineCode>);
    expect(screen.getByRole("button", { name: "docs/story.md" })).toBeInTheDocument();
  });
});
