import { ApiError } from "../api/client";

export type Result<T> = { ok: true; data: T } | { ok: false; error: ApiError };

export function ok<T>(data: T): Result<T> {
  return { ok: true, data };
}

export function err(error: ApiError): Result<never> {
  return { ok: false, error };
}

/**
 * Wrap an async call and convert thrown errors into a typed `Result`.
 * Non-ApiError throws are normalised to an `internal` envelope so callers
 * never have to branch on raw `unknown`.
 */
export async function tryCall<T>(fn: () => Promise<T>): Promise<Result<T>> {
  try {
    return ok(await fn());
  } catch (e) {
    if (e instanceof ApiError) return err(e);
    const msg = e instanceof Error ? e.message : String(e);
    const { makeApiError } = await import("../api/client");
    return err(makeApiError("internal", msg, 500));
  }
}
