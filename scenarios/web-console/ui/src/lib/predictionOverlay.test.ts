import { describe, expect, it, vi } from "vitest";
import { createPredictionOverlay } from "./predictionOverlay";

function fixture() {
  const container = document.createElement("div");
  const parent = document.createElement("div");
  const screen = document.createElement("div");
  screen.className = "xterm-screen";
  parent.appendChild(screen);
  container.appendChild(parent);
  document.body.appendChild(container);
  const bufferListeners: Array<() => void> = [];
  const resizeListeners: Array<() => void> = [];
  const terminal = {
    cols: 80,
    rows: 24,
    buffer: {
      onBufferChange: (listener: () => void) => {
        bufferListeners.push(listener);
        return { dispose: () => {} };
      },
    },
    onResize: (listener: () => void) => {
      resizeListeners.push(listener);
      return { dispose: () => {} };
    },
  } as any;
  return { container, screen, terminal, bufferListeners, resizeListeners };
}

describe("prediction overlay", () => {
  it("renders outside the xterm buffer and retires by cumulative offset", () => {
    const { container, terminal } = fixture();
    const overlay = createPredictionOverlay(terminal, container);
    overlay.add("a", 1, 2, 1);
    overlay.add("b", 2, 2, 2);
    const layer = container.querySelector("[data-testid='terminal-prediction-overlay']");
    expect(layer?.textContent).toBe("ab");
    overlay.retireThrough(1);
    expect(layer?.textContent).toBe("b");
    overlay.dispose();
    expect(container.querySelector("[data-testid='terminal-prediction-overlay']")).toBeNull();
  });

  it("clears every pending prediction", () => {
    const { container, terminal } = fixture();
    const overlay = createPredictionOverlay(terminal, container);
    overlay.add("x", 0, 0, 4);
    overlay.add("y", 1, 0, 8);
    overlay.clear();
    expect(container.querySelector("[data-testid='terminal-prediction-overlay']")?.textContent).toBe("");
  });

  it("ignores invalid entries and supports layout refresh", () => {
    const { container, screen, terminal } = fixture();
    const overlay = createPredictionOverlay(terminal, container);
    overlay.add("", 0, 0, 1);
    overlay.add("xy", 0, 0, 2);
    overlay.add("z", 0, 0, 3);
    expect(screen.parentElement?.querySelectorAll("span")).toHaveLength(1);
    overlay.updateLayout();
    overlay.retireThrough(99);
    expect(screen.parentElement?.querySelectorAll("span")).toHaveLength(0);
  });

  it("clears on a buffer switch and repositions when cell metrics change", () => {
    const { container, screen, terminal, bufferListeners, resizeListeners } = fixture();
    let width = 800;
    let height = 480;
    vi.spyOn(screen, "getBoundingClientRect").mockImplementation(() => ({
      width,
      height,
      top: 0,
      left: 0,
      right: width,
      bottom: height,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    } as DOMRect));
    const overlay = createPredictionOverlay(terminal, container);
    overlay.add("a", 1, 1, 1);
    const entry = container.querySelector("span") as HTMLSpanElement;
    expect(entry.style.left).toBe("10px");
    width = 1600;
    height = 960;
    overlay.updateLayout();
    expect(entry.style.left).toBe("20px");
    resizeListeners[0]?.();
    bufferListeners[0]?.();
    expect(container.querySelector("span")).toBeNull();
    overlay.dispose();
  });

  it("retires a matching cell and clears on a cursor mismatch", () => {
    const { container, terminal } = fixture();
    terminal.buffer.active = { cursorX: 5, cursorY: 1 };
    const overlay = createPredictionOverlay(terminal, container);
    overlay.add("a", 4, 1, 1);
    overlay.retireThrough(1, { col: 5, row: 1 });
    expect(container.querySelector("span")).toBeNull();

    overlay.add("b", 4, 1, 2);
    overlay.add("c", 5, 1, 3);
    overlay.retireThrough(3, { col: 20, row: 20 });
    expect(container.querySelector("span")).toBeNull();
    overlay.dispose();
  });

  it("only applies pending styling when latency marks a prediction unconfirmed", () => {
    const { container, terminal } = fixture();
    const overlay = createPredictionOverlay(terminal, container);
    overlay.add("a", 0, 0, 1, false);
    overlay.add("b", 1, 0, 2, true);
    const spans = [...container.querySelectorAll("span")];
    expect(spans[0]?.style.textDecoration).toBe("none");
    expect(spans[0]?.style.opacity).toBe("1");
    expect(spans[1]?.style.textDecoration).toBe("underline");
    expect(spans[1]?.style.opacity).toBe("0.72");
    overlay.dispose();
  });

  it("returns a safe no-op overlay when xterm has no screen", () => {
    const container = document.createElement("div");
    const overlay = createPredictionOverlay({ cols: 80, rows: 24 } as never, container);
    expect(() => {
      overlay.add("a", 0, 0, 1);
      overlay.retireThrough(1);
      overlay.clear();
      overlay.updateLayout();
      overlay.dispose();
    }).not.toThrow();
  });
});
