import { describe, it, expect } from "vitest";

import { ApiError, makeApiError } from "../api/client";
import { ok, err, tryCall } from "./result";

describe("ok / err", () => {
  it("wraps data in a success envelope", () => {
    const r = ok({ value: 1 });
    expect(r).toEqual({ ok: true, data: { value: 1 } });
  });

  it("wraps an ApiError in a failure envelope", () => {
    const e = makeApiError("not_found", "missing", 404);
    const r = err(e);
    expect(r.ok).toBe(false);
    if (!r.ok) {
      expect(r.error).toBe(e);
    }
  });
});

// Reject with an arbitrary value (Error or not). Typed `unknown` so the
// non-Error case stays clean under prefer-promise-reject-errors.
const rejectWith = (value: unknown): Promise<never> =>
  // Intentionally rejects with arbitrary (possibly non-Error) values to exercise
  // tryCall's normalisation of non-Error throws.
  // eslint-disable-next-line @typescript-eslint/prefer-promise-reject-errors
  Promise.reject(value);

describe("tryCall", () => {
  it("returns ok with the resolved value", async () => {
    const r = await tryCall(() => Promise.resolve(42));
    expect(r).toEqual({ ok: true, data: 42 });
  });

  it("passes an ApiError through unchanged", async () => {
    const e = makeApiError("unavailable", "down", 503);
    const r = await tryCall(() => rejectWith(e));
    expect(r.ok).toBe(false);
    if (!r.ok) {
      expect(r.error).toBe(e);
      expect(r.error.code).toBe("unavailable");
    }
  });

  it("normalises a plain Error into an internal ApiError", async () => {
    const r = await tryCall(() => rejectWith(new Error("boom")));
    expect(r.ok).toBe(false);
    if (!r.ok) {
      expect(r.error).toBeInstanceOf(ApiError);
      expect(r.error.code).toBe("internal");
      expect(r.error.status).toBe(500);
      expect(r.error.message).toContain("boom");
    }
  });

  it("normalises a non-Error throw via String()", async () => {
    const r = await tryCall(() => rejectWith("just-a-string"));
    expect(r.ok).toBe(false);
    if (!r.ok) {
      expect(r.error.code).toBe("internal");
      expect(r.error.message).toContain("just-a-string");
    }
  });
});
