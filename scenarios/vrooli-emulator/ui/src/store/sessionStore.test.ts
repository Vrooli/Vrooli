import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import sessionStoreSource from "./sessionStore.ts?raw";

vi.mock("../lib/api/sessions", () => ({
  createSession: vi.fn(),
  destroySession: vi.fn(),
  heartbeatSession: vi.fn(),
  getSession: vi.fn(),
  listSessions: vi.fn(),
  launchApp: vi.fn(),
  executeSessionControl: vi.fn(),
  buildVncWsUrl: vi.fn(() => "ws://localhost/ws"),
}));

vi.mock("../lib/bridge", () => ({
  postSessionEvent: vi.fn(() => true),
}));

const sampleSession = {
  id: "session-1",
  scenario_name: "demo",
  state: "running" as const,
  vnc_port: 5900,
  ws_port: 6080,
  width: 1280,
  height: 720,
  created_at: "2026-01-01T00:00:00Z",
  last_heartbeat: "2026-01-01T00:00:00Z",
  is_recording: false,
  network_mode: "normal" as const,
  dark_mode: false,
  app_running: false,
  platform: "linux",
};

describe("sessionStore", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    const { useSessionStore } = await import("./sessionStore");
    useSessionStore.getState().reset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("startSession sets session state and posts session.created", async () => {
    const api = await import("../lib/api/sessions");
    const bridge = await import("../lib/bridge");
    (api.createSession as ReturnType<typeof vi.fn>).mockResolvedValueOnce(sampleSession);

    const { useSessionStore } = await import("./sessionStore");
    const session = await useSessionStore.getState().startSession({
      scenario_name: "demo",
      platform: "linux",
      width: 1280,
      height: 720,
    });

    expect(session).toEqual(sampleSession);
    expect(useSessionStore.getState().activeSession).toEqual(sampleSession);
    expect(useSessionStore.getState().connectionStatus).toBe("connecting");
    expect(bridge.postSessionEvent).toHaveBeenCalledWith(
      expect.objectContaining({ type: "session.created" }),
    );
  });

  it("startSession with backend error posts session.error", async () => {
    const api = await import("../lib/api/sessions");
    const bridge = await import("../lib/bridge");
    (api.createSession as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      ...sampleSession,
      state: "error",
      error: "boom",
    });

    const { useSessionStore } = await import("./sessionStore");
    const result = await useSessionStore.getState().startSession({ scenario_name: "demo" });
    expect(result).toBeNull();
    expect(useSessionStore.getState().connectionStatus).toBe("failed");
    expect(useSessionStore.getState().error).toBe("boom");
    expect(bridge.postSessionEvent).toHaveBeenCalledWith(
      expect.objectContaining({ type: "session.error" }),
    );
  });

  it("stopSession posts session.destroyed and clears active session", async () => {
    const api = await import("../lib/api/sessions");
    const bridge = await import("../lib/bridge");
    (api.createSession as ReturnType<typeof vi.fn>).mockResolvedValueOnce(sampleSession);
    (api.destroySession as ReturnType<typeof vi.fn>).mockResolvedValueOnce(undefined);

    const { useSessionStore } = await import("./sessionStore");
    await useSessionStore.getState().startSession({ scenario_name: "demo" });
    await useSessionStore.getState().stopSession();

    expect(useSessionStore.getState().activeSession).toBeNull();
    expect(useSessionStore.getState().connectionStatus).toBe("disconnected");
    expect(bridge.postSessionEvent).toHaveBeenCalledWith(
      expect.objectContaining({ type: "session.destroyed" }),
    );
  });

  it("setConnectionStatus posts session.state_changed when the status changes", async () => {
    const api = await import("../lib/api/sessions");
    const bridge = await import("../lib/bridge");
    (api.createSession as ReturnType<typeof vi.fn>).mockResolvedValueOnce(sampleSession);

    const { useSessionStore } = await import("./sessionStore");
    await useSessionStore.getState().startSession({ scenario_name: "demo" });
    (bridge.postSessionEvent as ReturnType<typeof vi.fn>).mockClear();

    useSessionStore.getState().setConnectionStatus("connected");
    expect(bridge.postSessionEvent).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "session.state_changed",
        payload: expect.objectContaining({ status: "connected" }),
      }),
    );

    (bridge.postSessionEvent as ReturnType<typeof vi.fn>).mockClear();
    useSessionStore.getState().setConnectionStatus("connected");
    expect(bridge.postSessionEvent).not.toHaveBeenCalled();
  });

  it("executeControl stores a lastCapture for capture actions and does not call any captures store", async () => {
    const api = await import("../lib/api/sessions");
    (api.createSession as ReturnType<typeof vi.fn>).mockResolvedValueOnce(sampleSession);
    (api.getSession as ReturnType<typeof vi.fn>).mockResolvedValue(sampleSession);
    (api.executeSessionControl as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      status: "ok",
      data: { capture_id: "cap-1", path: "/tmp/shot.png" },
    });

    const { useSessionStore } = await import("./sessionStore");
    await useSessionStore.getState().startSession({ scenario_name: "demo" });
    await useSessionStore.getState().executeControl("screenshot");

    const { lastCapture } = useSessionStore.getState();
    expect(lastCapture?.type).toBe("screenshot");
    expect(lastCapture?.path).toBe("/tmp/shot.png");
  });

  it("reset clears all session state", async () => {
    const api = await import("../lib/api/sessions");
    (api.createSession as ReturnType<typeof vi.fn>).mockResolvedValueOnce(sampleSession);
    const { useSessionStore } = await import("./sessionStore");
    await useSessionStore.getState().startSession({ scenario_name: "demo" });
    useSessionStore.getState().reset();
    expect(useSessionStore.getState().activeSession).toBeNull();
    expect(useSessionStore.getState().connectionStatus).toBe("disconnected");
    expect(useSessionStore.getState().lastCapture).toBeNull();
  });
});

describe("sessionStore source constraints", () => {
  const source = sessionStoreSource;

  it("does not reference any captures store", () => {
    const capturesStoreHook = ["use", "Captures", "Store"].join("");
    expect(source).not.toContain(capturesStoreHook);
    expect(source).not.toContain("capturesStore");
  });

  it("does not expose isOpen/open/close drawer slice", () => {
    expect(source).not.toMatch(/\bisOpen\b/);
    expect(source).not.toMatch(/^\s*open:\s*\(/m);
    expect(source).not.toMatch(/^\s*close:\s*\(/m);
  });

  it("does not reference any Desktop* identifier", () => {
    const legacyIdentifiers = [
      ["Live", "Desktop"].join(""),
      ["Desktop", "Session"].join(""),
      ["Desktop", "Controls"].join(""),
      ["Desktop", "Toolbar"].join(""),
      ["Desktop", "Drawer"].join(""),
    ];
    for (const identifier of legacyIdentifiers) {
      expect(source).not.toContain(identifier);
    }
  });
});
