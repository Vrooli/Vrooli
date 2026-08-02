/**
 * Self-tests for the spatial mock builders.
 *
 * Test-utils that ship without their own tests are how fakes drift
 * silently. The trio below pins the contract every consumer relies on:
 *
 *   - GamepadInputManager mock is a fresh instance per build (no
 *     shared mutable state across tests)
 *   - The constructor double captures the `onAction` callback the hook
 *     passed in, so consumers can assert the latest closure is invoked
 *   - SpatialNavController mock's `registerGroup` returns a callable
 *     cleanup that's also exposed via the `.cleanup` slot
 *
 * If a builder grows a new method, extend this file before any hook
 * test starts depending on it.
 */
import { describe, expect, it, vi } from "vitest";

import {
  makeGamepadInputManagerCtor,
  makeMockGamepadInputManager,
  makeMockSpatialNavController,
} from "./spatial";

describe("makeMockGamepadInputManager", () => {
  it("returns a fresh instance with start/dispose vi.fn slots", () => {
    const a = makeMockGamepadInputManager();
    const b = makeMockGamepadInputManager();
    expect(a).not.toBe(b);
    expect(vi.isMockFunction(a.start)).toBe(true);
    expect(vi.isMockFunction(a.dispose)).toBe(true);
    expect(a.onAction).toBeUndefined();
  });
});

describe("makeGamepadInputManagerCtor", () => {
  it("captures the onAction callback onto the instance and returns it", () => {
    const instance = makeMockGamepadInputManager();
    const Ctor = makeGamepadInputManagerCtor(instance);
    const handler = vi.fn();

    const got = Ctor({ onAction: handler }) as unknown;

    expect(got).toBe(instance);
    expect(instance.onAction).toBe(handler);
  });
});

describe("makeMockSpatialNavController", () => {
  it("returns a fresh controller with vi.fn slots and a callable cleanup", () => {
    const c = makeMockSpatialNavController();
    expect(vi.isMockFunction(c.registerGroup)).toBe(true);
    expect(vi.isMockFunction(c.pushScope)).toBe(true);
    expect(vi.isMockFunction(c.popScope)).toBe(true);
    expect(vi.isMockFunction(c.dispose)).toBe(true);
    expect(vi.isMockFunction(c.cleanup)).toBe(true);
  });

  it("registerGroup returns the same cleanup fn the .cleanup slot exposes", () => {
    const c = makeMockSpatialNavController();
    const fakeEl = document.createElement("div");
    const returned = c.registerGroup(fakeEl, "spatial");
    expect(returned).toBe(c.cleanup);
  });
});
