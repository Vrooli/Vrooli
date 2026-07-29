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
  provisionCredential: vi.fn(),
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
  api.provisionCredential.mockResolvedValue({ status: "provisioned" });
});

describe("V2 onboarding wizard steps", () => {
  it("derives resources and lets operators select optional scenarios", async () => {
    const onToggle = vi.fn();
    renderWithProviders(<><StepSelectScenarios selected={new Set()} onToggle={onToggle} /><StepDerivedResources selected={new Set(["writer"])} /></>);
    expect(await screen.findByTestId("scenario-card-writer")).toHaveAttribute("aria-pressed", "false");
    fireEvent.click(screen.getByTestId("scenario-card-writer"));
    expect(onToggle).toHaveBeenCalledWith("writer");
    expect(await screen.findByText("ollama")).toBeInTheDocument();
    expect(screen.getByText("postgres")).toBeInTheDocument();
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
});
