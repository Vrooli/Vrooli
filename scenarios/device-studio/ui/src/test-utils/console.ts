type ConsoleTarget = Pick<Console, "error" | "warn">;
type ErrorFilter = (args: unknown[]) => boolean;

/** Capture unexpected output without relying on mutable Vitest spy history. */
export function installConsoleGuard(target: ConsoleTarget = console) {
  const originalError = target.error;
  const originalWarn = target.warn;
  const unexpected: string[] = [];
  target.error = (...args: unknown[]) => {
    // jsdom cannot parse some modern component-library stylesheets.
    if (!String(args[0]).includes("Could not parse CSS stylesheet")) {
      unexpected.push(`console.error: ${args.map(String).join(" ")}`);
    }
  };
  target.warn = (...args: unknown[]) => {
    unexpected.push(`console.warn: ${args.map(String).join(" ")}`);
  };
  return () => {
    target.error = originalError;
    target.warn = originalWarn;
    if (unexpected.length > 0) {
      throw new Error(`Unexpected console output during test:\n${unexpected.join("\n")}`);
    }
  };
}

/** Synchronous, narrowly scoped exception; unmatched errors reach the guard. */
export function withExpectedConsoleErrors<T>(
  expected: ErrorFilter,
  operation: () => T,
  target: ConsoleTarget = console,
): T {
  const original = target.error;
  target.error = (...args: unknown[]) => {
    if (!expected(args)) original.apply(target, args);
  };
  try {
    return operation();
  } finally {
    target.error = original;
  }
}
