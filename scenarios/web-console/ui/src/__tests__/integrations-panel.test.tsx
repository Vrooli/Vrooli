import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { renderWithProviders } from "@vrooli/api-base/testing";
import IntegrationsPanel from "../components/IntegrationsPanel";
import { i18n } from "../i18n";
import type { CapabilitiesResponse, CapabilityState } from "../api/capabilities";
import type { ManagedConnection } from "../api/monetization";

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:4200/api/v1",
  buildApiUrl: (path: string) => `http://localhost:4200/api/v1${path}`,
  buildWsUrl: (path: string) => `ws://localhost:4200/api/v1${path}`,
  createScenarioConnectTransport: () => ({}),
}));

const mocks = vi.hoisted(() => ({
  fetchCapabilities: vi.fn(),
  runCapabilityAction: vi.fn(),
  getConnections: vi.fn(),
  getCommercialContext: vi.fn(),
  testOpenRouterKey: vi.fn(),
  removeOpenRouterKey: vi.fn(),
  fetchScenarioLink: vi.fn(),
  getWebAccessToken: vi.fn(),
}));

vi.mock("../api/capabilities", () => ({
  fetchCapabilities: mocks.fetchCapabilities,
  runCapabilityAction: mocks.runCapabilityAction,
}));

vi.mock("../api/monetization", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/monetization")>();
  return {
    ...actual,
    getConnections: mocks.getConnections,
    getCommercialContext: mocks.getCommercialContext,
    testOpenRouterKey: mocks.testOpenRouterKey,
    removeOpenRouterKey: mocks.removeOpenRouterKey,
  };
});

vi.mock("../api/notifications", () => ({ fetchScenarioLink: mocks.fetchScenarioLink }));
vi.mock("../lib/auth", () => ({ getWebAccessToken: mocks.getWebAccessToken }));

function cap(overrides: Partial<CapabilityState> & Pick<CapabilityState, "id">): CapabilityState {
  return {
    name: overrides.id,
    description: "",
    dependencyKind: "resource",
    dependencySlug: overrides.id,
    features: [],
    status: "available",
    ...overrides,
  };
}

const audioTools = cap({
  id: "audio-tools",
  name: "audio-tools",
  description: "All audio capabilities flow through audio-tools via audioports.Remote*.",
  dependencyKind: "scenario",
  features: ["voice-input", "voice-streaming"],
  message: "scenario is healthy",
  // The catalogue always carries the remedy; it must not be offered while healthy.
  actionKind: "scenario_start",
  actionLabel: "Start Audio Tools",
  operatorCommand: "vrooli scenario start audio-tools --json",
  checkedAt: "2026-03-17T00:00:00Z",
});
const bridgeOff = cap({
  id: "vrooli-bridge",
  name: "Remote Terminals",
  dependencyKind: "scenario",
  status: "unavailable",
  message: "Bridge URL is not configured",
  reasonCode: "bridge_url_missing",
  actionKind: "scenario_start",
  actionLabel: "Start Bridge",
  operatorCommand: "vrooli scenario start vrooli-bridge --json",
});
const openrouterOff = cap({
  id: "openrouter",
  status: "unavailable",
  message: "OPENROUTER_API_KEY not configured",
  actionKind: "owner_guidance",
  actionLabel: "Review dependency health",
});
const claude = cap({ id: "claude", name: "Claude Code", message: "2.1.312 (Claude Code)", actionKind: "operator_command", actionLabel: "Install Claude Code" });
const grokMissing = cap({ id: "grok", name: "Grok", status: "unavailable", message: "grok not found", actionKind: "operator_command", actionLabel: "Install Grok" });

const catalogue: CapabilitiesResponse = {
  capabilities: [
    audioTools,
    bridgeOff,
    cap({ id: "ollama", message: "Ollama is running" }),
    openrouterOff,
    cap({ id: "session-backend-standard", message: "available" }),
    cap({ id: "session-backend-persistent", message: "available" }),
    claude,
    grokMissing,
  ],
  timestamp: "2026-03-17T00:00:00Z",
};

