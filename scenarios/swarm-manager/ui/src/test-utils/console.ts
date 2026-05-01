import { vi } from "vitest";

type ConsoleMethod = "error" | "warn";

export async function withExpectedConsoleMessage<T>(
  method: ConsoleMethod,
  expected: RegExp,
  fn: () => T | Promise<T>,
): Promise<T> {
  const original = console[method];
  const spy = vi.spyOn(console, method).mockImplementation((...args: unknown[]) => {
    const message = args.map(String).join(" ");
    if (!expected.test(message)) {
      original(...args);
    }
  });

  try {
    return await fn();
  } finally {
    spy.mockRestore();
  }
}
