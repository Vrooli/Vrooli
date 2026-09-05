import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Mode } from "@vrooli/proto-types/tunnel-manager/v1/config/config_pb";

import { renderWithProviders } from "../test-utils";
import {
  makeConfigMocks,
  makeConfigReadiness,
  makeConfigResponse,
  makeSyncResponse,
  makeTunnelConfig,
} from "../test-utils/mocks/config";

vi.mock("../api/config", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/config")>();
  return { ...actual, ...makeConfigMocks() };
});

import { SettingsPage } from "./SettingsPage";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";

describe("SettingsPage", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders local-mode readiness with missing Cloudflare fields", async () => {
    renderWithProviders(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.settingsPage.configPanel)).toBeInTheDocument();
      expect(screen.getByTestId(selectors.settingsPage.currentMode)).toHaveTextContent("Local");
    });
    expect(screen.getByTestId(selectors.settingsPage.remoteAvailable)).toHaveTextContent("Unavailable");
    expect(screen.getByTestId(selectors.settingsPage.missingFields)).toHaveTextContent("CLOUDFLARE_API_TOKEN");
    expect(screen.getByTestId(selectors.settingsPage.credentialPolicy)).toHaveTextContent("credential authority");
    expect(screen.getByTestId(selectors.settingsPage.credentialNextAction)).toHaveTextContent("Enter the missing fields");
    expect(screen.getByTestId(selectors.settingsPage.remoteModeButton)).toBeDisabled();
  });

  it("previews sync and shows the typed result message", async () => {
    const { configClient } = await import("../api/config");
    vi.mocked(configClient.sync).mockResolvedValueOnce(
      makeSyncResponse({ setupRequired: true, missingFields: ["CLOUDFLARE_ACCOUNT_ID"], message: "" }),
    );

    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    await user.click(await screen.findByTestId(selectors.settingsPage.syncDryRunButton));
    await waitFor(() => {
      expect(configClient.sync).toHaveBeenCalledWith({ dryRun: true });
    });
    expect(screen.getByTestId(selectors.settingsPage.syncResult)).toHaveTextContent("CLOUDFLARE_ACCOUNT_ID");
  });

  it("enables remote mode when remote credentials are available", async () => {
    const { configClient } = await import("../api/config");
    vi.mocked(configClient.getConfig).mockResolvedValueOnce(
      makeConfigResponse({
        config: makeTunnelConfig({ mode: Mode.LOCAL, accountId: "account", tunnelId: "tunnel" }),
        readiness: makeConfigReadiness({
          remoteAvailable: true,
          missingFields: [],
          credentialSource: "credential-authority",
          credentialRef: "vrooli/tunnel-manager:cloudflare-api-token",
          syncReady: true,
        }),
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    await user.click(await screen.findByTestId(selectors.settingsPage.remoteModeButton));
    await waitFor(() => {
      expect(configClient.switchMode).toHaveBeenCalledWith({ targetMode: Mode.REMOTE });
    });
  });

  it("shows authority-backed credential fields without an environment fallback", async () => {
    const { configClient } = await import("../api/config");
    vi.mocked(configClient.getConfig).mockResolvedValueOnce(
      makeConfigResponse({
        readiness: makeConfigReadiness({
          remoteAvailable: true,
          missingFields: [],
          credentialSource: "credential-authority",
          syncReady: true,
          credentialFields: [
            { name: "CLOUDFLARE_API_TOKEN", present: true, source: "credential-authority", writable: true },
            { name: "CLOUDFLARE_ACCOUNT_ID", present: true, source: "credential-authority", writable: true },
            { name: "CLOUDFLARE_TUNNEL_ID", present: true, source: "credential-authority", writable: true },
          ],
        }),
      }),
    );

    renderWithProviders(<SettingsPage />);

    expect(await screen.findByTestId(selectors.settingsPage.credentialFields)).toHaveTextContent("credential-authority");
    expect(screen.queryByTestId(selectors.settingsPage.credentialShadowWarning)).not.toBeInTheDocument();
  });

  it("saves write-only Cloudflare credentials and does not render the token", async () => {
    const { configClient } = await import("../api/config");
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    await user.type(await screen.findByTestId(selectors.settingsPage.accountIdInput), "acct-1");
    await user.type(screen.getByTestId(selectors.settingsPage.tunnelIdInput), "tun-1");
    await user.type(screen.getByTestId(selectors.settingsPage.apiTokenInput), "secret-token");
    await user.click(screen.getByTestId(selectors.settingsPage.credentialsSaveButton));

    await waitFor(() => {
      expect(configClient.setCloudflareCredentials).toHaveBeenCalledWith({
        accountId: "acct-1",
        tunnelId: "tun-1",
        apiToken: "secret-token",
      });
    });
    expect(screen.getByTestId(selectors.settingsPage.apiTokenInput)).toHaveValue("");
  });

  it("verifies remote capabilities and renders per-check scope remediation", async () => {
    const { configClient } = await import("../api/config");
    vi.mocked(configClient.verifyCredentials).mockResolvedValueOnce({
      ready: false,
      checks: [
        { name: "cloudflare.zone_dns_edit", state: 4, detail: "example.com", remediation: "Add Zone:DNS:Edit." },
        { name: "cloudflare.access_apps_edit", state: 1, detail: "access reachable", remediation: "" },
      ],
    } as never);

    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);
    await user.click(await screen.findByTestId(selectors.settingsPage.credentialVerificationButton));

    await waitFor(() => expect(configClient.verifyCredentials).toHaveBeenCalledWith({}));
    expect(screen.getByTestId(selectors.settingsPage.credentialVerificationResult)).toHaveTextContent("Insufficient scope");
    expect(screen.getByTestId(selectors.settingsPage.credentialVerificationResult)).toHaveTextContent("Add Zone:DNS:Edit.");
    expect(screen.getByTestId(selectors.settingsPage.credentialVerificationResult)).toHaveTextContent("Passed");
  });

  it("clears saved Cloudflare credentials", async () => {
    const { configClient } = await import("../api/config");
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    await user.click(await screen.findByTestId(selectors.settingsPage.credentialsClearButton));
    await waitFor(() => {
      expect(configClient.clearCloudflareCredentials).toHaveBeenCalledWith({ fields: ["all"] });
    });
  });

  it("explains that switching to remote no longer auto-pushes ingress", async () => {
    renderWithProviders(<SettingsPage />);
    expect(await screen.findByTestId(selectors.settingsPage.switchModeNote)).toHaveTextContent(
      "no longer auto-pushes ingress",
    );
    expect(screen.getByTestId(selectors.settingsPage.reviewDriftLink)).toHaveAttribute("href", "/drift");
  });

  it("toggles the global public-exposure switch and reflects the persisted state", async () => {
    const { configClient } = await import("../api/config");
    vi.mocked(configClient.getConfig).mockResolvedValueOnce(
      makeConfigResponse({ config: makeTunnelConfig({ publicExposureEnabled: false }) }),
    );

    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    const toggle = await screen.findByTestId(selectors.settingsPage.publicExposureToggle);
    expect(screen.getByTestId(selectors.settingsPage.publicExposureState)).toHaveTextContent("off");
    await user.click(toggle);

    await waitFor(() => {
      expect(configClient.setPublicExposure).toHaveBeenCalledWith({ enabled: true });
    });
    expect(screen.getByTestId(selectors.settingsPage.publicExposureStatusLink)).toHaveAttribute("href", "/drift");
  });

  it("reconciles additively via sync({}) and shows the result", async () => {
    const { configClient } = await import("../api/config");
    vi.mocked(configClient.sync).mockResolvedValueOnce(
      makeSyncResponse({ noChanges: false, message: "", added: ["a.itsagitime.com"], removed: [] }),
    );

    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    await user.click(await screen.findByTestId(selectors.settingsPage.reconcileNowButton));
    await waitFor(() => {
      expect(configClient.sync).toHaveBeenCalledWith({});
    });
    expect(screen.getByTestId(selectors.settingsPage.reconcileResult)).toHaveTextContent("added 1");
  });
});
