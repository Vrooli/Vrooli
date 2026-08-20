import { fireEvent, screen, waitFor } from "@testing-library/react";
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
  fetchCapabilities: vi.fn(),
  previewCapability: vi.fn(),
  applyCapability: vi.fn(),
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
    integrations: [{ name: "alpha/github-oauth", category: "integration", status: "deferred", required: true, detail: "Read project issues. Connection setup is deferred until the integration capability is available." }, { name: "release-authority", category: "system", status: "ready" }],
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
  api.fetchCapabilities.mockResolvedValue({ capabilities: [], count: 0 });
  api.previewCapability.mockResolvedValue({ capability_id: "demo-capability", plan_id: "demo-plan", state: "ready_to_preview", mutations: [{ id: "demo-write", summary: "write a verified demo artifact", reversible: true }] });
  api.applyCapability.mockResolvedValue({ capability_id: "demo-capability", state: "ready", outcome: "demo_ready", retryable: true, evidence: [{ kind: "demo", artifact_identity: "demo-artifact", observed_at: "now", verified: true }] });
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

  it("renders the deferred integration contract", async () => {
    renderWithProviders(<StepIntegrationsDeferred />);
    expect(screen.getByRole("status")).toHaveTextContent("Integration setup is deferred");
    const declared = await screen.findByTestId("declared-integrations");
    expect(declared).toHaveTextContent("alpha/github-oauth");
    expect(declared).toHaveTextContent("Read project issues");
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
    expect(screen.getByText("alpha/github-oauth")).toBeInTheDocument();
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
    expect(screen.getByTestId("credential-obtain-link")).toHaveAttribute("href", "https://example.test/key");
    fireEvent.click(screen.getByTestId("apply-confirm"));
    await waitFor(() => expect(api.applyOnboarding).toHaveBeenCalled());
    expect(await screen.findByTestId("apply-report")).toHaveTextContent("postgres");
    expect(await screen.findByTestId("apply-report")).toHaveTextContent("applied");
  });

  it("surfaces capability status and provider evidence without owner-specific rendering", async () => {
    api.fetchCapabilities.mockResolvedValueOnce({
      count: 1,
      capabilities: [{
        descriptor: {
          version: "operator-capability/v1", id: "demo-capability", owner: "demo.owner", title: "Protect a demo artifact",
          description: "A provider-defined action.", inputs: [{ id: "destination", kind: "path", label: "Destination", required: true }],
          policy: { requires_confirmation: true, idempotent: true, retryable: true }, evidence: { secret_free: true, kinds: ["demo"] },
        },
        state: "needs_operator_input", missing_inputs: ["destination"], remediation: "Choose a destination.", updated_at: "now",
      }],
    });
    renderWithProviders(<StepReadiness title="Credentials" />);
    expect(await screen.findByTestId("capability-card-demo-capability")).toHaveTextContent("Protect a demo artifact");
    fireEvent.change(screen.getByLabelText("Destination"), { target: { value: "/mnt/approved" } });
    fireEvent.click(screen.getByTestId("capability-confirm-demo-capability"));
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    await waitFor(() => expect(api.previewCapability).toHaveBeenCalledWith({ capability_id: "demo-capability", confirm: false, inputs: { destination: "/mnt/approved" } }));
    fireEvent.click(await screen.findByRole("button", { name: "Apply reviewed capability" }));
    await waitFor(() => expect(api.applyCapability).toHaveBeenCalledWith({ capability_id: "demo-capability", confirm: true, inputs: { destination: "/mnt/approved" } }));
    expect(screen.getByTestId("capability-result-demo-capability")).toHaveTextContent("demo_ready");
    expect(screen.queryByText("secret-value")).not.toBeInTheDocument();
  });

  it("renders evidence-only providers without inventing an action control", async () => {
    api.fetchCapabilities.mockResolvedValueOnce({
      count: 1,
      capabilities: [{
        descriptor: {
          version: "operator-capability/v1", id: "durable-backup-evidence", owner: "data-backup-manager", title: "Durable backup and recovery evidence",
          description: "Read-only owner evidence.", inputs: [],
          policy: { requires_confirmation: false, idempotent: true, retryable: true }, evidence: { secret_free: true, kinds: ["recovery-drill"] },
        },
        state: "degraded", evidence: [{ kind: "recovery-drill", artifact_identity: "data-backup-manager/drill/drill-1", observed_at: "now", verified: true }],
        remediation: "run a recovery drill",
      }],
    });
    renderWithProviders(<StepReadiness title="Credentials" />);
    expect(await screen.findByTestId("capability-card-durable-backup-evidence")).toHaveTextContent("recovery-drill · verified");
    expect(screen.getByTestId("capability-card-durable-backup-evidence")).toHaveTextContent("run a recovery drill");
    expect(screen.queryByRole("button", { name: "Preview" })).not.toBeInTheDocument();
  });

  it("keeps capability secrets write-only and clears them after apply", async () => {
    api.previewCapability.mockClear();
    api.applyCapability.mockClear();
    api.fetchCapabilities.mockResolvedValueOnce({
      count: 1,
      capabilities: [{
        descriptor: {
          version: "operator-capability/v1", id: "secret-capability", owner: "demo.owner", title: "Protect a secret",
          inputs: [
            { id: "destination", kind: "path", label: "Destination", required: true },
            { id: "passphrase", kind: "secret", label: "Passphrase", required: true },
          ],
          policy: { requires_confirmation: true, idempotent: true, retryable: true }, evidence: { secret_free: true, kinds: ["demo"] },
        },
        state: "needs_operator_input", missing_inputs: ["destination", "passphrase"], remediation: "Choose a destination and enter the passphrase.", updated_at: "now",
      }],
    });
    renderWithProviders(<StepReadiness title="Credentials" />);
    expect(await screen.findByTestId("capability-card-secret-capability")).toBeInTheDocument();
    const destination = screen.getByLabelText("Destination");
    fireEvent.change(destination, { target: { value: "/mnt/approved" } });
    await waitFor(() => expect(destination).toHaveValue("/mnt/approved"));
    const passphrase = screen.getByLabelText("Passphrase");
    fireEvent.change(passphrase, { target: { value: "ephemeral-passphrase" } });
    await waitFor(() => expect(passphrase).toHaveValue("ephemeral-passphrase"));
    fireEvent.click(screen.getByTestId("capability-confirm-secret-capability"));
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    await waitFor(() => expect(api.previewCapability).toHaveBeenCalledWith({ capability_id: "secret-capability", confirm: false, inputs: { destination: "/mnt/approved", passphrase: "ephemeral-passphrase" } }));
    fireEvent.click(await screen.findByRole("button", { name: "Apply reviewed capability" }));
    await waitFor(() => expect(api.applyCapability).toHaveBeenCalledWith({ capability_id: "secret-capability", confirm: true, inputs: { destination: "/mnt/approved", passphrase: "ephemeral-passphrase" } }));
    expect(screen.getByLabelText("Passphrase")).toHaveValue("");
    expect(screen.getByTestId("capability-result-secret-capability")).not.toHaveTextContent("ephemeral-passphrase");
  });

  it("reports preview and apply failures without claiming success", async () => {
    api.fetchCapabilities.mockResolvedValueOnce({
      count: 1,
      capabilities: [{
        descriptor: {
          version: "operator-capability/v1", id: "failing-capability", owner: "demo.owner", title: "A failing capability",
          inputs: [
            { id: "destination", kind: "path", label: "Destination", required: true },
            { id: "enabled", kind: "boolean", label: "Enabled", required: true },
          ],
          policy: { requires_confirmation: true, idempotent: true, retryable: true }, evidence: { secret_free: true, kinds: ["demo"] },
        },
        state: "needs_operator_input", missing_inputs: ["destination", "enabled"], remediation: "Choose a destination.", updated_at: "now",
      }],
    });
    api.previewCapability.mockRejectedValueOnce(new Error("preview unavailable"));
    renderWithProviders(<StepReadiness title="Credentials" />);
    fireEvent.change(await screen.findByLabelText("Destination"), { target: { value: "/mnt/approved" } });
    fireEvent.click(screen.getByLabelText(/Enabled/));
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("The capability preview failed");

    api.previewCapability.mockResolvedValueOnce({ capability_id: "failing-capability", plan_id: "demo-plan", state: "ready_to_preview", mutations: [] });
    api.applyCapability.mockRejectedValueOnce(new Error("apply unavailable"));
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    await waitFor(() => expect(screen.getByTestId("capability-preview-failing-capability")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("capability-confirm-failing-capability"));
    fireEvent.click(screen.getByRole("button", { name: "Apply reviewed capability" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("The capability could not be applied");
  });

  it("does not render manual recovery commands in the onboarding action surface", async () => {
    api.fetchV2Readiness.mockResolvedValueOnce({
      status: "missing",
      scenarios: [],
      resources: [],
      credentials: [],
      hosts: [],
      integrations: [],
      checked_at: "2026-07-29T00:00:00Z",
    });
    renderWithProviders(<StepReadiness title="Validation" />);
    expect(await screen.findByTestId("readiness-summary")).toHaveTextContent("missing");
    expect(screen.queryByText(/secrets-manager backup export/i)).not.toBeInTheDocument();
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


});
