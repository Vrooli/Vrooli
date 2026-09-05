import { act, fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ContextMenu } from "../../../../components/ContextMenu/versions/1.3.0/ContextMenu";

describe("ContextMenu interaction contract", () => {
  it("renders an explicitly positioned menu and invokes its item", () => {
    const onSelect = vi.fn();
    render(<ContextMenu open position={{ x: 120, y: 80 }} title="Actions" closeLabel="Close actions" items={[{ id: "open", label: "Open", onSelect }]} />);
    fireEvent.click(screen.getByRole("menuitem", { name: "Open" }));
    expect(onSelect).toHaveBeenCalledOnce();
  });

  it("captures a right-click origin from its trigger", () => {
    const onOpenAt = vi.fn();
    render(
      <ContextMenu open={false} title="Actions" closeLabel="Close actions" items={[]} onOpenAt={onOpenAt}>
        <button type="button">Row</button>
      </ContextMenu>,
    );
    fireEvent.contextMenu(screen.getByRole("button", { name: "Row" }), { clientX: 31, clientY: 47 });
    expect(onOpenAt).toHaveBeenCalledWith({ x: 31, y: 47, pointerType: "mouse" });
  });

  it("opens from a long press without requiring a second trigger", () => {
    vi.useFakeTimers();
    const onOpenAt = vi.fn();
    render(
      <ContextMenu open={false} title="Actions" closeLabel="Close actions" items={[]} onOpenAt={onOpenAt}>
        <button type="button">Row</button>
      </ContextMenu>,
    );
    const row = screen.getByRole("button", { name: "Row" });
    fireEvent.pointerDown(row, { pointerId: 1, pointerType: "touch", clientX: 10, clientY: 20, button: 0 });
    act(() => vi.advanceTimersByTime(450));
    expect(onOpenAt).toHaveBeenCalledWith({ x: 10, y: 20, pointerType: "touch" });
    vi.useRealTimers();
  });
});
