import { fireEvent, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { StepDerivedResources } from "./StepDerivedResources";
import { StepHostRequirements } from "./StepHostRequirements";
import { StepIntegrationsDeferred } from "./StepIntegrationsDeferred";
import { StepOperatingMode } from "./StepOperatingMode";
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
    safeguards: [{ name: "firewall", required: false, reason: "network safety", status: "optional", risk: "medium" }],
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
    renderWithProviders(<StepHostRequirements onTool={onTool} onSafeguard={onSafeguard} />);
    expect(await screen.findByText("git")).toBeInTheDocument();
    const firewall = screen.getByRole("checkbox", { name: /firewall/i });
    fireEvent.click(firewall);
    expect(onSafeguard).toHaveBeenCalledWith("firewall", true);
    expect(onTool).not.toHaveBeenCalled();
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

    renderWithProviders(<StepReadiness />);
    expect(await screen.findByText("Host requirements")).toBeInTheDocument();
    expect(screen.getByText("Integrations")).toBeInTheDocument();
    expect(screen.getByText("OAuth")).toBeInTheDocument();
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
    renderWithProviders(<StepReadiness />);
    expect(await screen.findByTestId("backend-diagnosis")).toHaveTextContent("Available");
    expect(screen.getByTestId("store-init-guidance")).toBeInTheDocument();
    expect(screen.getByTestId("credential-obtain-link")).toHaveAttribute("href", "https://example.test/key");
    fireEvent.click(screen.getByTestId("apply-confirm"));
    await waitFor(() => expect(api.applyOnboarding).toHaveBeenCalled());
    expect(await screen.findByTestId("apply-report")).toHaveTextContent("postgres");
    expect(await screen.findByTestId("apply-report")).toHaveTextContent("applied");
  });
});
