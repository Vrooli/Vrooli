import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@/test-utils";
import { DesktopControlsMenu } from "./DesktopControlsMenu";
import {
  DesktopNetworkMode,
  DesktopSessionState,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/evidence_pb";

// Mock the store
const mockState = {
  activeSession: {
    sessionId: "test-session",
    scenarioName: "test",
    state: DesktopSessionState.RUNNING,
    vncPort: 5900,
    websocketPort: 6080,
    width: 1280,
    height: 720,
    recording: false,
    networkMode: DesktopNetworkMode.NORMAL,
    darkMode: false,
    appRunning: false,
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
  executeDesktopControl: vi.fn().mockResolvedValue({}),
  controlResultString: vi.fn(),
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
    mockState.activeSession.recording = true;
    render(<DesktopControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));

    expect(screen.getByText("Stop Recording")).toBeInTheDocument();
    mockState.activeSession.recording = false;
  });

  it("shows toggle active state for offline mode", () => {
    mockState.activeSession.networkMode = DesktopNetworkMode.OFFLINE;
    render(<DesktopControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));

    // Offline toggle indicator should be active (emerald dot)
    const offlineButton = screen.getByText("Offline Mode").closest("button");
    expect(offlineButton).toBeInTheDocument();
    const toggleDot = offlineButton?.querySelector(".bg-emerald-400");
    expect(toggleDot).toBeInTheDocument();

    mockState.activeSession.networkMode = DesktopNetworkMode.NORMAL;
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

    await waitFor(() => {
      expect(executeDesktopControl).toHaveBeenCalledWith("test-session", {
        action: "launch_app",
        params: undefined,
      });
    });
  });

  it("calls executeDesktopControl with params for dark mode toggle", async () => {
    const { executeDesktopControl } = await import("../../lib/api/livedesktop");
    render(<DesktopControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));
    fireEvent.click(screen.getByText("Dark Mode"));

    await waitFor(() => {
      expect(executeDesktopControl).toHaveBeenCalledWith("test-session", {
        action: "dark_mode",
        params: { enabled: true },
      });
    });
  });

  it("applies environment, clipboard, locale, bandwidth, and resize controls", async () => {
    const { executeDesktopControl } = await import("../../lib/api/livedesktop");
    render(<DesktopControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));

    fireEvent.click(screen.getByText("Env Variables..."));
    fireEvent.change(screen.getByRole("textbox", { name: "Environment variables" }), {
      target: { value: "ONE=1\nINVALID\n TWO = two " },
    });
    fireEvent.click(screen.getByText("Inject"));
    await waitFor(() => {
      expect(executeDesktopControl).toHaveBeenCalledWith("test-session", {
        action: "inject_env",
        params: { vars: { ONE: "1", TWO: "two" } },
      });
    });

    fireEvent.click(screen.getByText("Write Clipboard..."));
    fireEvent.change(screen.getByRole("textbox", { name: "Clipboard content" }), {
      target: { value: "desktop evidence" },
    });
    fireEvent.click(screen.getByText("Write"));
    await waitFor(() => {
      expect(executeDesktopControl).toHaveBeenCalledWith("test-session", {
        action: "clipboard_write",
        params: { content: "desktop evidence" },
      });
    });

    fireEvent.click(screen.getByText("Locale..."));
    fireEvent.change(screen.getByRole("textbox", { name: "Locale" }), {
      target: { value: "fr_FR.UTF-8" },
    });
    fireEvent.click(screen.getByText("Apply"));
    await waitFor(() => {
      expect(executeDesktopControl).toHaveBeenCalledWith("test-session", {
        action: "locale",
        params: { locale: "fr_FR.UTF-8" },
      });
    });

    fireEvent.click(screen.getByText("Slow Connection"));
    fireEvent.change(screen.getByLabelText("Bandwidth limit"), { target: { value: "512" } });
    fireEvent.click(screen.getByText("Apply"));
    await waitFor(() => {
      expect(executeDesktopControl).toHaveBeenCalledWith("test-session", {
        action: "slow_connection",
        params: { enabled: true, bandwidth_kbps: 512 },
      });
    });
  });

  it("executes app, capture, network toggle, recording, and display resize actions", async () => {
    const { executeDesktopControl } = await import("../../lib/api/livedesktop");
    mockState.activeSession.appRunning = true;
    mockState.activeSession.recording = true;
    mockState.activeSession.networkMode = DesktopNetworkMode.SLOW;
    render(<DesktopControlsMenu />);
    fireEvent.click(screen.getByText("Controls"));

    fireEvent.click(screen.getByText("Quit App"));
    fireEvent.click(screen.getByText("Screenshot"));
    fireEvent.click(screen.getByText("Stop Recording"));
    fireEvent.click(screen.getByText("Offline Mode"));
    fireEvent.click(screen.getByText("Slow Connection"));
    fireEvent.click(screen.getByText("Resize Display..."));
    fireEvent.change(screen.getByLabelText("Desktop width"), { target: { value: "1600" } });
    fireEvent.change(screen.getByLabelText("Desktop height"), { target: { value: "1000" } });
    fireEvent.click(screen.getByText("Apply"));

    await waitFor(() => {
      expect(executeDesktopControl).toHaveBeenCalledWith("test-session", { action: "quit_app", params: undefined });
      expect(executeDesktopControl).toHaveBeenCalledWith("test-session", { action: "screenshot", params: undefined });
      expect(executeDesktopControl).toHaveBeenCalledWith("test-session", { action: "stop_recording", params: undefined });
      expect(executeDesktopControl).toHaveBeenCalledWith("test-session", { action: "offline_mode", params: { enabled: true } });
      expect(executeDesktopControl).toHaveBeenCalledWith("test-session", { action: "slow_connection", params: { enabled: false } });
      expect(executeDesktopControl).toHaveBeenCalledWith("test-session", { action: "resize_display", params: { width: 1600, height: 1000 } });
    });
    mockState.activeSession.appRunning = false;
    mockState.activeSession.recording = false;
    mockState.activeSession.networkMode = DesktopNetworkMode.NORMAL;
  });
});
