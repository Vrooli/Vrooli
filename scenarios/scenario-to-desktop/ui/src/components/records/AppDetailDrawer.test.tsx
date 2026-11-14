import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { useLiveDesktopStore } from "../../store/liveDesktopStore";
import { AppDetailDrawer } from "./AppDetailDrawer";

vi.mock("../captures/CapturesSection", () => ({
  CapturesSection: () => <div>Capture evidence</div>,
}));

vi.mock("./SigningBadge", () => ({
  SigningBadge: () => <span>Signing ready</span>,
}));

const item = {
  record: {
    id: "record-1",
    build_id: "build-1",
    scenario_name: "desktop-app",
    app_display_name: "Desktop App",
    template_type: "basic",
    framework: "electron",
    location_mode: "proper",
    output_path: "/tmp/desktop-app",
    destination_path: "/exports/desktop-app",
    deployment_mode: "bundled",
  },
  has_build: true,
  build_state: "completed",
  smoke_test_id: "smoke-1",
  screen_recording: {
    recorded: true,
    duration_ms: 2500,
    file_size_bytes: 1024,
  },
  build_status: {
    status: "completed",
    metadata: {
      version: "1.2.3",
      git_branch: "main",
      git_commit_hash: "abcdef123456",
      git_dirty: true,
    },
  },
};

describe("AppDetailDrawer", () => {
  const onClose = vi.fn();
  const onMove = vi.fn();
  const onDelete = vi.fn();
  const onSwitchTemplate = vi.fn();
  const onEditSigning = vi.fn();
  const onRebuildWithSigning = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    useLiveDesktopStore.setState({
      activeSession: null,
      connectionStatus: "disconnected",
      error: null,
      isOpen: false,
      scenarioName: null,
      appPath: null,
    });
    vi.spyOn(window, "confirm").mockReturnValue(true);
  });

  function renderDrawer() {
    return renderWithProviders(
      <AppDetailDrawer
        item={item}
        open
        onClose={onClose}
        onMove={onMove}
        onDelete={onDelete}
        movePending={false}
        onSwitchTemplate={onSwitchTemplate}
        onEditSigning={onEditSigning}
        onRebuildWithSigning={onRebuildWithSigning}
      />,
    );
  }

  it("shows build provenance and opens the local interactive desktop", async () => {
    const user = userEvent.setup();
    renderDrawer();

    expect(screen.getByText("Desktop App")).toBeInTheDocument();
    expect(
      screen.getByText("Built with uncommitted changes"),
    ).toBeInTheDocument();
    expect(screen.getByText("abcdef1")).toBeInTheDocument();
    expect(screen.getByText("Capture evidence")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(useLiveDesktopStore.getState().isOpen).toBe(true);
    expect(useLiveDesktopStore.getState().scenarioName).toBe("desktop-app");
  });

  it("moves to a chosen path and exposes signing and template recovery actions", async () => {
    const user = userEvent.setup();
    renderDrawer();

    await user.click(screen.getByRole("button", { name: "Choose" }));
    await user.type(
      screen.getByPlaceholderText("/path/to/destination"),
      "/exports/custom",
    );
    const moveButtons = screen.getAllByRole("button", { name: "Move" });
    const destinationMove = moveButtons[1];
    if (!destinationMove)
      throw new Error("Destination move action is not mounted");
    await user.click(destinationMove);
    await user.click(screen.getByRole("button", { name: "Configure" }));
    await user.click(screen.getByRole("button", { name: "Rebuild" }));
    await user.click(screen.getByRole("button", { name: "Change" }));

    expect(onMove).toHaveBeenCalledWith(
      "record-1",
      "custom",
      "/exports/custom",
    );
    expect(onEditSigning).toHaveBeenCalledWith("desktop-app");
    expect(onRebuildWithSigning).toHaveBeenCalledWith("desktop-app");
    expect(onSwitchTemplate).toHaveBeenCalledWith("desktop-app", "basic");
  });

  it("requires confirmation before deleting only the generated desktop wrapper", async () => {
    const user = userEvent.setup();
    renderDrawer();

    await user.click(screen.getByRole("button", { name: "Delete" }));

    expect(window.confirm).toHaveBeenCalledWith(
      'Delete desktop build for "desktop-app"? This removes platforms/electron for that scenario.',
    );
    expect(onDelete).toHaveBeenCalledWith("desktop-app");
  });
});
