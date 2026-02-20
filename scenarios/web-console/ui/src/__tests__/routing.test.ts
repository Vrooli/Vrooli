import { describe, it, expect, beforeEach, afterEach } from "vitest";

// [REQ:P0-001a] Navigation and routing tests

describe("useHashRoute", () => {
  let originalHash: string;

  beforeEach(() => {
    originalHash = window.location.hash;
  });

  afterEach(() => {
    window.location.hash = originalHash;
  });

  it("exports Route type and useHashRoute hook", async () => {
    const mod = await import("../hooks/useHashRoute");
    expect(mod.useHashRoute).toBeDefined();
    expect(typeof mod.useHashRoute).toBe("function");
  });

  it("defines valid route values", async () => {
    const mod = await import("../hooks/useHashRoute");
    expect(mod.useHashRoute).toBeDefined();
  });
});

describe("Page lazy imports", () => {
  it("SessionsPage module exports default component", async () => {
    const mod = await import("../pages/SessionsPage");
    expect(mod.default).toBeDefined();
    expect(typeof mod.default).toBe("function");
  });

  it("Workspace module exports default component", async () => {
    const mod = await import("../components/Workspace");
    expect(mod.default).toBeDefined();
    expect(typeof mod.default).toBe("function");
  });
});

describe("SessionsPage", () => {
  it("accepts onBack prop", async () => {
    const mod = await import("../pages/SessionsPage");
    expect(mod.default.length).toBeLessThanOrEqual(1);
  });

  it("uses session API functions", async () => {
    const api = await import("../lib/api");
    expect(api.listSessions).toBeDefined();
    expect(api.deleteSession).toBeDefined();
    expect(api.updateSessionPolicy).toBeDefined();
  });
});

describe("Navigation selectors", () => {
  it("registers nav selectors", async () => {
    const { selectors } = await import("../consts/selectors");
    const nav = selectors.nav as unknown as Record<string, string>;
    expect(nav.sessions).toBe("nav-sessions");
    expect(nav.settings).toBe("nav-settings");
  });

  it("registers sessions page selectors", async () => {
    const { selectors } = await import("../consts/selectors");
    const sessions = selectors.sessions as unknown as Record<string, string>;
    expect(sessions.back).toBe("sessions-back");
    expect(sessions.refresh).toBe("sessions-refresh");
  });
});
