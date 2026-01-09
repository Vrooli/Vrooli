/**
 * @typedef {(tab: import("../types.d.ts").TerminalTab, cols: number, rows: number) => void} ResizeHandler
 * @typedef {(tab: import("../types.d.ts").TerminalTab, data: string) => void} DataHandler
 */

const terminalHandlersDefaults = {
  onResize: /** @type {ResizeHandler | null} */ (null),
  onData: /** @type {DataHandler | null} */ (null),
};

let terminalHandlers = { ...terminalHandlersDefaults };

/**
 * @param {{ onResize?: ResizeHandler; onData?: DataHandler }} [handlers]
 */
export function setTerminalEventHandlers({ onResize, onData } = {}) {
  terminalHandlers = {
    onResize: typeof onResize === "function" ? onResize : null,
    onData: typeof onData === "function" ? onData : null,
  };
}

export function getTerminalHandlers() {
  return terminalHandlers;
}
