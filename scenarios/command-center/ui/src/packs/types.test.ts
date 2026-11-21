import { describe, expect, it, vi } from "vitest";
import { registerEnabledPacks, type ConnectorPack } from "./types";

describe("connector pack catalog membership", () => {
  it("registers bespoke visuals only when enabled", () => {
    const scene = vi.fn() as ConnectorPack["compositions"][string];
    const pack: ConnectorPack = { id: "vrooli", sourceDescriptors: ["swarm-manager"], readouts: { ladder: "Ladder" }, compositions: { meridian: scene } };
    const engine = { readouts: { hero: "Hero" }, compositions: {} };
    expect(registerEnabledPacks(engine, [pack], new Set()).readouts).toEqual({ hero: "Hero" });
    expect(registerEnabledPacks(engine, [pack], new Set(["vrooli"])).compositions.meridian).toBe(scene);
  });
});
