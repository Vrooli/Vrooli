import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { VirtualList } from "../../../../components/VirtualList/versions/1.1.0/VirtualList";

type Row = { id: number; label: string };
const rows: Row[] = Array.from({ length: 200 }, (_, id) => ({ id, label: `Row ${id}` }));

class ResizeObserverStub {
  static instances: ResizeObserverStub[] = [];
  callback: ResizeObserverCallback;
  target?: Element;
  constructor(callback: ResizeObserverCallback) { this.callback = callback; ResizeObserverStub.instances.push(this); }
  observe(target: Element) { this.target = target; }
  disconnect() {}
  emit(height: number) { this.callback([{ contentRect: { height } } as ResizeObserverEntry], this as unknown as ResizeObserver); }
}

describe("VirtualList", () => {
  afterEach(() => { ResizeObserverStub.instances = []; vi.unstubAllGlobals(); });

  it("re-measures a row above the viewport without losing the visible scroll anchor", () => {
    vi.stubGlobal("ResizeObserver", ResizeObserverStub);
    render(<VirtualList items={rows} height={360} initialScrollTop={7200} overscan={70} getItemKey={(row) => String(row.id)} renderItem={(row) => <span>{row.label}</span>} />);
    const viewport = screen.getByTestId("data-display.virtual-list").querySelector("[data-rcl-virtual-list-viewport]") as HTMLElement;
    expect(screen.getByText("Row 100")).toBeTruthy();
    const row40 = ResizeObserverStub.instances.find((observer) => observer.target?.textContent === "Row 40");
    expect(row40).toBeTruthy();
    act(() => row40?.emit(192));
    expect(viewport.scrollTop).toBe(7320);
  });

  it("reports scroll changes through the owned viewport", async () => {
    const onScroll = vi.fn();
    render(<VirtualList items={rows.slice(0, 3)} onScrollPositionChange={onScroll} renderItem={(row) => <span>{row.label}</span>} />);
    const viewport = screen.getByTestId("data-display.virtual-list").querySelector("[data-rcl-virtual-list-viewport]") as HTMLElement;
    viewport.scrollTop = 24;
    await act(async () => {
      fireEvent.scroll(viewport);
      await new Promise((resolve) => setTimeout(resolve, 25));
    });
    expect(onScroll).toHaveBeenCalledWith(24);
  });
});
