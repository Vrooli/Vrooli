import { describe, expect, it } from "vitest";
import {
  CATALOG_SINGULAR,
  SETTINGS_TASKS,
  describeEntry,
  roomsBindingSignal,
  roomsUsingTheme,
  seedEntry,
  slugify,
  taskById,
} from "./settingsTasks";
import type { CatalogEntry, Catalogs } from "./api";

const cosmos: CatalogEntry = { id: "cosmos", label: "Cosmos" };
const ember: CatalogEntry = { id: "ember" };
const orbital: CatalogEntry = { id: "orbital-field", slots: { running: { shape: "scalar" }, healthy: { shape: "scalar" } } };
const activeSignal: CatalogEntry = { id: "active_scenarios", label: "Active scenarios", shape: "scalar", coverage: "NOW" };
const rowsSignal: CatalogEntry = { id: "throughput_stats", shape: "rows", coverage: "IN-REACH" };
const missionControl: CatalogEntry = { id: "mission-control", title: "Mission Control", theme: "cosmos", composition: "orbital-field", bind: { running: "active_scenarios" } };
const theForge: CatalogEntry = { id: "the-forge", title: "The Forge", theme: "cosmos", composition: "orbital-field", bind: {} };
const swarmConnector: CatalogEntry = { id: "swarm", enabled: true, transport: "connect", pack: true };
const offConnector: CatalogEntry = { id: "off", enabled: false };

const catalogs: Catalogs = {
  themes: [cosmos, ember],
  compositions: [orbital],
  signals: [activeSignal, rowsSignal],
  rooms: [missionControl, theForge],
  connectors: [swarmConnector, offConnector],
};

describe("task registry", () => {
  it("exposes a stable set of tasks resolvable by id", () => {
    expect(SETTINGS_TASKS.length).toBeGreaterThanOrEqual(6);
    expect(taskById("recolor")?.catalog).toBe("themes");
    expect(taskById("add-room")?.create).toBe(true);
    expect(taskById(null)).toBeUndefined();
    expect(taskById("nope")).toBeUndefined();
  });
});

describe("relationship queries", () => {
  it("finds rooms binding a signal and using a theme", () => {
    expect(roomsBindingSignal("active_scenarios", catalogs)).toEqual(["Mission Control"]);
    expect(roomsBindingSignal("nobody", catalogs)).toEqual([]);
    expect(roomsUsingTheme("cosmos", catalogs)).toEqual(["Mission Control", "The Forge"]);
    expect(roomsUsingTheme("ember", catalogs)).toEqual([]);
  });
});

describe("describeEntry", () => {
  it("returns an empty chain for a missing entry", () => {
    expect(describeEntry("rooms", undefined, catalogs)).toEqual([]);
  });

  it("chains theme → room → composition → slots for a room, flagging unbound slots", () => {
    const chain = describeEntry("rooms", missionControl, catalogs);
    expect(chain.map((node) => node.kind)).toEqual(["theme", "room", "composition", "running slot", "healthy slot"]);
    expect(chain.find((node) => node.kind === "running slot")?.tone).toBe("ok");
    expect(chain.find((node) => node.kind === "healthy slot")?.tone).toBe("muted"); // unbound
  });

  it("flags a shape-mismatched binding as a warning", () => {
    const room: CatalogEntry = { id: "x", title: "X", theme: "cosmos", composition: "orbital-field", bind: { running: "throughput_stats" } };
    const chain = describeEntry("rooms", room, { ...catalogs, rooms: [room] });
    expect(chain.find((node) => node.kind === "running slot")?.tone).toBe("warn");
  });

  it("warns when a room names no theme or composition", () => {
    const chain = describeEntry("rooms", { id: "y", title: "Y" }, catalogs);
    expect(chain.find((node) => node.kind === "theme")?.tone).toBe("warn");
    expect(chain.find((node) => node.kind === "composition")?.tone).toBe("warn");
  });

  it("summarises a theme by how many rooms it paints", () => {
    expect(describeEntry("themes", cosmos, catalogs)[1]?.label).toBe("paints 2 rooms");
    expect(describeEntry("themes", ember, catalogs)[1]?.tone).toBe("muted");
  });

  it("summarises a signal by shape, coverage, and room usage", () => {
    const chain = describeEntry("signals", activeSignal, catalogs);
    expect(chain.map((node) => node.label)).toContain("scalar");
    expect(chain.map((node) => node.label)).toContain("bound in 1 room");
  });

  it("describes a connector's state, transport, and pack flag", () => {
    const chain = describeEntry("connectors", swarmConnector, catalogs);
    expect(chain.map((node) => node.label)).toEqual(["swarm", "enabled", "connect", "connector pack"]);
    expect(describeEntry("connectors", offConnector, catalogs)[1]?.tone).toBe("muted");
  });

  it("falls back to a single node for an unmodelled catalog", () => {
    expect(describeEntry("readouts", { id: "scalar-figure" }, catalogs)).toEqual([{ label: "scalar-figure", kind: "readouts", tone: "ok" }]);
  });

  it("labels a slot bound to an unknown signal by its id", () => {
    const room: CatalogEntry = { id: "z", title: "Z", theme: "cosmos", composition: "orbital-field", bind: { running: "ghost_signal" } };
    const node = describeEntry("rooms", room, { ...catalogs, rooms: [room] }).find((entry) => entry.kind === "running slot");
    expect(node?.label).toBe("ghost_signal");
    expect(node?.tone).toBe("ok"); // unknown signal can't be shape-checked, so not flagged
  });

  it("uses singular phrasing for a theme painting exactly one room", () => {
    const single = describeEntry("themes", cosmos, { ...catalogs, rooms: [missionControl] });
    expect(single[1]?.label).toBe("paints 1 room");
    expect(single[1]?.tone).toBe("ok");
  });

  it("warns on a shapeless signal bound in no rooms", () => {
    const shapeless: CatalogEntry = { id: "raw", coverage: "MISSING" };
    const chain = describeEntry("signals", shapeless, { ...catalogs, signals: [shapeless], rooms: [] });
    expect(chain.find((node) => node.kind === "shape")?.tone).toBe("warn");
    expect(chain.find((node) => node.label === "bound in no rooms")?.tone).toBe("muted");
  });
});

describe("create helpers", () => {
  it("seeds a room with the first theme and composition", () => {
    const room = seedEntry("rooms", "new-room", catalogs);
    expect(room).toMatchObject({ id: "new-room", theme: "cosmos", composition: "orbital-field", bind: {} });
  });

  it("seeds a connector enabled with defaults", () => {
    expect(seedEntry("connectors", "src", catalogs)).toMatchObject({ id: "src", enabled: true, transport: "http-json" });
  });

  it("seeds a bare entry for other catalogs", () => {
    expect(seedEntry("signals", "sig", catalogs)).toEqual({ id: "sig" });
  });

  it("falls back to default theme and composition when catalogs are empty", () => {
    expect(seedEntry("rooms", "r", {})).toMatchObject({ theme: "cosmos", composition: "orbital-field" });
  });

  it("slugifies titles into stable ids", () => {
    expect(slugify("  My New Room!! ")).toBe("my-new-room");
    expect(slugify("A/B — Test 2")).toBe("a-b-test-2");
    expect(CATALOG_SINGULAR.rooms).toBe("room");
  });
});
