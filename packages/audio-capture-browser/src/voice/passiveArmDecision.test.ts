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

describe("package-owned passive wake-word lifecycle", () => {
  it("enters only when enabled, configured, visible, and idle", () => {
    expect(decidePassiveArm(base)).toBe("enter");
    expect(decidePassiveArm({ ...base, wakeWordConfigured: false })).toBe("none");
    expect(decidePassiveArm({ ...base, documentVisible: false })).toBe("none");
    expect(decidePassiveArm({ ...base, voiceState: "recording" })).toBe("none");
  });

  it("holds active listeners and exits when ownership is disabled", () => {
    expect(decidePassiveArm({ ...base, listenerActive: true })).toBe("none");
    expect(decidePassiveArm({ ...base, listenerActive: true, wakeWordEnabled: false })).toBe("exit");
    expect(decidePassiveArm({ ...base, listenerActive: true, voiceEnabled: false })).toBe("exit");
    expect(decidePassiveArm({ ...base, startBlocked: true })).toBe("none");
  });
});
