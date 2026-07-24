import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@/test-utils";
import { DesktopControlsMenu } from "./DesktopControlsMenu";

// Mock the store
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
  },
  connectionStatus: "connected" as const,
  error: null,
  isOpen: true,
  scenarioName: "test",
  appPath: null,
  refreshSession: vi.fn(),
};

vi.mock("../../store/liveDesktopStore", () => ({
  useLiveDesktopStore: Object.assign(
    (selector: (state: typeof mockState) => unknown) => selector(mockState),
    {
      getState: () => mockState,
    },
  ),
}));

// Mock the API
vi.mock("../../lib/api/livedesktop", () => ({
  executeDesktopControl: vi.fn().mockResolvedValue({ status: "ok" }),
}));

describe("DesktopControlsMenu", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the Controls trigger button", () => {
    render(<DesktopControlsMenu />);
    expect(screen.getByText("Controls")).toBeInTheDocument();
  });

  it("opens popover with section headers on click", () => {
    render(<DesktopControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));

    expect(screen.getByText("App")).toBeInTheDocument();
    expect(screen.getByText("Environment")).toBeInTheDocument();
    expect(screen.getByText("Advanced")).toBeInTheDocument();
  });

  it("renders all action items", () => {
    render(<DesktopControlsMenu />);
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
    render(<DesktopControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));

    expect(screen.getByText("Stop Recording")).toBeInTheDocument();
    mockState.activeSession.is_recording = false;
  });

  it("shows toggle active state for offline mode", () => {
    (mockState.activeSession as { network_mode: string }).network_mode = "offline";
    render(<DesktopControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));

    // Offline toggle indicator should be active (emerald dot)
    const offlineButton = screen.getByText("Offline Mode").closest("button");
    expect(offlineButton).toBeInTheDocument();
    const toggleDot = offlineButton?.querySelector(".bg-emerald-400");
    expect(toggleDot).toBeInTheDocument();

    (mockState.activeSession as { network_mode: string }).network_mode = "normal";
  });

  it("opens resize sub-form when clicking Resize Display", () => {
    render(<DesktopControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));
    fireEvent.click(screen.getByText("Resize Display..."));

    expect(screen.getByText("Apply")).toBeInTheDocument();
  });

  it("opens env variables sub-form when clicking Env Variables", () => {
    render(<DesktopControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));
    fireEvent.click(screen.getByText("Env Variables..."));

    expect(screen.getByText("Inject")).toBeInTheDocument();
  });

  it("calls executeDesktopControl when clicking Launch App", async () => {
    const { executeDesktopControl } = await import("../../lib/api/livedesktop");
    render(<DesktopControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));
    fireEvent.click(screen.getByText("Launch App"));

    await waitFor(() => expect(executeDesktopControl).toHaveBeenCalledWith("test-session", {
      action: "launch_app",
      params: undefined,
    }));
  });

  it("calls executeDesktopControl with params for dark mode toggle", async () => {
    const { executeDesktopControl } = await import("../../lib/api/livedesktop");
    render(<DesktopControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));
    fireEvent.click(screen.getByText("Dark Mode"));

    await waitFor(() => expect(executeDesktopControl).toHaveBeenCalledWith("test-session", {
      action: "dark_mode",
      params: { enabled: true },
    }));
  });
});
