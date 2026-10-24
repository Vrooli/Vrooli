import { act, fireEvent, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../../test-utils";
import { previewModel, previewProps } from "../../../test-utils/filePreview";
import { strings } from "../../../consts/strings";
import { copyText } from "../../../lib/clipboard";
import { parseStructure } from "../parseStructure";
import { StructuredPreview } from "../renderers/StructuredPreview";

vi.mock("../../markdown", () => ({ MarkdownRenderer: () => null }));
vi.mock("../useStructuredData", () => ({ useStructuredData: (source: string, json: boolean, truncated: boolean) => truncated ? { error: "limit" } : parseStructure(source, json) }));
vi.mock("@vrooli/react-component-library/useResizeObserver", () => ({ useResizeObserver: () => ({ ref: () => {}, rect: { width: viewport.width, height: 600 } }) }));
vi.mock("../../../lib/clipboard", () => ({ copyText: vi.fn().mockResolvedValue(true) }));
const viewport = vi.hoisted(() => ({ width: 390 }));
const c = strings.readerPreview;
function props(content = '{"group":{"number":900719925474099312345,"empty":""},"enabled":false}', path = "/config.json") {
  return previewProps({ model: previewModel({ previewId: "config", kind: "code", basename: path.slice(1), resolvedPath: path }), text: { resolvedPath: path, kind: "code", mimeType: "text/plain", content, truncated: false } });
}
beforeEach(() => { vi.clearAllMocks(); viewport.width = 390; });
describe("configuration inspector", () => {
  it("finds collapsed values and copies original numeric spelling and unambiguous property paths", async () => {
    render(<StructuredPreview {...props()} />);
    fireEvent.change(screen.getByRole("searchbox", { name: c.filterStructure }), { target: { value: "number" } });
    fireEvent.click(screen.getByRole("button", { name: /number 900719925474099312345/ }));
    const detail = screen.getByTestId("reader-structure-detail");
    expect(detail).toHaveTextContent("$.group.number");
    expect(screen.getByRole("button", { name: c.backStructure })).toHaveFocus();
    await act(async () => { fireEvent.click(within(detail).getByRole("button", { name: c.copyValue })); await Promise.resolve(); });
    expect(copyText).toHaveBeenLastCalledWith("900719925474099312345");
    await act(async () => { fireEvent.click(within(detail).getByRole("button", { name: c.copyPath })); await Promise.resolve(); });
    expect(copyText).toHaveBeenLastCalledWith("$.group.number");
    fireEvent.click(screen.getByRole("button", { name: c.backStructure }));
    expect(screen.getByTestId("reader-structure-tree")).toBeInTheDocument();
  });
  it("expands/collapses, supports keyboard navigation and distinguishes empty strings from null", () => {
    render(<StructuredPreview {...props()} />);
    fireEvent.click(screen.getByRole("button", { name: c.expandAll }));
    const rows = Array.from(document.querySelectorAll<HTMLButtonElement>("[data-structure-inspect]"));
    const first = rows[0]; if (!first) throw new Error("Missing root");
    act(() => { first.focus(); }); fireEvent.keyDown(first, { key: "ArrowDown" });
    expect(document.activeElement).toHaveTextContent("group");
    fireEvent.click(screen.getByRole("button", { name: /^empty/ }));
    expect(screen.getByTestId("reader-structure-detail")).toHaveTextContent(c.emptyString);
    fireEvent.click(screen.getByRole("button", { name: c.backStructure }));
    fireEvent.click(screen.getByRole("button", { name: c.collapseAll }));
    expect(document.querySelectorAll("[data-structure-inspect]")).toHaveLength(1);
  });
  it("switches YAML documents, retains references without expansion and resets selection", () => {
    render(<StructuredPreview {...props('value: &shared One\ncopy: *shared\n---\nnext: Two', '/config.yaml')} />);
    fireEvent.click(screen.getByRole("button", { name: /copy \*shared/ }));
    expect(screen.getByTestId("reader-structure-detail")).toHaveTextContent(c.referenceHint);
    fireEvent.change(screen.getByRole("combobox", { name: c.yamlDocument }), { target: { value: "1" } });
    expect(screen.getByRole("button", { name: /next "Two"/ })).toBeInTheDocument();
    expect(screen.queryByTestId("reader-structure-detail")).not.toBeInTheDocument();
  });
  it("formats JSON as a separate view and keeps copying/downloading the original source", () => {
    const content = '{"id":900719925474099312345,"name":"file"}'; render(<StructuredPreview {...props(content)} />);
    fireEvent.click(screen.getByRole("button", { name: c.formatJson }));
    expect(screen.getByText(c.formattedView)).toBeInTheDocument();
    expect(screen.getByTestId("file-preview-code")).toHaveTextContent("900719925474099312345");
    expect(screen.getByTestId("file-preview-download")).toHaveAttribute("href", "/preview/blob");
  });
  it("falls back to original source on invalid or truncated input", () => {
    const first = props('{"broken":'); const { rerender } = render(<StructuredPreview {...first} />);
    expect(screen.getByText(c.invalidStructure)).toBeInTheDocument(); expect(screen.getByTestId("file-preview-code")).toBeInTheDocument();
    const next = props("{}");
    rerender(<StructuredPreview {...next} model={{ ...next.model, previewId: "next" }} text={next.text ? { ...next.text, truncated: true } : null} />);
    expect(screen.getByText(c.structureLimit)).toBeInTheDocument();
  });
  it("keeps desktop details beside the tree and navigates collections by keyboard", async () => {
    viewport.width = 1000;
    render(<StructuredPreview {...props()} />);
    expect(screen.getByTestId("reader-structure-tree")).toBeInTheDocument();
    expect(screen.getByTestId("reader-structure-detail")).toBeInTheDocument();
    const root = document.querySelector<HTMLButtonElement>('[data-structure-inspect="0"]');
    if (!root) throw new Error("Missing root");
    act(() => { root.focus(); });
    fireEvent.keyDown(root, { key: "ArrowRight" });
    const group = document.activeElement; if (!(group instanceof HTMLButtonElement)) throw new Error("Missing group");
    expect(group).toHaveTextContent("group");
    fireEvent.keyDown(group, { key: "ArrowRight" }); fireEvent.keyDown(group, { key: "ArrowRight" });
    expect(document.activeElement).toHaveTextContent("number");
    fireEvent.keyDown(document.activeElement ?? root, { key: "End" }); expect(document.activeElement).toHaveTextContent("enabled");
    fireEvent.keyDown(document.activeElement ?? root, { key: "Home" }); expect(root).toHaveFocus();
    fireEvent.keyDown(root, { key: "ArrowLeft" }); expect(document.querySelectorAll("[data-structure-inspect]")).toHaveLength(1);
    fireEvent.click(screen.getByRole("button", { name: c.expandAll }));
    fireEvent.click(screen.getByTitle("$.group.empty"));
    const detail = screen.getByTestId("reader-structure-detail");
    await act(async () => { fireEvent.click(within(detail).getByRole("button", { name: c.copySource })); await Promise.resolve(); });
    expect(copyText).toHaveBeenLastCalledWith('""');
  });
  it("restores original JSON, copies its exact bytes, and reports an empty filter result", async () => {
    const content = '{"id":900719925474099312345}'; render(<StructuredPreview {...props(content)} />);
    fireEvent.click(screen.getByRole("button", { name: c.formatJson }));
    fireEvent.click(screen.getByRole("button", { name: c.originalSource }));
    expect(screen.queryByText(c.formattedView)).not.toBeInTheDocument();
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: c.copySource })); await Promise.resolve(); });
    expect(copyText).toHaveBeenLastCalledWith(content);
    fireEvent.click(screen.getByRole("button", { name: c.showStructure }));
    fireEvent.change(screen.getByRole("searchbox", { name: c.filterStructure }), { target: { value: "missing" } });
    expect(screen.getByText(c.noProperties)).toBeInTheDocument();
    fireEvent.keyDown(screen.getByRole("searchbox"), { key: "Escape" });
    expect(screen.getByRole("searchbox")).toHaveValue("");
  });
  it("copies the actual empty source of an implicit YAML null and announces clipboard feedback", async () => {
    render(<StructuredPreview {...props("value:\n", "/config.yaml")} />);
    fireEvent.click(screen.getByTitle("$.value"));
    const detail = screen.getByTestId("reader-structure-detail");
    await act(async () => { fireEvent.click(within(detail).getByRole("button", { name: c.copySource })); await Promise.resolve(); });
    expect(copyText).toHaveBeenLastCalledWith("");
    expect(screen.getAllByRole("status", { hidden: true }).some((element) => element.textContent === strings.messagesFileViewer.copied)).toBe(true);
  });
  it("renders oversized source as one text node instead of invoking syntax or per-line rendering", () => {
    const source = "line\n".repeat(60_001);
    render(<StructuredPreview {...props(source)} />);
    const fallback = screen.getByTestId("file-preview-code");
    expect(fallback.tagName).toBe("PRE");
    expect(fallback).toHaveAttribute("tabindex", "0");
    expect(fallback.childNodes).toHaveLength(1);
    expect(fallback.textContent).toBe(source);
  });
});
