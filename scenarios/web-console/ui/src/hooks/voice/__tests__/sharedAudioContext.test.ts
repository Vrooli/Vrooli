import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  getSharedAudioContext,
  ensureAudioContextOnGesture,
  closeSharedAudioContext,
} from "../sharedAudioContext";

// --- Mock AudioContext in jsdom ---

class MockAudioContext {
  state = "running";
  resume = vi.fn().mockResolvedValue(undefined);
  close = vi.fn().mockResolvedValue(undefined);
}

globalThis.AudioContext = MockAudioContext as unknown as typeof AudioContext;

describe("sharedAudioContext", () => {
  afterEach(() => {
    closeSharedAudioContext();
  });

  it("getSharedAudioContext returns an AudioContext", () => {
    const ctx = getSharedAudioContext();
    expect(ctx).toBeInstanceOf(MockAudioContext);
  });

  it("multiple calls return the same instance (singleton)", () => {
    const a = getSharedAudioContext();
    const b = getSharedAudioContext();
    expect(a).toBe(b);
  });

  it("after closeSharedAudioContext, next call creates a new instance", () => {
    const first = getSharedAudioContext();
    closeSharedAudioContext();
    const second = getSharedAudioContext();
    expect(second).not.toBe(first);
    expect(second).toBeInstanceOf(MockAudioContext);
  });

  it("ensureAudioContextOnGesture installs event listener and creates context on pointerdown", () => {
    const addSpy = vi.spyOn(document, "addEventListener");

    ensureAudioContextOnGesture();

    // Should have installed pointerdown and keydown listeners
    const pointerCall = addSpy.mock.calls.find(([evt]) => evt === "pointerdown");
    const keydownCall = addSpy.mock.calls.find(([evt]) => evt === "keydown");
    expect(pointerCall).toBeDefined();
    expect(keydownCall).toBeDefined();

    // Dispatch pointerdown to trigger context creation
    document.dispatchEvent(new Event("pointerdown", { bubbles: true }));

    const ctx = getSharedAudioContext();
    expect(ctx).toBeInstanceOf(MockAudioContext);

    addSpy.mockRestore();
  });

  it("ensureAudioContextOnGesture is idempotent (multiple calls don't add multiple listeners)", () => {
    const addSpy = vi.spyOn(document, "addEventListener");

    ensureAudioContextOnGesture();
    const countAfterFirst = addSpy.mock.calls.filter(
      ([evt]) => evt === "pointerdown" || evt === "keydown",
    ).length;

    ensureAudioContextOnGesture();
    const countAfterSecond = addSpy.mock.calls.filter(
      ([evt]) => evt === "pointerdown" || evt === "keydown",
    ).length;

    // No additional listeners should have been added
    expect(countAfterSecond).toBe(countAfterFirst);

    addSpy.mockRestore();
  });
});
