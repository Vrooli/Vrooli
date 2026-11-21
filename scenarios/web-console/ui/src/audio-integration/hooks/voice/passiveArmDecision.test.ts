// Regression for the "always-on wake word never listens" bug: nothing in the
// app entered passive mode, so the toggle flipped a config bit but the mic was
// never opened. decidePassiveArm is the reconciliation that drives the
// always-on cycle (passive → wake → record → idle → passive). These tests pin
// the enter/exit/hold transitions across that lifecycle.

import { describe, expect, it } from "vitest";

import { decidePassiveArm, type PassiveArmInput } from "./passiveArmDecision";

const base: PassiveArmInput = {
  voiceEnabled: true,
  wakeWordEnabled: true,
  wakeWordConfigured: true,
  voiceState: "idle",
  listenerActive: false,
  startBlocked: false,
  documentVisible: true,
};

describe("decidePassiveArm", () => {
  it("arms passive listening when enabled, configured, and idle", () => {
    expect(decidePassiveArm(base)).toBe("enter");
  });

  it("does not arm until a template is configured", () => {
    expect(decidePassiveArm({ ...base, wakeWordConfigured: false })).toBe("none");
  });

  it("holds (no double-enter) while a listener is already active", () => {
    expect(decidePassiveArm({ ...base, listenerActive: true })).toBe("none");
  });

  it("holds while the mic is mid-flight, then re-arms once idle again", () => {
    // The always-on cycle: detection → preparing/recording → back to idle.
    for (const voiceState of ["preparing", "recording", "listening", "transcribing"] as const) {
      expect(decidePassiveArm({ ...base, voiceState })).toBe("none");
    }
    // Turn ended; idle with no listener → re-arm.
    expect(decidePassiveArm({ ...base, voiceState: "idle" })).toBe("enter");
  });

  it("does not retry-storm after a passive start failure (latch held)", () => {
    expect(decidePassiveArm({ ...base, startBlocked: true })).toBe("none");
  });

  it("does not arm while the document is hidden (iOS-PWA background-mic leak)", () => {
    // Even fully enabled + configured + idle, a hidden tab must not open the mic.
    expect(decidePassiveArm({ ...base, documentVisible: false })).toBe("none");
  });

  it("tears down the listener when the wake-word toggle goes off", () => {
    expect(decidePassiveArm({ ...base, wakeWordEnabled: false, listenerActive: true })).toBe("exit");
  });

  it("tears down when voice input is disabled entirely", () => {
    expect(decidePassiveArm({ ...base, voiceEnabled: false, listenerActive: true })).toBe("exit");
  });

  it("is a no-op when off and nothing is listening", () => {
    expect(decidePassiveArm({ ...base, wakeWordEnabled: false })).toBe("none");
    expect(decidePassiveArm({ ...base, voiceEnabled: false })).toBe("none");
  });
});
