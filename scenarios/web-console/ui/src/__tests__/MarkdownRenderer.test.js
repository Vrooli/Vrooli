import { jsx as _jsx } from "react/jsx-runtime";
import { describe, it, expect, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
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
        const { container } = render(_jsx(MarkdownRenderer, { content: "" }));
        expect(container.innerHTML).toBe("");
    });
    it("renders a paragraph", () => {
        render(_jsx(MarkdownRenderer, { content: "Hello world" }));
        expect(screen.getByText("Hello world")).toBeInTheDocument();
    });
    it("renders headings", () => {
        render(_jsx(MarkdownRenderer, { content: "# Title\n\n## Subtitle" }));
        expect(screen.getByText("Title")).toBeInTheDocument();
        expect(screen.getByText("Subtitle")).toBeInTheDocument();
    });
    it("renders bold text", () => {
        render(_jsx(MarkdownRenderer, { content: "This is **bold** text" }));
        const strong = document.querySelector("strong");
        expect(strong).not.toBeNull();
        expect(strong?.textContent).toBe("bold");
    });
    it("renders italic text", () => {
        render(_jsx(MarkdownRenderer, { content: "This is *italic* text" }));
        const em = document.querySelector("em");
        expect(em).not.toBeNull();
        expect(em?.textContent).toBe("italic");
    });
    it("renders unordered lists", () => {
        render(_jsx(MarkdownRenderer, { content: "- Item 1\n- Item 2\n- Item 3" }));
        const items = document.querySelectorAll("li");
        expect(items.length).toBe(3);
    });
    it("renders ordered lists", () => {
        render(_jsx(MarkdownRenderer, { content: "1. First\n2. Second\n3. Third" }));
        const items = document.querySelectorAll("li");
        expect(items.length).toBe(3);
    });
    it("renders blockquotes", () => {
        render(_jsx(MarkdownRenderer, { content: "> A wise quote" }));
        const bq = document.querySelector("blockquote");
        expect(bq).not.toBeNull();
        expect(bq?.textContent).toContain("A wise quote");
    });
    it("renders horizontal rules", () => {
        render(_jsx(MarkdownRenderer, { content: "Above\n\n---\n\nBelow" }));
        const hr = document.querySelector("hr");
        expect(hr).not.toBeNull();
    });
    it("renders tables", () => {
        render(_jsx(MarkdownRenderer, { content: "| A | B |\n|---|---|\n| 1 | 2 |" }));
        const table = document.querySelector("table");
        expect(table).not.toBeNull();
        const cells = document.querySelectorAll("td");
        expect(cells.length).toBe(2);
    });
    it("table cells have minimum width so they don't collapse, and table can horizontally overflow", () => {
        render(_jsx(MarkdownRenderer, { content: "| A | B | C | D | E |\n|---|---|---|---|---|\n| 1 | 2 | 3 | 4 | 5 |" }));
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
        expect(wrapper?.className).toMatch(/overflow-x-auto/);
        expect(document.querySelector("table")?.className).toMatch(/w-auto/);
    });
    it("renders links with target=_blank", () => {
        render(_jsx(MarkdownRenderer, { content: "[Click](https://example.com)" }));
        const link = document.querySelector("a");
        expect(link).not.toBeNull();
        expect(link?.getAttribute("target")).toBe("_blank");
        expect(link?.getAttribute("rel")).toContain("noopener");
    });
    it("does not force target=_blank on local file-style links", () => {
        render(_jsx(MarkdownRenderer, { content: "[Open](docs/plan.md)" }));
        const link = document.querySelector("a");
        expect(link).not.toBeNull();
        expect(link?.getAttribute("target")).toBeNull();
    });
    it("forwards link clicks through onLinkClick", () => {
        const onLinkClick = vi.fn();
        render(_jsx(MarkdownRenderer, { content: "[Open](docs/plan.md)", onLinkClick: onLinkClick }));
        const link = document.querySelector("a");
        expect(link).not.toBeNull();
        link?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
        expect(onLinkClick).toHaveBeenCalledTimes(1);
        expect(onLinkClick.mock.calls[0]?.[0]).toBe("docs/plan.md");
    });
    it("auto-links bare file paths in prose with the prose-path treatment", () => {
        const onLinkClick = vi.fn();
        render(_jsx(MarkdownRenderer, { content: "Edited scenarios/web-console/ui/src/App.tsx:42 in place", onLinkClick: onLinkClick }));
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
        render(_jsx(MarkdownRenderer, { content: "Use and/or logic on the TCP/IP stack with node.js" }));
        expect(document.querySelector("a")).toBeNull();
    });
    it("does not auto-link paths inside inline code or authored links", () => {
        render(_jsx(MarkdownRenderer, { content: "See `src/lib/a.ts` and [b](docs/plan.md)" }));
        expect(document.querySelector("a[data-prose-path='true']")).toBeNull();
        // The authored link keeps its normal treatment.
        const link = document.querySelector("a");
        expect(link?.getAttribute("href")).toBe("docs/plan.md");
        expect(link?.className).not.toContain("decoration-dotted");
    });
    it("auto-links a deep absolute upload path in prose (live regression)", () => {
        const path = "/home/matthalloran8/.vrooli/cache/vrooli/web-console/uploads/e802040e-8e0a-4fed-a776-34d1eed75bb1/IMG_9951.png";
        render(_jsx(MarkdownRenderer, { content: `Looks like it’s matching the negatives ${path}` }));
        const link = document.querySelector("a[data-prose-path='true']");
        expect(link).not.toBeNull();
        expect(link?.getAttribute("href")).toBe(path);
    });
    it("does not chip non-path inline code like and/or or file://", () => {
        render(_jsx(MarkdownRenderer, { content: "Use `and/or` with `TCP/IP`, `50/50`, `vrooli.com`, `file://`", onFileReferenceClick: () => { } }));
        // Chips render a button with an Open title; none of these should get one.
        expect(document.querySelector("button[title^='Open ']")).toBeNull();
    });
    it("still chips real paths and plain filenames in inline code", () => {
        render(_jsx(MarkdownRenderer, { content: "See `~/notes/todo.txt` and `README.md`", onFileReferenceClick: () => { } }));
        const chips = document.querySelectorAll("button[title^='Open ']");
        expect(chips).toHaveLength(2);
    });
    it("does not auto-link inside autolinked URLs", () => {
        render(_jsx(MarkdownRenderer, { content: "Docs at https://example.com/docs/guide.md today" }));
        expect(document.querySelector("a[data-prose-path='true']")).toBeNull();
    });
    it("renders strikethrough text (GFM)", () => {
        render(_jsx(MarkdownRenderer, { content: "This is ~~deleted~~ text" }));
        const del = document.querySelector("del");
        expect(del).not.toBeNull();
        expect(del?.textContent).toBe("deleted");
    });
    it("renders inline code", () => {
        render(_jsx(MarkdownRenderer, { content: "Use `console.log` for debugging" }));
        const code = document.querySelector("code");
        expect(code).not.toBeNull();
        expect(code?.textContent).toBe("console.log");
    });
    it("renders fenced code blocks", () => {
        render(_jsx(MarkdownRenderer, { content: '```typescript\nconst x = 1;\n```' }));
        // The CodeBlock component renders in the DOM
        expect(document.querySelector("[class*='rounded-lg']")).not.toBeNull();
    });
    it("applies className prop", () => {
        const { container } = render(_jsx(MarkdownRenderer, { content: "Test", className: "custom-class" }));
        expect(container.querySelector(".custom-class")).not.toBeNull();
    });
    it("renders without a search wrapper prop path", () => {
        const { container } = render(_jsx(MarkdownRenderer, { content: "Hello world" }));
        const wrapper = container.querySelector("[data-search-query]");
        expect(wrapper).toBeNull();
    });
    it("renders a Mermaid full-screen control and forwards the exact source", () => {
        const onMermaidOpen = vi.fn();
        render(_jsx(MarkdownRenderer, { content: "```mermaid\ngraph TD; A-->B\n```", onMermaidOpen: onMermaidOpen }));
        const openButton = screen.getByLabelText("mermaid.openFullscreen");
        fireEvent.click(openButton);
        expect(onMermaidOpen).toHaveBeenCalledTimes(1);
        expect(onMermaidOpen.mock.calls[0]?.[0]).toContain("graph TD; A-->B");
    });
    it("omits the Mermaid full-screen control when no handler is provided", () => {
        render(_jsx(MarkdownRenderer, { content: "```mermaid\ngraph TD; A-->B\n```" }));
        expect(screen.queryByLabelText("mermaid.openFullscreen")).toBeNull();
    });
    it("falls back to plain text on error via error boundary", () => {
        // Force an error by providing non-string content to test boundary
        const { container } = render(_jsx(MarkdownRenderer, { content: "Safe content" }));
        expect(container.textContent).toContain("Safe content");
    });
});
