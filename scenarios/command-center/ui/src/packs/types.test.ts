import { describe, expect, it, vi } from "vitest";
import { registerEnabledPacks, type ConnectorPack } from "./types";

describe("connector pack catalog membership", () => {
  it("registers bespoke visuals only when enabled", () => {
    const scene = vi.fn() as ConnectorPack["compositions"][number]["create"];
    const meridian = { id: "meridian", create: scene, slots: {} };
    const pack: ConnectorPack = { id: "vrooli", sourceDescriptors: ["swarm-manager"], readouts: { ladder: "Ladder" }, compositions: [meridian] };
    const engine = { readouts: { hero: "Hero" }, compositions: [] };
    expect(registerEnabledPacks(engine, [pack], new Set()).readouts).toEqual({ hero: "Hero" });
    expect(registerEnabledPacks(engine, [pack], new Set()).compositions).toEqual([]);
    expect(registerEnabledPacks(engine, [pack], new Set(["vrooli"])).compositions).toEqual([meridian]);
  });

  it("rejects a pack composition whose id collides with an engine composition", () => {
    const scene = vi.fn() as ConnectorPack["compositions"][number]["create"];
    const pack: ConnectorPack = { id: "vrooli", sourceDescriptors: [], readouts: {}, compositions: [{ id: "orbital-field", create: scene, slots: {} }] };
    const engine = { readouts: {}, compositions: [{ id: "orbital-field", create: scene, slots: {} }] };
    expect(() => registerEnabledPacks(engine, [pack], new Set(["vrooli"]))).toThrow('duplicate composition id "orbital-field"');
  });
});
