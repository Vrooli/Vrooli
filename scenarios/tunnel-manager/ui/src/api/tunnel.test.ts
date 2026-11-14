import { describe, expect, it, vi } from "vitest";
import { getStatus, listMetrics, tunnelClient } from "./tunnel";

describe("tunnel API helpers", () => {
  it("returns tunnel status", async () => {
    const response = { status: "healthy" };
    const spy = vi.spyOn(tunnelClient, "getStatus").mockResolvedValueOnce(response as never);
    await expect(getStatus()).resolves.toBe(response);
    expect(spy).toHaveBeenCalledWith({});
  });
  it("converts a metrics window to timestamps", async () => {
    const samples = [{ value: 1 }];
    const spy = vi.spyOn(tunnelClient, "listMetrics").mockResolvedValueOnce({ samples } as never);
    const from = new Date("2026-08-21T12:00:00.000Z");
    const to = new Date("2026-08-21T13:00:00.000Z");
    await expect(listMetrics(from, to)).resolves.toBe(samples);
    expect(spy).toHaveBeenCalledWith({
      from: expect.objectContaining({ seconds: 1787313600n }),
      to: expect.objectContaining({ seconds: 1787317200n }),
    });
  });

  it("omits timestamps when no metrics window is supplied", async () => {
    const samples = [{ value: 1 }];
    const spy = vi.spyOn(tunnelClient, "listMetrics").mockResolvedValueOnce({ samples } as never);

    await expect(listMetrics()).resolves.toBe(samples);
    expect(spy).toHaveBeenCalledWith({ from: undefined, to: undefined });
  });
});
