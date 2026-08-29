import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders as render } from "../../test-utils";
import type { SnippetDTO } from "../../api/snippets";

const hook = vi.hoisted(() => ({ useSnippets: vi.fn() }));
vi.mock("../../hooks/useSnippets", () => hook);

import { SnippetPicker } from "./SnippetPicker";

const snippets: SnippetDTO[] = [
  { id: "plain", name: "Plain", body: "Demand exact evidence", color: "#38d9c0", pinned: false, use_count: 2, last_used_at: "", sort_order: 0, created_at: "", updated_at: "" },
  { id: "vars", name: "Investigate", body: "Check {{scenario}} with {{owner}}", color: "#4dabf7", pinned: true, use_count: 1, last_used_at: "", sort_order: 0, created_at: "", updated_at: "" },
];

describe("SnippetPicker", () => {
  const touch = vi.fn();
  const save = vi.fn();
  beforeEach(() => {
    vi.clearAllMocks();
    touch.mockResolvedValue(undefined);
    save.mockImplementation((input) => Promise.resolve({ ...snippets[0], ...input } as SnippetDTO));
    hook.useSnippets.mockReturnValue({ snippets, status: "ready", touch, save });
  });

  it("filters by body text and keeps rows at the 44px floor", () => {
    render(<SnippetPicker open onClose={vi.fn()} onInsert={vi.fn()} />);
    fireEvent.change(screen.getByTestId("snippet-filter"), { target: { value: "exact evidence" } });
    const row = screen.getByTestId("snippet-row-plain");
    expect(row.className).toContain("min-h-11");
    expect(screen.queryByTestId("snippet-row-vars")).toBeNull();
  });

  it("inserts exact plain text and touches only after insertion resolves", async () => {
    let resolve!: () => void;
    const onInsert = vi.fn(() => new Promise<void>((done) => { resolve = done; }));
    render(<SnippetPicker open onClose={vi.fn()} onInsert={onInsert} />);
    fireEvent.click(screen.getByTestId("snippet-row-plain"));
    expect(onInsert).toHaveBeenCalledWith("Demand exact evidence", snippets[0]);
    expect(touch).not.toHaveBeenCalled();
    resolve();
    await waitFor(() => { expect(touch).toHaveBeenCalledWith("plain"); });
  });

  it("detours unresolved names through the variable sheet", () => {
    const onInsert = vi.fn();
    render(<SnippetPicker open onClose={vi.fn()} onInsert={onInsert} autoValues={{ scenario: "web-console" }} />);
    fireEvent.click(screen.getByTestId("snippet-row-vars"));
    expect(onInsert).not.toHaveBeenCalled();
    expect(screen.getByTestId("snippet-variable-readonly-scenario")).toHaveTextContent("web-console");
    expect(screen.getByTestId("snippet-variable-input-owner")).toBeTruthy();
  });

  it("does not touch when dismissed", () => {
    const onClose = vi.fn();
    render(<SnippetPicker open onClose={onClose} onInsert={vi.fn()} />);
    fireEvent.keyDown(window, { key: "Escape" });
    expect(touch).not.toHaveBeenCalled();
  });

  it("filters through one shared field carrying its own search adornment", () => {
    render(<SnippetPicker open onClose={vi.fn()} onInsert={vi.fn()} />);
    // The field is a grouped control, not a bare input hand-styled at the call
    // site: the group owns the border and supplies the leading search glyph.
    const group = screen.getByTestId("snippet-filter-group");
    expect(group).toHaveAttribute("data-rcl-input-group");
    expect(within(group).getByTestId("snippet-filter-group-adornment-leading")).toBeTruthy();
    expect(within(group).getByTestId("snippet-filter")).toHaveAttribute("data-rcl-input");
  });

  it("offers clearing only while a filter is set and restores every row", () => {
    render(<SnippetPicker open onClose={vi.fn()} onInsert={vi.fn()} />);
    expect(screen.queryByTestId("snippet-filter-clear")).toBeNull();
    fireEvent.change(screen.getByTestId("snippet-filter"), { target: { value: "exact evidence" } });
    expect(screen.queryByTestId("snippet-row-vars")).toBeNull();
    fireEvent.click(screen.getByTestId("snippet-filter-clear"));
    expect(screen.getByTestId("snippet-row-vars")).toBeTruthy();
    expect(screen.queryByTestId("snippet-filter-clear")).toBeNull();
  });

  it("keeps each list count in the accessible name, not only in the badge", () => {
    render(<SnippetPicker open onClose={vi.fn()} onInsert={vi.fn()} />);
    // The shared strip marks its badge `aria-hidden`, so a count that lived
    // only there would be invisible to assistive technology.
    const strip = screen.getByTestId("navigation.tabs");
    expect(within(strip).getByTestId("snippet-segment-all")).toHaveAccessibleName(/\(2\)/);
    expect(within(strip).getByTestId("snippet-segment-pinned")).toHaveAccessibleName(/\(1\)/);
  });

  it("selects lists through the shared tab strip and narrows Pinned to pinned rows", () => {
    render(<SnippetPicker open onClose={vi.fn()} onInsert={vi.fn()} />);
    const strip = screen.getByTestId("navigation.tabs");
    expect(within(strip).getByTestId("snippet-segment-recent")).toHaveAttribute("aria-selected", "true");
    fireEvent.click(within(strip).getByTestId("snippet-segment-pinned"));
    expect(screen.getByTestId("snippet-row-vars")).toBeTruthy();
    expect(screen.queryByTestId("snippet-row-plain")).toBeNull();
  });

  it("states an empty result instead of an empty box, and the state clears the filter", () => {
    render(<SnippetPicker open onClose={vi.fn()} onInsert={vi.fn()} />);
    fireEvent.change(screen.getByTestId("snippet-filter"), { target: { value: "nothing matches this" } });
    const list = screen.getByTestId("snippet-list");
    expect(within(list).getByText("snippets.picker.noMatches")).toBeTruthy();
    fireEvent.click(within(list).getByRole("button", { name: "snippets.picker.clearFilter" }));
    expect(screen.getByTestId("snippet-row-plain")).toBeTruthy();
  });

  it("reveals pin and edit on a row without making them the only route", () => {
    render(<SnippetPicker open onClose={vi.fn()} onInsert={vi.fn()} />);
    const track = screen.getByTestId("snippet-swipe-plain");
    fireEvent.click(within(track).getByTestId("snippet-swipe-plain.action.pin"));
    expect(save).toHaveBeenCalledWith(expect.objectContaining({ id: "plain", pinned: true }));
  });

  it("opens the save sheet in edit mode with the row's own text", async () => {
    render(<SnippetPicker open onClose={vi.fn()} onInsert={vi.fn()} />);
    const track = screen.getByTestId("snippet-swipe-vars");
    fireEvent.click(within(track).getByTestId("snippet-swipe-vars.action.edit"));
    const sheet = await screen.findByTestId("snippet-save-sheet");
    expect(within(sheet).getByTestId("snippet-save-name")).toHaveValue("Investigate");
  });

  it("shows how many variables a row will ask for before it is picked", () => {
    render(<SnippetPicker open onClose={vi.fn()} onInsert={vi.fn()} />);
    const row = screen.getByTestId("snippet-row-vars");
    expect(within(row).getByLabelText("snippets.variableCount")).toHaveTextContent("2");
    expect(within(screen.getByTestId("snippet-row-plain")).queryByLabelText("snippets.variableCount")).toBeNull();
  });
});
