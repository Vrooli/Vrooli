import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { renderWithProviders as render } from "../../../test-utils";
import { previewModel, previewProps } from "../../../test-utils/filePreview";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { useEbook } from "../useEbook";
import { EbookPreview } from "../renderers/EbookPreview";
import type { EbookData } from "../epubData";
vi.mock("../useEbook", () => ({ useEbook: vi.fn() }));
const geometry = vi.hoisted(() => ({ width: 390 }));
vi.mock("@vrooli/react-component-library/useResizeObserver", () => ({ useResizeObserver: () => ({ ref: () => {}, rect: { width: geometry.width } }) }));
const encode = (source: string) => new TextEncoder().encode(source);
const book: EbookData = { title: "A useful book", creator: "Vrooli", fixed: false, chapters: [{ path: "one.xhtml", title: "First chapter" }, { path: "two.xhtml", title: "Second chapter" }], contents: [{ chapter: 0, title: "First chapter", depth: 0, fragment: "" }, { chapter: 1, title: "Second chapter", depth: 0, fragment: "detail" }], parts: { "one.xhtml": encode('<html><body><h1 id="intro">First chapter</h1><p>Find this useful phrase.</p><a href="two.xhtml#detail">Read more</a></body></html>'), "two.xhtml": encode('<html><body><h1 id="detail">Second chapter</h1><p>Another useful phrase.</p><a href="one.xhtml#intro">Return</a></body></html>') } };
const c = strings.ebookPreview;
function props() { return previewProps({ model: previewModel({ kind: "ebook", basename: "book.epub" }) }); }
beforeEach(() => { geometry.width = 390; vi.mocked(useEbook).mockReturnValue({ data: book }); Object.defineProperty(HTMLElement.prototype, "scrollIntoView", { configurable: true, value: vi.fn() }); });
describe("EPUB reading flow", () => {
  it("replaces the chapter with phone contents and returns to a focused fragment", async () => {
    render(<EbookPreview {...props()} />); await screen.findByRole("heading", { name: "First chapter" });
    fireEvent.click(screen.getByRole("button", { name: c.contents })); expect(screen.getByTestId(selectors.ebookPreview.pane)).toHaveClass("hidden");
    fireEvent.click(within(screen.getByRole("navigation", { name: c.contents })).getByRole("button", { name: "Second chapter" }));
    await waitFor(() => { expect(screen.getByRole("heading", { name: "Second chapter" })).toHaveFocus(); }); expect(screen.queryByRole("navigation", { name: c.contents })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("link", { name: "Return" })); await waitFor(() => { expect(screen.getByRole("heading", { name: "First chapter" })).toHaveFocus(); });
  });
  it("keeps desktop contents beside the chapter and searches without closing the drawer", async () => {
    geometry.width = 1000; render(<EbookPreview {...props()} />); await screen.findByRole("heading", { name: "First chapter" }); fireEvent.click(screen.getByRole("button", { name: c.contents })); expect(screen.getByTestId(selectors.ebookPreview.pane)).not.toHaveClass("hidden");
    fireEvent.click(screen.getByRole("button", { name: c.find })); const input = screen.getByRole("searchbox", { name: c.find }); expect(input).toHaveFocus(); fireEvent.change(input, { target: { value: "useful phrase" } }); await waitFor(() => { expect(screen.getByRole("button", { name: strings.artifactPreview.nextMatch })).toBeEnabled(); }); fireEvent.keyDown(input, { key: "Escape" }); expect(input).not.toBeInTheDocument(); expect(screen.getByRole("button", { name: c.find })).toHaveFocus();
  });
  it("offers reading preferences, bound chapter navigation and download recovery", async () => {
    const { unmount } = render(<EbookPreview {...props()} />); await screen.findByRole("heading", { name: "First chapter" }); expect(screen.getByRole("button", { name: c.previous })).toBeDisabled(); fireEvent.click(screen.getByRole("button", { name: c.options })); fireEvent.click(screen.getByRole("radio", { name: "22 px" })); expect(screen.getByTestId(selectors.ebookPreview.article).parentElement).toHaveStyle({ fontSize: "22px" }); expect(screen.getByRole("link", { name: strings.messagesFileViewer.download })).toHaveAttribute("download", "book.epub"); unmount();
    vi.mocked(useEbook).mockReturnValue({ error: "encrypted" }); render(<EbookPreview {...props()} />); expect(screen.getByText(c.encrypted)).toBeInTheDocument(); expect(screen.getByTestId("file-preview-download")).toBeInTheDocument();
  });
});
