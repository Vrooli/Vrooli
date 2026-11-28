// [REQ:ONB-CORE-SUPERVISION-AUTHORITY]
import { fireEvent, screen, waitFor, within } from "../../test-utils";
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
import { resetCredentialTargetState } from "../../hooks/useCredentialTargetState";
import { encodeOperatorQuestionDraft } from "../../lib/operatorQuestionDraft";

const api = vi.hoisted(() => ({
  fetchHostRequirements: vi.fn(),
  fetchTargets: vi.fn().mockResolvedValue({ targets: [], error: undefined }),
}));
const credentialsApi = vi.hoisted(() => ({ fetchCredentials: vi.fn(), provisionCredential: vi.fn(), revealCredential: vi.fn() }));
const capabilitiesApi = vi.hoisted(() => ({
  fetchCapabilities: vi.fn(),
  previewCapability: vi.fn(),
  applyCapability: vi.fn(),
  verifyCapability: vi.fn(),
}));
const selectionApi = vi.hoisted(() => ({
  fetchScenarios: vi.fn(),
  fetchCoreSet: vi.fn(),
  fetchClosure: vi.fn(),
}));
const resourcesApi = vi.hoisted(() => ({ fetchDerivedResources: vi.fn() }));
const coreAuthorityApi = vi.hoisted(() => ({ readCoreAuthority: vi.fn().mockResolvedValue({ revision: "r1", authority: { seed: [], trusted_base: [], important: [], members: [] } }) }));

const applyApi = vi.hoisted(() => ({
  startApply: vi.fn(),
  reviewApply: vi.fn(),
  fetchApplyPlan: vi.fn(),
  fetchApplyRun: vi.fn(),
  cancelApply: vi.fn(),
}));

const readinessApi = vi.hoisted(() => ({ fetchReadiness: vi.fn(), acknowledgeDegraded: vi.fn() }));
const operatorInputsApi = vi.hoisted(() => ({ fetchOperatorInputs: vi.fn(), resolveOperatorInputs: vi.fn() }));
const draftApi = vi.hoisted(() => ({ fetchDraft: vi.fn(), saveDraft: vi.fn(), discardDraft: vi.fn() }));

vi.mock("../../api/host", () => api);
vi.mock("../../api/credentials", () => credentialsApi);
vi.mock("../../api/capabilities", () => capabilitiesApi);
vi.mock("../../api/selection", () => selectionApi);
vi.mock("../../api/resources", () => resourcesApi);
vi.mock("../../api/coreAuthority", () => coreAuthorityApi);
vi.mock("../../api/apply", () => applyApi);
vi.mock("../../api/readiness", () => readinessApi);
vi.mock("../../api/operatorinputs", () => operatorInputsApi);
vi.mock("../../api/session", () => draftApi);
vi.mock("../../api/operatorstate", () => ({ fetchOperatorState: vi.fn().mockResolvedValue({ version: "v1" }) }));

const scenarios = {
  scenarios: [
    { name: "control-plane", systemRequired: true, enabled: true, autoRestart: true, resources: ["postgres"], description: "Required" },
    { name: "writer", systemRequired: false, enabled: false, autoRestart: false, resources: ["ollama"], description: "Optional" },
  ],
  count: 2,
};

