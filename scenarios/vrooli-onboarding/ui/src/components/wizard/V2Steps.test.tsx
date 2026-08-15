import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { StepDerivedResources } from "./StepDerivedResources";
import { StepHostRequirements } from "./StepHostRequirements";
import { StepIntegrationsDeferred } from "./StepIntegrationsDeferred";
import { StepOperatingMode } from "./StepOperatingMode";
import { StepApply } from "./StepApply";
import { StepReadiness } from "./StepReadiness";
import { StepSelectScenarios } from "./StepSelectScenarios";

const api = vi.hoisted(() => ({
  fetchV2Scenarios: vi.fn(),
  fetchV2HostRequirements: vi.fn(),
  fetchV2Readiness: vi.fn(),
  fetchV2Closure: vi.fn(),
  fetchV2Resources: vi.fn(),
  provisionCredential: vi.fn(),
  applyOnboarding: vi.fn(),
  fetchV2ApplyPlan: vi.fn(),
  fetchV2ApplyStatus: vi.fn(),
  fetchOperatorInputs: vi.fn(),
  fetchCredentialStoreStatus: vi.fn(),
  selectCredentialBackend: vi.fn(),
  reselectCredentialBackend: vi.fn(),
  initializeCredentialStore: vi.fn(),
  unlockCredentialStore: vi.fn(),
  changeCredentialStorePassphrase: vi.fn(),
  rewrapCredentialStore: vi.fn(),
  resolveOperatorInputs: vi.fn(),
}));

vi.mock("../../lib/api", () => api);

const scenarios = {
  scenarios: [
    { name: "control-plane", system_required: true, enabled: true, auto_restart: true, resources: ["postgres"], description: "Required" },
    { name: "writer", system_required: false, enabled: false, auto_restart: false, resources: ["ollama"], description: "Optional" },
  ],
  count: 2,
};

beforeEach(() => {
  api.fetchV2Scenarios.mockResolvedValue(scenarios);
  api.fetchV2HostRequirements.mockResolvedValue({
    tools: [{ name: "git", required: true, reason: "source control", status: "required" }],
    safeguards: [{ name: "firewall", required: false, reason: "network safety", status: "optional", risk: "medium", config_schema: { type: "object", properties: { target: { type: "string", description: "collector target" } } } }],
  });
  api.fetchV2Readiness.mockResolvedValue({
    status: "degraded",
    scenarios: ["control-plane"],
    resources: ["postgres"],
    credentials: [{ resource: "openrouter", logical_id: "openrouter", field: "api_key", label: "OpenRouter key", required: true, status: "unconfigured" }],
    hosts: [{ name: "git", status: "ready", kind: "tool", required: true }],
    integrations: [{ name: "OAuth", status: "deferred", detail: "Owned by integration-hub" }],
    checked_at: "2026-07-29T00:00:00Z",
  });
  api.fetchV2Closure.mockResolvedValue({ resources: [{ name: "postgres", required: true, direct: true, provenance: [] }, { name: "ollama", required: false, direct: false, provenance: [] }], scenarios: [] });
  api.fetchV2Resources.mockResolvedValue({
    resources: [{ name: "postgres", category: "database", enabled: true, installed: true }, { name: "ollama", category: "ai", enabled: false, installed: true }],
    required: [{ name: "postgres", category: "database", enabled: true, installed: true }],
    optional: [{ name: "ollama", category: "ai", enabled: false, installed: true }],
    standalone: [{ name: "qdrant", category: "search", enabled: false, installed: true }], count: 3,
  });
  api.provisionCredential.mockResolvedValue({ status: "provisioned" });
  api.applyOnboarding.mockResolvedValue({ status: "applied", items: [{ name: "postgres", outcome: "applied" }] });
  api.fetchV2ApplyPlan.mockResolvedValue({ items: [{ id: "resource:postgres", kind: "resource", name: "postgres", required: true }] });
  api.fetchV2ApplyStatus.mockResolvedValue({ run_id: "apply-test", status: "applied", items: [{ name: "postgres", outcome: "applied" }] });
  api.fetchOperatorInputs.mockResolvedValue({ version: 1, updated_at: "now", requests: [] });
  api.fetchCredentialStoreStatus.mockResolvedValue({ initialized: true, active: true, entries: 1, active_wrap: "native-wrap", active_key_store: "keychain" });
  api.selectCredentialBackend.mockResolvedValue({ status: "selected", backend: "native" });
  api.initializeCredentialStore.mockResolvedValue({ initialized: true, active: true, entries: 0 });
  api.unlockCredentialStore.mockResolvedValue({ status: "unlocked", active_wrap: "encrypted-file" });
  api.changeCredentialStorePassphrase.mockResolvedValue({ status: "changed" });
  api.rewrapCredentialStore.mockResolvedValue({ status: "rewrapped", provider: "native", key_store: "keychain" });
  api.resolveOperatorInputs.mockResolvedValue({ status: "resolved", configuration_pending: false });
});

