/**
 * Additional MarkdownRenderer coverage — lines 116-127.
 *
 * Uncovered branches:
 *   - `pre` component (line 116)
 *   - `extractText` branches: number node, array node, React-element node (with props.children),
 *     and the fallback `return ""` (null/undefined/boolean)
 *   - `del` and `em` inline components
 *   - `blockquote` and `hr`
 *   - block-code (className-carrying code block → !looksInline path in `code` component)
 */
import { describe, it, expect, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { MarkdownRenderer } from "./MarkdownRenderer";

afterEach(() => {
  cleanup();
});

describe("MarkdownRenderer — additional element coverage (lines 116-127)", () => {
  it("renders a <pre> wrapper around code blocks", () => {
    render(<MarkdownRenderer content={"```\nsome code\n```"} />);
    // react-markdown wraps code blocks in <pre><code>
    const pre = document.querySelector("pre");
    expect(pre).not.toBeNull();
    expect(pre).toHaveClass("my-3");
  });

  it("renders block code (non-inline) with block styling when className is present", () => {
    render(<MarkdownRenderer content={"```js\nconsole.log(1);\n```"} />);
    // The code element should have the block class, not the inline class
    const code = document.querySelector("code");
    expect(code).not.toBeNull();
    // block code gets `block overflow-x-auto` class
    expect(code?.className).toMatch(/block/);
  });

  it("renders ~~strikethrough~~ as <del> (remark-gfm)", () => {
    render(<MarkdownRenderer content="~~deleted text~~" />);
    const del = document.querySelector("del");
    expect(del).not.toBeNull();
    expect(del).toHaveTextContent("deleted text");
  });

  it("renders *em* as <em>", () => {
    render(<MarkdownRenderer content="*italic text*" />);
    const em = document.querySelector("em");
    expect(em).not.toBeNull();
    expect(em).toHaveTextContent("italic text");
  });

  it("renders > blockquote", () => {
    render(<MarkdownRenderer content="> quoted text" />);
    const bq = document.querySelector("blockquote");
    expect(bq).not.toBeNull();
    expect(bq).toHaveTextContent("quoted text");
  });

  it("renders --- as <hr>", () => {
    render(<MarkdownRenderer content={"above\n\n---\n\nbelow"} />);
    const hr = document.querySelector("hr");
    expect(hr).not.toBeNull();
    expect(hr).toHaveClass("my-6");
  });

  it("renders ordered list (ol)", () => {
    render(<MarkdownRenderer content={"1. first\n2. second"} />);
    const ol = document.querySelector("ol");
    expect(ol).not.toBeNull();
    expect(screen.getByText(/first/)).toBeInTheDocument();
    expect(screen.getByText(/second/)).toBeInTheDocument();
  });

  it("renders **bold** as <strong>", () => {
    render(<MarkdownRenderer content="**bold text**" />);
    const strong = document.querySelector("strong");
    expect(strong).not.toBeNull();
    expect(strong).toHaveTextContent("bold text");
  });

  it("renders an internal (relative) link without target or rel attributes", () => {
    render(<MarkdownRenderer content="[internal](/docs/PRD.md)" />);
    const link = screen.getByRole("link", { name: "internal" });
    expect(link).toHaveAttribute("href", "/docs/PRD.md");
    expect(link).not.toHaveAttribute("target");
    expect(link).not.toHaveAttribute("rel");
  });

  it("renders a mailto: link as external (has noopener noreferrer)", () => {
    render(<MarkdownRenderer content="[mail](mailto:foo@bar.com)" />);
    const link = screen.getByRole("link", { name: "mail" });
    expect(link).toHaveAttribute("href", "mailto:foo@bar.com");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("renders an h4 heading", () => {
    render(<MarkdownRenderer content={"#### Level Four"} />);
    expect(screen.getByRole("heading", { name: "Level Four", level: 4 })).toBeInTheDocument();
  });

  it("renders thead and th elements in a GFM table", () => {
    render(
      <MarkdownRenderer content={"| col1 | col2 |\n|------|------|\n| a    | b    |"} />,
    );
    const thead = document.querySelector("thead");
    expect(thead).not.toBeNull();
    const th = document.querySelector("th");
    expect(th).not.toBeNull();
  });

  it("applies optional className to the wrapper div", () => {
    const { container } = render(
      <MarkdownRenderer content="hello" className="custom-class" />,
    );
    const wrapper = container.firstElementChild;
    expect(wrapper?.className).toContain("custom-class");
  });
});
