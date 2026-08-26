import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { i18n } from "../i18n";
import WorkspaceSection from "../components/settings/WorkspaceSection";

const mockStoreState: Record<string, unknown> = {
  isMinimapVisible: true,
  setMinimapVisible: vi.fn(),
  displayMode: "grid",
  setDisplayMode: vi.fn(),
  toolbarLayout: "expanded",
  setToolbarLayout: vi.fn(),
  keepScreenAwake: true,
  setKeepScreenAwake: vi.fn(),
  adaptiveChrome: true,
  setAdaptiveChrome: vi.fn(),
  touchScrollSensitivity: 1,
  wheelScrollSensitivity: 1,
  setTouchScrollSensitivity: vi.fn(),
  setWheelScrollSensitivity: vi.fn(),
  tmuxMouseMode: false,
  setTmuxMouseMode: vi.fn(),
  predictionLatencyThresholdMs: 20,
  setPredictionLatencyThresholdMs: vi.fn(),
  resetScrollSensitivities: vi.fn(),
};

let mockWakeLockStatus = "active";

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) => selector(mockStoreState),
}));

vi.mock("../stores/useWakeLockStatus", () => ({
  useWakeLockStatus: (selector: (state: { status: string }) => unknown) =>
    selector({ status: mockWakeLockStatus }),
}));

describe("WorkspaceSection", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    mockStoreState.keepScreenAwake = true;
    mockWakeLockStatus = "active";
    await i18n.changeLanguage("en");
  });

  it("renders keep-screen-awake toggle checked when enabled", () => {
    render(<WorkspaceSection />);
    const toggle = screen.getByTestId("keep-screen-awake-toggle");
    expect(toggle).toHaveAttribute("aria-checked", "true");
  });

  it("renders keep-screen-awake toggle unchecked when disabled", () => {
    mockStoreState.keepScreenAwake = false;
    render(<WorkspaceSection />);
    const toggle = screen.getByTestId("keep-screen-awake-toggle");
    expect(toggle).toHaveAttribute("aria-checked", "false");
  });

  it("calls setKeepScreenAwake when toggle is clicked", () => {
    render(<WorkspaceSection />);
    const toggle = screen.getByTestId("keep-screen-awake-toggle");
    fireEvent.click(toggle);
    expect(mockStoreState.setKeepScreenAwake).toHaveBeenCalledWith(false);
  });

  it("selects sidebar display mode", () => {
    render(<WorkspaceSection />);
    fireEvent.click(screen.getByTestId("display-mode-sidebar"));
    expect(mockStoreState.setDisplayMode).toHaveBeenCalledWith("sidebar");
  });

  it("shows unsupported hint when wake lock API is not available", () => {
    mockWakeLockStatus = "unsupported";
    render(<WorkspaceSection />);
    expect(screen.getByText(/doesn't support screen wake lock/i)).toBeTruthy();
  });

  it("shows denied hint when wake lock is denied", () => {
    mockWakeLockStatus = "denied";
    render(<WorkspaceSection />);
    expect(screen.getByText(/was denied/i)).toBeTruthy();
  });

  it("shows active hint with accent color when status is active", () => {
    mockWakeLockStatus = "active";
    render(<WorkspaceSection />);
    const hint = screen.getByText(/Screen is being kept awake/i);
    expect(hint).toBeTruthy();
    expect(hint.className).toContain("text-wc-accent");
  });

  it("shows default hint when toggle is off regardless of status", () => {
    mockStoreState.keepScreenAwake = false;
    mockWakeLockStatus = "unsupported";
    render(<WorkspaceSection />);
    expect(screen.getByText(/Prevent the device from dimming/i)).toBeTruthy();
  });

  it("shows denied hint with error color", () => {
    mockWakeLockStatus = "denied";
    render(<WorkspaceSection />);
    const hint = screen.getByText(/was denied/i);
    expect(hint.className).toContain("text-wc-error");
  });

  it("shows re-acquiring hint with amber color when released", () => {
    mockWakeLockStatus = "released";
    render(<WorkspaceSection />);
    const hint = screen.getByText(/Re-acquiring/i);
    expect(hint).toBeTruthy();
    expect(hint.className).toContain("text-yellow-500");
  });

  it("wires all workspace display, toolbar, sensitivity, and device controls", () => {
    render(<WorkspaceSection />);
    fireEvent.click(screen.getByTestId("display-mode-grid"));
    fireEvent.click(screen.getByTestId("display-mode-tabs"));
    fireEvent.click(screen.getByTestId("display-mode-sidebar"));
    fireEvent.click(screen.getByTestId("toolbar-layout-compact"));
    fireEvent.click(screen.getByTestId("toolbar-layout-expanded"));
    fireEvent.click(screen.getByTestId("minimap-toggle"));
    fireEvent.click(screen.getByTestId("adaptive-chrome-toggle"));
    fireEvent.click(screen.getByTestId("keep-screen-awake-toggle"));
    fireEvent.change(screen.getByLabelText("Touch scroll sensitivity"), { target: { value: "1.5" } });
    fireEvent.change(screen.getByLabelText("Wheel scroll sensitivity"), { target: { value: "2.0" } });
    const deviceInput = screen.getByRole("textbox");
    fireEvent.change(deviceInput, { target: { value: "phone" } });

    expect(mockStoreState.setDisplayMode).toHaveBeenCalledWith("sidebar");
    expect(mockStoreState.setToolbarLayout).toHaveBeenCalledWith("expanded");
    expect(mockStoreState.setMinimapVisible).toHaveBeenCalledWith(false);
    expect(mockStoreState.setAdaptiveChrome).toHaveBeenCalledWith(false);
    expect(mockStoreState.setTouchScrollSensitivity).toHaveBeenCalledWith(1.5);
    expect(mockStoreState.setWheelScrollSensitivity).toHaveBeenCalledWith(2);
  });

  it("exposes the off-by-default tmux mouse choice and reset control", () => {
    render(<WorkspaceSection />);
    expect(screen.getByTestId("tmux-mouse-mode-default-toggle")).toHaveAttribute("aria-checked", "false");
    fireEvent.click(screen.getByTestId("tmux-mouse-mode-default-toggle"));
    fireEvent.click(screen.getByRole("button", { name: "Reset scroll sensitivities" }));
    expect(mockStoreState.setTmuxMouseMode).toHaveBeenCalledWith(true);
    expect(mockStoreState.resetScrollSensitivities).toHaveBeenCalled();
  });
});
