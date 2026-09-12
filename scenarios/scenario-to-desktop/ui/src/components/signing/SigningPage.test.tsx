import { fireEvent, render, screen, waitFor } from "@/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SigningPage } from "./SigningPage";

const {
  checkSigningReadinessMock,
  deleteSigningConfigMock,
  discoverCertificatesMock,
  fetchScenarioDesktopStatusMock,
  fetchSigningConfigMock,
  fetchSigningPrerequisitesMock,
  generateLinuxSigningKeyMock,
  saveSigningConfigMock,
  validateSigningConfigMock,
} = vi.hoisted(() => ({
  checkSigningReadinessMock: vi.fn(),
  deleteSigningConfigMock: vi.fn(),
  discoverCertificatesMock: vi.fn(),
  fetchScenarioDesktopStatusMock: vi.fn(),
  fetchSigningConfigMock: vi.fn(),
  fetchSigningPrerequisitesMock: vi.fn(),
  generateLinuxSigningKeyMock: vi.fn(),
  saveSigningConfigMock: vi.fn(),
  validateSigningConfigMock: vi.fn(),
}));

vi.mock("../../lib/api", () => ({
  fetchScenarioDesktopStatus: fetchScenarioDesktopStatusMock,
}));
vi.mock("../../lib/api/connect", () => ({
  signingConnectClient: {
    getSigningConfig: fetchSigningConfigMock,
    getSigningReadiness: checkSigningReadinessMock,
    listSigningPrerequisites: fetchSigningPrerequisitesMock,
    putSigningConfig: saveSigningConfigMock,
    validateSigningConfig: validateSigningConfigMock,
    deleteSigningConfig: deleteSigningConfigMock,
    discoverSigningCertificates: discoverCertificatesMock,
    generateLinuxSigningKey: generateLinuxSigningKeyMock,
  },
}));

vi.mock("./WindowsSigningForm", () => ({
  WindowsSigningForm: () => <div>Windows signing settings</div>,
}));
vi.mock("./MacOSSigningForm", () => ({
  MacOSSigningForm: () => <div>macOS signing settings</div>,
}));
vi.mock("./LinuxSigningForm", () => ({
  LinuxSigningForm: ({
    config,
    onChange,
  }: {
    config?: { gpg_key_id?: string };
    onChange: (value: { gpg_key_id: string }) => void;
  }) => (
    <>
      <button
        type="button"
        onClick={() => {
          onChange({ gpg_key_id: "ABC123" });
        }}
      >
        Set Linux signing key
      </button>
      <div>Linux key: {config?.gpg_key_id || "not configured"}</div>
    </>
  ),
}));
vi.mock("./PrerequisitesPanel", () => ({
  PrerequisitesPanel: () => <div>Signing tools status</div>,
}));

describe("SigningPage", () => {
  beforeEach(() => {
    fetchScenarioDesktopStatusMock.mockResolvedValue({
      scenarios: [
        {
          name: "secrets-manager",
          display_name: "Secrets Manager",
          has_desktop: true,
        },
      ],
      stats: { total: 1, with_desktop: 1, built: 0, web_only: 0 },
    });
    fetchSigningConfigMock.mockResolvedValue({ config: null });
    checkSigningReadinessMock.mockResolvedValue({
      ready: false,
      message: "Configure a signing identity before publishing.",
      platforms: [
        { platform: 1, ready: false, message: "Certificate required" },
        { platform: 2, ready: false, message: "Identity required" },
        { platform: 3, ready: false, message: "GPG key required" },
      ],
    });
    fetchSigningPrerequisitesMock.mockResolvedValue({ tools: [] });
    saveSigningConfigMock.mockResolvedValue({ config: { enabled: true } });
    validateSigningConfigMock.mockResolvedValue({
      valid: true,
      errors: [],
      warnings: [],
    });
    deleteSigningConfigMock.mockResolvedValue({ deleted: true });
    discoverCertificatesMock.mockResolvedValue({ certificates: [] });
    generateLinuxSigningKeyMock.mockResolvedValue({
      fingerprint: "ABC123",
      homedir: "/tmp/keys",
    });
  });

  it("asks for a scenario before showing configuration controls", async () => {
    render(<SigningPage />);

    expect(
      await screen.findByText("Select a scenario to configure code signing."),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText("Enable Signing")).not.toBeInTheDocument();
  });

  it("lets an operator enable, configure, and save signing for the selected scenario", async () => {
    const onScenarioChange = vi.fn();
    render(
      <SigningPage
        initialScenario="secrets-manager"
        onScenarioChange={onScenarioChange}
      />,
    );

    const enableSigning = await screen.findByLabelText("Enable Signing");
    expect(await screen.findByText("Signing Readiness")).toBeInTheDocument();
    expect(await screen.findByText("GPG key required")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Save Configuration" }),
    ).toBeDisabled();

    fireEvent.click(enableSigning);
    expect(screen.getByText("Windows signing settings")).toBeInTheDocument();
    expect(screen.getByText("macOS signing settings")).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "Set Linux signing key" }),
    );
    expect(screen.getByText("Linux key: ABC123")).toBeInTheDocument();
    expect(screen.getByText("Unsaved changes")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Save Configuration" }));
    await waitFor(() => {
      expect(saveSigningConfigMock).toHaveBeenCalledTimes(1);
    });
    const saveRequest = saveSigningConfigMock.mock.calls[0]?.[0] as unknown as {
      scenarioName: string;
      config: { enabled: boolean; linux?: { gpgKeyId?: string } };
    };
    const savedConfig = saveRequest.config;
    expect(saveRequest.scenarioName).toBe("secrets-manager");
    expect(savedConfig.enabled).toBe(true);
    expect(savedConfig.linux?.gpgKeyId).toBe("ABC123");
    expect(onScenarioChange).not.toHaveBeenCalled();
  });

  it("refreshes signing evidence and scans the selected certificate platform", async () => {
    render(<SigningPage initialScenario="secrets-manager" />);
    await screen.findByText("Signing Readiness");
    const initialConfigCalls = fetchSigningConfigMock.mock.calls.length;
    const initialReadinessCalls = checkSigningReadinessMock.mock.calls.length;
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    await waitFor(() => {
      expect(fetchSigningConfigMock.mock.calls.length).toBeGreaterThan(
        initialConfigCalls,
      );
      expect(checkSigningReadinessMock.mock.calls.length).toBeGreaterThan(
        initialReadinessCalls,
      );
    });
    fireEvent.change(
      screen.getByRole("combobox", { name: "Certificate discovery platform" }),
      {
        target: { value: "linux" },
      },
    );
    fireEvent.click(screen.getByRole("button", { name: "Scan" }));
    await waitFor(() => {
      expect(discoverCertificatesMock).toHaveBeenCalled();
    });
  });
});
