import { fireEvent, render, screen, waitFor } from "@/test-utils";
import { act } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { Platform } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import { useLiveDesktopStore } from "../../store/liveDesktopStore";
import { LiveDesktopDrawer } from "./LiveDesktopDrawer";

vi.mock("./DesktopToolbar", () => ({
  DesktopToolbar: () => <div>Desktop toolbar</div>,
}));
vi.mock("./VncCanvas", () => ({
  VncCanvas: ({ sessionId }: { sessionId: string }) => <div>VNC session {sessionId}</div>,
}));

describe("LiveDesktopDrawer", () => {
  const close = vi.fn();
  const startSession = vi.fn();
  const setError = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    useLiveDesktopStore.setState({
      activeSession: null,
      connectionStatus: "disconnected",
      error: null,
      isOpen: true,
      scenarioName: "canvas-lab",
      appPath: "/artifacts/canvas-lab.AppImage",
      close,
      startSession,
      setError,
    });
  });

  it("starts a Linux session with the chosen dimensions and artifact", async () => {
    render(<LiveDesktopDrawer />);
    expect(screen.getByText("Desktop — canvas-lab")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Width"), { target: { value: "1440" } });
    fireEvent.change(screen.getByLabelText("Height"), { target: { value: "900" } });
    fireEvent.click(screen.getByRole("button", { name: "Start Session" }));

    await waitFor(() => {
      expect(startSession).toHaveBeenCalledWith({
        width: 1440,
        height: 900,
        scenarioName: "canvas-lab",
        artifactPath: "/artifacts/canvas-lab.AppImage",
        platform: Platform.LINUX,
      });
    });
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(close).toHaveBeenCalledOnce();
  });

  it("communicates startup and active VNC desktop states", () => {
    act(() => {
      useLiveDesktopStore.setState({ connectionStatus: "connecting" });
    });
    const { rerender } = render(<LiveDesktopDrawer />);
    expect(screen.getByText("Starting desktop session...")).toBeInTheDocument();

    act(() => {
      useLiveDesktopStore.setState({
        connectionStatus: "connected",
        activeSession: { sessionId: "desktop-1" } as never,
      });
    });
    rerender(<LiveDesktopDrawer />);
    expect(screen.getByText("Desktop toolbar")).toBeInTheDocument();
    expect(screen.getByText("VNC session desktop-1")).toBeInTheDocument();
    expect(screen.queryByText("Desktop — canvas-lab")).not.toBeInTheDocument();
  });

  it("clears a session error and retries using the current configuration", async () => {
    useLiveDesktopStore.setState({ error: "VNC tunnel unavailable" });
    render(<LiveDesktopDrawer />);
    expect(screen.getByText("Desktop Session Error")).toBeInTheDocument();
    expect(screen.getByText("VNC tunnel unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    expect(setError).toHaveBeenCalledWith(null);
    await waitFor(() => {
      expect(startSession).toHaveBeenCalledWith(expect.objectContaining({
        scenarioName: "canvas-lab",
        width: 1280,
        height: 720,
      }));
    });
  });

  it("does not issue a session request without a selected scenario", () => {
    useLiveDesktopStore.setState({ scenarioName: null, appPath: null });
    render(<LiveDesktopDrawer />);
    expect(screen.getByText("Interactive Desktop")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Start Session" }));
    expect(startSession).not.toHaveBeenCalled();
  });
});
