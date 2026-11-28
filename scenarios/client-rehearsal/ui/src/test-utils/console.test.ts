import { describe, expect, it, vi } from "vitest";
import { installConsoleGuard, withExpectedConsoleErrors } from "./console";

describe("console guard", () => {
  it("reports both unexpected channels and restores the original methods", () => {
    const target = { error: vi.fn(), warn: vi.fn() };
    const originals = { ...target };
    const finish = installConsoleGuard(target);
    target.error("broken", 7);
    target.warn("warning");
    expect(finish).toThrow("console.error: broken 7\nconsole.warn: warning");
    expect(target.error).toBe(originals.error);
    expect(target.warn).toBe(originals.warn);
  });

  it("permits only the documented jsdom stylesheet diagnostic", () => {
    const target = { error: vi.fn(), warn: vi.fn() };
    const finish = installConsoleGuard(target);
    target.error(new Error("Could not parse CSS stylesheet"));
    expect(finish).not.toThrow();
  });

  it("forwards unrelated errors even inside an expected-error scope", () => {
    const target = { error: vi.fn(), warn: vi.fn() };
    const finish = installConsoleGuard(target);
    withExpectedConsoleErrors(args => args[0] === "expected", () => {
      target.error("expected");
      target.error("unrelated failure");
    }, target);
    expect(finish).toThrow("unrelated failure");
  });

  it("does not allow an expected diagnostic after its scope ends", () => {
    const target = { error: vi.fn(), warn: vi.fn() };
    const finish = installConsoleGuard(target);
    withExpectedConsoleErrors(args => args[0] === "expected", () => target.error("expected"), target);
    target.error("expected");
    expect(finish).toThrow("console.error: expected");
  });

  it("restores the guard if the scoped operation throws", () => {
    const target = { error: vi.fn(), warn: vi.fn() };
    const finish = installConsoleGuard(target);
    const guardedError = target.error;
    expect(() => withExpectedConsoleErrors(() => true, () => {
      throw new Error("operation failed");
    }, target)).toThrow("operation failed");
    expect(target.error).toBe(guardedError);
    expect(finish).not.toThrow();
  });
});
