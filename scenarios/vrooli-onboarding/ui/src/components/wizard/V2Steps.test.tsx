// [REQ:ONB-CORE-SUPERVISION-AUTHORITY]
import { fireEvent, screen, waitFor } from "../../test-utils";
import { vi } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { DerivedResourceStep } from "./DerivedResourceStep";
import { HostRequirementStep } from "./HostRequirementStep";
import { StepIntegrationsDeferred } from "./StepIntegrationsDeferred";
import { StepOperatingMode } from "./StepOperatingMode";
import { StepApply } from "./StepApply";
import { StepCredentials } from "./StepCredentials";
import { StepReady } from "./StepReady";
import { ScenarioCatalogStep } from "./ScenarioCatalogStep";
import { StepCoreSet } from "./StepCoreSet";
import { ApplyRunState, ApplyStepState } from "@vrooli/proto-types/vrooli-onboarding/v1/apply/apply_pb";

const api = vi.hoisted(() => ({
  fetchHostRequirements: vi.fn(),
}));
const credentialsApi = vi.hoisted(() => ({ provisionCredential: vi.fn() }));
const capabilitiesApi = vi.hoisted(() => ({
  fetchCapabilities: vi.fn(),
  previewCapability: vi.fn(),
  applyCapability: vi.fn(),
}));
const selectionApi = vi.hoisted(() => ({
  fetchScenarios: vi.fn(),
  fetchCoreSet: vi.fn(),
  fetchClosure: vi.fn(),
}));
const resourcesApi = vi.hoisted(() => ({ fetchDerivedResources: vi.fn() }));

const applyApi = vi.hoisted(() => ({
  startApply: vi.fn(),
  reviewApply: vi.fn(),
  fetchApplyPlan: vi.fn(),
  fetchApplyRun: vi.fn(),
  cancelApply: vi.fn(),
}));

const readinessApi = vi.hoisted(() => ({ fetchReadiness: vi.fn(), acknowledgeDegraded: vi.fn() }));
const operatorInputsApi = vi.hoisted(() => ({ fetchOperatorInputs: vi.fn(), resolveOperatorInputs: vi.fn() }));

vi.mock("../../api/host", () => api);
vi.mock("../../api/credentials", () => credentialsApi);
vi.mock("../../api/capabilities", () => capabilitiesApi);
vi.mock("../../api/selection", () => selectionApi);
vi.mock("../../api/resources", () => resourcesApi);
vi.mock("../../api/apply", () => applyApi);
vi.mock("../../api/readiness", () => readinessApi);
vi.mock("../../api/operatorinputs", () => operatorInputsApi);

const scenarios = {
  scenarios: [
    { name: "control-plane", systemRequired: true, enabled: true, autoRestart: true, resources: ["postgres"], description: "Required" },
    { name: "writer", systemRequired: false, enabled: false, autoRestart: false, resources: ["ollama"], description: "Optional" },
  ],
  count: 2,
};

