import { describe, expect, it, vi } from "vitest";

const rpc = vi.hoisted(() => {
  const methods = [
    "reviewApply", "startApply", "getApplyRun", "cancelApply", "getApplyPlan",
    "listCapabilities", "getCapabilityStatus", "previewCapability", "applyCapability",
    "searchConfiguration", "listCredentials", "provisionCredential", "diagnoseCredentials",
    "searchGlossary", "getHostFacts", "listHostRequirements", "listTargets",
    "patchHostSafeguardConfig", "setNotificationRecipient", "listOperatorInputs",
    "resolveOperatorInputs", "getOperatorState", "patchOperatorState", "getReadiness",
    "acknowledgeDegradedReadiness", "listResources", "getResource", "getResourceHealth",
    "listDerivedResources", "listScenarios", "getCoreSet", "getRecommendation",
    "acceptRecommendation", "getClosure", "getUnion", "getSession", "advanceSessionStep",
    "listProfiles", "evaluateProfile",
    "getStepModel", "getDraft", "saveDraft", "discardDraft",
  ] as const;
  return Object.fromEntries(methods.map((method) => [method, vi.fn()])) as { [K in typeof methods[number]]: ReturnType<typeof vi.fn> };
});

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: ({ appendSuffix = false }: { appendSuffix?: boolean } = {}) =>
    appendSuffix ? "http://onboarding.test/api/v1/" : "http://onboarding.test/api/v1",
  createScenarioConnectTransport: () => ({}),
  buildApiUrl: (path: string, options: { baseUrl: string }) => `${options.baseUrl}${path}`,
}));

vi.mock("@connectrpc/connect", () => ({ createClient: () => rpc }));

vi.mock("@bufbuild/protobuf", async () => ({
  ...(await vi.importActual<typeof import("@bufbuild/protobuf")>("@bufbuild/protobuf")),
  fromJson: (_schema: unknown, value: unknown) => ({ jsonValue: value }),
}));

import { CapabilityState } from "@vrooli/proto-types/vrooli-onboarding/v1/capabilities/capabilities_pb";
import { SafeguardDisposition } from "@vrooli/proto-types/vrooli-onboarding/v1/host/host_pb";
import type { Answer } from "@vrooli/proto-types/vrooli-onboarding/v1/operatorinputs/operatorinputs_pb";
import { ReadinessState } from "@vrooli/proto-types/vrooli-onboarding/v1/readiness/readiness_pb";
import { API_BASE, REST_API_BASE, onboardingTransport } from "./base";
import { cancelApply, fetchApplyPlan, fetchApplyRun, reviewApply, startApply } from "./apply";
import { applyCapability, fetchCapabilities, fetchCapabilityStatus, previewCapability } from "./capabilities";
import { searchConfiguration } from "./configuration";
import { diagnoseCredentials, fetchCredentials, provisionCredential } from "./credentials";
import { fetchGlossary } from "./glossary";
import { fetchHealth } from "./health";
import { fetchHostFacts, fetchHostRequirements, fetchTargets, patchHostSafeguardConfig, setNotificationRecipient } from "./host";
import { fetchOperatorInputs, resolveOperatorInputs } from "./operatorinputs";
import { fetchOperatorState, saveOperatorState } from "./operatorstate";
import { acknowledgeDegraded, fetchReadiness } from "./readiness";
import { fetchDerivedResources, fetchResource, fetchResourceHealth, fetchResources } from "./resources";
import { evaluateProfile, fetchProfiles } from "./profiles";
import { acceptRecommendation, fetchClosure, fetchCoreSet, fetchRecommendation, fetchScenarios, fetchUnion } from "./selection";
import { advanceSessionStep, discardDraft, fetchDraft, fetchSession, fetchStepModel, saveDraft } from "./session";

const stamp = { seconds: 1_700_000_000n, nanos: 0 };

function resetRpc() {
  for (const method of Object.values(rpc)) method.mockReset();
}

