import type { Terminal } from "@xterm/xterm";

export interface PredictionOverlay {
  add: (char: string, col: number, row: number, offset: number, unconfirmed?: boolean) => void;
  retireThrough: (offset: number, cursor?: { col: number; row: number }) => void;
  clear: () => void;
  updateLayout: () => void;
  dispose: () => void;
}

interface PredictionEntry {
  element: HTMLSpanElement;
  offset: number;
  col: number;
  row: number;
}

function nextCell(col: number, row: number, cols: number, rows: number): { col: number; row: number } {
  if (cols <= 0) return { col, row };
  if (col + 1 < cols) return { col: col + 1, row };
  return { col: 0, row: Math.min(Math.max(0, rows - 1), row + 1) };
}

/**
 * Renders speculative input above xterm without ever changing its buffer.
 * The terminal remains the authoritative selection/copy surface; entries are
 * disposable until the cumulative stdin acknowledgement retires them.
 */
export function createPredictionOverlay(
  terminal: Terminal,
  container: HTMLElement,
): PredictionOverlay {
  const screen = container.querySelector<HTMLElement>(".xterm-screen");
  if (!screen?.parentElement) {
    return {
      add: () => {},
      retireThrough: () => {},
      clear: () => {},
      updateLayout: () => {},
      dispose: () => {},
    };
  }

  const parent = screen.parentElement;
  const previousPosition = parent.style.position;
  if (!previousPosition) parent.style.position = "relative";
  const layer = document.createElement("div");
  layer.dataset.testid = "terminal-prediction-overlay";
  layer.style.position = "absolute";
  layer.style.inset = "0";
  layer.style.pointerEvents = "none";
  layer.style.overflow = "hidden";
  layer.setAttribute("aria-hidden", "true");
  parent.appendChild(layer);

  const entries: PredictionEntry[] = [];
  const updateEntry = (entry: PredictionEntry) => {
    const bounds = screen.getBoundingClientRect();
    const cellWidth = terminal.cols > 0 ? bounds.width / terminal.cols : 0;
    const cellHeight = terminal.rows > 0 ? bounds.height / terminal.rows : 0;
    const cursor = entry.element.dataset.cell?.split(",").map(Number) ?? [0, 0];
    entry.element.style.left = `${(cursor[0] ?? 0) * cellWidth}px`;
    entry.element.style.top = `${(cursor[1] ?? 0) * cellHeight}px`;
    entry.element.style.width = `${cellWidth}px`;
    entry.element.style.height = `${cellHeight}px`;
    entry.element.style.lineHeight = `${cellHeight}px`;
    entry.element.style.font = getComputedStyle(screen).font;
  };
  const updateLayout = () => entries.forEach(updateEntry);
  const clear = () => {
    entries.splice(0).forEach((entry) => entry.element.remove());
  };

  // Buffer switches (normal ↔ alternate) invalidate cell coordinates. The
  // terminal remains authoritative, so speculative cells must disappear
  // rather than being painted over a newly-selected buffer.
  const bufferDisposable = typeof terminal.buffer?.onBufferChange === "function"
    ? terminal.buffer.onBufferChange(() => clear())
    : undefined;
  const resizeDisposable = typeof terminal.onResize === "function"
    ? terminal.onResize(() => {
        clear();
        updateLayout();
      })
    : undefined;
  const resizeObserver = typeof ResizeObserver === "function"
    ? new ResizeObserver(() => updateLayout())
    : undefined;
  resizeObserver?.observe(screen);

  return {
    add(char, col, row, offset, unconfirmed = false) {
      if (!char || char.length !== 1) return;
      const previous = entries[entries.length - 1];
      if (previous && previous.col === col && previous.row === row) {
        const cell = nextCell(previous.col, previous.row, terminal.cols, terminal.rows);
        col = cell.col;
        row = cell.row;
      }
      const element = document.createElement("span");
      element.textContent = char;
      element.dataset.cell = `${col},${row}`;
      element.style.position = "absolute";
      element.style.color = "inherit";
      element.style.opacity = unconfirmed ? "0.72" : "1";
      element.style.textDecoration = unconfirmed ? "underline" : "none";
      element.style.pointerEvents = "none";
      const entry = { element, offset, col, row };
      entries.push(entry);
      layer.appendChild(element);
      updateEntry(entry);
    },
    retireThrough(offset, cursor) {
      const retiring = entries.filter((entry) => entry.offset <= offset);
      const last = retiring[retiring.length - 1];
      if (last && cursor) {
        const expected = nextCell(last.col, last.row, terminal.cols, terminal.rows);
        if (cursor.col !== expected.col || cursor.row !== expected.row) {
          clear();
          return;
        }
      }
      for (let i = entries.length - 1; i >= 0; i -= 1) {
        const entry = entries[i];
        if (!entry || entry.offset > offset) continue;
        entry.element.remove();
        entries.splice(i, 1);
      }
    },
    clear,
    updateLayout,
    dispose() {
      clear();
      bufferDisposable?.dispose();
      resizeDisposable?.dispose();
      resizeObserver?.disconnect();
      layer.remove();
      if (!previousPosition) parent.style.position = "";
    },
  };
}
