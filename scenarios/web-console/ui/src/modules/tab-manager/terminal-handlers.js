let terminalHandlers = {
  onResize: null,
  onData: null,
};

export function setTerminalEventHandlers({ onResize, onData } = {}) {
  terminalHandlers = {
    onResize: typeof onResize === "function" ? onResize : null,
    onData: typeof onData === "function" ? onData : null,
  };
}

export function getTerminalHandlers() {
  return terminalHandlers;
}
