import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { makeHealthResponse, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SettingsPage } from "./SettingsPage";
import { fetchOptionalCapabilities } from "../api/capabilities";
import { fetchHealth } from "../api/health";

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

  it("shows offline optional capabilities without gating local work", async () => {
    vi.mocked(fetchOptionalCapabilities).mockResolvedValue([
      { id: "ai-gateway", name: "AI Gateway", status: "unavailable", message: "AI Gateway is unavailable from this app; local project work remains available", checkedAt: "2026-10-21T12:00:00Z" },
      { id: "workspace-sandbox", name: "Workspace Sandbox", status: "unavailable", message: "Workspace Sandbox is unavailable from this app; local project work remains available", checkedAt: "2026-10-21T12:00:00Z" },
    ]);

    renderWithProviders(<SettingsPage />);
    await waitFor(() => expect(screen.getByTestId("settings-capability-ai-gateway")).toHaveAttribute("data-status", "unavailable"));
    expect(screen.getByTestId("settings-capability-workspace-sandbox")).toHaveAttribute("data-status", "unavailable");
    expect(screen.getByTestId("settings-capability-ai-gateway")).toHaveTextContent("local project work remains available");
    expect(screen.getByTestId(selectors.settingsPage.accountConnect)).toBeDisabled();
  });

  it("reports observed optional capabilities when local health checks succeed", async () => {
    vi.mocked(fetchOptionalCapabilities).mockResolvedValue([
      { id: "ai-gateway", name: "AI Gateway", status: "available", message: "AI Gateway responded to a local health check", checkedAt: "2026-10-21T12:00:00Z", actionLabel: "Start AI Gateway", operatorCommand: "vrooli scenario start ai-gateway --json" },
      { id: "workspace-sandbox", name: "Workspace Sandbox", status: "available", message: "Workspace Sandbox responded to a local health check", checkedAt: "2026-10-21T12:00:00Z" },
    ]);

    renderWithProviders(<SettingsPage />);
    await waitFor(() => expect(screen.getByTestId("settings-capability-ai-gateway")).toHaveAttribute("data-status", "available"));
    expect(screen.getByTestId("settings-capability-workspace-sandbox")).toHaveAttribute("data-status", "available");
    expect(screen.getByTestId("settings-capability-ai-gateway")).toHaveTextContent("responded to a local health check");
    expect(screen.getByTestId("settings-capability-ai-gateway")).toHaveTextContent(strings.pages.settings.capabilities.source);
    expect(screen.getByTestId("settings-capability-ai-gateway")).toHaveTextContent("vrooli scenario start ai-gateway --json");
  });

  it("keeps account linking and credit sharing unavailable in an ordinary browser", () => {
    renderWithProviders(<SettingsPage />);

    expect(screen.getByTestId(selectors.settingsPage.accountStatus)).toHaveTextContent("pages.settings.account.states.unavailable");
    expect(screen.getByText("pages.settings.account.creditStates.unavailable")).toBeInTheDocument();
    expect(screen.getByTestId(selectors.settingsPage.accountConnect)).toBeDisabled();
    expect(screen.getByTestId("settings-credit-balance")).toHaveTextContent("pages.settings.account.balanceUnavailable");
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
    await waitFor(() => expect(screen.getByTestId("settings-credit-balance")).toHaveTextContent("875 credits"));
    expect(screen.getByTestId("settings-credit-balance")).toHaveTextContent("LPBS");
    fireEvent.click(screen.getByRole("button", { name: "pages.settings.account.balanceRefresh" }));
    await waitFor(() => expect(screen.getByText("pages.settings.account.balanceStale")).toBeInTheDocument());
    expect(screen.getByTestId("settings-credit-balance")).toHaveTextContent("875 credits");
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
    await waitFor(() => expect(screen.getByText("pages.settings.account.creditStates.not_shared")).toBeInTheDocument());
    fireEvent.click(screen.getByText("pages.settings.account.creditShare"));
    await waitFor(() => expect(setCreditSessionSharing).toHaveBeenCalledWith(true));
    expect(screen.getByText("pages.settings.account.creditStates.shared")).toBeInTheDocument();
    fireEvent.click(screen.getByText("pages.settings.account.creditStop"));
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
