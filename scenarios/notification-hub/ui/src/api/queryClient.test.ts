import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, it } from "vitest";

import { createQueryClient, retryTransientOnce } from "./queryClient";

describe("retryTransientOnce", () => {
  it("retries a transient failure once, then stops", () => {
    const outage = new ConnectError("down", Code.Unavailable);
    expect(retryTransientOnce(0, outage)).toBe(true);
    expect(retryTransientOnce(1, outage)).toBe(false);
  });

  it("never retries a refused session", () => {
    expect(retryTransientOnce(0, new ConnectError("no session", Code.Unauthenticated))).toBe(false);
  });
});

describe("createQueryClient", () => {
  it("applies the retry policy to every query", () => {
    expect(createQueryClient().getDefaultOptions().queries?.retry).toBe(retryTransientOnce);
  });
});
