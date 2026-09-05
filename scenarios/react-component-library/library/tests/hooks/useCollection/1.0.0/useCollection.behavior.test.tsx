import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { createCommandRegistry } from "../../../../services/CommandRegistry/versions/1.0.0/CommandRegistry";
import { useCollection } from "../../../../hooks/useCollection/versions/1.0.0/useCollection";

type Row = { id: string; label: string; disabled?: boolean };
const rows: Row[] = [
  { id: "a", label: "Alpha" },
  { id: "b", label: "Beta", disabled: true },
  { id: "c", label: "Gamma" },
];
const getKey = (row: Row) => row.id;
const selectable = (row: Row) => row.disabled ? "Unavailable" : false;

function Harness({ selected, onChange }: { selected?: string[]; onChange?: (keys: string[]) => void }) {
  const collection = useCollection(rows, {
    getKey,
    selection: { mode: "multi", selected, onChange, selectable },
  });
  return <div {...collection.getContainerProps()}>{collection.rows.map((row) => {
    const state = collection.rowStateFor(row);
    return <button key={row.id} {...collection.getRowProps(row)}>{row.label}:{String(state.selection.selected)}</button>;
  })}</div>;
}

function ActionHarness({ onOpen = vi.fn(), getSearchText }: { onOpen?: (row: Row) => void; getSearchText?: (row: Row) => string }) {
  const collection = useCollection(rows, {
    getKey,
    getSearchText,
    onOpen,
    actions: [
      { id: "open", label: "Open", shortcut: "Enter", onSelect: (selected) => onOpen(selected[0]!) },
      { id: "hidden", label: "Hidden", hidden: (row) => row.id === "a", onSelect: vi.fn() },
      { id: "disabled", label: "Disabled", disabled: () => "Unavailable", onSelect: vi.fn() },
    ],
    selection: { mode: "multi", enterOn: ["shortcut"] },
  });
  return <div {...collection.getContainerProps()}>{collection.rows.map((row) => <button key={row.id} {...collection.getRowProps(row)}>{row.label}:{collection.cursorKey}</button>)}</div>;
}

function KeyboardEntryHarness() {
  const collection = useCollection(rows, { getKey, selection: { mode: "none", enterOn: ["shortcut"] } });
  return <div {...collection.getContainerProps()}>{collection.rows.map((row) => <button key={row.id} {...collection.getRowProps(row)}>{row.label}:{String(collection.rowStateFor(row).selection.selected)}</button>)}</div>;
}

function BulkHarness({ announce, onOutcome }: { announce: (message: string) => void; onOutcome: (outcomes: unknown[]) => void }) {
  const collection = useCollection(rows, {
    getKey,
    announce,
    actions: [{
      id: "archive",
      label: "Archive",
      bulk: true,
      onSelect: async ([row]) => {
        if (row?.id === "c") throw new Error("Gamma failed");
      },
    }],
    selection: { mode: "multi" },
  });
  return <div {...collection.getContainerProps()}>
    {collection.rows.map((row) => <button key={row.id} {...collection.getRowProps(row)}>{row.label}</button>)}
    <button type="button" onClick={async () => onOutcome(await collection.bulk.run("archive"))}>Run bulk</button>
  </div>;
}

function AnnounceHarness({ announce }: { announce: (message: string) => void }) {
  const collection = useCollection(rows, { getKey, announce, selection: { mode: "multi" } });
  return <div {...collection.getContainerProps()}>{collection.rows.map((row) => <button key={row.id} {...collection.getRowProps(row)}>{row.label}</button>)}</div>;
}

function RequeryHarness() {
  const [query, setQuery] = useState("");
  const collection = useCollection(rows, {
    getKey,
    query,
    search: (row, value) => row.label.toLowerCase().includes(value.toLowerCase()),
    selection: { mode: "multi" },
  });
  return <>
    <button type="button" onClick={() => setQuery("a")}>Filter</button>
    <div {...collection.getContainerProps()}>{collection.rows.map((row) => <button key={row.id} {...collection.getRowProps(row)}>{row.label}:{String(collection.rowStateFor(row).selection.selected)}</button>)}</div>
  </>;
}

