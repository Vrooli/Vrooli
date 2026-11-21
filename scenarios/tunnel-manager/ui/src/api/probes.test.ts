import { describe, expect, it, vi } from "vitest";
import { classify, listProbes, probesClient, runProbes } from "./probes";

describe("probes API helpers", () => {
  it("unwraps probe runs", async () => {
    const results = [{ subdomain: "alpha" }];
    const spy = vi.spyOn(probesClient, "runProbes").mockResolvedValueOnce({ results } as never);
    await expect(runProbes()).resolves.toBe(results);
    expect(spy).toHaveBeenCalledWith({});
  });
  it("forwards history filters", async () => {
    const results = [{ subdomain: "alpha" }];
    const spy = vi.spyOn(probesClient, "listProbes").mockResolvedValueOnce({ results } as never);
    await expect(listProbes("alpha", 25)).resolves.toBe(results);
    expect(spy).toHaveBeenCalledWith({ subdomain: "alpha", limit: 25 });
  });
  it("unwraps classifications", async () => {
    const classifications = [{ subdomain: "alpha" }];
    const spy = vi.spyOn(probesClient, "classify").mockResolvedValueOnce({ classifications } as never);
    await expect(classify()).resolves.toBe(classifications);
    expect(spy).toHaveBeenCalledWith({});
  });
});