describe("typed onboarding API adapters", () => {
  it("uses deployment-aware API bases and transports", () => {
    expect(API_BASE).toBe("http://onboarding.test/api/v1");
    expect(REST_API_BASE).toBe("http://onboarding.test/api/v1/");
    expect(onboardingTransport()).toEqual({});
  });

  it("maps apply requests and plan responses at the wire boundary", async () => {
    resetRpc();
    rpc.reviewApply.mockResolvedValue({ planId: "p" });
    rpc.startApply.mockResolvedValue({ runId: "r" });
    rpc.getApplyRun.mockResolvedValue({ runId: "r" });
    rpc.cancelApply.mockResolvedValue({ cancelled: true });
    rpc.getApplyPlan.mockResolvedValue({
      items: [{ id: "i", kind: "resource", name: "postgres", required: true, privileged: false, observedState: "ready" }],
      target: "remote", planId: "p", planDigest: "digest", revision: "4", expiresAt: stamp,
    });
    await reviewApply({ target: "remote", plan_id: "p", plan_digest: "digest", expected_revision: "3" });
    await startApply({ target: "remote", plan_id: "p", plan_digest: "digest", expected_revision: "3", consent_receipt_id: "c", idempotency_key: "k" });
    await fetchApplyRun("r", "remote");
    await cancelApply("r", "remote");
    await expect(fetchApplyPlan("remote")).resolves.toMatchObject({
      target: "remote", plan_id: "p", plan_digest: "digest", revision: "4", expires_at: new Date(1_700_000_000_000).toISOString(),
      items: [{ id: "i", state: "ready" }],
    });
    expect(rpc.reviewApply).toHaveBeenCalledWith({ target: "remote", planId: "p", planDigest: "digest", expectedRevision: "3" });
    expect(rpc.startApply).toHaveBeenCalledWith({ target: "remote", planId: "p", planDigest: "digest", expectedRevision: "3", consentReceiptId: "c", idempotencyKey: "k" });
    rpc.getApplyPlan.mockResolvedValueOnce({ items: [], target: "local", planId: "", planDigest: "", revision: "", expiresAt: undefined });
    await expect(fetchApplyPlan()).resolves.toMatchObject({ items: [], expires_at: undefined });
  });

  it("maps capability status, preview, and result metadata", async () => {
    resetRpc();
    const candidate = { id: "candidate", kind: "tool", label: "Tool", location: "/bin/tool", stableIdentity: "tool@1", deviceIdentity: "node", writable: true, physicalIndependence: "independent", status: "ready", risk: "low", remediation: "none", metadata: { source: "host" } };
    const evidence = { kind: "probe", artifactIdentity: "artifact", sourceGeneration: "g1", checksum: "sha", coverage: ["host"], observedAt: stamp, verified: true, remediation: "none" };
    const descriptor = {
      version: "1", id: "cap", owner: "provider", title: "Capability", description: "description", risk: "low",
      scope: "host", purpose: "purpose", sensitivity: "none", disposition: "available", dispositionReason: "reason", referenceUrl: "https://example.test",
      applicability: { platforms: ["linux"], environments: ["dev"], targets: ["local"] },
      provenance: { requester: "operator", scope: "scope", grantSource: "local", revocationLimit: "session" },
      lifecycle: { preview: true, apply: true, verify: true, revoke: false, recover: true, recovery: "retry" },
      inputs: [{ id: "input", kind: "choice", label: "Input", description: "desc", required: true, declinable: true, options: ["one"], defaultValue: "one", candidates: [candidate], validation: "valid", constraints: { minLength: 1, maxLength: 4, minDuration: "1s", maxDuration: "2s" } }],
      prerequisites: ["ready"], policy: { requiresConfirmation: true, idempotent: true, retryable: true, protectedRoots: ["/tmp"], remediation: "none" },
      evidence: { kinds: ["probe"], requiredFields: ["checksum"], secretFree: true, freshness: "1h" }, remediation: "none",
    };
    const status = { descriptor, state: CapabilityState.READY, candidates: [candidate], missingInputs: [], evidence: [evidence], remediation: "none", updatedAt: stamp };
    rpc.listCapabilities.mockResolvedValue({ capabilities: [status], count: 1 });
    rpc.getCapabilityStatus.mockResolvedValue({ statuses: [status], count: 1 });
    rpc.previewCapability.mockResolvedValue({ capabilityId: "cap", planId: "plan", state: CapabilityState.READY_TO_PREVIEW, mutations: [{ id: "m", summary: "change", reversible: true }], candidates: [candidate], remediation: "none", expiresAt: stamp });
    rpc.applyCapability.mockResolvedValue({ capabilityId: "cap", state: CapabilityState.READY, outcome: "applied", retryable: false, errorCode: "", remediation: "none", evidence: [evidence], mutations: [{ id: "m", summary: "change", reversible: true }], completedAt: stamp });
    await expect(fetchCapabilities("remote")).resolves.toMatchObject({ count: 1, capabilities: [{ state: "ready", descriptor: { id: "cap", policy: { requires_confirmation: true }, inputs: [{ constraints: { min_length: 1 } }] }, evidence: [{ observed_at: new Date(1_700_000_000_000).toISOString() }] }] });
    await expect(fetchCapabilityStatus()).resolves.toMatchObject({ capabilities: [{ state: "ready" }] });
    await expect(previewCapability({ capability_id: "cap", idempotency_key: "k", confirm: true, inputs: { choice: "one" } })).resolves.toMatchObject({ capability_id: "cap", state: "ready_to_preview" });
    await expect(applyCapability({ capability_id: "cap", confirm: true, inputs: {} }, "remote")).resolves.toMatchObject({ capability_id: "cap", state: "ready", outcome: "applied", completed_at: new Date(1_700_000_000_000).toISOString() });
    expect(rpc.previewCapability).toHaveBeenCalledWith({ target: "local", action: { capabilityId: "cap", idempotencyKey: "k", confirm: true, inputs: { choice: "one" } } });
  });

  it("keeps sparse provider responses safe and preserves every capability state", async () => {
    resetRpc();
    const sparseDescriptor = { inputs: [], policy: {}, evidence: {} };
    const states = [CapabilityState.DISCOVERED, CapabilityState.NEEDS_OPERATOR_INPUT, CapabilityState.READY_TO_PREVIEW, CapabilityState.APPLYING, CapabilityState.VERIFYING, CapabilityState.READY, CapabilityState.RETRYABLE_FAILURE, CapabilityState.DEGRADED, CapabilityState.UNSUPPORTED, CapabilityState.UNSPECIFIED];
    rpc.listCapabilities.mockResolvedValue({ capabilities: states.map((state, index) => ({ descriptor: index === 0 ? undefined : sparseDescriptor, state, candidates: [], missingInputs: [], evidence: [{ kind: "probe", artifactIdentity: String(index), verified: false, coverage: undefined, observedAt: undefined }], remediation: undefined, updatedAt: undefined })), count: states.length });
    await expect(fetchCapabilities()).resolves.toMatchObject({ capabilities: [
      { state: "discovered", descriptor: { version: "" } },
      { state: "needs_operator_input" }, { state: "ready_to_preview" }, { state: "applying" }, { state: "verifying" },
      { state: "ready" }, { state: "retryable_failure" }, { state: "degraded" }, { state: "unsupported" }, { state: "discovered" },
    ] });
    rpc.previewCapability.mockResolvedValue({ capabilityId: "cap", planId: "plan", state: CapabilityState.UNSPECIFIED, mutations: [], candidates: [], remediation: undefined, expiresAt: undefined });
    rpc.applyCapability.mockResolvedValue({ capabilityId: "cap", state: CapabilityState.UNSPECIFIED, outcome: "unknown", retryable: false, errorCode: undefined, remediation: undefined, evidence: [], mutations: [], completedAt: undefined });
    await expect(previewCapability({ capability_id: "cap", confirm: false, inputs: {} })).resolves.toEqual({ capability_id: "cap", plan_id: "plan", state: "discovered", mutations: [], candidates: [], remediation: undefined, expires_at: undefined });
    await expect(applyCapability({ capability_id: "cap", confirm: false, inputs: {} })).resolves.toMatchObject({ state: "discovered", completed_at: undefined });
  });

  it("forwards the remaining typed domain calls with their public request shape", async () => {
    resetRpc();
    rpc.searchConfiguration.mockResolvedValue({ results: [{ id: "config" }] });
    rpc.listCredentials.mockResolvedValue({ credentials: [] });
    rpc.provisionCredential.mockResolvedValue({ status: "provisioned" });
    rpc.diagnoseCredentials.mockResolvedValue({ status: "ready" });
    rpc.searchGlossary.mockResolvedValue({ entries: [] });
    rpc.getHostFacts.mockResolvedValue({ available: true, reason: "", cpuCount: 4, memoryTotalBytes: 10n, memoryAvailableBytes: 5n, diskFreeBytes: 2n, gpus: ["gpu"], platform: "linux" });
    rpc.listHostRequirements.mockResolvedValue({ tools: [{ name: "tool", required: true, reason: "required", notes: "note", description: "desc", risk: "low", privilege: "user", bundling: "bundle", platforms: ["linux"], commands: ["tool --version"], configSchema: { type: "object" }, config: { enabled: true }, status: "ready", disposition: SafeguardDisposition.PERMISSION_DENIED, detail: "detail", remediation: "fix" }], safeguards: [] });
    rpc.listTargets.mockResolvedValue({ targets: [{ id: "node", name: "Node", status: "online", os: "linux", architecture: "amd64", kind: "control-plane", online: true, available: true, reason: "", nextAction: "none", capabilities: ["test"], scopes: ["read"], readiness: [{ identity: "health", label: "Health", passed: true, state: "ready", detail: "ok", recoveryAction: "none" }] }], error: "" });
    rpc.patchHostSafeguardConfig.mockResolvedValue({});
    rpc.setNotificationRecipient.mockResolvedValue({});
    rpc.listOperatorInputs.mockResolvedValue({ inputs: [] });
    rpc.resolveOperatorInputs.mockResolvedValue({ inputs: [] });
    rpc.getOperatorState.mockResolvedValue({ state: { version: "1" } });
    rpc.patchOperatorState.mockResolvedValue({ state: { version: "2" } });
    rpc.listResources.mockResolvedValue({ resources: [] });
    rpc.getResource.mockResolvedValue({ name: "postgres" });
    rpc.getResourceHealth.mockResolvedValue({ status: "ready" });
    rpc.listDerivedResources.mockResolvedValue({ resources: [] });
    rpc.listScenarios.mockResolvedValue({ scenarios: [] });
    rpc.getCoreSet.mockResolvedValue({ scenarios: [] });
    rpc.getRecommendation.mockResolvedValue({});
    rpc.acceptRecommendation.mockResolvedValue({});
    rpc.getClosure.mockResolvedValue({});
    rpc.getUnion.mockResolvedValue({});
    rpc.getSession.mockResolvedValue({ step: 1 });
    rpc.advanceSessionStep.mockResolvedValue({ step: 2 });
    rpc.getStepModel.mockResolvedValue({ steps: [] });
    rpc.getDraft.mockResolvedValue({ target: "local", actor: "operator" });
    rpc.saveDraft.mockResolvedValue({ target: "local", actor: "operator" });
    rpc.discardDraft.mockResolvedValue({ target: "local", actor: "operator" });

    await searchConfiguration("query", "remote");
    await fetchCredentials("remote");
    await provisionCredential({ logical_id: "demo", field: "key", value: "secret" }, "remote");
    await diagnoseCredentials();
    await fetchGlossary("term");
    await expect(fetchHostFacts()).resolves.toMatchObject({ cpu_count: 4, memory_total_bytes: 10, platform: "linux" });
    await expect(fetchHostRequirements()).resolves.toMatchObject({ tools: [{ disposition: "permission_denied", config_schema: { type: "object" } }] });
    await expect(fetchTargets()).resolves.toMatchObject({ targets: [{ id: "node", readiness: [{ recovery_action: "none" }] }] });
    await patchHostSafeguardConfig({ safeguard_name: "firewall", config_key: "enabled", value: true }, "remote");
    await setNotificationRecipient("operator@example.test", "remote");
    await fetchOperatorInputs("remote");
    const answer: Answer = { $typeName: "vrooli.vrooli_onboarding.v1.operatorinputs.Answer", requestId: "choice", value: "one", declined: false };
    await resolveOperatorInputs([answer], "remote");
    await expect(fetchOperatorState()).resolves.toEqual({ version: "1" });
    await expect(saveOperatorState({ version: "2" }, "remote")).resolves.toEqual({ version: "2" });
    rpc.getOperatorState.mockResolvedValueOnce({ state: undefined });
    rpc.patchOperatorState.mockResolvedValueOnce({ state: undefined });
    await expect(fetchOperatorState()).resolves.toEqual({});
    await expect(saveOperatorState({ version: "3" })).resolves.toEqual({});
    await fetchResources("remote");
    await fetchResource("postgres");
    await fetchResourceHealth();
    await fetchDerivedResources();
    await fetchScenarios("remote");
    await fetchCoreSet(["z", "a"], "remote");
    await fetchCoreSet();
    await fetchRecommendation();
    await acceptRecommendation("remote");
    await acceptRecommendation("remote", "develop-and-publish", ["scenario-b", "scenario-a", "scenario-a"]);
    await fetchClosure();
    await fetchUnion("remote");
    rpc.listProfiles.mockResolvedValue({ profiles: [{ id: "local-use" }] });
    rpc.evaluateProfile.mockResolvedValue({ profile: { id: "local-use" }, valid: true });
    await fetchProfiles("remote");
    await evaluateProfile("local-use", { hosting: "local" }, "remote");
    await fetchSession("remote");
    await advanceSessionStep("credentials", "remote");
    await fetchStepModel();
    await fetchDraft("remote", "operator");
    await saveDraft({ target: "remote", actor: "operator", baseRevision: "1", expectedRevision: "1", stepId: "welcome", choices: { mode: "shared" } });
    await discardDraft("remote", "operator");

    expect(rpc.searchConfiguration).toHaveBeenCalledWith({ query: "query", target: "remote" });
    expect(rpc.patchHostSafeguardConfig).toHaveBeenCalledWith({ target: "remote", safeguardName: "firewall", configKey: "enabled", value: { jsonValue: true } });
    expect(rpc.getCoreSet).toHaveBeenCalledWith({ target: "remote", seed: ["a", "z"] });
    expect(rpc.advanceSessionStep).toHaveBeenCalledWith({ target: "remote", stepId: "credentials" });
    expect(rpc.acceptRecommendation).toHaveBeenCalledWith({
      target: "remote",
      profile: "develop-and-publish",
      selection: expect.objectContaining({ scenarios: ["scenario-a", "scenario-b"] }),
    });
    expect(rpc.listProfiles).toHaveBeenCalledWith({ target: "remote" });
    expect(rpc.evaluateProfile).toHaveBeenCalledWith({
      target: "remote",
      profileId: "local-use",
      answers: { hosting: { jsonValue: "local" } },
    });
  });

  it("maps readiness states and keeps health failures explicit", async () => {
    resetRpc();
    rpc.getReadiness.mockResolvedValue({
      status: ReadinessState.DEGRADED, scenarios: ["demo"], resources: ["postgres"],
      credentials: [{ resource: "openrouter", logicalId: "key", field: "apiKey", label: "API key", description: "secret", obtainUrl: "https://example.test", required: true, provisioning: "manual", derivedFrom: "operator", legacyStatus: "missing", detail: "missing" }],
      hosts: [{ item: { name: "tool", category: "system", status: ReadinessState.MISSING, detail: "missing", remediation: "install", required: true }, kind: "tool", required: true }],
      integrations: [{ name: "bridge", category: "integration", status: ReadinessState.DEFERRED, detail: "deferred", remediation: "configure", required: false }],
      checkedAt: stamp, credentialDiagnosis: { fields: { apiKey: "missing" } },
      recovery: { receiptExists: true, exportedAt: "today", entryCount: 1, uncovered: ["x"], requiredAbsent: ["y"], requiredAbsentDetails: [{ address: "y", description: "required" }], rootCopyIssues: ["z"] },
      blockers: [{ kind: "credential", name: "key", reason: "missing", remediation: "provide" }], degraded: [{ kind: "host", name: "tool", reason: "missing", remediation: "install" }], degradedDigest: "digest", degradedAcknowledged: false,
    });
    rpc.acknowledgeDegradedReadiness.mockResolvedValue({ acknowledged: true });
    await expect(fetchReadiness("remote")).resolves.toMatchObject({ status: "degraded", checked_at: new Date(1_700_000_000_000).toISOString(), hosts: [{ status: "missing", kind: "tool" }], integrations: [{ status: "deferred" }], recovery: { required_absent_details: [{ address: "y" }] }, blockers: [{ kind: "credential" }] });
    await acknowledgeDegraded("digest", "remote");
    expect(rpc.acknowledgeDegradedReadiness).toHaveBeenCalledWith({ target: "remote", readinessDigest: "digest" });

    const originalFetch = globalThis.fetch;
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({ status: "ok", service: "onboarding", timestamp: "now" }) }) as typeof fetch;
    await expect(fetchHealth()).resolves.toMatchObject({ status: "ok", service: "onboarding" });
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: false, status: 503 });
    await expect(fetchHealth()).rejects.toThrow("API request failed: 503");
    globalThis.fetch = originalFetch;
  });

  it("normalizes sparse host and readiness records with conservative defaults", async () => {
    resetRpc();
    rpc.getHostFacts.mockResolvedValue({ available: false, reason: "offline", cpuCount: 0, gpus: [], platform: "" });
    rpc.listHostRequirements.mockResolvedValue({ tools: [{ name: "tool", required: false, reason: "optional", notes: "", description: "", risk: "", privilege: "", bundling: "", platforms: [], commands: [], configSchema: undefined, config: {}, status: "missing", disposition: SafeguardDisposition.UNSPECIFIED, detail: "", remediation: "" }], safeguards: [] });
    rpc.getReadiness.mockResolvedValue({
      status: ReadinessState.UNSPECIFIED, scenarios: [], resources: [], credentials: [], hosts: [], integrations: [], checkedAt: undefined,
      credentialDiagnosis: undefined, recovery: undefined, blockers: [], degraded: [], degradedDigest: "", degradedAcknowledged: false,
    });
    await expect(fetchHostFacts()).resolves.toEqual({ available: false, reason: "offline", cpu_count: 0, memory_total_bytes: undefined, memory_available_bytes: undefined, disk_free_bytes: undefined, gpus: [], platform: undefined });
    await expect(fetchHostRequirements()).resolves.toMatchObject({ tools: [{ disposition: undefined, notes: undefined, description: undefined, risk: undefined, privilege: undefined, bundling: undefined, detail: undefined, remediation: undefined }] });
    await expect(fetchReadiness()).resolves.toMatchObject({ status: "deferred", checked_at: "", credentials: [], hosts: [], integrations: [], recovery: undefined });
    rpc.getReadiness.mockResolvedValueOnce({ status: ReadinessState.READY, scenarios: [], resources: [], credentials: [{ resource: "r", logicalId: "id", field: "field", label: "label", required: false, status: ReadinessState.READY }], hosts: [{ item: { name: "host", status: ReadinessState.READY }, kind: "other" }], integrations: [{ name: "other", category: "other", status: ReadinessState.READY }], blockers: [], degraded: [], degradedDigest: "", degradedAcknowledged: false });
    await expect(fetchReadiness()).resolves.toMatchObject({ credentials: [{ status: "ready" }], hosts: [{ kind: undefined, required: undefined }], integrations: [{ category: undefined, kind: undefined }] });

    rpc.listHostRequirements.mockResolvedValue({
      tools: [SafeguardDisposition.READY, SafeguardDisposition.MISSING, SafeguardDisposition.PERMISSION_DENIED, SafeguardDisposition.CONTENT_MISMATCH, SafeguardDisposition.DEFERRED, SafeguardDisposition.UNSUPPORTED, SafeguardDisposition.NOT_APPLICABLE].map((disposition, index) => ({ name: `tool-${index}`, required: false, reason: "optional", status: "observed", disposition })),
      safeguards: [],
    });
    await fetchHostRequirements();

    for (const status of [ReadinessState.READY, ReadinessState.MISSING, ReadinessState.DEGRADED, ReadinessState.UNSUPPORTED, ReadinessState.DEFERRED, ReadinessState.NOT_APPLICABLE, ReadinessState.UNSPECIFIED]) {
      rpc.getReadiness.mockResolvedValueOnce({ status, scenarios: [], resources: [], credentials: [], hosts: [], integrations: [], blockers: [], degraded: [], degradedDigest: "", degradedAcknowledged: false });
      await fetchReadiness();
    }
  });
});
