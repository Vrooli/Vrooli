import { describe, it, expect } from "vitest";

describe("Page lazy imports", () => {
  it("Workspace module exports default component", async () => {
    const mod = await import("../components/Workspace");
    expect(mod.default).toBeDefined();
    expect(typeof mod.default).toBe("function");
  });
});

describe("Navigation selectors", () => {
  it("registers settings selector", async () => {
    const { selectors } = await import("../consts/selectors");
    const nav = selectors.nav as unknown as Record<string, string>;
    expect(nav.settings).toBe("nav-settings");
  });

  it("registers settings action selectors", async () => {
    const { selectors } = await import("../consts/selectors");
    const settings = selectors.settings as unknown as Record<string, string>;
    expect(settings.error).toBe("settings-error");
    expect(settings.createProfile).toBe("create-profile");
  });
});
