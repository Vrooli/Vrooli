import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../../test-utils";
import { previewModel, previewProps } from "../../../test-utils/filePreview";
import { strings } from "../../../consts/strings";
import { MarkdownPreview } from "../renderers/MarkdownPreview";

vi.mock("@vrooli/react-component-library/useResizeObserver", () => ({ useResizeObserver: () => ({ ref: () => {}, rect: { width: 390, height: 600 } }) }));
const c = strings.readerPreview;
function props(content = '# Guide\n\nRead **this phrase** and this phrase.\n\n## Setup\n\n[Config](./config.yaml)\n\n[Setup](#setup)\n\n```text\n# not a heading\n```') {
  return previewProps({ model: previewModel({ previewId: "markdown", kind: "markdown", basename: "guide.md", resolvedPath: "/project/docs/guide.md" }), text: { resolvedPath: "/project/docs/guide.md", kind: "markdown", mimeType: "text/markdown", content, truncated: false } });
}
beforeEach(() => { Object.defineProperty(HTMLElement.prototype, "scrollIntoView", { configurable: true, value: vi.fn() }); });
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });
describe("Markdown document reader", () => {
  it("builds an outline from rendered headings, ignores fenced source, and returns to the chosen section", async () => {
    render(<MarkdownPreview {...props()} />);
    fireEvent.click(screen.getByRole("button", { name: c.contents }));
    const outline = screen.getByRole("navigation", { name: c.contents });
    await waitFor(() => { expect(within(outline).getAllByRole("button")).toHaveLength(2); });
    fireEvent.click(within(outline).getByRole("button", { name: "Setup" }));
    expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
    await waitFor(() => { expect(document.activeElement).toBe(screen.getByRole("heading", { name: "Setup" })); });
  });
  it("searches across inline formatting, treats patterns literally, and restores focus after closing search", async () => {
    render(<MarkdownPreview {...props()} />);
    fireEvent.click(screen.getByRole("button", { name: c.findDocument }));
    const search = screen.getByRole("searchbox", { name: c.findDocument }); expect(search).toHaveFocus();
    fireEvent.change(search, { target: { value: "Read this phrase" } });
    await waitFor(() => { expect(screen.getByRole("button", { name: strings.artifactPreview.nextMatch })).toBeEnabled(); });
    fireEvent.change(search, { target: { value: ".*" } });
    await waitFor(() => { expect(screen.getByRole("button", { name: strings.artifactPreview.nextMatch })).toBeDisabled(); });
    fireEvent.keyDown(search, { key: "Escape" });
    expect(search).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: c.findDocument })).toHaveFocus();
  });
  it("opens sibling file references relative to the document and keeps source and text controls available", () => {
    const navigate = vi.fn(); render(<MarkdownPreview {...props()} onNavigate={navigate} />);
    fireEvent.click(screen.getByRole("link", { name: "Config" })); expect(navigate).toHaveBeenCalledWith("/project/docs/./config.yaml");
    fireEvent.change(screen.getByLabelText(c.textSize), { target: { value: "20" } });
    expect(document.querySelector(".wc-markdown-reader")).toHaveStyle({ fontSize: "20px" });
    fireEvent.click(screen.getByRole("button", { name: c.showSource }));
    expect(screen.getByTestId("file-preview-code")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: c.showRead }));
    expect(screen.getByRole("heading", { name: "Guide" })).toBeInTheDocument();
  });
  it("resets modes between files, preserves truncation warnings and keeps source text inert", () => {
    const first = props(); const { rerender } = render(<MarkdownPreview {...first} />);
    fireEvent.click(screen.getByRole("button", { name: c.showSource }));
    const next = props('<script>window.bad = true</script>\n\n# Next');
    rerender(<MarkdownPreview {...next} model={{ ...next.model, previewId: "next" }} text={next.text ? { ...next.text, truncated: true } : null} />);
    expect(screen.getByRole("heading", { name: "Next" })).toBeInTheDocument();
    expect(screen.getByText(strings.messagesFileViewer.truncatedNotice)).toBeInTheDocument();
    expect(document.querySelector("script")).toBeNull();
  });
  it("highlights actual DOM ranges across inline markup and releases highlights when source opens", async () => {
    const highlights = new Map<string, Set<Range>>();
    vi.stubGlobal("CSS", { highlights });
    vi.stubGlobal("Highlight", class extends Set<Range> { constructor(...ranges: Range[]) { super(ranges); } });
    render(<MarkdownPreview {...props("# Guide\n\nalpha **beta** gamma alpha **beta**")} />);
    fireEvent.click(screen.getByRole("button", { name: c.findDocument }));
    const input = screen.getByRole("searchbox", { name: c.findDocument });
    fireEvent.change(input, { target: { value: "alpha beta" } });
    await waitFor(() => { expect([...highlights.values()].some((ranges) => ranges.size === 2)).toBe(true); });
    const all = [...highlights.entries()].find(([name]) => !name.endsWith("-active"))?.[1];
    expect([...all ?? []].map((range) => range.toString())).toEqual(["alpha beta", "alpha beta"]);
    const active = () => [...highlights.entries()].find(([name]) => name.endsWith("-active"))?.[1].values().next().value;
    const first = active();
    fireEvent.keyDown(input, { key: "Enter" }); expect(active()).not.toBe(first);
    fireEvent.keyDown(input, { key: "Enter", shiftKey: true }); expect(active()).toBe(first);
    fireEvent.click(screen.getByRole("button", { name: c.showSource }));
    await waitFor(() => { expect([...highlights.values()].every((ranges) => ranges.size === 0)).toBe(true); });
  });
  it("resolves repeated heading fragments without navigating the application", async () => {
    render(<MarkdownPreview {...props("# Setup\n\n## Setup\n\n[Jump](#setup-1)")} />);
    const target = screen.getAllByRole("heading", { name: "Setup" })[1];
    await waitFor(() => { expect(target).toHaveAttribute("tabindex", "-1"); });
    const hash = window.location.hash;
    fireEvent.click(screen.getByRole("link", { name: "Jump" }));
    expect(target).toHaveFocus(); expect(window.location.hash).toBe(hash);
  });
});