describe("useCollection", () => {
  it("keeps disabled rows out of select-all and reports controlled selection", () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);
    fireEvent.keyDown(screen.getByRole("listbox"), { key: "a", ctrlKey: true });
    expect(onChange).toHaveBeenLastCalledWith(["a", "c"]);
    expect(screen.getByText("Beta:false")).toBeTruthy();
  });

  it("hydrates controlled selection and supports keyboard cursor selection", () => {
    const onChange = vi.fn();
    render(<Harness selected={["c"]} onChange={onChange} />);
    const list = screen.getByRole("listbox");
    fireEvent.keyDown(list, { key: "ArrowUp" });
    fireEvent.keyDown(list, { key: " " });
    expect(onChange).toHaveBeenLastCalledWith(["c", "a"]);
  });

  it("enters selection mode on a stationary long press when configured", () => {
    vi.useFakeTimers();
    const onChange = vi.fn();
    function LongPressHarness() {
      const collection = useCollection(rows, {
        getKey,
        selection: { mode: "none", enterOn: ["long-press"], onChange },
      });
      return <div {...collection.getContainerProps()}>{collection.rows.map((row) => <button key={row.id} {...collection.getRowProps(row)}>{row.label}:{String(collection.rowStateFor(row).selection.selected)}</button>)}</div>;
    }
    render(<LongPressHarness />);
    const row = screen.getAllByRole("listitem")[0];
    act(() => {
      fireEvent.pointerDown(row, { pointerId: 1, pointerType: "touch", clientX: 10, clientY: 10 });
      vi.advanceTimersByTime(450);
    });
    expect(screen.getByRole("listbox")).toBeTruthy();
    expect(screen.getByText("Alpha:true")).toBeTruthy();
    expect(onChange).toHaveBeenCalledWith(["a"]);
    vi.useRealTimers();
  });

  it("moves the cursor without selecting, then selects with space and ranges with shift", () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);
    const list = screen.getByRole("listbox");
    fireEvent.keyDown(list, { key: "ArrowDown" });
    expect(screen.getAllByRole("option")[1]).toHaveAttribute("tabindex", "0");
    expect(onChange).not.toHaveBeenCalled();
    fireEvent.keyDown(list, { key: " " });
    fireEvent.keyDown(list, { key: "ArrowDown", shiftKey: true });
    expect(onChange).toHaveBeenLastCalledWith(["c"]);
  });

  it("supports shift-click and modifier-click selection", () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);
    const rowsInDom = screen.getAllByRole("option");
    fireEvent.click(rowsInDom[0]);
    fireEvent.click(rowsInDom[2], { shiftKey: true });
    expect(onChange).toHaveBeenLastCalledWith(["a", "c"]);
    fireEvent.click(rowsInDom[0], { ctrlKey: true });
    expect(onChange).toHaveBeenLastCalledWith(["c"]);
  });

  it("exits selection mode with escape and enters it with the keyboard shortcut", () => {
    const onChange = vi.fn();
    render(<KeyboardEntryHarness />);
    const list = screen.getByRole("list");
    fireEvent.keyDown(list, { key: "Escape" });
    expect(screen.getByRole("list")).toBeTruthy();
    fireEvent.keyDown(screen.getByRole("list"), { key: "a", ctrlKey: true });
    expect(screen.getByRole("listbox")).toBeTruthy();
  });

  it("runs the primary action with one row and supports typeahead", () => {
    const onOpen = vi.fn();
    render(<ActionHarness onOpen={onOpen} getSearchText={(row) => row.label} />);
    const list = screen.getByRole("listbox");
    fireEvent.keyDown(list, { key: "g" });
    fireEvent.keyDown(list, { key: "Enter" });
    expect(onOpen).toHaveBeenCalledWith(rows[2]);
  });

  it("resolves hidden and disabled actions without running disabled work", () => {
    const onSelect = vi.fn();
    function ActionProbe() {
      const collection = useCollection(rows, { getKey, actions: [{ id: "hidden", label: "Hidden", hidden: () => true, onSelect }, { id: "blocked", label: "Blocked", disabled: () => "Unavailable", onSelect }] });
      return <>{collection.actionsFor(rows[0]).map((action) => <button key={action.id} onClick={() => void action.run()}>{action.label}:{action.disabledReason ?? "enabled"}</button>)}</>;
    }
    render(<ActionProbe />);
    expect(screen.queryByText(/Hidden/)).toBeNull();
    fireEvent.click(screen.getByText("Blocked:Unavailable"));
    expect(onSelect).not.toHaveBeenCalled();
  });

  it("registers and unregisters row commands with the mount lifecycle", () => {
    const registry = createCommandRegistry();
    function CommandHarness() {
      useCollection(rows, { getKey, commandRegistry: registry, registerCommands: true, actions: [{ id: "open", label: "Open", shortcut: "Enter", onSelect: vi.fn() }] });
      return null;
    }
    const view = render(<CommandHarness />);
    expect(registry.getSnapshot().commands).toHaveLength(3);
    view.unmount();
    expect(registry.getSnapshot().commands).toHaveLength(0);
  });

  it("reports per-row bulk outcomes and announces partial failure", async () => {
    const announce = vi.fn();
    const onOutcome = vi.fn();
    render(<BulkHarness announce={announce} onOutcome={onOutcome} />);
    const rowButtons = screen.getAllByRole("option");
    fireEvent.click(rowButtons[0]);
    fireEvent.click(rowButtons[2]);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Run bulk" }));
    });
    expect(onOutcome).toHaveBeenCalledWith([
      { id: "a", status: "success" },
      { id: "c", status: "failed", error: "Gamma failed" },
    ]);
    expect(announce).toHaveBeenCalledWith("1 action failed");
  });

  it("announces selection count changes", () => {
    const announce = vi.fn();
    render(<AnnounceHarness announce={announce} />);
    fireEvent.click(screen.getAllByRole("option")[0]);
    expect(announce).toHaveBeenCalledWith("1 selected");
  });

  it("does not enter selection mode when long-press is not configured", () => {
    vi.useFakeTimers();
    render(<KeyboardEntryHarness />);
    const row = screen.getAllByRole("listitem")[0];
    fireEvent.pointerDown(row, { pointerId: 1, pointerType: "touch", clientX: 10, clientY: 10 });
    act(() => { vi.advanceTimersByTime(600); });
    expect(screen.queryByRole("listbox")).toBeNull();
    fireEvent.pointerUp(row, { pointerId: 1, pointerType: "touch" });
    vi.useRealTimers();
  });

  it("cancels a long-press when the pointer moves beyond the gesture tolerance", () => {
    vi.useFakeTimers();
    function MovableLongPressHarness() {
      const collection = useCollection(rows, { getKey, selection: { mode: "none", enterOn: ["long-press"] } });
      return <div {...collection.getContainerProps()}>{collection.rows.map((row) => <button key={row.id} {...collection.getRowProps(row)}>{row.label}:{String(collection.rowStateFor(row).selection.selected)}</button>)}</div>;
    }
    render(<MovableLongPressHarness />);
    const row = screen.getAllByRole("listitem")[0];
    fireEvent.pointerDown(row, { pointerId: 1, pointerType: "touch", clientX: 10, clientY: 10 });
    fireEvent.pointerMove(row, { pointerId: 1, pointerType: "touch", clientX: 30, clientY: 10 });
    act(() => { vi.advanceTimersByTime(600); });
    expect(screen.queryByRole("listbox")).toBeNull();
    fireEvent.pointerCancel(row, { pointerId: 1, pointerType: "touch" });
    vi.useRealTimers();
  });

  it("keeps the selected anchor coherent when filtering the visible rows", () => {
    render(<RequeryHarness />);
    const list = screen.getByRole("listbox");
    fireEvent.click(screen.getAllByRole("option")[0]);
    fireEvent.click(screen.getAllByRole("option")[2], { shiftKey: true });
    fireEvent.click(screen.getByRole("button", { name: "Filter" }));
    expect(screen.getByText("Alpha:true")).toBeTruthy();
    expect(screen.getByText("Gamma:true")).toBeTruthy();
    expect(list).toBeTruthy();
  });

  it("keeps the headless controller free of component and primitive imports", () => {
    const source = readFileSync(resolve(process.cwd(), "../library/hooks/useCollection/versions/1.0.0/useCollection.ts"), "utf8");
    expect(source).not.toMatch(/@vrooli\/react-component-library\/(components|primitives)\//);
  });
});
