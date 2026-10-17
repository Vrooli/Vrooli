import { beforeEach, describe, expect, it, vi } from "vitest";
import { deviceIdentity, setDeviceLabel } from "./deviceIdentity";
describe("deviceIdentity", () => {
  beforeEach(() => localStorage.clear());
  it("keeps the generated identifier across reload-like reads", () => {
    const first = deviceIdentity();
    expect(deviceIdentity().id).toBe(first.id);
    setDeviceLabel("Desk");
    expect(deviceIdentity().label).toBe("Desk");
  });

  it("uses platform-independent labels and the random fallback when needed", () => {
    expect(deviceIdentity().label).toBe("Desktop");
    localStorage.clear();
    vi.stubGlobal("crypto", {});
    vi.stubGlobal("screen", { width: 500, height: 900 });
    vi.stubGlobal("navigator", { userAgent: "Android" });
    expect(deviceIdentity().label).toBe("Android phone");
    localStorage.clear();
    vi.stubGlobal("screen", { width: 1200, height: 800 });
    vi.stubGlobal("navigator", { userAgent: "iPad" });
    expect(deviceIdentity().label).toBe("Tablet");
    vi.unstubAllGlobals();
  });
});
