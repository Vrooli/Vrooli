import { describe, expect, it } from "vitest";
import { makeReading } from "../test-utils/readings";
import { roomSurfaceStatus } from "./roomSurface";

describe("roomSurfaceStatus", () => {
  it("marks a room ready when every rendered figure is live and verified", () => {
    expect(roomSurfaceStatus([makeReading()], { swarm: { from_cache: false, staleness_ts: null, integration_status: "available" } })).toEqual({ state: "ready" });
  });

  it("marks the room partial when a source answered health but its figure is untrusted", () => {
    const reading = makeReading({ trust: "UNTRUSTED", trustReason: "selector returned no number" });
    const status = roomSurfaceStatus([reading], { lpbs: { from_cache: false, staleness_ts: null, integration_status: "available" } });
    expect(status.state).toBe("partial");
    expect(status.statusMessage).toMatch(/Each figure shows its own source state/);
  });

  it("marks the room partial when its source is stale or unavailable", () => {
    const reading = makeReading();
    expect(roomSurfaceStatus([reading], { swarm: { from_cache: true, staleness_ts: "2026-11-20T00:00:00Z", integration_status: "available" } }).state).toBe("partial");
    expect(roomSurfaceStatus([reading], { swarm: { from_cache: false, staleness_ts: null, integration_status: "unavailable" } }).state).toBe("partial");
  });

  it("marks a room with no readings empty", () => {
    expect(roomSurfaceStatus([], {})).toEqual({ state: "empty" });
  });
});
