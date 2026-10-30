import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";

import { makeHealthResponse, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SettingsPage } from "./SettingsPage";
import { fetchOptionalCapabilities } from "../api/capabilities";
import { fetchHealth } from "../api/health";

const C = strings.pages.settings.capabilities;

vi.mock("../api/capabilities", () => ({ fetchOptionalCapabilities: vi.fn() }));
vi.mock("../api/health", () => ({ fetchHealth: vi.fn() }));

const linked = {
  state: "linked" as const,
  localPrincipal: "local-principal-1",
  businessAccountId: "business-account-7",
  scopes: ["device-studio:read"],
};

describe("SettingsPage desktop account", () => {
  beforeEach(() => {
    vi.mocked(fetchOptionalCapabilities).mockReturnValue(new Promise(() => {}));
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
  });

  afterEach(() => {
    cleanup();
    delete window.desktop;
  });

  it("calls out offline optional services without gating local work", async () => {
    vi.mocked(fetchOptionalCapabilities).mockResolvedValue([
      { id: "ai-gateway", name: "AI Gateway", status: "unavailable", message: "AI Gateway is unavailable from this app; local project work remains available", checkedAt: "2026-10-21T12:00:00Z", operatorCommand: "vrooli scenario start ai-gateway --json" },
      { id: "workspace-sandbox", name: "Workspace Sandbox", status: "available", message: "Workspace Sandbox responded to a local health check", checkedAt: "2026-10-21T12:00:00Z" },
    ]);

    renderWithProviders(<SettingsPage />);
    const section = screen.getByTestId("settings-optional-capabilities");
    await waitFor(() => expect(within(section).getByText("1 of 2 ready")).toBeInTheDocument());
    // The off service is called out with what still works; Device Studio cannot start it, so no button.
    expect(within(section).getByText(C.items.aiGateway.problem)).toBeInTheDocument();
    expect(within(section).queryByRole("button", { name: /start/i })).not.toBeInTheDocument();

    // Its detail view keeps the health message and the start command under technical details.
    fireEvent.click(within(section).getByRole("button", { name: new RegExp(C.items.aiGateway.name) }));
    expect(within(section).getByText("vrooli scenario start ai-gateway --json")).toBeInTheDocument();
    expect(within(section).getByText(/local project work remains available/)).toBeInTheDocument();
  });

  it("reports a failed check and checks again on request", async () => {
    vi.mocked(fetchOptionalCapabilities).mockRejectedValueOnce(new Error("down"));
    renderWithProviders(<SettingsPage />);
    const section = screen.getByTestId("settings-optional-capabilities");
    expect(await within(section).findByText(C.checkFailed)).toBeInTheDocument();

    vi.mocked(fetchOptionalCapabilities).mockResolvedValue([
      { id: "ai-gateway", name: "AI Gateway", status: "available" },
      { id: "workspace-sandbox", name: "Workspace Sandbox", status: "available" },
    ]);
    fireEvent.click(within(section).getByRole("button", { name: /Check again/ }));
    expect(await within(section).findByText("All 2 ready")).toBeInTheDocument();
  });

  it("keeps account linking and credit sharing unavailable in an ordinary browser", () => {
    renderWithProviders(<SettingsPage />);

    // A browser has no desktop owner: explain where the feature lives instead of showing dead controls.
    expect(screen.getByTestId(selectors.settingsPage.accountStatus)).toHaveTextContent("pages.settings.account.desktopOnly");
    expect(screen.getByText("pages.settings.account.desktopOnlyCredits")).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.settingsPage.accountConnect)).not.toBeInTheDocument();
    expect(screen.queryByRole("switch")).not.toBeInTheDocument();
    expect(screen.queryByTestId("settings-credit-balance")).not.toBeInTheDocument();
  });

  it("keeps account settings available when the health API returns an invalid timestamp", async () => {
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse({ timestamp: "not-a-date" }));

    renderWithProviders(<SettingsPage />);

    expect(await screen.findByTestId(selectors.settingsPage.accountStatus)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.health.card)).toBeInTheDocument();
  });

  it("shows the fresh LPBS balance with its source and preserves it as stale after a failed refresh", async () => {
    const getCreditBalance = vi.fn()
      .mockResolvedValueOnce({ state: "available" as const, balanceCredits: "875", displayCreditsLabel: "credits", displayCreditsMultiplier: 1, source: "LPBS" as const, fetchedAt: "2026-10-20T12:00:00.000Z", walletUpdatedAt: "2026-10-19T12:00:00.000Z" })
      .mockResolvedValueOnce({ state: "unavailable" as const });
    window.desktop = { auth: {
      getCreditBalance,
      getDesktopLinkStatus: vi.fn(async () => ({ state: "unlinked" as const, scopes: [] })),
      connectDesktop: vi.fn(async () => ({ state: "unlinked" as const, scopes: [] })),
      disconnectDesktop: vi.fn(async () => ({ localCleared: true, remoteRevocation: "confirmed" as const })),
    } };

    renderWithProviders(<SettingsPage />);
    await waitFor(() => expect(screen.getByTestId("settings-credit-balance")).toHaveTextContent("pages.settings.account.balanceValue"));
    expect(screen.getByTestId("settings-credit-balance-details")).toHaveTextContent("LPBS");
    fireEvent.click(screen.getByRole("button", { name: /pages\.settings\.account\.balanceRefresh/ }));
    await waitFor(() => expect(screen.getByText("pages.settings.account.balanceStale")).toBeInTheDocument());
    expect(screen.getByTestId("settings-credit-balance")).toHaveTextContent("pages.settings.account.balanceValue");
    expect(screen.getByTestId("settings-credit-balance-details")).toHaveTextContent("LPBS");
  });

  it("shares the LPBS refresh session only after an explicit desktop-owner action", async () => {
    const getCreditSessionShareStatus = vi.fn(async () => ({ state: "not_shared" as const }));
    const setCreditSessionSharing = vi.fn(async (enabled: boolean) => enabled
      ? { state: "shared" as const }
      : { state: "not_shared" as const, localCleared: true, remoteRevocation: "uncertain" as const });
    window.desktop = { auth: {
      getDesktopLinkStatus: vi.fn(async () => linked),
      connectDesktop: vi.fn(async () => linked),
      disconnectDesktop: vi.fn(async () => ({ localCleared: true, remoteRevocation: "uncertain" as const })),
      getCreditSessionShareStatus,
      setCreditSessionSharing,
    } };

    renderWithProviders(<SettingsPage />);
    const share = screen.getByRole("switch", { name: "pages.settings.account.creditStatusLabel" });
    await waitFor(() => expect(share).toBeEnabled());
    expect(share).not.toBeChecked();
    expect(setCreditSessionSharing).not.toHaveBeenCalled();
    fireEvent.click(share);
    await waitFor(() => expect(setCreditSessionSharing).toHaveBeenCalledWith(true));
    await waitFor(() => expect(share).toBeChecked());
    fireEvent.click(share);
    await waitFor(() => expect(setCreditSessionSharing).toHaveBeenCalledWith(false));
    expect(screen.getByText("pages.settings.account.creditRemoteUncertain")).toBeInTheDocument();
  });

  it("shows owner-confirmed account scopes and lets the user switch through browser PKCE", async () => {
    const connectDesktop = vi.fn(async () => ({ ...linked, businessAccountId: "business-account-8" }));
    const disconnectDesktop = vi.fn(async () => ({ localCleared: true, remoteRevocation: "confirmed" as const }));
    window.desktop = { auth: {
      getDesktopLinkStatus: vi.fn(async () => linked),
      connectDesktop,
      disconnectDesktop,
    } };

    renderWithProviders(<SettingsPage />);
    await waitFor(() => expect(screen.getByTestId(selectors.settingsPage.accountStatus)).toHaveTextContent("pages.settings.account.states.linked"));
    expect(screen.getByText("business-account-7")).toBeInTheDocument();
    expect(screen.getByText("device-studio:read")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId(selectors.settingsPage.accountConnect));
    await waitFor(() => expect(disconnectDesktop).toHaveBeenCalledWith({ resource: "device-studio" }));
    await waitFor(() => expect(connectDesktop).toHaveBeenCalledWith({
      resource: "device-studio",
      audience: "scenario:device-studio",
      scopes: ["device-studio:read"],
    }));
    await waitFor(() => expect(screen.getByText("business-account-8")).toBeInTheDocument());
  });

  it("reports uncertain remote revocation while keeping local clear and sign-out", async () => {
    const disconnectDesktop = vi.fn(async () => ({ localCleared: true as const, remoteRevocation: "uncertain" as const }));
    window.desktop = { auth: {
      getDesktopLinkStatus: vi.fn(async () => linked),
      connectDesktop: vi.fn(async () => linked),
      disconnectDesktop,
    } };

    renderWithProviders(<SettingsPage />);
    const button = await screen.findByTestId(selectors.settingsPage.accountDisconnect);
    fireEvent.click(button);

    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("pages.settings.account.remoteUncertain"));
    expect(disconnectDesktop).toHaveBeenCalledWith({ resource: "device-studio" });
  });
});
