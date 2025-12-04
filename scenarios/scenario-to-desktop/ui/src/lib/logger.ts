type LogMethod = (message: string, ...args: unknown[]) => void;

const noop: LogMethod = () => {};

const safeConsole = typeof console !== "undefined" ? console : ({ log: noop, warn: noop, error: noop, info: noop, debug: noop } as Console);

export const logger = {
  info: ((msg: string, ...args: unknown[]) => safeConsole.info?.(msg, ...args)) as LogMethod,
  warn: ((msg: string, ...args: unknown[]) => safeConsole.warn?.(msg, ...args)) as LogMethod,
  error: ((msg: string, ...args: unknown[]) => safeConsole.error?.(msg, ...args)) as LogMethod,
  debug: ((msg: string, ...args: unknown[]) => safeConsole.debug?.(msg, ...args)) as LogMethod
};
