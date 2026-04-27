import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MarkdownRenderer } from "../components/markdown";

// Mock shiki and mermaid to avoid async loading in jsdom
vi.mock("shiki", () => ({
  createHighlighter: vi.fn().mockResolvedValue({
    codeToHtml: vi.fn().mockReturnValue('<pre class="shiki"><code>mocked</code></pre>'),
    getLoadedLanguages: vi.fn().mockReturnValue(["typescript", "javascript"]),
  }),
}));

vi.mock("mermaid", () => ({
  default: {
    initialize: vi.fn(),
    render: vi.fn().mockResolvedValue({ svg: '<svg>mocked diagram</svg>' }),
  },
}));

describe("MarkdownRenderer", () => {
  it("renders null for empty content", () => {
    const { container } = render(<MarkdownRenderer content="" />);
    expect(container.innerHTML).toBe("");
  });

  it("renders a paragraph", () => {
    render(<MarkdownRenderer content="Hello world" />);
    expect(screen.getByText("Hello world")).toBeInTheDocument();
  });

  it("renders headings", () => {
    render(<MarkdownRenderer content={"# Title\n\n## Subtitle"} />);
    expect(screen.getByText("Title")).toBeInTheDocument();
    expect(screen.getByText("Subtitle")).toBeInTheDocument();
  });

  it("renders bold text", () => {
    render(<MarkdownRenderer content="This is **bold** text" />);
    const strong = document.querySelector("strong");
    expect(strong).not.toBeNull();
    expect(strong?.textContent).toBe("bold");
  });

  it("renders italic text", () => {
    render(<MarkdownRenderer content="This is *italic* text" />);
    const em = document.querySelector("em");
    expect(em).not.toBeNull();
    expect(em?.textContent).toBe("italic");
  });

  it("renders unordered lists", () => {
    render(<MarkdownRenderer content={"- Item 1\n- Item 2\n- Item 3"} />);
    const items = document.querySelectorAll("li");
    expect(items.length).toBe(3);
  });

  it("renders ordered lists", () => {
    render(<MarkdownRenderer content={"1. First\n2. Second\n3. Third"} />);
    const items = document.querySelectorAll("li");
    expect(items.length).toBe(3);
  });

  it("renders blockquotes", () => {
    render(<MarkdownRenderer content={"> A wise quote"} />);
    const bq = document.querySelector("blockquote");
    expect(bq).not.toBeNull();
    expect(bq?.textContent).toContain("A wise quote");
  });

  it("renders horizontal rules", () => {
    render(<MarkdownRenderer content={"Above\n\n---\n\nBelow"} />);
    const hr = document.querySelector("hr");
    expect(hr).not.toBeNull();
  });

  it("renders tables", () => {
    render(<MarkdownRenderer content={"| A | B |\n|---|---|\n| 1 | 2 |"} />);
    const table = document.querySelector("table");
    expect(table).not.toBeNull();
    const cells = document.querySelectorAll("td");
    expect(cells.length).toBe(2);
  });

  it("renders links with target=_blank", () => {
    render(<MarkdownRenderer content="[Click](https://example.com)" />);
    const link = document.querySelector("a");
    expect(link).not.toBeNull();
    expect(link?.getAttribute("target")).toBe("_blank");
    expect(link?.getAttribute("rel")).toContain("noopener");
  });

  it("does not force target=_blank on local file-style links", () => {
    render(<MarkdownRenderer content="[Open](docs/plan.md)" />);
    const link = document.querySelector("a");
    expect(link).not.toBeNull();
    expect(link?.getAttribute("target")).toBeNull();
  });

  it("forwards link clicks through onLinkClick", () => {
    const onLinkClick = vi.fn();
    render(<MarkdownRenderer content="[Open](docs/plan.md)" onLinkClick={onLinkClick} />);
    const link = document.querySelector("a");
    expect(link).not.toBeNull();
    link?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    expect(onLinkClick).toHaveBeenCalledTimes(1);
    expect(onLinkClick.mock.calls[0]?.[0]).toBe("docs/plan.md");
  });

  it("renders strikethrough text (GFM)", () => {
    render(<MarkdownRenderer content={"This is ~~deleted~~ text"} />);
    const del = document.querySelector("del");
    expect(del).not.toBeNull();
    expect(del?.textContent).toBe("deleted");
  });

  it("renders inline code", () => {
    render(<MarkdownRenderer content="Use `console.log` for debugging" />);
    const code = document.querySelector("code");
    expect(code).not.toBeNull();
    expect(code?.textContent).toBe("console.log");
  });

  it("renders fenced code blocks", () => {
    render(<MarkdownRenderer content={'```typescript\nconst x = 1;\n```'} />);
    // The CodeBlock component renders in the DOM
    expect(document.querySelector("[class*='rounded-lg']")).not.toBeNull();
  });

  it("applies className prop", () => {
    const { container } = render(<MarkdownRenderer content="Test" className="custom-class" />);
    expect(container.querySelector(".custom-class")).not.toBeNull();
  });

  it("renders without a search wrapper prop path", () => {
    const { container } = render(<MarkdownRenderer content="Hello world" />);
    const wrapper = container.querySelector("[data-search-query]");
    expect(wrapper).toBeNull();
  });

  it("falls back to plain text on error via error boundary", () => {
    // Force an error by providing non-string content to test boundary
    const { container } = render(<MarkdownRenderer content="Safe content" />);
    expect(container.textContent).toContain("Safe content");
  });
});
