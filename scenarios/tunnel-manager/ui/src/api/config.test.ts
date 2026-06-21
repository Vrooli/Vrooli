import { describe, expect, it, vi } from "vitest";

import {
  adoptIngress,
  configClient,
  getConfig,
  getConfigState,
  getDrift,
  ignoreIngress,
  pruneIngress,
  sync,
} from "./config";
import { makeConfigResponse, makeDriftResponse, makeIngressEntry } from "../test-utils/mocks/config";

describe("config API helpers", () => {
  it("returns the full config state from the generated client", async () => {
    const resp = makeConfigResponse();
    const spy = vi.spyOn(configClient, "getConfig").mockResolvedValueOnce(resp);

    await expect(getConfigState()).resolves.toBe(resp);
    expect(spy).toHaveBeenCalledWith({});
  });

  it("unwraps just the persisted config", async () => {
    const resp = makeConfigResponse();
    vi.spyOn(configClient, "getConfig").mockResolvedValueOnce(resp);

    await expect(getConfig()).resolves.toBe(resp.config);
  });

  it("reconciles additively by default and threads prune", async () => {
    const spy = vi.spyOn(configClient, "sync").mockResolvedValue({} as never);

    await sync();
    expect(spy).toHaveBeenCalledWith({ dryRun: false, prune: false });

    await sync({ dryRun: true, prune: true });
    expect(spy).toHaveBeenCalledWith({ dryRun: true, prune: true });
  });

  it("returns the drift report", async () => {
    const resp = makeDriftResponse();
    const spy = vi.spyOn(configClient, "getDrift").mockResolvedValueOnce(resp);

    await expect(getDrift()).resolves.toBe(resp);
    expect(spy).toHaveBeenCalledWith({});
  });

  it("adopts a hostname with the given provenance hints", async () => {
    const entry = makeIngressEntry();
    const spy = vi.spyOn(configClient, "adoptIngress").mockResolvedValueOnce({ entry } as never);

    await expect(adoptIngress({ hostname: "a.itsagitime.com", target: "http://127.0.0.1:9000" })).resolves.toEqual({
      entry,
    });
    expect(spy).toHaveBeenCalledWith({ hostname: "a.itsagitime.com", target: "http://127.0.0.1:9000" });
  });

  it("ignores a hostname with an optional note", async () => {
    const entry = makeIngressEntry();
    const spy = vi.spyOn(configClient, "ignoreIngress").mockResolvedValueOnce({ entry } as never);

    await ignoreIngress({ hostname: "a.itsagitime.com", note: "ack" });
    expect(spy).toHaveBeenCalledWith({ hostname: "a.itsagitime.com", note: "ack" });
  });

  it("prunes a single hostname", async () => {
    const spy = vi.spyOn(configClient, "pruneIngress").mockResolvedValueOnce({ pruned: true } as never);

    await expect(pruneIngress("a.itsagitime.com")).resolves.toEqual({ pruned: true });
    expect(spy).toHaveBeenCalledWith({ hostname: "a.itsagitime.com" });
  });
});
