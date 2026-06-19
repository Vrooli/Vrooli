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
    expect(screen.getByTestId(selectors.settingsPage.remoteModeButton)).toBeDisabled();
  });

  it("previews sync and shows the typed result message", async () => {
    const { configClient } = await import("../api/config");
    vi.mocked(configClient.sync).mockResolvedValueOnce(
      makeSyncResponse({ setupRequired: true, missingFields: ["CLOUDFLARE_ACCOUNT_ID"], message: "" }) as never,
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
          credentialSource: "env:CLOUDFLARE_*",
          credentialRef: "env:CLOUDFLARE_API_TOKEN",
          syncReady: true,
        }),
      }) as never,
    );

    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    await user.click(await screen.findByTestId(selectors.settingsPage.remoteModeButton));
    await waitFor(() => {
      expect(configClient.switchMode).toHaveBeenCalledWith({ targetMode: Mode.REMOTE });
    });
  });
});
