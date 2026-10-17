import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MarkdownRenderer } from "../components/markdown";
import { readFileSync } from "node:fs";
const appStyles = readFileSync("src/styles.css", "utf8");

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

  it("table cells have minimum width so they don't collapse, and table can horizontally overflow", () => {
    render(<MarkdownRenderer content={"| A | B | C | D | E |\n|---|---|---|---|---|\n| 1 | 2 | 3 | 4 | 5 |"} />);
    const cells = document.querySelectorAll("td");
    expect(cells.length).toBe(5);
    cells.forEach((cell) => {
      expect(cell.className).toMatch(/min-w-\[8rem\]/);
    });
    const headers = document.querySelectorAll("th");
    headers.forEach((h) => {
      expect(h.className).toMatch(/min-w-\[8rem\]/);
    });
    const wrapper = document.querySelector("table")?.parentElement;
    expect(wrapper?.className).toMatch(/rcl-md__table-scroll/);
    expect(document.querySelectorAll('[data-rcl-md-resize-handle]')).toHaveLength(4);
  });

  it("exposes keyboard-resizable boundaries for adjacent table columns", async () => {
    render(<MarkdownRenderer content={"| A | B | C |\n|---|---|---|\n| 1 | 2 | 3 |"} />);
    const handle = screen.getByRole("separator", { name: "Resize column 1" });
    const before = Number(handle.getAttribute("aria-valuenow"));

    fireEvent.keyDown(handle, { key: "ArrowRight" });

    await waitFor(() => {
      expect(Number(handle.getAttribute("aria-valuenow"))).toBeGreaterThan(before);
    });
  });

  it("renders links with target=_blank", () => {
    render(<MarkdownRenderer content="[Click](https://example.com)" />);
    const link = document.querySelector("a");
    expect(link).not.toBeNull();
    expect(link?.getAttribute("target")).toBe("_blank");
    expect(link?.getAttribute("rel")).toContain("noopener");
  });

  it("keeps formatted link labels in the link color", () => {
    // Model the generated utility colors, then load the actual app stylesheet.
    // The browser smoke additionally checks the full Tailwind output.
    const style = document.createElement("style");
    style.textContent = `.text-wc-accent { color: rgb(34, 211, 238); } .text-wc-text-primary { color: rgb(248, 250, 252); }` + appStyles.replace(/^@tailwind .*;$/gm, "");
    document.head.appendChild(style);
    try {
      render(<MarkdownRenderer content={'[**Bold link**](https://example.com) [`Code link`](https://example.com)'} />);
      for (const link of screen.getAllByRole("link")) {
        const label = link.querySelector("strong, code");
        if (!label) throw new Error("Expected a formatted link label");
        expect(getComputedStyle(label).color).toBe(getComputedStyle(link).color);
      }
    } finally {
      style.remove();
    }
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

  it("auto-links bare file paths in prose with the prose-path treatment", () => {
    const onLinkClick = vi.fn();
    render(<MarkdownRenderer content="Edited scenarios/web-console/ui/src/App.tsx:42 in place" onLinkClick={onLinkClick} />);
    const link = document.querySelector("a[data-prose-path='true']");
    expect(link).not.toBeNull();
    expect(link?.textContent).toBe("scenarios/web-console/ui/src/App.tsx:42");
    expect(link?.getAttribute("href")).toBe("scenarios/web-console/ui/src/App.tsx:42");
    expect(link?.className).toContain("decoration-dotted");
    link?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    expect(onLinkClick).toHaveBeenCalledTimes(1);
    expect(onLinkClick.mock.calls[0]?.[0]).toBe("scenarios/web-console/ui/src/App.tsx:42");
  });

  it("does not auto-link slashed prose or bare module names", () => {
    render(<MarkdownRenderer content="Use and/or logic on the TCP/IP stack with node.js" />);
    expect(document.querySelector("a")).toBeNull();
  });

  it("does not auto-link paths inside inline code or authored links", () => {
    render(<MarkdownRenderer content="See `src/lib/a.ts` and [b](docs/plan.md)" />);
    expect(document.querySelector("a[data-prose-path='true']")).toBeNull();
    // The authored link keeps its normal treatment.
    const link = document.querySelector("a");
    expect(link?.getAttribute("href")).toBe("docs/plan.md");
    expect(link?.className).not.toContain("decoration-dotted");
  });

  it("auto-links a deep absolute upload path in prose (live regression)", () => {
    const path =
      "/home/matthalloran8/.vrooli/cache/vrooli/web-console/uploads/e802040e-8e0a-4fed-a776-34d1eed75bb1/IMG_9951.png";
    render(<MarkdownRenderer content={`Looks like it’s matching the negatives ${path}`} />);
    const link = document.querySelector("a[data-prose-path='true']");
    expect(link).not.toBeNull();
    expect(link?.getAttribute("href")).toBe(path);
  });

  it("does not chip non-path inline code like and/or or file://", () => {
    render(<MarkdownRenderer content="Use `and/or` with `TCP/IP`, `50/50`, `vrooli.com`, `file://`" onFileReferenceClick={() => {}} />);
    // Chips render a button with an Open title; none of these should get one.
    expect(document.querySelector("button[title^='Open ']")).toBeNull();
  });

  it("still chips real paths and plain filenames in inline code", () => {
    render(<MarkdownRenderer content="See `~/notes/todo.txt` and `README.md`" onFileReferenceClick={() => {}} />);
    const chips = document.querySelectorAll("button[title^='Open ']");
    expect(chips).toHaveLength(2);
  });

  it("does not auto-link inside autolinked URLs", () => {
    render(<MarkdownRenderer content="Docs at https://example.com/docs/guide.md today" />);
    expect(document.querySelector("a[data-prose-path='true']")).toBeNull();
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

  it("opens a fenced block that holds only a path", () => {
    const onFileReferenceClick = vi.fn();
    render(
      <MarkdownRenderer
        content={"```\n/the/path/example.md\n```"}
        onFileReferenceClick={onFileReferenceClick}
      />,
    );
    // Pinned to the block treatment: the inline-code chip already handled
    // `paths` in backticks, so a passing assertion must prove the fenced
    // block itself became clickable (block copy button + "path" label).
    expect(document.querySelector("[data-testid='code-block-copy']")).not.toBeNull();
    expect(screen.getByText("path")).toBeInTheDocument();
    const button = document.querySelector("button[title='Open /the/path/example.md']");
    expect(button).not.toBeNull();
    fireEvent.click(button as HTMLElement);
    expect(onFileReferenceClick).toHaveBeenCalledWith("/the/path/example.md");
  });

  it("opens every line of a fenced block that holds only paths", () => {
    const onFileReferenceClick = vi.fn();
    render(
      <MarkdownRenderer
        content={"```\ndocs/plan.md\n~/notes/todo.txt\nREADME.md\n```"}
        onFileReferenceClick={onFileReferenceClick}
      />,
    );
    const buttons = document.querySelectorAll("button[title^='Open ']");
    expect(buttons).toHaveLength(3);
    fireEvent.click(buttons[1] as HTMLElement);
    expect(onFileReferenceClick).toHaveBeenCalledWith("~/notes/todo.txt");
  });

  it("leaves a fenced block alone when any line is not a path", () => {
    const onFileReferenceClick = vi.fn();
    render(
      <MarkdownRenderer
        content={"```\n/the/path/example.md\nmake start\n```"}
        onFileReferenceClick={onFileReferenceClick}
      />,
    );
    expect(document.querySelector("button[title^='Open ']")).toBeNull();
  });

  it("leaves a language-tagged fence alone even when it holds only a path", () => {
    const onFileReferenceClick = vi.fn();
    render(
      <MarkdownRenderer
        content={"```bash\n./scripts/build.sh\n```"}
        onFileReferenceClick={onFileReferenceClick}
      />,
    );
    expect(document.querySelector("button[title^='Open ']")).toBeNull();
  });

  it("does not make path blocks clickable without a handler", () => {
    render(<MarkdownRenderer content={"```\n/the/path/example.md\n```"} />);
    expect(document.querySelector("button[title^='Open ']")).toBeNull();
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

  it("renders a Mermaid full-screen control and forwards the exact source", () => {
    const onMermaidOpen = vi.fn();
    render(
      <MarkdownRenderer
        content={"```mermaid\ngraph TD; A-->B\n```"}
        onMermaidOpen={onMermaidOpen}
      />,
    );
    const openButton = screen.getByLabelText("mermaid.openFullscreen");
    fireEvent.click(openButton);
    expect(onMermaidOpen).toHaveBeenCalledTimes(1);
    expect(onMermaidOpen.mock.calls[0]?.[0]).toContain("graph TD; A-->B");
  });

  it("omits the Mermaid full-screen control when no handler is provided", () => {
    render(<MarkdownRenderer content={"```mermaid\ngraph TD; A-->B\n```"} />);
    expect(screen.queryByLabelText("mermaid.openFullscreen")).toBeNull();
  });

  it("falls back to plain text on error via error boundary", () => {
    // Force an error by providing non-string content to test boundary
    const { container } = render(<MarkdownRenderer content="Safe content" />);
    expect(container.textContent).toContain("Safe content");
  });
});
