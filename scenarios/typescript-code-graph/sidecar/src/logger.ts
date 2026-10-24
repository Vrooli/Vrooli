// Stderr-only logger. Nothing in the sidecar writes to stdout except the
// IPC framer in framing.ts — stdout is reserved for line-delimited JSON
// responses and any stray write would corrupt the protocol.

function format(level: string, msg: string, extra?: Record<string, unknown>): string {
  const base = `[sidecar] ${level} ${msg}`;
  if (!extra) return base;
  try {
    return `${base} ${JSON.stringify(extra)}`;
  } catch {
    return base;
  }
}

export const logger = {
  debug(msg: string, extra?: Record<string, unknown>): void {
    if (process.env["SIDECAR_DEBUG"]) {
      process.stderr.write(format("debug", msg, extra) + "\n");
    }
  },
  info(msg: string, extra?: Record<string, unknown>): void {
    process.stderr.write(format("info", msg, extra) + "\n");
  },
  warn(msg: string, extra?: Record<string, unknown>): void {
    process.stderr.write(format("warn", msg, extra) + "\n");
  },
  error(msg: string, extra?: Record<string, unknown>): void {
    process.stderr.write(format("error", msg, extra) + "\n");
  },
};