const openrouterAccount: ManagedConnection = {
  id: "conn-1",
  provider: "openrouter",
  connection_name: "OpenRouter",
  account_label: "API key",
  status: "connected",
  scopes: ["models"],
  supported_actions: ["test", "delete"],
};

const summary = () => document.querySelector("[data-rcl-integrations-summary-title]")?.textContent;

function groupNamed(label: string): HTMLElement {
  const heading = screen.getByText(label, { selector: "[data-rcl-settings-group-label]" });
  const group = heading.closest("[data-rcl-settings-group]");
  if (!(group instanceof HTMLElement)) throw new Error(`no group ${label}`);
  return group;
}

beforeEach(async () => {
  await i18n.changeLanguage("en");
  for (const mock of Object.values(mocks)) mock.mockReset();
  mocks.getConnections.mockResolvedValue({ connections: [] });
  mocks.fetchScenarioLink.mockResolvedValue("");
  mocks.getWebAccessToken.mockReturnValue(null);
});

describe("IntegrationsPanel", () => {
  it("does not fetch while the tab is closed", () => {
    renderWithProviders(<IntegrationsPanel open={false} />);
    expect(mocks.fetchCapabilities).not.toHaveBeenCalled();
    expect(mocks.getConnections).not.toHaveBeenCalled();
  });

  it("shows the first check as in progress", () => {
    mocks.fetchCapabilities.mockReturnValue(new Promise(() => undefined));
    renderWithProviders(<IntegrationsPanel open />);
    expect(summary()).toBe("Checking…");
  });

  it("summarises what is ready and groups everything by what it is for", async () => {
    mocks.fetchCapabilities.mockResolvedValue(catalogue);
    renderWithProviders(<IntegrationsPanel open />);

    await waitFor(() => { expect(summary()).toBe("5 of 8 ready"); });
    expect(within(groupNamed("Powered by other apps")).getByText("Voice")).toBeInTheDocument();
    expect(within(groupNamed("Powered by other apps")).getByText("Remote terminals")).toBeInTheDocument();
    expect(within(groupNamed("AI models")).getByText("Ollama")).toBeInTheDocument();
    expect(within(groupNamed("Terminal sessions")).getByText("Persistent")).toBeInTheDocument();
    expect(within(groupNamed("Coding agents")).getByText("Claude Code")).toBeInTheDocument();
    expect(within(groupNamed("Accounts")).getByText("No accounts connected yet.")).toBeInTheDocument();
  });

  it("describes services in plain words instead of their catalogue text", async () => {
    mocks.fetchCapabilities.mockResolvedValue(catalogue);
    renderWithProviders(<IntegrationsPanel open />);

    expect(await screen.findByText("Dictation, read-aloud and wake word")).toBeInTheDocument();
    expect(screen.queryByText(/audioports\.Remote/)).not.toBeInTheDocument();
    expect(screen.queryByText("scenario is healthy")).not.toBeInTheDocument();
  });

  it("shows an installed agent's version", async () => {
    mocks.fetchCapabilities.mockResolvedValue(catalogue);
    renderWithProviders(<IntegrationsPanel open />);
    expect(await screen.findByText("Version 2.1.312")).toBeInTheDocument();
  });

  it("lists each broken service with its fix, and offers no fix it cannot run", async () => {
    mocks.fetchCapabilities.mockResolvedValue(catalogue);
    renderWithProviders(<IntegrationsPanel open />);

    expect(await screen.findByText("Remote terminals are off")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Start Bridge" })).toBeInTheDocument();
    expect(screen.getByText("OpenRouter isn't set up")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Review dependency health" })).not.toBeInTheDocument();
  });

  it("does not offer to start a service that is already running", async () => {
    mocks.fetchCapabilities.mockResolvedValue(catalogue);
    renderWithProviders(<IntegrationsPanel open />);
    await screen.findByText("Voice");
    expect(screen.queryByRole("button", { name: "Start Audio Tools" })).not.toBeInTheDocument();
  });

  it("starts a stopped service and shows the result", async () => {
    const running = { ...catalogue, capabilities: catalogue.capabilities.map((c) => (c.id === "vrooli-bridge" ? { ...c, status: "available" as const, message: "Bridge is ready" } : c)), timestamp: "2026-03-17T00:01:00Z" };
    mocks.fetchCapabilities.mockResolvedValueOnce(catalogue).mockResolvedValue(running);
    mocks.runCapabilityAction.mockResolvedValue({
      success: true,
      status: "healthy",
      message: "",
      capabilityId: "vrooli-bridge",
      actionKind: "scenario_start",
      capabilities: running.capabilities,
      timestamp: running.timestamp,
    });
    renderWithProviders(<IntegrationsPanel open />);

    fireEvent.click(await screen.findByRole("button", { name: "Start Bridge" }));

    await waitFor(() => { expect(mocks.runCapabilityAction).toHaveBeenCalledWith("vrooli-bridge", "scenario_start"); });
    await waitFor(() => { expect(screen.queryByText("Remote terminals are off")).not.toBeInTheDocument(); });
    expect(within(groupNamed("Powered by other apps")).getByText("Done.")).toBeInTheDocument();
  });

  it("explains a failed start", async () => {
    mocks.fetchCapabilities.mockResolvedValue(catalogue);
    mocks.runCapabilityAction.mockRejectedValue(new Error("wait timed out"));
    renderWithProviders(<IntegrationsPanel open />);

    fireEvent.click(await screen.findByRole("button", { name: "Start Bridge" }));
    expect(await screen.findByText("That didn't work: wait timed out")).toBeInTheDocument();
  });

  it("explains an unsuccessful result the service reports", async () => {
    mocks.fetchCapabilities.mockResolvedValue(catalogue);
    mocks.runCapabilityAction.mockResolvedValue({
      success: false,
      status: "failed",
      message: "scenario did not become healthy",
      capabilityId: "vrooli-bridge",
      actionKind: "scenario_start",
      capabilities: catalogue.capabilities,
      timestamp: catalogue.timestamp,
    });
    renderWithProviders(<IntegrationsPanel open />);

    fireEvent.click(await screen.findByRole("button", { name: "Start Bridge" }));
    expect(await screen.findByText("That didn't work: scenario did not become healthy")).toBeInTheDocument();
  });

  it("installs a coding agent that is missing", async () => {
    mocks.fetchCapabilities.mockResolvedValue(catalogue);
    mocks.runCapabilityAction.mockResolvedValue({
      success: true,
      status: "installed",
      message: "",
      capabilityId: "grok",
      actionKind: "operator_command",
      capabilities: catalogue.capabilities,
      timestamp: catalogue.timestamp,
    });
    renderWithProviders(<IntegrationsPanel open />);

    const agents = await waitFor(() => groupNamed("Coding agents"));
    expect(within(agents).getByText("Not installed")).toBeInTheDocument();
    fireEvent.click(within(agents).getByRole("button", { name: "Install" }));
    await waitFor(() => { expect(mocks.runCapabilityAction).toHaveBeenCalledWith("grok", "operator_command"); });
  });

  it("opens a service's detail view with what it adds and its technical details", async () => {
    mocks.fetchCapabilities.mockResolvedValue({
      ...catalogue,
      capabilities: [{ ...audioTools, featureStatus: { "voice-input": "available", "voice-streaming": "unavailable" } }],
    });
    mocks.fetchScenarioLink.mockResolvedValue("https://audio-tools.example.test");
    renderWithProviders(<IntegrationsPanel open header={<h2>Integrations</h2>} />);

    fireEvent.click(await screen.findByRole("button", { name: "Voice" }));

    const detail = within(document.querySelector<HTMLElement>("[data-rcl-integrations-view='detail']") ?? document.body);
    expect(detail.getByRole("heading", { name: "Voice" })).toHaveFocus();
    expect(detail.getByText("Provided by Audio Tools")).toBeInTheDocument();
    expect(detail.getByText("Some Voice features are unavailable")).toBeInTheDocument();
    expect(detail.getByText("Dictate into the composer")).toBeInTheDocument();
    expect(detail.getByText("Unavailable")).toBeInTheDocument();
    expect(detail.getByText("Technical details")).toBeInTheDocument();
    expect(detail.getByText("audio-tools")).toBeInTheDocument();
    expect(await screen.findByTestId("integration-open-audio-tools")).toHaveAttribute("href", "https://audio-tools.example.test");
    expect(screen.getByRole("heading", { name: "Integrations", hidden: true })).not.toBeVisible();

    fireEvent.click(detail.getByRole("button", { name: "Integrations" }));
    expect(screen.getByRole("button", { name: "Voice" })).toHaveFocus();
  });

  it("counts a service with some features down as limited, not broken", async () => {
    mocks.fetchCapabilities.mockResolvedValue({
      ...catalogue,
      capabilities: [{ ...audioTools, featureStatus: { "voice-input": "available", "voice-streaming": "unavailable" } }],
    });
    renderWithProviders(<IntegrationsPanel open />);

    await waitFor(() => { expect(summary()).toBe("0 of 1 ready"); });
    expect(screen.getByText("Limited")).toBeInTheDocument();
    // Naming what is missing, rather than pointing at a card that cannot be opened.
    expect(screen.getAllByText("Unavailable right now: Live transcription while you talk.")[0]).toBeInTheDocument();
    // Audio Tools is running; starting it again would not bring the features back.
    expect(screen.queryByRole("button", { name: "Start Audio Tools" })).not.toBeInTheDocument();
  });

  it("shows a connected account and its controls on its detail view", async () => {
    mocks.fetchCapabilities.mockResolvedValue(catalogue);
    mocks.getConnections.mockResolvedValue({ connections: [openrouterAccount] });
    renderWithProviders(<IntegrationsPanel open />);

    const accounts = await waitFor(() => {
      const group = groupNamed("Accounts");
      within(group).getByText("Connected");
      return group;
    });
    fireEvent.click(within(accounts).getByRole("button", { name: "OpenRouter" }));
    expect(screen.getByRole("button", { name: "Test connection" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument();
    expect(screen.getByText("models")).toBeInTheDocument();
  });

  async function openAccount() {
    mocks.fetchCapabilities.mockResolvedValue(catalogue);
    mocks.getConnections.mockResolvedValue({ connections: [openrouterAccount] });
    renderWithProviders(<IntegrationsPanel open />);
    const accounts = await waitFor(() => {
      const group = groupNamed("Accounts");
      within(group).getByText("Connected");
      return group;
    });
    fireEvent.click(within(accounts).getByRole("button", { name: "OpenRouter" }));
  }

  it("tests an account's key and reports the result", async () => {
    mocks.testOpenRouterKey.mockResolvedValueOnce({ valid: true, source: "vault", checked_at: "now" }).mockResolvedValueOnce({ valid: false, source: "vault", checked_at: "now" }).mockRejectedValueOnce(new Error("down"));
    await openAccount();

    fireEvent.click(screen.getByRole("button", { name: "Test connection" }));
    expect(await screen.findByText("Connection verified.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Test connection" }));
    expect(await screen.findByText("Connection test failed.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Test connection" }));
    expect(await screen.findByText("Connection test unavailable.")).toBeInTheDocument();
  });

  it("removes an account's key only after confirmation", async () => {
    const confirm = vi.spyOn(window, "confirm").mockReturnValueOnce(false).mockReturnValueOnce(true).mockReturnValueOnce(true);
    mocks.removeOpenRouterKey.mockResolvedValueOnce(undefined).mockRejectedValueOnce(new Error("denied"));
    await openAccount();

    fireEvent.click(screen.getByRole("button", { name: "Remove" }));
    expect(mocks.removeOpenRouterKey).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Remove" }));
    expect(await screen.findByText("Connection removed.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Remove" }));
    expect(await screen.findByText("Connection could not be removed.")).toBeInTheDocument();
    confirm.mockRestore();
  });

  it("flags an account that needs attention with its next step", async () => {
    mocks.fetchCapabilities.mockResolvedValue({ ...catalogue, capabilities: [] });
    mocks.getConnections.mockResolvedValue({ connections: [{ ...openrouterAccount, status: "expired", next_action: "Paste a new key." }] });
    renderWithProviders(<IntegrationsPanel open />);

    expect(await screen.findByText("OpenRouter needs attention")).toBeInTheDocument();
    expect(screen.getByText("Paste a new key.")).toBeInTheDocument();
    expect(screen.getByText("Expired")).toBeInTheDocument();
  });

  it("keeps checking services when accounts cannot be loaded", async () => {
    mocks.fetchCapabilities.mockResolvedValue(catalogue);
    mocks.getConnections.mockRejectedValue(new Error("integration-hub down"));
    renderWithProviders(<IntegrationsPanel open />);

    expect(await screen.findByText("Accounts can't be loaded right now. Everything else here is still checked.")).toBeInTheDocument();
    expect(screen.getByText("Voice")).toBeInTheDocument();
  });

  it("reports a failed check and keeps accounts visible", async () => {
    mocks.fetchCapabilities.mockRejectedValue(new Error("boom"));
    mocks.getConnections.mockResolvedValue({ connections: [openrouterAccount] });
    renderWithProviders(<IntegrationsPanel open />);

    await waitFor(() => { expect(summary()).toBe("Couldn't check what's connected"); });
    expect(screen.getByText("Failed to load integrations: boom")).toBeInTheDocument();
    expect(await screen.findByText("OpenRouter")).toBeInTheDocument();
  });

  it("checks again on request", async () => {
    mocks.fetchCapabilities.mockResolvedValue(catalogue);
    renderWithProviders(<IntegrationsPanel open />);
    const again = await screen.findByRole("button", { name: "Check again" });
    const before = mocks.fetchCapabilities.mock.calls.length;
    fireEvent.click(again);
    await waitFor(() => { expect(mocks.fetchCapabilities.mock.calls.length).toBeGreaterThan(before); });
  });

  it("puts a capability it has no place for under Other, and shows it as checking until it is", async () => {
    mocks.fetchCapabilities.mockResolvedValue({ ...catalogue, capabilities: [cap({ id: "future-capability", name: "future-capability", status: "unknown" })] });
    renderWithProviders(<IntegrationsPanel open />);

    const other = await waitFor(() => groupNamed("Other"));
    expect(within(other).getByText("Future capability")).toBeInTheDocument();
    expect(within(other).getByText("Checking")).toBeInTheDocument();
  });

  it("shows an account recommendation and lets it be dismissed", async () => {
    mocks.getWebAccessToken.mockReturnValue("token");
    mocks.fetchCapabilities.mockResolvedValue(catalogue);
    mocks.getCommercialContext.mockResolvedValue({
      stale: false,
      content: [
        {
          content_id: "rec-1",
          placement: "integrations",
          title: "Use hosted models",
          description: "Add credits to use cloud models without a key.",
          priority: "normal",
          eligible: true,
          cta_label: "See plans",
          cta_destination: "/account",
          expires_at: "2999-01-01T00:00:00Z",
          dismissible: true,
        },
      ],
    });
    renderWithProviders(<IntegrationsPanel open />);

    expect(await screen.findByText("Use hosted models")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(screen.queryByText("Use hosted models")).not.toBeInTheDocument();
  });
});
