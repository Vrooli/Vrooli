import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { WineInstallDialog } from "./WineInstallDialog";

const {
  checkWineStatusMock,
  startWineInstallMock,
  fetchWineInstallStatusMock,
} = vi.hoisted(() => ({
  checkWineStatusMock: vi.fn(),
  startWineInstallMock: vi.fn(),
  fetchWineInstallStatusMock: vi.fn(),
}));

vi.mock("../../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../lib/api")>();
  return {
    ...actual,
    checkWineStatus: checkWineStatusMock,
    startWineInstall: startWineInstallMock,
    fetchWineInstallStatus: fetchWineInstallStatusMock,
  };
});

describe("WineInstallDialog", () => {
  const onClose = vi.fn();
  const onInstallComplete = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    fetchWineInstallStatusMock.mockResolvedValue(null);
  });

  it("lets an operator close the dialog when Wine is already ready for Windows builds", async () => {
    checkWineStatusMock.mockResolvedValue({
      installed: true,
      version: "wine-9.0",
    });
    const user = userEvent.setup();

    renderWithProviders(
      <WineInstallDialog
        onClose={onClose}
        onInstallComplete={onInstallComplete}
      />,
    );

    expect(await screen.findByText("Wine is Installed")).toBeInTheDocument();
    expect(screen.getByText("wine-9.0")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Continue" }));

    expect(onClose).toHaveBeenCalledOnce();
    expect(startWineInstallMock).not.toHaveBeenCalled();
  });

  it("requires selecting an installer before starting a Windows-build prerequisite install", async () => {
    checkWineStatusMock.mockResolvedValue({
      installed: false,
      recommendedMethod: "appimage",
      installMethods: [
        {
          id: "appimage",
          name: "Wine AppImage",
          description: "Portable Wine for isolated desktop builds",
          requiresSudo: false,
          steps: ["Download https://example.test/wine.AppImage"],
          estimated: "2 minutes",
        },
      ],
    });
    startWineInstallMock.mockResolvedValue({ install_id: "wine-install-1" });
    const user = userEvent.setup();

    renderWithProviders(
      <WineInstallDialog
        onClose={onClose}
        onInstallComplete={onInstallComplete}
      />,
    );

    const continueButton = await screen.findByRole("button", {
      name: "Continue",
    });
    expect(continueButton).toBeDisabled();
    expect(screen.getByText("Recommended")).toBeInTheDocument();

    await user.click(screen.getByText("Wine AppImage"));
    await waitFor(() => { expect(continueButton).toBeEnabled(); });
    await user.click(continueButton);

    expect(startWineInstallMock).toHaveBeenCalledWith("appimage");
  });

  it("returns the operator to Windows builds after a completed Flatpak installation", async () => {
    checkWineStatusMock.mockResolvedValue({
      installed: false,
      installMethods: [
        {
          id: "flatpak",
          name: "Wine Flatpak",
          description: "Sandboxed Wine runtime",
          requiresSudo: false,
          steps: ["Install Flatpak"],
          estimated: "3 minutes",
        },
      ],
    });
    startWineInstallMock.mockResolvedValue({ install_id: "wine-install-2" });
    fetchWineInstallStatusMock.mockResolvedValue({
      installId: "wine-install-2",
      status: "completed",
      method: "flatpak",
      startedAt: "2026-07-27T10:00:00Z",
      log: ["Wine installed"],
      errorLog: [],
    });
    const user = userEvent.setup();

    renderWithProviders(
      <WineInstallDialog
        onClose={onClose}
        onInstallComplete={onInstallComplete}
      />,
    );

    await user.click(await screen.findByText("Wine Flatpak"));
    await user.click(screen.getByRole("button", { name: "Continue" }));

    expect(
      await screen.findByText("Installation Complete"),
    ).toBeInTheDocument();
    expect(screen.getByText("Wine installed")).toBeInTheDocument();
    await user.click(
      screen.getByRole("button", { name: "Continue with Windows Build" }),
    );

    expect(onInstallComplete).toHaveBeenCalledOnce();
  });
});