beforeEach(() => {
  window.localStorage.clear();
  selectionApi.fetchScenarios.mockResolvedValue(scenarios);
  selectionApi.fetchCoreSet.mockResolvedValue({
    available: true,
    seed: ["control-plane", "writer"],
    trustedBase: ["control-plane"],
    memberCounts: { scenario: 2, resource: 1 },
    members: [
      { name: "control-plane", kind: "scenario", supervisionIntent: "must_start" },
      { name: "writer", kind: "scenario", supervisionIntent: "must_start" },
      { name: "postgres", kind: "resource", supervisionIntent: "must_serve" },
    ],
  });
  api.fetchHostRequirements.mockResolvedValue({
    tools: [{ name: "git", required: true, reason: "source control", status: "required" }],
    safeguards: [{ name: "firewall", required: false, reason: "network safety", status: "optional", risk: "medium", config_schema: { type: "object", properties: { target: { type: "string", description: "collector target" } } } }],
  });
  readinessApi.fetchReadiness.mockResolvedValue({
    status: "degraded",
    scenarios: ["control-plane"],
    resources: ["postgres"],
    credentials: [{ resource: "openrouter", logical_id: "openrouter", field: "api_key", label: "OpenRouter key", required: true, status: "unconfigured" }],
    hosts: [{ name: "git", status: "ready", kind: "tool", required: true }],
    integrations: [{ name: "alpha/github-oauth", category: "integration", status: "deferred", required: true, detail: "Read project issues. Connection setup is deferred until the integration capability is available." }, { name: "release-authority", category: "system", status: "ready" }],
    checked_at: "2026-07-29T00:00:00Z",
    blockers: [],
    degraded: [],
  });
  readinessApi.acknowledgeDegraded.mockResolvedValue({ status: "acknowledged", readinessDigest: "digest-under-test" });
  operatorInputsApi.fetchOperatorInputs.mockResolvedValue({ requests: [] });
  operatorInputsApi.resolveOperatorInputs.mockResolvedValue({ configurationPending: false, outcomes: [] });
  selectionApi.fetchClosure.mockResolvedValue({ resources: [{ name: "postgres", required: true, direct: true, provenance: [] }, { name: "ollama", required: false, direct: false, provenance: [] }], scenarios: [] });
  resourcesApi.fetchDerivedResources.mockResolvedValue({
    resources: [{ name: "postgres", category: "database", enabled: true, installed: true }, { name: "ollama", category: "ai", enabled: false, installed: true }],
    required: [{ name: "postgres", category: "database", enabled: true, installed: true }],
    optional: [{ name: "ollama", category: "ai", enabled: false, installed: true }],
    standalone: [{ name: "qdrant", category: "search", enabled: false, installed: true }], count: 3,
  });
  credentialsApi.provisionCredential.mockResolvedValue({ status: "provisioned" });
  applyApi.startApply.mockResolvedValue({ run: { runId: "apply-test", status: ApplyRunState.APPLIED, legacyStatus: "applied", steps: [{ name: "postgres", state: ApplyStepState.APPLIED, legacyOutcome: "applied" }] } });
  applyApi.reviewApply.mockResolvedValue({ target: "local", planId: "plan-test", planDigest: "digest-test", revision: "revision-test", consentReceiptId: "receipt-test" });
  applyApi.fetchApplyPlan.mockResolvedValue({ target: "local", plan_id: "plan-test", plan_digest: "digest-test", revision: "revision-test", items: [{ id: "resource:postgres", kind: "resource", name: "postgres", required: true, privileged: false, state: "pending" }] });
  applyApi.fetchApplyRun.mockResolvedValue({ runId: "apply-test", status: ApplyRunState.APPLIED, legacyStatus: "applied", steps: [{ name: "postgres", state: ApplyStepState.APPLIED, legacyOutcome: "applied" }] });
  applyApi.cancelApply.mockResolvedValue({ run: { runId: "apply-test", status: ApplyRunState.CANCELLED, legacyStatus: "cancelled", steps: [] } });
  capabilitiesApi.fetchCapabilities.mockResolvedValue({ capabilities: [], count: 0 });
  capabilitiesApi.previewCapability.mockResolvedValue({ capability_id: "demo-capability", plan_id: "demo-plan", state: "ready_to_preview", mutations: [{ id: "demo-write", summary: "write a verified demo artifact", reversible: true }] });
  capabilitiesApi.applyCapability.mockResolvedValue({ capability_id: "demo-capability", state: "ready", outcome: "demo_ready", retryable: true, evidence: [{ kind: "demo", artifact_identity: "demo-artifact", observed_at: "now", verified: true }] });
});

