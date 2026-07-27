import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useLiveDesktopStore } from "../../store/liveDesktopStore";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { VncCanvas } from "./VncCanvas";

const { rfbInstances, buildVncWsUrlMock } = vi.hoisted(() => ({
  rfbInstances: [] as Array<{
    scaleViewport: boolean;
    resizeSession: boolean;
    clipViewport: boolean;
    disconnect: ReturnType<typeof vi.fn>;
    addEventListener: ReturnType<typeof vi.fn>;
  }>,
  buildVncWsUrlMock: vi.fn(),
}));

vi.mock("@novnc/novnc", () => ({
  default: class RFBMock {
    scaleViewport = false;
    resizeSession = true;
    clipViewport = true;
    disconnect = vi.fn();
    addEventListener = vi.fn();

    constructor() {
      rfbInstances.push(this);
    }
  },
}));

vi.mock("../../lib/api/livedesktop", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../../lib/api/livedesktop")>();
  return { ...actual, buildVncWsUrl: buildVncWsUrlMock };
});

class ResizeObserverMock {
  observe = vi.fn();
  disconnect = vi.fn();
}

function latestRfb() {
  const rfb = rfbInstances[0];
  if (!rfb) {
    throw new Error("VNC client was not created");
  }
  return rfb;
}

describe("VncCanvas", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    rfbInstances.length = 0;
    buildVncWsUrlMock.mockReturnValue("ws://127.0.0.1:6080/vnc");
    vi.stubGlobal("ResizeObserver", ResizeObserverMock);
    useLiveDesktopStore.setState({
      activeSession: null,
      connectionStatus: "disconnected",
      error: null,
      isOpen: true,
      scenarioName: "desktop-app",
      appPath: null,
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("configures a local VNC transport and reports successful connection", () => {
    renderWithProviders(<VncCanvas sessionId="session-1" />);

    const rfb = latestRfb();
    expect(buildVncWsUrlMock).toHaveBeenCalledWith("session-1");
    expect(rfb).toBeDefined();
    expect(rfb.scaleViewport).toBe(true);
    expect(rfb.resizeSession).toBe(false);
    expect(rfb.clipViewport).toBe(false);

    const connect = rfb.addEventListener.mock.calls.find(
      ([event]) => event === "connect",
    )?.[1] as (() => void) | undefined;
    connect?.();

    expect(useLiveDesktopStore.getState().connectionStatus).toBe("connected");
  });

  it("surfaces an unclean disconnect and disconnects on unmount", () => {
    const { unmount } = renderWithProviders(
      <VncCanvas sessionId="session-2" />,
    );
    const rfb = latestRfb();
    const disconnect = rfb.addEventListener.mock.calls.find(
      ([event]) => event === "disconnect",
    )?.[1] as ((event: CustomEvent<{ clean: boolean }>) => void) | undefined;

    disconnect?.(new CustomEvent("disconnect", { detail: { clean: false } }));
    expect(useLiveDesktopStore.getState().error).toBe("VNC connection lost");

    unmount();
    expect(rfb.disconnect).toHaveBeenCalledOnce();
  });

  it("reports the server-provided reason when VNC security negotiation fails", () => {
    renderWithProviders(<VncCanvas sessionId="session-3" />);
    const rfb = latestRfb();
    const securityFailure = rfb.addEventListener.mock.calls.find(
      ([event]) => event === "securityfailure",
    )?.[1] as ((event: CustomEvent<{ reason?: string }>) => void) | undefined;

    securityFailure?.(
      new CustomEvent("securityfailure", {
        detail: { reason: "auth rejected" },
      }),
    );

    expect(useLiveDesktopStore.getState().error).toBe(
      "VNC security error: auth rejected",
    );
  });
});
