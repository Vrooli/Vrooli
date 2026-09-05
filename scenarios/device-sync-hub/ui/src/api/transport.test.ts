import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { authedFetch, DEVICE_TOKEN_HEADER } from "./transport";
import { clearSession, saveSession } from "../features/session/store";

describe("authedFetch dual-credential wrapper", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn().mockResolvedValue(new Response("{}", { status: 200 }));
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    clearSession();
  });

  const headersOf = () => new Headers((fetchSpy.mock.calls[0]?.[1] as RequestInit).headers);

  it("attaches no credential headers when the session is empty", async () => {
    await authedFetch("/x");
    const headers = headersOf();
    expect(headers.has(DEVICE_TOKEN_HEADER)).toBe(false);
    expect(headers.has("Authorization")).toBe(false);
  });

  it("attaches the device token header when paired", async () => {
    saveSession({ deviceToken: "dt-1", device: null, ownerToken: null, ownerEmail: null });
    await authedFetch("/x");
    expect(headersOf().get(DEVICE_TOKEN_HEADER)).toBe("dt-1");
  });

  it("attaches both device + owner headers when both are present", async () => {
    saveSession({ deviceToken: "dt-1", device: null, ownerToken: "owner-jwt", ownerEmail: null });
    await authedFetch("/x");
    const headers = headersOf();
    expect(headers.get(DEVICE_TOKEN_HEADER)).toBe("dt-1");
    expect(headers.get("Authorization")).toBe("Bearer owner-jwt");
  });

  it("reads credentials fresh per call (pairing mid-session takes effect)", async () => {
    await authedFetch("/x");
    expect(headersOf().has(DEVICE_TOKEN_HEADER)).toBe(false);

    saveSession({ deviceToken: "late", device: null, ownerToken: null, ownerEmail: null });
    await authedFetch("/y");
    const second = new Headers((fetchSpy.mock.calls[1]?.[1] as RequestInit).headers);
    expect(second.get(DEVICE_TOKEN_HEADER)).toBe("late");
  });
});
