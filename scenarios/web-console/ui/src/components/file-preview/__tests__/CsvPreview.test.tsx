import { act, fireEvent, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../../test-utils";
import { previewModel, previewProps } from "../../../test-utils/filePreview";
import { strings } from "../../../consts/strings";
import { CsvPreview } from "../renderers/CsvPreview";

vi.mock("../../markdown", () => ({ MarkdownRenderer: () => null }));
const c = strings.artifactPreview;
function props(content = "name,amount,status\nAlpha,10,settled\nBeta,2,pending\nGamma,100,settled") {
  return previewProps({ model: previewModel({ kind: "csv", basename: "orders.csv", resolvedPath: "/orders.csv" }), text: { resolvedPath: "/orders.csv", kind: "csv", mimeType: "text/csv", content, truncated: false } });
}
function tableRows() { return Array.from(document.querySelectorAll("tbody tr")); }

describe("table inspector", () => {
  it("sorts numerically, filters all loaded fields and restores original order", () => {
    render(<CsvPreview {...props()} />);
    const amount = screen.getByRole("button", { name: "amount" });
    fireEvent.click(amount);
    expect(tableRows()[0]).toHaveTextContent("Beta");
    expect(amount.closest("th")).toHaveAttribute("aria-sort", "ascending");
    fireEvent.click(amount);
    expect(tableRows()[0]).toHaveTextContent("Gamma");
    fireEvent.click(amount);
    expect(tableRows()[0]).toHaveTextContent("Alpha");
    fireEvent.change(screen.getByRole("searchbox", { name: c.filterRows }), { target: { value: "pending" } });
    expect(tableRows()).toHaveLength(1);
    expect(tableRows()[0]).toHaveTextContent("Beta");
  });
  it("provides full record details on demand with copy actions and a return path", () => {
    render(<CsvPreview {...props()} />);
    fireEvent.change(screen.getByRole("combobox", { name: c.tableView }), { target: { value: "records" } });
    const list = screen.getByTestId("artifact-table-records");
    const second = within(list).getAllByRole("button", { name: c.openRow })[1];
    if (!second) throw new Error("Missing second record");
    fireEvent.click(second);
    const detail = screen.getByTestId("artifact-table-record");
    expect(detail).toHaveTextContent("Beta");
    expect(detail).toHaveTextContent("pending");
    expect(within(detail).getAllByRole("button", { name: c.copyCell })).toHaveLength(3);
    fireEvent.click(screen.getByRole("button", { name: c.backToRows }));
    expect(screen.getByTestId("artifact-table-records")).toBeInTheDocument();
  });
  it("keeps extra fields visible and supports keyboard column resizing", () => {
    render(<CsvPreview {...props("name\nAlpha,extra,retained")} />);
    expect(screen.getByText("retained")).toBeInTheDocument();
    const handle = screen.getAllByRole("separator", { name: c.resizeColumn })[0];
    if (!handle) throw new Error("Missing column resize handle");
    expect(handle).toHaveAttribute("aria-valuenow", "180");
    fireEvent.keyDown(handle, { key: "ArrowRight" });
    expect(handle).toHaveAttribute("aria-valuenow", "196");
  });
  it("paginates without dropping rows and returns to page one when filtering", () => {
    const content = "name,value\n" + Array.from({ length: 51 }, (_, index) => `row-${String(index)},${String(index)}`).join("\n");
    render(<CsvPreview {...props(content)} />);
    expect(tableRows()).toHaveLength(50);
    fireEvent.click(screen.getByRole("button", { name: c.nextPage }));
    expect(tableRows()).toHaveLength(1);
    expect(tableRows()[0]).toHaveTextContent("row-50");
    fireEvent.change(screen.getByRole("searchbox", { name: c.filterRows }), { target: { value: "row-0" } });
    expect(tableRows()[0]).toHaveTextContent("row-0");
    expect(screen.getByRole("button", { name: c.previousPage })).toBeDisabled();
  });
  it("keeps one cell in the tab order and navigates adjacent cells with arrow keys", () => {
    render(<CsvPreview {...props()} />);
    const cells = Array.from(document.querySelectorAll<HTMLButtonElement>("td button"));
    expect(cells.filter((cell) => cell.tabIndex === 0)).toHaveLength(1);
    const first = cells[0];
    if (!first) throw new Error("Missing first cell");
    act(() => { first.focus(); });
    fireEvent.keyDown(first, { key: "ArrowRight" });
    expect(document.activeElement).toHaveTextContent("10");
    fireEvent.keyDown(document.activeElement ?? first, { key: "ArrowDown" });
    expect(document.activeElement).toHaveTextContent("2");
  });
  it("shows original source for malformed quoting and never interprets cell HTML", () => {
    const { rerender } = render(<CsvPreview {...props('name\n"unfinished')} />);
    expect(screen.getByText(c.malformedCsv)).toBeInTheDocument();
    expect(screen.getByTestId("file-preview-code")).toBeInTheDocument();
    rerender(<CsvPreview {...props('name\n<img src=x onerror=alert(1)>')} />);
    expect(screen.getByText("<img src=x onerror=alert(1)>")).toBeInTheDocument();
    expect(document.querySelector("img")).toBeNull();
  });
});
