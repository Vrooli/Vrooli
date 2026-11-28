import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SessionControlsMenu } from "./SessionControlsMenu";

const mockState = {
  activeSession: {
    id: "test-session",
    scenario_name: "test",
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
  },
  connectionStatus: "connected" as const,
  error: null,
  refreshSession: vi.fn(),
};

vi.mock("../../store/sessionStore", () => ({
  useSessionStore: Object.assign(
    (selector: (state: typeof mockState) => unknown) => selector(mockState),
    {
      getState: () => mockState,
    },
  ),
}));

vi.mock("../../lib/api/sessions", () => ({
  executeSessionControl: vi.fn().mockResolvedValue({ status: "ok" }),
}));

describe("SessionControlsMenu", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the Controls trigger button", () => {
    render(<SessionControlsMenu />);
    expect(screen.getByText("Controls")).toBeInTheDocument();
  });

  it("opens popover with section headers on click", () => {
    render(<SessionControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));

    expect(screen.getByText("App")).toBeInTheDocument();
    expect(screen.getByText("Environment")).toBeInTheDocument();
    expect(screen.getByText("Advanced")).toBeInTheDocument();
  });

  it("renders all action items", () => {
    render(<SessionControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));

    expect(screen.getByText("Launch App")).toBeInTheDocument();
    expect(screen.getByText("Quit App")).toBeInTheDocument();
    expect(screen.getByText("Screenshot")).toBeInTheDocument();
    expect(screen.getByText("Record")).toBeInTheDocument();
    expect(screen.getByText("Offline Mode")).toBeInTheDocument();
    expect(screen.getByText("Slow Connection")).toBeInTheDocument();
    expect(screen.getByText("Resize Display...")).toBeInTheDocument();
    expect(screen.getByText("Env Variables...")).toBeInTheDocument();
    expect(screen.getByText("Read Clipboard")).toBeInTheDocument();
    expect(screen.getByText("Write Clipboard...")).toBeInTheDocument();
    expect(screen.getByText("Dark Mode")).toBeInTheDocument();
    expect(screen.getByText("Locale...")).toBeInTheDocument();
  });

  it("shows recording state when session is recording", () => {
    mockState.activeSession.is_recording = true;
    render(<SessionControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));

    expect(screen.getByText("Stop Recording")).toBeInTheDocument();
    mockState.activeSession.is_recording = false;
  });

  it("shows toggle active state for offline mode", () => {
    (mockState.activeSession as { network_mode: string }).network_mode = "offline";
    render(<SessionControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));

    const offlineButton = screen.getByText("Offline Mode").closest("button");
    expect(offlineButton).toBeInTheDocument();
    const toggleDot = offlineButton?.querySelector(".bg-emerald-400");
    expect(toggleDot).toBeInTheDocument();

    (mockState.activeSession as { network_mode: string }).network_mode = "normal";
  });

  it("opens resize sub-form when clicking Resize Display", () => {
    render(<SessionControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));
    fireEvent.click(screen.getByText("Resize Display..."));

    expect(screen.getByText("Apply")).toBeInTheDocument();
  });

  it("opens env variables sub-form when clicking Env Variables", () => {
    render(<SessionControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));
    fireEvent.click(screen.getByText("Env Variables..."));

    expect(screen.getByText("Inject")).toBeInTheDocument();
  });

  it("calls executeSessionControl when clicking Launch App", async () => {
    const { executeSessionControl } = await import("../../lib/api/sessions");
    render(<SessionControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));
    fireEvent.click(screen.getByText("Launch App"));

    expect(executeSessionControl).toHaveBeenCalledWith("test-session", {
      action: "launch_app",
      params: undefined,
    });
  });

  it("calls executeSessionControl with params for dark mode toggle", async () => {
    const { executeSessionControl } = await import("../../lib/api/sessions");
    render(<SessionControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));
    fireEvent.click(screen.getByText("Dark Mode"));

    expect(executeSessionControl).toHaveBeenCalledWith("test-session", {
      action: "dark_mode",
      params: { enabled: true },
    });
  });
});