describe("V2 onboarding wizard steps", () => {
  it("previews the supervision closure and prevents trusted-base removal", async () => {
    const onChange = vi.fn();
    renderWithProviders(<StepCoreSet seed={new Set(["control-plane"])} trustedBase={new Set(["control-plane"])} onChange={onChange} />);
    expect(await screen.findByText("2 scenarios · 1 resources")).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Supervise control-plane" })).toBeDisabled();
    fireEvent.click(screen.getByRole("checkbox", { name: "Supervise writer" }));
    expect(onChange).not.toHaveBeenCalled();
    const confirm = screen.getByRole("button", { name: "Confirm supervision set" });
    await waitFor(() => expect(confirm).toBeEnabled());
    fireEvent.click(confirm);
    expect(onChange).toHaveBeenCalledWith(["control-plane", "writer"]);
  });

  it("keeps the seed visible when closure computation is unavailable", async () => {
    selectionApi.fetchCoreSet.mockResolvedValueOnce({
      available: false,
      seed: ["control-plane"],
      trustedBase: ["control-plane"],
      error: "catalog unavailable",
    });
    renderWithProviders(<StepCoreSet seed={new Set(["control-plane"])} trustedBase={new Set(["control-plane"])} onChange={vi.fn()} />);
    expect(await screen.findByText(/Seed: control-plane/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Confirm supervision set" })).toBeDisabled();
  });

  it("derives resources and lets operators select optional scenarios", async () => {
    const onToggle = vi.fn();
    const onResourceToggle = vi.fn();
    renderWithProviders(<><ScenarioCatalogStep selected={new Set()} onToggle={onToggle} /><DerivedResourceStep selected={new Set(["writer"])} operatorState={{ version: "1", updatedAt: "now", resources: { ollama: { enabled: true } } }} onToggle={onResourceToggle} /></>);
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
    fireEvent.change(screen.getByRole("searchbox", { name: "Search scenarios" }), { target: { value: "does-not-exist" } });
    expect(screen.getByText("No scenarios match this filter. Clear it to see the full catalog.")).toBeInTheDocument();
    fireEvent.change(screen.getByRole("searchbox", { name: "Search scenarios" }), { target: { value: "" } });
    fireEvent.click(screen.getByRole("button", { name: /Available/ }));
    expect(screen.getByTestId("scenario-card-writer")).toBeInTheDocument();
    expect(screen.queryByTestId("scenario-card-control-plane")).not.toBeInTheDocument();
  });

  it("shows manifest-derived host requirements and sends opt-ins to the owner", async () => {
    const onTool = vi.fn();
    const onSafeguard = vi.fn();
    renderWithProviders(<HostRequirementStep onTool={onTool} onSafeguard={onSafeguard} onHostConfig={vi.fn()} />);
    expect(await screen.findByText("git")).toBeInTheDocument();
    const firewall = screen.getByRole("checkbox", { name: /firewall/i });
    fireEvent.click(firewall);
    expect(onSafeguard).toHaveBeenCalledWith("firewall", true);
    expect(onTool).not.toHaveBeenCalled();
  });

  it("renders generic manifest config fields and emits their values", async () => {
    const onConfig = vi.fn();
    renderWithProviders(<HostRequirementStep onTool={vi.fn()} onSafeguard={vi.fn()} onHostConfig={(kind, name, config) => onConfig(kind, name, config)} />);
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
    const checkbox = await screen.findByRole("switch", { name: "Keep writer running" });
    fireEvent.click(checkbox);
    expect(onAutoRestart).toHaveBeenCalledWith("writer", true);
  });

  it("provisions a credential without exposing its value and renders validation groups", async () => {
    renderWithProviders(<StepCredentials />);
    const input = await screen.findByLabelText("Value for OpenRouter key");
    fireEvent.change(input, { target: { value: "secret-value" } });
    fireEvent.click(screen.getByRole("button", { name: "Save securely" }));
    await waitFor(() => expect(credentialsApi.provisionCredential).toHaveBeenCalledWith({ logical_id: "openrouter", field: "api_key", value: "secret-value" }, "local"));
    expect(input).toHaveValue("");

    renderWithProviders(<StepReady />);
    expect(await screen.findByText("Host requirements")).toBeInTheDocument();
    expect(screen.getByText("Integrations")).toBeInTheDocument();
    expect(screen.getByText("alpha/github-oauth")).toBeInTheDocument();
  });

  it("surfaces provider failures", async () => {
    readinessApi.fetchReadiness.mockRejectedValueOnce(new Error("probe unavailable"));
    renderWithProviders(<StepReady />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Readiness could not be checked");
  });

  // The acknowledgement is durable operator state, not component state. A page
  // reload must not silently discard the operator's acceptance, and accepting
  // one gap must not authorise completion over a different one.
  it("records the degraded acknowledgement through the API rather than in component state", async () => {
    readinessApi.fetchReadiness.mockResolvedValue({
      status: "degraded", scenarios: [], resources: [], credentials: [], hosts: [], integrations: [], checked_at: "now",
      blockers: [],
      degraded: [{ kind: "credential", name: "vrooli/remote-desktop:username", reason: "the credential is declared and not configured", remediation: "Provide it on the credentials step." }],
      degraded_digest: "digest-under-test",
      degraded_acknowledged: false,
    });
    readinessApi.acknowledgeDegraded.mockResolvedValue({ status: "acknowledged", readinessDigest: "digest-under-test" });
    renderWithProviders(<StepReady />);
    expect(await screen.findByTestId("readiness-degraded")).toHaveTextContent("vrooli/remote-desktop:username");
    fireEvent.click(await screen.findByTestId("readiness-continue-degraded"));
    await waitFor(() => expect(readinessApi.acknowledgeDegraded).toHaveBeenCalledWith("digest-under-test", "local"));
  });

  // The gate is at the marker write, but the operator must still be told why
  // the flow will not report completion.
  it("names blocking items and states that completion is withheld", async () => {
    readinessApi.fetchReadiness.mockResolvedValue({
      status: "missing", scenarios: [], resources: [], credentials: [], hosts: [], integrations: [], checked_at: "now",
      blockers: [{ kind: "credential", name: "vrooli/calendar:jwt-secret", reason: "the credential is declared and not configured", remediation: "Provide it on the credentials step." }],
      degraded: [],
    });
    renderWithProviders(<StepReady />);
    expect(await screen.findByTestId("readiness-blockers")).toHaveTextContent("vrooli/calendar:jwt-secret");
    expect(screen.getByTestId("finish-blocked")).toBeInTheDocument();
    expect(screen.queryByTestId("readiness-continue-degraded")).toBeNull();
  });

  it("supports enum, boolean, and numeric host configuration fields", async () => {
    api.fetchHostRequirements.mockResolvedValueOnce({
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
    renderWithProviders(<HostRequirementStep onTool={vi.fn()} onSafeguard={vi.fn()} onHostConfig={(kind, name, config) => onConfig(kind, name, config)} />);
    fireEvent.change(await screen.findByLabelText("mode"), { target: { value: "enforce" } });
    fireEvent.click(screen.getByLabelText("enabled"));
    fireEvent.change(screen.getByLabelText("retries"), { target: { value: "3" } });
    expect(onConfig).toHaveBeenLastCalledWith("host_safeguards", "firewall", { mode: "enforce", enabled: true, retries: 3 });
  });

  it("keeps provider guidance out of the focused apply report", async () => {
    readinessApi.fetchReadiness.mockResolvedValueOnce({
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
    renderWithProviders(<StepApply />);
    expect(await screen.findByRole("heading", { level: 1, name: "Review and apply" })).toBeInTheDocument();
    expect(screen.queryByTestId("backend-diagnosis")).not.toBeInTheDocument();
    expect(screen.queryByTestId("credential-obtain-link")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Apply selection" }));
    await waitFor(() => expect(applyApi.startApply).toHaveBeenCalled());
    expect(await screen.findByTestId("run-ladder")).toHaveTextContent("postgres");
    expect(await screen.findByTestId("run-ladder")).toHaveTextContent("succeeded");
  });

  it("surfaces capability status and provider evidence without owner-specific rendering", async () => {
    capabilitiesApi.fetchCapabilities.mockResolvedValueOnce({
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
    renderWithProviders(<StepCredentials />);
    expect(await screen.findByTestId("capability-card-demo-capability")).toHaveTextContent("Protect a demo artifact");
    fireEvent.change(screen.getByLabelText("Destination"), { target: { value: "/mnt/approved" } });
    fireEvent.click(screen.getByTestId("capability-confirm-demo-capability"));
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    await waitFor(() => expect(capabilitiesApi.previewCapability).toHaveBeenCalledWith({ capability_id: "demo-capability", confirm: false, inputs: { destination: "/mnt/approved" } }));
    fireEvent.click(await screen.findByRole("button", { name: "Apply reviewed capability" }));
    await waitFor(() => expect(capabilitiesApi.applyCapability).toHaveBeenCalledWith({ capability_id: "demo-capability", confirm: true, inputs: { destination: "/mnt/approved" } }));
    expect(screen.getByTestId("capability-result-demo-capability")).toHaveTextContent("demo_ready");
    expect(screen.queryByText("secret-value")).not.toBeInTheDocument();
  });

  it("renders evidence-only providers without inventing an action control", async () => {
    capabilitiesApi.fetchCapabilities.mockResolvedValueOnce({
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
    renderWithProviders(<StepCredentials />);
    expect(await screen.findByTestId("capability-card-durable-backup-evidence")).toHaveTextContent("recovery-drill · verified");
    expect(screen.getByTestId("capability-card-durable-backup-evidence")).toHaveTextContent("run a recovery drill");
    expect(screen.queryByRole("button", { name: "Preview" })).not.toBeInTheDocument();
  });

  it("renders provider provenance and unsupported disposition generically", async () => {
    capabilitiesApi.fetchCapabilities.mockResolvedValueOnce({
      count: 1,
      capabilities: [{
        descriptor: {
          version: "operator-capability/v1", id: "platform-permission-fixture", owner: "fixture.owner", title: "Platform permission",
          scope: "selected host", purpose: "request a declared permission", sensitivity: "operator", disposition: "unsupported", disposition_reason: "target does not expose the owner",
          provenance: { requester: "onboarding operator", scope: "selected host", grant_source: "explicit consent", revocation_limit: "owner revoke only" },
          inputs: [{ id: "permission", kind: "enum", label: "Permission", required: true, options: ["notifications"] }],
          policy: { requires_confirmation: true, idempotent: true, retryable: true }, evidence: { secret_free: true, kinds: ["permission"] },
        },
        state: "unsupported", missing_inputs: ["permission"], updated_at: "now",
      }],
    });
    renderWithProviders(<StepCredentials />);
    const card = await screen.findByTestId("capability-card-platform-permission-fixture");
    expect(card).toHaveTextContent("selected host");
    expect(card).toHaveTextContent("owner revoke only");
    expect(screen.getByTestId("capability-blocked-platform-permission-fixture")).toHaveTextContent("target does not expose the owner");
    expect(screen.queryByRole("button", { name: "Preview" })).not.toBeInTheDocument();
  });

  it("keeps capability secrets write-only and clears them after apply", async () => {
    capabilitiesApi.previewCapability.mockClear();
    capabilitiesApi.applyCapability.mockClear();
    capabilitiesApi.fetchCapabilities.mockResolvedValueOnce({
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
    renderWithProviders(<StepCredentials />);
    expect(await screen.findByTestId("capability-card-secret-capability")).toBeInTheDocument();
    const destination = screen.getByLabelText("Destination");
    fireEvent.change(destination, { target: { value: "/mnt/approved" } });
    await waitFor(() => expect(destination).toHaveValue("/mnt/approved"));
    const passphrase = screen.getByLabelText("Passphrase");
    fireEvent.change(passphrase, { target: { value: "ephemeral-passphrase" } });
    await waitFor(() => expect(passphrase).toHaveValue("ephemeral-passphrase"));
    fireEvent.click(screen.getByTestId("capability-confirm-secret-capability"));
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    await waitFor(() => expect(capabilitiesApi.previewCapability).toHaveBeenCalledWith({ capability_id: "secret-capability", confirm: false, inputs: { destination: "/mnt/approved", passphrase: "ephemeral-passphrase" } }));
    fireEvent.click(await screen.findByRole("button", { name: "Apply reviewed capability" }));
    await waitFor(() => expect(capabilitiesApi.applyCapability).toHaveBeenCalledWith({ capability_id: "secret-capability", confirm: true, inputs: { destination: "/mnt/approved", passphrase: "ephemeral-passphrase" } }));
    expect(screen.getByLabelText("Passphrase")).toHaveValue("");
    expect(screen.getByTestId("capability-result-secret-capability")).not.toHaveTextContent("ephemeral-passphrase");
  });

  it("reports preview and apply failures without claiming success", async () => {
    capabilitiesApi.fetchCapabilities.mockResolvedValueOnce({
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
    capabilitiesApi.previewCapability.mockRejectedValueOnce(new Error("preview unavailable"));
    renderWithProviders(<StepCredentials />);
    fireEvent.change(await screen.findByLabelText("Destination"), { target: { value: "/mnt/approved" } });
    fireEvent.click(screen.getByLabelText(/Enabled/));
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("The capability preview failed");

    capabilitiesApi.previewCapability.mockResolvedValueOnce({ capability_id: "failing-capability", plan_id: "demo-plan", state: "ready_to_preview", mutations: [] });
    capabilitiesApi.applyCapability.mockRejectedValueOnce(new Error("apply unavailable"));
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    await waitFor(() => expect(screen.getByTestId("capability-preview-failing-capability")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("capability-confirm-failing-capability"));
    fireEvent.click(screen.getByRole("button", { name: "Apply reviewed capability" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("The capability could not be applied");
  });

  it("does not render manual recovery commands in the onboarding action surface", async () => {
    readinessApi.fetchReadiness.mockResolvedValueOnce({
      status: "missing",
      scenarios: [],
      resources: [],
      credentials: [],
      hosts: [],
      integrations: [],
      checked_at: "2026-07-29T00:00:00Z",
    });
    renderWithProviders(<StepReady />);
    expect(await screen.findByTestId("readiness-summary")).toHaveTextContent("missing");
    expect(screen.queryByText(/secrets-manager backup export/i)).not.toBeInTheDocument();
  });

  it("keeps the apply route as a distinct step identity", async () => {
    renderWithProviders(<StepApply />);
    expect(await screen.findByRole("heading", { level: 1, name: "Review and apply" })).toBeInTheDocument();
  });

  it("waits for an async partial apply and exposes retry evidence", async () => {
    const initialApplyCalls = applyApi.startApply.mock.calls.length;
    applyApi.startApply.mockResolvedValueOnce({ run: { runId: "apply-pending", status: ApplyRunState.PENDING, legacyStatus: "pending", steps: [] } });
    applyApi.fetchApplyRun.mockResolvedValueOnce({ runId: "apply-pending", status: ApplyRunState.PARTIALLY_APPLIED, legacyStatus: "partially_applied", steps: [
      { name: "firewall", state: ApplyStepState.FAILED, legacyOutcome: "failed", error: "permission denied" },
      { name: "writer", state: ApplyStepState.BLOCKED, legacyOutcome: "blocked", error: "blocked by firewall" },
    ] });
    renderWithProviders(<StepApply />);
    fireEvent.click(await screen.findByRole("button", { name: "Apply selection" }));
    expect(await screen.findByTestId("skipped-note")).toHaveTextContent("Some items were skipped or failed");
    fireEvent.click(screen.getByTestId("retry"));
    await waitFor(() => expect(applyApi.startApply).toHaveBeenCalledTimes(initialApplyCalls + 2));
  });

  it("requests cancellation and preserves the run identity", async () => {
    applyApi.startApply.mockResolvedValueOnce({ run: { runId: "apply-cancel", status: ApplyRunState.PENDING, legacyStatus: "pending", steps: [] } });
    applyApi.cancelApply.mockResolvedValueOnce({ run: { runId: "apply-cancel", status: ApplyRunState.CANCELLED, legacyStatus: "cancelled", steps: [] } });
    renderWithProviders(<StepApply />);
    fireEvent.click(await screen.findByRole("button", { name: "Apply selection" }));
    fireEvent.click(await screen.findByTestId("apply-cancel"));
    await waitFor(() => expect(applyApi.cancelApply).toHaveBeenCalledWith("apply-cancel", "local"));
    expect(screen.getByTestId("apply-state")).toHaveTextContent("apply stopped safely");
    expect(screen.getByTestId("apply-state")).toHaveTextContent("apply-cancel");
  });

  it("reports credential provisioning failure without clearing the input", async () => {
    credentialsApi.provisionCredential.mockRejectedValueOnce(new Error("authority unavailable"));
    renderWithProviders(<StepCredentials />);
    const input = await screen.findByLabelText("Value for OpenRouter key");
    fireEvent.change(input, { target: { value: "secret-value" } });
    fireEvent.click(screen.getByRole("button", { name: "Save securely" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Credential provisioning failed");
    expect(input).toHaveValue("secret-value");
  });

  it("renders and submits every declared operator-input kind", async () => {
    operatorInputsApi.fetchOperatorInputs.mockResolvedValueOnce({
      requests: [
        { id: "secret", kind: 1, title: "Secret", description: "Sensitive", required: true, options: [], candidates: [], validation: "required", declinable: true },
        { id: "choice", kind: 2, title: "Choice", description: "Pick one", required: true, options: ["one", "two"], candidates: [], validation: "select" },
        { id: "confirm", kind: 3, title: "Confirm", description: "Confirm it", required: false, options: [], candidates: [] },
        { id: "path", kind: 4, title: "Path", description: "Path", required: false },
        { id: "enum", kind: 5, title: "Enum", description: "Enum", required: false, options: ["value"] },
        { id: "boolean", kind: 6, title: "Boolean", description: "Boolean", required: false },
        { id: "duration", kind: 7, title: "Duration", description: "Duration", required: false },
        { id: "confirmation", kind: 8, title: "Confirmation", description: "Confirmation", required: false },
        { id: "unknown", kind: 99, title: "Unknown", description: "Fallback", required: false },
      ],
    });
    readinessApi.fetchReadiness.mockResolvedValueOnce({ status: "ready", scenarios: [], resources: [], credentials: [], hosts: [], integrations: [], checked_at: "now", blockers: [], degraded: [] });
    renderWithProviders(<StepCredentials target="remote" />);
    expect(await screen.findByTestId("target-question-set")).toHaveTextContent("Secret");
    expect(screen.getByLabelText("Secret")).toBeInTheDocument();
    expect(screen.getByText("Choice")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Secret"), { target: { value: "temporary" } });
    fireEvent.click(screen.getByRole("button", { name: "Submit secret answers" }));
    await waitFor(() => expect(operatorInputsApi.resolveOperatorInputs).toHaveBeenCalledWith(expect.arrayContaining([expect.objectContaining({ requestId: "secret", value: "temporary", declined: false })]), "remote"));
    expect(await screen.findByTestId("target-question-status")).toHaveTextContent("Answers submitted");
  });


});