beforeEach(() => {
  ["local", "node-7", "remote"].forEach(resetCredentialTargetState);
  draftApi.fetchDraft.mockResolvedValue({ target: "local", actor: "onboarding-questionnaire", baseRevision: "v1", revision: "", stepId: "credentials", choices: {} });
  draftApi.saveDraft.mockResolvedValue({ target: "local", actor: "onboarding-questionnaire", baseRevision: "v1", revision: "draft-r1", stepId: "credentials", choices: {} });
  draftApi.discardDraft.mockResolvedValue({});
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
  api.fetchTargets.mockResolvedValue({ targets: [], error: undefined });
  readinessApi.fetchReadiness.mockResolvedValue({
    status: "degraded",
    scenarios: ["control-plane"],
    resources: ["postgres"],
    credentials: [{ resource: "openrouter", logical_id: "openrouter", field: "api_key", label: "OpenRouter key", required: true, status: "unconfigured" }],
    hosts: [{ name: "git", status: "ready", kind: "tool", required: true }],
    integrations: [{ name: "alpha/github-oauth", category: "integration", status: "deferred", required: true, detail: "Read project issues. Onboarding does not bind this declaration to an Integration Hub connection." }, { name: "release-authority", category: "system", status: "ready" }],
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
  credentialsApi.revealCredential.mockResolvedValue("revealed-secret-value");
  credentialsApi.fetchCredentials.mockResolvedValue({ credentials: [{ resource: "openrouter", logical_id: "openrouter", field: "api_key", label: "OpenRouter key", required: true, status: "unconfigured" }], count: 1 });
  applyApi.startApply.mockResolvedValue({ run: { runId: "apply-test", status: ApplyRunState.APPLIED, legacyStatus: "applied", steps: [{ name: "postgres", state: ApplyStepState.APPLIED, legacyOutcome: "applied" }] } });
  applyApi.reviewApply.mockResolvedValue({ target: "local", planId: "plan-test", planDigest: "digest-test", revision: "revision-test", consentReceiptId: "receipt-test" });
  applyApi.fetchApplyPlan.mockResolvedValue({ target: "local", plan_id: "plan-test", plan_digest: "digest-test", revision: "revision-test", items: [{ id: "resource:postgres", kind: "resource", name: "postgres", required: true, privileged: false, state: "pending" }] });
  applyApi.fetchApplyRun.mockResolvedValue({ runId: "apply-test", status: ApplyRunState.APPLIED, legacyStatus: "applied", steps: [{ name: "postgres", state: ApplyStepState.APPLIED, legacyOutcome: "applied" }] });
  applyApi.cancelApply.mockResolvedValue({ run: { runId: "apply-test", status: ApplyRunState.CANCELLED, legacyStatus: "cancelled", steps: [] } });
  capabilitiesApi.fetchCapabilities.mockResolvedValue({ capabilities: [], count: 0 });
  capabilitiesApi.previewCapability.mockResolvedValue({ capability_id: "demo-capability", plan_id: "demo-plan", state: "ready_to_preview", mutations: [{ id: "demo-write", summary: "write a verified demo artifact", reversible: true }] });
  capabilitiesApi.applyCapability.mockResolvedValue({ capability_id: "demo-capability", state: "ready", outcome: "demo_ready", retryable: true, evidence: [{ kind: "demo", artifact_identity: "demo-artifact", observed_at: "now", verified: true }] });
  capabilitiesApi.verifyCapability.mockResolvedValue([{ kind: "demo", artifact_identity: "demo-artifact", observed_at: "now", verified: true }]);
});

async function openCredentialDetails(tab: "actions" | "questions" = "actions") {
  fireEvent.click(await screen.findByRole("button", { name: "Review credential setup" }));
  await screen.findByRole("dialog", { name: "Credential setup details" });
  fireEvent.click(screen.getByRole("tab", { name: tab === "questions" ? "Setup questions" : "Provider actions" }));
}

describe("V2 onboarding wizard steps", () => {
  it("previews the supervision closure and prevents trusted-base removal", async () => {
    const onChange = vi.fn();
    renderWithProviders(<StepCoreSet seed={new Set(["control-plane"])} trustedBase={new Set(["control-plane"])} selectedScenarios={new Set(["writer"])} onChange={onChange} />);
    expect(await screen.findByText("2 scenarios · 1 resources")).toBeInTheDocument();
    expect(screen.getByText("Required for recovery · always included")).toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Supervise control-plane" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("checkbox", { name: "Supervise writer" }));
    expect(onChange).not.toHaveBeenCalled();
    const confirm = screen.getByRole("button", { name: "Confirm supervision set" });
    await waitFor(() => expect(confirm).toBeEnabled());
    fireEvent.click(confirm);
    expect(onChange).toHaveBeenCalledWith(["control-plane", "writer"], [], [], "r1");
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
    expect(await screen.findByRole("checkbox", { name: "Enable ollama" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Enable postgres" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Enable qdrant" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("checkbox", { name: "Enable ollama" }));
    expect(onResourceToggle).toHaveBeenCalledWith("ollama", false);
    fireEvent.click(screen.getByRole("checkbox", { name: "Enable qdrant" }));
    expect(onResourceToggle).toHaveBeenCalledWith("qdrant", true);
    expect(screen.getByRole("checkbox", { name: "Enable postgres" })).toBeDisabled();
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
    // Once the save resolves the secret is gone from the form, however the
    // credential row re-renders after the refetch.
    await waitFor(() => expect(screen.queryByDisplayValue("secret-value")).toBeNull());

    renderWithProviders(<StepReady />);
    expect(await screen.findByText("Host requirements")).toBeInTheDocument();
    expect(screen.getByText("Integrations")).toBeInTheDocument();
    expect(screen.getByText("alpha/github-oauth")).toBeInTheDocument();
  });

  it("reveals a stored credential only on explicit action and hides it again", async () => {
    credentialsApi.fetchCredentials.mockResolvedValueOnce({
      credentials: [{ resource: "landing-page-business-suite", logical_id: "vrooli/landing-page-business-suite", field: "admin-default-password", label: "Seeded admin password", required: true, provisioning: "operator", status: "configured" }],
      count: 1,
    });
    renderWithProviders(<StepCredentials />);

    const revealButton = await screen.findByTestId("credential-reveal");
    expect(screen.queryByTestId("credential-revealed-value")).toBeNull();

    fireEvent.click(revealButton);
    await waitFor(() => expect(credentialsApi.revealCredential).toHaveBeenCalledWith({ logical_id: "vrooli/landing-page-business-suite", field: "admin-default-password" }, "local"));
    expect(await screen.findByTestId("credential-revealed-value")).toHaveValue("revealed-secret-value");

    fireEvent.click(revealButton);
    await waitFor(() => expect(screen.queryByTestId("credential-revealed-value")).toBeNull());
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
    fireEvent.click(await screen.findByRole("button", { name: "mode" }));
    fireEvent.click(screen.getByRole("option", { name: "enforce" }));
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
      blockers: [],
      degraded: [],
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
    renderWithProviders(<StepCredentials target="remote" />);
    await openCredentialDetails();
    expect(await screen.findByTestId("capability-card-demo-capability")).toHaveTextContent("Protect a demo artifact");
    fireEvent.change(screen.getByLabelText("Destination"), { target: { value: "/mnt/approved" } });
    fireEvent.click(screen.getByTestId("capability-confirm-demo-capability"));
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    await waitFor(() => expect(capabilitiesApi.previewCapability).toHaveBeenCalledWith({ capability_id: "demo-capability", confirm: false, inputs: { destination: "/mnt/approved" } }, "remote"));
    fireEvent.click(await screen.findByRole("button", { name: "Apply reviewed capability" }));
    await waitFor(() => expect(capabilitiesApi.applyCapability).toHaveBeenCalledWith({ capability_id: "demo-capability", confirm: true, inputs: { destination: "/mnt/approved" } }, "remote"));
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
    await openCredentialDetails();
    expect(await screen.findByTestId("capability-card-durable-backup-evidence")).toHaveTextContent("recovery-drill · verified");
    expect(screen.getByTestId("capability-card-durable-backup-evidence")).toHaveTextContent("run a recovery drill");
    expect(screen.queryByRole("button", { name: "Preview" })).not.toBeInTheDocument();
  });

  it("binds capability verification to the selected target", async () => {
    capabilitiesApi.fetchCapabilities.mockResolvedValueOnce({
      count: 1,
      capabilities: [{
        descriptor: {
          version: "operator-capability/v1", id: "target-verification", owner: "demo.owner", title: "Verify a target",
          inputs: [], lifecycle: { preview: false, apply: false, verify: true, revoke: false, recover: false },
          policy: { requires_confirmation: false, idempotent: true, retryable: true }, evidence: { secret_free: true, kinds: ["demo"] },
        },
        state: "degraded", updated_at: "now",
      }],
    });
    renderWithProviders(<StepCredentials target="node-7" />);
    await openCredentialDetails();
    fireEvent.click(await screen.findByRole("button", { name: "Verify now" }));
    await waitFor(() => expect(capabilitiesApi.verifyCapability).toHaveBeenCalledWith("target-verification", "node-7"));
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
    await openCredentialDetails();
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
    await openCredentialDetails();
    expect(await screen.findByTestId("capability-card-secret-capability")).toBeInTheDocument();
    const destination = screen.getByLabelText("Destination");
    fireEvent.change(destination, { target: { value: "/mnt/approved" } });
    await waitFor(() => expect(destination).toHaveValue("/mnt/approved"));
    const passphrase = screen.getByLabelText("Passphrase");
    fireEvent.change(passphrase, { target: { value: "ephemeral-passphrase" } });
    await waitFor(() => expect(passphrase).toHaveValue("ephemeral-passphrase"));
    fireEvent.click(screen.getByTestId("capability-confirm-secret-capability"));
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    await waitFor(() => expect(capabilitiesApi.previewCapability).toHaveBeenCalledWith({ capability_id: "secret-capability", confirm: false, inputs: { destination: "/mnt/approved", passphrase: "ephemeral-passphrase" } }, "local"));
    fireEvent.click(await screen.findByRole("button", { name: "Apply reviewed capability" }));
    await waitFor(() => expect(capabilitiesApi.applyCapability).toHaveBeenCalledWith({ capability_id: "secret-capability", confirm: true, inputs: { destination: "/mnt/approved", passphrase: "ephemeral-passphrase" } }, "local"));
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
    await openCredentialDetails();
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
      blockers: [],
      degraded: [],
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
      { name: "firewall", state: ApplyStepState.NEEDS_ELEVATION, legacyOutcome: "needs_elevation", error: "permission denied" },
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

  it("surfaces an interrupted run as uncertain and blocks retry until target evidence resolves it", async () => {
    const initialApplyCalls = applyApi.startApply.mock.calls.length;
    applyApi.fetchApplyPlan.mockResolvedValueOnce({ target: "local", plan_id: "retained-plan", plan_digest: "retained-digest", revision: "r3", items: [
      { id: "resource:postgres", kind: "resource", name: "postgres", required: true, privileged: false, state: "unknown" },
      { id: "scenario:writer", kind: "scenario", name: "writer", required: true, privileged: false, state: "unknown" },
    ] });
    applyApi.fetchApplyRun.mockResolvedValueOnce({
      runId: "interrupted-run",
      status: ApplyRunState.INDETERMINATE,
      legacyStatus: "interrupted",
      steps: [
        { id: "tool:git", name: "git", state: ApplyStepState.APPLIED, legacyOutcome: "applied" },
        { id: "resource:postgres", name: "postgres", state: ApplyStepState.UNSPECIFIED, legacyOutcome: "indeterminate", error: "The process stopped before confirming this item." },
        { id: "scenario:writer", name: "writer", state: ApplyStepState.PENDING, legacyOutcome: "pending" },
      ],
    });
    window.localStorage.setItem("vrooli-onboarding.apply-run.local", "interrupted-run");

    renderWithProviders(<StepApply />);
    expect(await screen.findByTestId("apply-state")).toHaveTextContent("The run outcome is unknown");
    expect(screen.getByTestId("apply-progress")).toHaveTextContent("The process stopped before confirming this item.");
    expect(screen.getByTestId("apply-retry-blocked")).toHaveTextContent("retry is available only after each uncertain item is observed satisfied or removed");
    expect(screen.queryByTestId("retry")).not.toBeInTheDocument();
    expect(applyApi.startApply).toHaveBeenCalledTimes(initialApplyCalls);
  });

  it("blocks retry when an interrupted run contains an unknown outcome", async () => {
    applyApi.fetchApplyRun.mockResolvedValueOnce({
      runId: "future-outcome-run",
      status: ApplyRunState.INDETERMINATE,
      legacyStatus: "interrupted",
      steps: [{ id: "resource:postgres", name: "postgres", state: ApplyStepState.UNSPECIFIED, legacyOutcome: "future_outcome" }],
    });
    window.localStorage.setItem("vrooli-onboarding.apply-run.local", "future-outcome-run");

    renderWithProviders(<StepApply />);
    expect(await screen.findByTestId("apply-state")).toHaveTextContent("The run outcome is unknown");
    expect(screen.getByTestId("apply-retry-blocked")).toBeInTheDocument();
    expect(screen.queryByTestId("retry")).not.toBeInTheDocument();
  });

  it("recovers an interrupted apply through recheck, review, and retry", async () => {
    let currentPlan = { target: "local", plan_id: "recovery-plan-r1", plan_digest: "digest-r1", revision: "r1", items: [
      { id: "resource:postgres", kind: "resource", name: "postgres", required: true, privileged: false, state: "unknown" },
      { id: "scenario:writer", kind: "scenario", name: "writer", required: true, privileged: false, state: "unknown" },
    ] };
    applyApi.fetchApplyPlan.mockImplementation(async () => currentPlan);
    applyApi.fetchApplyRun.mockResolvedValueOnce({
      runId: "recheck-run",
      status: ApplyRunState.INDETERMINATE,
      legacyStatus: "interrupted",
      steps: [
        { id: "resource:postgres", name: "postgres", state: ApplyStepState.UNSPECIFIED, legacyOutcome: "indeterminate" },
        { id: "scenario:writer", name: "writer", state: ApplyStepState.UNSPECIFIED, legacyOutcome: "indeterminate" },
      ],
    });
    applyApi.reviewApply.mockResolvedValueOnce({ target: "local", planId: "recovery-plan-r3", planDigest: "digest-r3", revision: "r3", consentReceiptId: "recovery-consent" });
    window.localStorage.setItem("vrooli-onboarding.apply-run.local", "recheck-run");

    renderWithProviders(<StepApply />);
    expect(await screen.findByTestId("apply-retry-blocked")).toBeInTheDocument();
    expect(screen.queryByTestId("retry")).not.toBeInTheDocument();

    let callsBeforeRecheck = applyApi.fetchApplyPlan.mock.calls.length;
    currentPlan = { ...currentPlan, plan_id: "recovery-plan-r2", plan_digest: "digest-r2", revision: "r2", items: [
      { id: "resource:postgres", kind: "resource", name: "postgres", required: true, privileged: false, state: "satisfied" },
      { id: "scenario:writer", kind: "scenario", name: "writer", required: true, privileged: false, state: "unknown" },
    ] };
    fireEvent.click(screen.getByTestId("apply-recheck"));
    await waitFor(() => expect(applyApi.fetchApplyPlan.mock.calls.length).toBeGreaterThan(callsBeforeRecheck));
    expect(await screen.findByTestId("apply-plan-summary")).toHaveTextContent("1 already in place, 1 not sampled");
    expect(screen.getByTestId("apply-retry-blocked")).toBeInTheDocument();
    expect(screen.queryByTestId("retry")).not.toBeInTheDocument();

    callsBeforeRecheck = applyApi.fetchApplyPlan.mock.calls.length;
    currentPlan = { ...currentPlan, plan_id: "recovery-plan-r3", plan_digest: "digest-r3", revision: "r3", items: [
      { id: "resource:postgres", kind: "resource", name: "postgres", required: true, privileged: false, state: "satisfied" },
      { id: "scenario:writer", kind: "scenario", name: "writer", required: true, privileged: false, state: "satisfied" },
    ] };
    fireEvent.click(screen.getByTestId("apply-recheck"));
    expect(await screen.findByTestId("retry")).toBeInTheDocument();
    expect(screen.queryByTestId("apply-retry-blocked")).not.toBeInTheDocument();
    await waitFor(() => expect(applyApi.fetchApplyPlan.mock.calls.length).toBeGreaterThan(callsBeforeRecheck));

    fireEvent.click(screen.getByTestId("retry"));
    await waitFor(() => expect(applyApi.startApply).toHaveBeenCalled());
    expect(applyApi.reviewApply).toHaveBeenCalledWith({ target: "local", plan_id: "recovery-plan-r3", plan_digest: "digest-r3", expected_revision: "r3" });
    expect(applyApi.startApply).toHaveBeenCalledWith(expect.objectContaining({
      target: "local", plan_id: "recovery-plan-r3", plan_digest: "digest-r3", expected_revision: "r3", consent_receipt_id: "recovery-consent",
    }));
  });

  it("enables retry when rechecked plan no longer selects the uncertain item", async () => {
    applyApi.fetchApplyPlan.mockResolvedValueOnce({ target: "local", plan_id: "plan-1", plan_digest: "digest-1", revision: "r1", items: [
      { id: "scenario:writer", kind: "scenario", name: "writer", required: true, privileged: false, state: "unknown" },
    ] });
    applyApi.fetchApplyPlan.mockResolvedValueOnce({ target: "local", plan_id: "plan-2", plan_digest: "digest-2", revision: "r2", items: [] });
    applyApi.fetchApplyRun.mockResolvedValueOnce({
      runId: "removed-run",
      status: ApplyRunState.INDETERMINATE,
      legacyStatus: "interrupted",
      steps: [{ id: "scenario:writer", name: "writer", state: ApplyStepState.UNSPECIFIED, legacyOutcome: "indeterminate" }],
    });
    window.localStorage.setItem("vrooli-onboarding.apply-run.local", "removed-run");

    renderWithProviders(<StepApply />);
    expect(await screen.findByTestId("apply-retry-blocked")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("apply-recheck"));
    expect(await screen.findByTestId("retry")).toBeInTheDocument();
  });

  it("reports credential provisioning failure without clearing the input", async () => {
    credentialsApi.provisionCredential.mockRejectedValueOnce(new Error("authority unavailable"));
    renderWithProviders(<StepCredentials />);
    const input = await screen.findByLabelText("Value for OpenRouter key");
    fireEvent.change(input, { target: { value: "secret-value" } });
    fireEvent.click(screen.getByRole("button", { name: "Save securely" }));
    expect(await screen.findByRole("alertdialog")).toHaveTextContent("Could not save this credential");
    expect(input).toHaveValue("secret-value");
  });

	it("keeps credential rows informative across provider-supplied and verified states", async () => {
    readinessApi.fetchReadiness.mockResolvedValueOnce({
      status: "degraded", scenarios: [], resources: [], hosts: [], integrations: [], checked_at: "now", blockers: [], degraded: [],
      credentials: [
        { resource: "derived", logical_id: "derived", field: "endpoint", label: "Derived endpoint", required: true, provisioning: "derived", derived_from: "gateway", status: "configured", evidence_status: "verified", evidence_detail: "Verified by the owning component." },
        { resource: "generated", logical_id: "generated", field: "token", label: "Generated token", required: false, provisioning: "generated", status: "configured" },
        { resource: "pending", logical_id: "pending", field: "key", label: "Pending key", required: true, status: "pending", detail: "Still checking." },
        { resource: "stored", logical_id: "stored", field: "key", label: "Stored key", required: true, status: "configured", evidence_status: "unverified", evidence_detail: "Verification is required before use." },
      ],
      });
    credentialsApi.fetchCredentials.mockResolvedValueOnce({ credentials: [
      { resource: "derived", logical_id: "derived", field: "endpoint", label: "Derived endpoint", required: true, provisioning: "derived", derived_from: "gateway", status: "configured" },
      { resource: "generated", logical_id: "generated", field: "token", label: "Generated token", required: false, provisioning: "generated", status: "configured", description: "Created by onboarding." },
      { resource: "pending", logical_id: "pending", field: "key", label: "Pending key", required: true, status: "pending", detail: "Still checking.", evidence_detail: "The provider has not responded yet." },
      { resource: "stored", logical_id: "stored", field: "key", label: "Stored key", required: true, status: "configured", evidence_status: "unverified", evidence_detail: "Verification is required before use." },
    ], count: 3 });
    renderWithProviders(<StepCredentials />);

    expect(await screen.findAllByTestId("credential-card")).toHaveLength(4);
    expect(screen.getByTestId("credential-outstanding-count")).toHaveTextContent("2");
    expect(screen.getByText("Provided by gateway after its source credential is available.")).toBeInTheDocument();
    expect(screen.getByText("Generated by the owning component on first start. There is nothing to enter.")).toBeInTheDocument();
    expect(screen.getByText("Still checking.")).toBeInTheDocument();
		expect(screen.getByLabelText("Value for Pending key")).toBeInTheDocument();
	});

	it("keeps provider unavailability distinct from a missing credential", async () => {
		readinessApi.fetchReadiness.mockResolvedValueOnce({
			status: "unsupported", scenarios: [], resources: [], hosts: [], integrations: [], checked_at: "now", blockers: [], degraded: [],
			credentials: [{ resource: "native-store", logical_id: "native-store", field: "token", label: "Native token", required: true, status: "unsupported", detail: "secure store is locked", provider_state: "unavailable", provider_detail: "secure store is locked", evidence_detail: "The credential address cannot be evaluated until its provider is available; readiness remains unknown." }],
		});
		credentialsApi.fetchCredentials.mockResolvedValueOnce({ credentials: [{ resource: "native-store", logical_id: "native-store", field: "token", label: "Native token", required: true, status: "unsupported" }], count: 1 });
		renderWithProviders(<StepCredentials />);

		expect(await screen.findByTestId("credential-card")).toHaveTextContent("secure store is locked");
	});

	it("renders the credential configuration provider diagnosis and refresh action", async () => {
    readinessApi.fetchReadiness.mockResolvedValueOnce({
      status: "degraded", scenarios: [], resources: [], hosts: [], integrations: [], checked_at: "now", blockers: [], degraded: [],
      credentials: [], credential_diagnosis: { provider: { backend: "verified escrow", condition: "attention", explanation: "The escrow is locked.", fix: "Unlock the escrow.", write_condition: "write unavailable", write_fix: "Reconnect the escrow." } },
    });
    renderWithProviders(<StepCredentials />);
    await openCredentialDetails();
    fireEvent.click(screen.getByRole("tab", { name: "Secure store" }));
    expect(await screen.findByTestId("credential-configuration")).toHaveTextContent("verified escrow");
  });

  it("keeps credential inputs on the page and defers configuration work to the dialog", async () => {
    capabilitiesApi.fetchCapabilities.mockClear();
    operatorInputsApi.fetchOperatorInputs.mockClear();
    renderWithProviders(<StepCredentials />);
    expect(await screen.findByTestId("credential-card")).toHaveTextContent("OpenRouter key");
    expect(capabilitiesApi.fetchCapabilities).not.toHaveBeenCalled();
    expect(operatorInputsApi.fetchOperatorInputs).not.toHaveBeenCalled();

    await openCredentialDetails();
    const dialog = await screen.findByRole("dialog", { name: "Credential setup details" });
    expect(within(dialog).queryByTestId("credential-card")).not.toBeInTheDocument();
    await waitFor(() => expect(capabilitiesApi.fetchCapabilities).toHaveBeenCalledWith("local"));
    await waitFor(() => expect(operatorInputsApi.fetchOperatorInputs).toHaveBeenCalledWith("local"));
  });

  it("shows a stable credential skeleton while readiness and inventory are loading", async () => {
    readinessApi.fetchReadiness.mockImplementationOnce(() => new Promise(() => undefined));
    credentialsApi.fetchCredentials.mockImplementationOnce(() => new Promise(() => undefined));
    renderWithProviders(<StepCredentials />);
    expect(await screen.findByRole("status", { name: /Checking credential requirements/ })).toBeInTheDocument();
  });

  it("renders and submits every declared operator-input kind", async () => {
    operatorInputsApi.fetchOperatorInputs.mockResolvedValueOnce({
      requests: [
        { id: "secret", kind: 1, title: "Secret", description: "Sensitive", required: true, options: [], candidates: [], validation: "required", declinable: true },
        { id: "choice", kind: 2, title: "Choice", description: "Pick one", required: true, options: ["one", "two"], candidates: [], validation: "select" },
        { id: "confirm", kind: 3, title: "Confirm", description: "Confirm it", required: false, options: [], candidates: [] },
        { id: "path", kind: 4, title: "Path", description: "Path", required: false, options: [], candidates: [] },
        { id: "enum", kind: 5, title: "Enum", description: "Enum", required: false, options: ["value"], candidates: [] },
        { id: "boolean", kind: 6, title: "Boolean", description: "Boolean", required: false, options: [], candidates: [] },
        { id: "duration", kind: 7, title: "Duration", description: "Duration", required: false, options: [], candidates: [] },
        { id: "confirmation", kind: 8, title: "Confirmation", description: "Confirmation", required: false, options: [], candidates: [] },
        { id: "unknown", kind: 99, title: "Unknown", description: "Fallback", required: false, options: [], candidates: [] },
      ],
    });
    readinessApi.fetchReadiness.mockResolvedValueOnce({ status: "ready", scenarios: [], resources: [], credentials: [], hosts: [], integrations: [], checked_at: "now", blockers: [], degraded: [] });
    renderWithProviders(<StepCredentials target="remote" />);
    await openCredentialDetails("questions");
    expect(await screen.findByTestId("target-question-set")).toHaveTextContent("Secret");
    expect(screen.getByLabelText("Secret")).toBeInTheDocument();
    expect(screen.getByText("Choice")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Secret"), { target: { value: "temporary" } });
    fireEvent.click(screen.getByRole("button", { name: "Submit secret answers" }));
    await waitFor(() => expect(operatorInputsApi.resolveOperatorInputs).toHaveBeenCalledWith(expect.arrayContaining([expect.objectContaining({ requestId: "secret", value: "temporary", declined: false })]), "remote"));
    expect(await screen.findByTestId("target-question-status")).toHaveTextContent("Answers submitted");
  });

  it("keeps unsaved secret answers across route remounts without drafting them", async () => {
    draftApi.fetchDraft.mockClear();
    draftApi.saveDraft.mockClear();
    operatorInputsApi.resolveOperatorInputs.mockReset();
    operatorInputsApi.fetchOperatorInputs.mockResolvedValue({
      requests: [{ id: "secret", kind: 1, title: "Secret", description: "Sensitive", required: true, options: [], candidates: [], validation: "required" }],
    });

    const firstRoute = renderWithProviders(<StepCredentials target="remote" />);
    await openCredentialDetails("questions");
    const input = await screen.findByLabelText("Secret");
    fireEvent.change(input, { target: { value: "temporary-secret" } });
    firstRoute.unmount();

    const recoveredRoute = renderWithProviders(<StepCredentials target="remote" />);
    await openCredentialDetails("questions");
    const recoveredInput = await screen.findByLabelText("Secret");
    await waitFor(() => expect(recoveredInput).toHaveValue("temporary-secret"));
    expect(window.localStorage.length).toBe(0);
    expect(draftApi.saveDraft).not.toHaveBeenCalled();

    operatorInputsApi.resolveOperatorInputs.mockRejectedValueOnce(new Error("temporary target write failure"));
    fireEvent.click(screen.getByRole("button", { name: "Submit secret answers" }));
    expect(await screen.findByTestId("target-question-status")).toHaveTextContent("target rejected these answers");
    expect(recoveredInput).toHaveValue("temporary-secret");

    operatorInputsApi.resolveOperatorInputs.mockResolvedValueOnce({ configurationPending: false, outcomes: [] });
    fireEvent.click(screen.getByRole("button", { name: "Submit secret answers" }));
    await waitFor(() => expect(screen.getByTestId("target-question-status")).toHaveTextContent("Answers submitted"));
    expect(recoveredInput).toHaveValue("");
    expect(window.localStorage.length).toBe(0);
    for (const [request] of draftApi.saveDraft.mock.calls) {
      expect(JSON.stringify(request.choices)).not.toContain("temporary-secret");
    }
  });

  it("recovers target questions through draft outage, remount, and same-target retry", async () => {
    operatorInputsApi.resolveOperatorInputs.mockClear();
    const question = { id: "region", kind: "choice", defaultValue: "" };
    const questionRequest = { id: "region", kind: 2, title: "Region", description: "Choose a region", required: true, options: ["eu-west", "asia", "us-east"], candidates: [], validation: "select" };
    const drafts: Record<string, { target: string; actor: string; baseRevision: string; revision: string; stepId: string; choices: Record<string, string> }> = {
      remote: { target: "remote", actor: "onboarding-questionnaire", baseRevision: "remote-base", revision: "remote-r1", stepId: "credentials", choices: encodeOperatorQuestionDraft([question], { region: "eu-west" }, {}) },
      local: { target: "local", actor: "onboarding-questionnaire", baseRevision: "local-base", revision: "local-r1", stepId: "credentials", choices: encodeOperatorQuestionDraft([question], { region: "us-east" }, {}) },
    };
    let loseFirstRemoteRead = true;
    draftApi.fetchDraft.mockImplementation(async (target: string) => {
      if (target === "remote" && loseFirstRemoteRead) {
        loseFirstRemoteRead = false;
        throw new Error("temporary draft owner outage");
      }
      const draft = drafts[target];
      if (!draft) throw new Error(`missing draft for ${target}`);
      return { ...draft, choices: { ...draft.choices } };
    });
    draftApi.saveDraft.mockImplementation(async (request: { target: string; actor: string; baseRevision: string; expectedRevision?: string; stepId?: string; choices?: Record<string, string> }) => {
      const current = drafts[request.target];
      if (!current) throw new Error(`missing draft for ${request.target}`);
      const saved = { ...current, ...request, revision: `${request.target}-r2`, stepId: request.stepId ?? current.stepId, choices: { ...(request.choices ?? {}) } };
      drafts[request.target] = saved;
      return saved;
    });
    operatorInputsApi.fetchOperatorInputs.mockResolvedValue({ requests: [questionRequest] });
    operatorInputsApi.resolveOperatorInputs.mockRejectedValueOnce(new Error("selected target provider unavailable"));

    const firstRoute = renderWithProviders(<StepCredentials target="remote" />);
    await openCredentialDetails("questions");
    expect(await screen.findByRole("alert")).toHaveTextContent("Saved setup answers could not be loaded");
    fireEvent.click(screen.getByRole("button", { name: "Retry restoring answers" }));
    const remoteInput = await screen.findByTestId("forms.generated-form");
    expect(remoteInput).toHaveValue("eu-west");

    fireEvent.change(remoteInput, { target: { value: "asia" } });
    await waitFor(() => expect(screen.getByTestId("question-draft-status")).toHaveTextContent("Non-secret answers are saved for this target"));
    firstRoute.unmount();

    const resumedRoute = renderWithProviders(<StepCredentials target="remote" />);
    await openCredentialDetails("questions");
    const resumedRemoteInput = await screen.findByTestId("forms.generated-form");
    expect(resumedRemoteInput).toHaveValue("asia");
    resumedRoute.rerender(<StepCredentials target="local" />);
    await waitFor(() => expect(screen.getByTestId("forms.generated-form")).toHaveValue("us-east"));
    resumedRoute.rerender(<StepCredentials target="remote" />);
    await waitFor(() => expect(screen.getByTestId("forms.generated-form")).toHaveValue("asia"));

    fireEvent.click(screen.getByRole("button", { name: "Submit answers" }));
    expect(await screen.findByTestId("target-question-status")).toHaveTextContent("target rejected these answers");
    expect(screen.getByTestId("forms.generated-form")).toHaveValue("asia");
    operatorInputsApi.resolveOperatorInputs.mockResolvedValueOnce({ configurationPending: false, outcomes: [] });
    fireEvent.click(screen.getByRole("button", { name: "Submit answers" }));
    await waitFor(() => expect(screen.getByTestId("target-question-status")).toHaveTextContent("Answers submitted"));
    expect(operatorInputsApi.resolveOperatorInputs).toHaveBeenNthCalledWith(1, expect.arrayContaining([expect.objectContaining({ requestId: "region", value: "asia" })]), "remote");
    expect(operatorInputsApi.resolveOperatorInputs).toHaveBeenNthCalledWith(2, expect.arrayContaining([expect.objectContaining({ requestId: "region", value: "asia" })]), "remote");
    expect(draftApi.fetchDraft.mock.calls.map(([target]) => target)).toContain("local");
    expect(draftApi.fetchDraft.mock.calls.map(([target]) => target)).toContain("remote");
  });


});
