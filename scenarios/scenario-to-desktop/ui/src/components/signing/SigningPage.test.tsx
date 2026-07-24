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
  checkSigningReadiness: checkSigningReadinessMock,
  deleteSigningConfig: deleteSigningConfigMock,
  discoverCertificates: discoverCertificatesMock,
  fetchScenarioDesktopStatus: fetchScenarioDesktopStatusMock,
  fetchSigningConfig: fetchSigningConfigMock,
  fetchSigningPrerequisites: fetchSigningPrerequisitesMock,
  generateLinuxSigningKey: generateLinuxSigningKeyMock,
  saveSigningConfig: saveSigningConfigMock,
  validateSigningConfig: validateSigningConfigMock,
}));

vi.mock("./WindowsSigningForm", () => ({ WindowsSigningForm: () => <div>Windows signing settings</div> }));
vi.mock("./MacOSSigningForm", () => ({ MacOSSigningForm: () => <div>macOS signing settings</div> }));
vi.mock("./LinuxSigningForm", () => ({
  LinuxSigningForm: ({ config, onChange }: { config?: { gpg_key_id?: string }; onChange: (value: { gpg_key_id: string }) => void }) => (
    <>
      <button type="button" onClick={() => onChange({ gpg_key_id: "ABC123" })}>Set Linux signing key</button>
      <div>Linux key: {config?.gpg_key_id || "not configured"}</div>
    </>
  ),
}));
vi.mock("./PrerequisitesPanel", () => ({ PrerequisitesPanel: () => <div>Signing tools status</div> }));

describe("SigningPage", () => {
  beforeEach(() => {
    fetchScenarioDesktopStatusMock.mockResolvedValue({
      scenarios: [{ name: "secrets-manager", display_name: "Secrets Manager", has_desktop: true }],
      stats: { total: 1, with_desktop: 1, built: 0, web_only: 0 },
    });
    fetchSigningConfigMock.mockResolvedValue({ config: null });
    checkSigningReadinessMock.mockResolvedValue({
      ready: false,
      platforms: {
        windows: { ready: false, reason: "Certificate required" },
        macos: { ready: false, reason: "Identity required" },
        linux: { ready: false, reason: "GPG key required" },
      },
      issues: ["Configure a signing identity before publishing."],
    });
    fetchSigningPrerequisitesMock.mockResolvedValue({ tools: [] });
    saveSigningConfigMock.mockResolvedValue({ config: { enabled: true } });
    validateSigningConfigMock.mockResolvedValue({ valid: true, errors: [], warnings: [] });
    deleteSigningConfigMock.mockResolvedValue({ deleted: true });
    discoverCertificatesMock.mockResolvedValue({ certificates: [] });
    generateLinuxSigningKeyMock.mockResolvedValue({ fingerprint: "ABC123", homedir: "/tmp/keys" });
  });

  it("asks for a scenario before showing configuration controls", async () => {
    render(<SigningPage />);

    expect(await screen.findByText("Select a scenario to configure code signing.")).toBeInTheDocument();
    expect(screen.queryByLabelText("Enable Signing")).not.toBeInTheDocument();
  });

  it("lets an operator enable, configure, and save signing for the selected scenario", async () => {
    const onScenarioChange = vi.fn();
    render(<SigningPage initialScenario="secrets-manager" onScenarioChange={onScenarioChange} />);

    const enableSigning = await screen.findByLabelText("Enable Signing");
    expect(await screen.findByText("Signing Readiness")).toBeInTheDocument();
    expect(await screen.findByText("GPG key required")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save Configuration" })).toBeDisabled();

    fireEvent.click(enableSigning);
    expect(screen.getByText("Windows signing settings")).toBeInTheDocument();
    expect(screen.getByText("macOS signing settings")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Set Linux signing key" }));
    expect(screen.getByText("Linux key: ABC123")).toBeInTheDocument();
    expect(screen.getByText("Unsaved changes")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Save Configuration" }));
    await waitFor(() => {
      expect(saveSigningConfigMock).toHaveBeenCalledTimes(1);
    });
    const savedConfig = saveSigningConfigMock.mock.calls[0]?.[1] as { enabled: boolean; linux?: { gpg_key_id?: string } };
    expect(saveSigningConfigMock.mock.calls[0]?.[0]).toBe("secrets-manager");
    expect(savedConfig.enabled).toBe(true);
    expect(savedConfig.linux?.gpg_key_id).toBe("ABC123");
    expect(onScenarioChange).not.toHaveBeenCalled();
  });
});