describe("V2 onboarding wizard steps", () => {
  it("derives resources and lets operators select optional scenarios", async () => {
    const onToggle = vi.fn();
    const onResourceToggle = vi.fn();
    renderWithProviders(<><StepSelectScenarios selected={new Set()} onToggle={onToggle} /><StepDerivedResources selected={new Set(["writer"])} operatorState={{ version: "1", updated_at: "now", resources: { ollama: { enabled: true } } }} onToggle={onResourceToggle} /></>);
    expect(await screen.findByTestId("scenario-card-writer")).toHaveAttribute("aria-pressed", "false");
    fireEvent.click(screen.getByTestId("scenario-card-writer"));
    expect(onToggle).toHaveBeenCalledWith("writer");
    expect(await screen.findByText("ollama")).toBeInTheDocument();
    expect(screen.getByText("postgres")).toBeInTheDocument();
    expect(screen.getByText("qdrant")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("checkbox", { name: /ollama/i }));
    expect(onResourceToggle).toHaveBeenCalledWith("ollama", false);
    fireEvent.click(screen.getByRole("checkbox", { name: /qdrant/i }));
    expect(onResourceToggle).toHaveBeenCalledWith("qdrant", true);
    expect(screen.getByRole("checkbox", { name: /postgres/i })).toBeDisabled();
    fireEvent.change(screen.getByTestId("scenario-search"), { target: { value: "does-not-exist" } });
    expect(screen.getByText("No scenarios match this filter. Clear it to see the full catalog.")).toBeInTheDocument();
    fireEvent.change(screen.getByTestId("scenario-search"), { target: { value: "" } });
    fireEvent.change(screen.getByTestId("scenario-filter"), { target: { value: "available" } });
    expect(screen.getByTestId("scenario-card-writer")).toBeInTheDocument();
    expect(screen.queryByTestId("scenario-card-control-plane")).not.toBeInTheDocument();
  });

  it("shows manifest-derived host requirements and sends opt-ins to the owner", async () => {
    const onTool = vi.fn();
    const onSafeguard = vi.fn();
    renderWithProviders(<StepHostRequirements onTool={onTool} onSafeguard={onSafeguard} onHostConfig={vi.fn()} />);
    expect(await screen.findByText("git")).toBeInTheDocument();
    const firewall = screen.getByRole("checkbox", { name: /firewall/i });
    fireEvent.click(firewall);
    expect(onSafeguard).toHaveBeenCalledWith("firewall", true);
    expect(onTool).not.toHaveBeenCalled();
  });

  it("renders generic manifest config fields and emits their values", async () => {
    const onConfig = vi.fn();
    renderWithProviders(<StepHostRequirements onTool={vi.fn()} onSafeguard={vi.fn()} onHostConfig={(kind, name, config) => onConfig(kind, name, config)} />);
    const field = await screen.findByLabelText("target");
    fireEvent.change(field, { target: { value: "collector.example:6666" } });
    expect(onConfig).toHaveBeenCalledWith("host_safeguards", "firewall", { target: "collector.example:6666" });
  });

  it("renders the deferred integration contract", () => {
    renderWithProviders(<StepIntegrationsDeferred />);
    expect(screen.getByRole("status")).toHaveTextContent("Integration setup is deferred");
    expect(screen.getByRole("link", { name: /read the integration contract/i })).toHaveAttribute("href", "/docs/configuration/integrations/connectors.md");
  });

  it("persists operating-mode choices through its owner callback", async () => {
    const onAutoRestart = vi.fn();
    renderWithProviders(<StepOperatingMode selected={new Set(["writer"])} onAutoRestart={onAutoRestart} />);
    const checkbox = await screen.findByRole("checkbox", { name: "Keep writer running" });
    fireEvent.click(checkbox);
    expect(onAutoRestart).toHaveBeenCalledWith("writer", true);
  });

  it("provisions a credential without exposing its value and renders validation groups", async () => {
    renderWithProviders(<StepReadiness title="Credentials" />);
    const input = await screen.findByLabelText("Value for OpenRouter key");
    fireEvent.change(input, { target: { value: "secret-value" } });
    fireEvent.click(screen.getByRole("button", { name: "Save securely" }));
    await waitFor(() => expect(api.provisionCredential).toHaveBeenCalledWith({ logical_id: "openrouter", field: "api_key", value: "secret-value" }));
    expect(input).toHaveValue("");

    renderWithProviders(<StepReadiness title="Validation" />);
    expect(await screen.findByText("Host requirements")).toBeInTheDocument();
    expect(screen.getByText("Integrations")).toBeInTheDocument();
    expect(screen.getByText("OAuth")).toBeInTheDocument();
  });

  it("surfaces provider failures and records deliberate degraded continuation", async () => {
    api.fetchV2Readiness.mockRejectedValueOnce(new Error("probe unavailable"));
    renderWithProviders(<StepReadiness title="Validation" />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Readiness could not be checked");

    api.fetchV2Readiness.mockResolvedValueOnce({
      status: "degraded", scenarios: [], resources: [], credentials: [], hosts: [], integrations: [], checked_at: "now",
    });
    renderWithProviders(<StepReadiness title="Validation" />);
    fireEvent.click(await screen.findByTestId("readiness-continue-degraded"));
    expect(await screen.findByText("Degraded continuation recorded for this session.")).toBeInTheDocument();
  });

  it("supports enum, boolean, and numeric host configuration fields", async () => {
    api.fetchV2HostRequirements.mockResolvedValueOnce({
      tools: [],
      safeguards: [{
        name: "firewall", required: false, reason: "network safety", status: "optional", risk: "medium",
        config_schema: { type: "object", properties: {
          mode: { type: "string", enum: ["audit", "enforce"] },
          enabled: { type: "boolean" },
          retries: { type: "integer" },
        } },
      }],
    });
    const onConfig = vi.fn();
    renderWithProviders(<StepHostRequirements onTool={vi.fn()} onSafeguard={vi.fn()} onHostConfig={(kind, name, config) => onConfig(kind, name, config)} />);
    fireEvent.change(await screen.findByLabelText("mode"), { target: { value: "enforce" } });
    fireEvent.click(screen.getByLabelText("enabled"));
    fireEvent.change(screen.getByLabelText("retries"), { target: { value: "3" } });
    expect(onConfig).toHaveBeenLastCalledWith("host_safeguards", "firewall", { mode: "enforce", enabled: true, retries: 3 });
  });

  it("renders provider guidance and the idempotent apply report", async () => {
    api.fetchV2Readiness.mockResolvedValueOnce({
      status: "ready",
      scenarios: [],
      resources: [],
      credentials: [{ resource: "openrouter", logical_id: "openrouter", field: "api_key", label: "OpenRouter key", description: "Used for hosted inference.", obtain_url: "https://example.test/key", required: false, status: "configured" }],
      hosts: [],
      integrations: [],
      checked_at: "2026-07-29T00:00:00Z",
      credential_diagnosis: { provider: { backend: "native", condition: "ready", explanation: "Available", fix: "None" } },
      recovery: { receipt_exists: false, entry_count: 0, uncovered: [] },
    });
    renderWithProviders(<StepReadiness title="Apply" />);
    expect(await screen.findByTestId("backend-diagnosis")).toHaveTextContent("Available");
    expect(screen.getByTestId("store-init-guidance")).toBeInTheDocument();
    expect(screen.getByTestId("credential-obtain-link")).toHaveAttribute("href", "https://example.test/key");
    fireEvent.click(screen.getByTestId("apply-confirm"));
    await waitFor(() => expect(api.applyOnboarding).toHaveBeenCalled());
    expect(await screen.findByTestId("apply-report")).toHaveTextContent("postgres");
    expect(await screen.findByTestId("apply-report")).toHaveTextContent("applied");
  });

  it("keeps the apply route as a distinct step identity", async () => {
    renderWithProviders(<StepApply />);
    expect(await screen.findByRole("heading", { name: "Apply" })).toBeInTheDocument();
  });

  it("waits for an async partial apply and exposes retry evidence", async () => {
    const initialApplyCalls = api.applyOnboarding.mock.calls.length;
    api.applyOnboarding.mockResolvedValueOnce({ run_id: "apply-pending", status: "pending", items: [] });
    api.fetchV2ApplyStatus.mockResolvedValueOnce({ run_id: "apply-pending", status: "partially_applied", items: [
      { name: "firewall", outcome: "failed", error: "permission denied" },
      { name: "writer", outcome: "blocked", error: "blocked by firewall" },
    ] });
    renderWithProviders(<StepReadiness title="Apply" />);
    fireEvent.click(await screen.findByTestId("apply-confirm"));
    expect(await screen.findByTestId("skipped-note")).toHaveTextContent("Some items were skipped or failed");
    fireEvent.click(screen.getByTestId("retry"));
    await waitFor(() => expect(api.applyOnboarding).toHaveBeenCalledTimes(initialApplyCalls + 2));
  });

  it("reports credential provisioning failure without clearing the input", async () => {
    api.provisionCredential.mockRejectedValueOnce(new Error("authority unavailable"));
    renderWithProviders(<StepReadiness title="Credentials" />);
    const input = await screen.findByLabelText("Value for OpenRouter key");
    fireEvent.change(input, { target: { value: "secret-value" } });
    fireEvent.click(screen.getByRole("button", { name: "Save securely" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Credential provisioning failed");
    expect(input).toHaveValue("secret-value");
  });

  it("resolves deferred operator inputs through the protected queue", async () => {
    api.fetchOperatorInputs.mockResolvedValueOnce({ version: 1, updated_at: "now", requests: [{ id: "mode", kind: "choice", title: "Operating mode", default: "desktop" }] });
    renderWithProviders(<StepReadiness title="Credentials" />);
    const input = await screen.findByTestId("operator-input");
    fireEvent.change(input, { target: { value: "server" } });
    fireEvent.click(screen.getByTestId("operator-input-resolve"));
    await waitFor(() => expect(api.resolveOperatorInputs).toHaveBeenCalledWith([{ request_id: "mode", value: "server" }]));
    expect(input).toHaveValue("");
  });

  it("supports credential store initialization and lifecycle controls", async () => {
    api.fetchCredentialStoreStatus.mockResolvedValueOnce({ initialized: false, active: false, entries: 0 });
    renderWithProviders(<StepReadiness title="Credentials" />);
    fireEvent.click(await screen.findByText("Manage protection"));
    fireEvent.change(screen.getByLabelText("New store passphrase"), { target: { value: "new-secret" } });
    fireEvent.click(screen.getByRole("button", { name: "Initialize" }));
    await waitFor(() => expect(api.initializeCredentialStore).toHaveBeenCalledWith("new-secret"));
    cleanup();

    api.fetchCredentialStoreStatus.mockResolvedValue({ initialized: true, active: true, entries: 0, active_wrap: "encrypted-file", active_key_store: "file" });
    renderWithProviders(<StepReadiness title="Credentials" />);
    const card = await screen.findAllByTestId("credential-store-card").then((cards) => cards[cards.length - 1]!);
    const cardQueries = within(card);
    fireEvent.click(cardQueries.getByText("Manage protection"));
    fireEvent.change(cardQueries.getByLabelText("Store passphrase"), { target: { value: "current" } });
    fireEvent.click(cardQueries.getByRole("button", { name: "Unlock" }));
    fireEvent.click(cardQueries.getByRole("button", { name: "Rewrap" }));
    fireEvent.change(cardQueries.getByLabelText("Current store passphrase"), { target: { value: "current" } });
    fireEvent.change(cardQueries.getByLabelText("New store passphrase"), { target: { value: "rotated" } });
    fireEvent.click(cardQueries.getByRole("button", { name: "Change passphrase" }));
    fireEvent.click(cardQueries.getByRole("button", { name: "Use native authority" }));
    fireEvent.click(cardQueries.getByRole("button", { name: "Use encrypted authority" }));
    await waitFor(() => {
      expect(api.unlockCredentialStore).toHaveBeenCalledWith("current");
      expect(api.rewrapCredentialStore).toHaveBeenCalledWith("current");
      expect(api.changeCredentialStorePassphrase).toHaveBeenCalledWith("current", "rotated");
    });
    expect(api.selectCredentialBackend).toHaveBeenCalledWith("native");
    expect(api.selectCredentialBackend).toHaveBeenCalledWith("encrypted-file");
  });

  it("exposes verified backend migration when credentials exist", async () => {
    api.fetchCredentialStoreStatus.mockResolvedValue({ initialized: true, active: true, entries: 1, active_wrap: "encrypted-file", active_key_store: "file" });
    api.reselectCredentialBackend.mockResolvedValue({ from: "encrypted-file", to: "native", attempted: ["credential"], verified: ["credential"], committed: true });
    renderWithProviders(<StepReadiness title="Credentials" />);
    const card = await screen.findAllByTestId("credential-store-card").then((cards) => cards[cards.length - 1]!);
    fireEvent.click(within(card).getByText("Manage protection"));
    fireEvent.click(within(card).getByRole("button", { name: "Re-evaluate and migrate safely" }));
    await waitFor(() => expect(api.reselectCredentialBackend).toHaveBeenCalled());
  });
});
