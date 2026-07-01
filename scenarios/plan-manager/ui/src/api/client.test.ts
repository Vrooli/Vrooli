import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  FindingTriage,
  LogEntryType,
  LogSeverity,
  PhaseStatus,
  PlanStatus,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

import { ApiError, decodeApiError, uploadFile } from "./client";

const createClientMock = vi.hoisted(() => vi.fn());

vi.mock("@connectrpc/connect", () => ({
  createClient: createClientMock,
}));

describe("api/client REST helpers", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.resetModules();
    createClientMock.mockReset();
  });

  it("throws ApiError with the typed envelope on non-2xx responses", async () => {
    const err = await decodeApiError(
      new Response(JSON.stringify({ code: "internal", message: "store down" }), {
        status: 500,
      }),
    );

    expect(err).toBeInstanceOf(ApiError);
    expect(err.code).toBe("internal");
    expect(err.status).toBe(500);
    expect(err.message).toContain("store down");
  });

  it("falls back to an internal envelope when the error body is malformed", async () => {
    const err = await decodeApiError(new Response("not json", { status: 502 }));

    expect(err.code).toBe("internal");
    expect(err.status).toBe(502);
    expect(err.message).toContain("unexpected 502 response");
  });

  it("posts multipart form data through the REST helper", async () => {
    const formData = new FormData();
    formData.set("file", new File(["hello"], "hello.txt", { type: "text/plain" }));
    fetchSpy.mockResolvedValueOnce(new Response("{}", { status: 200 }));

    await uploadFile("/things/thing-1/attachments", formData);

    const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect(url).toMatch(/\/api\/v1\/things\/thing-1\/attachments$/);
    expect(init).toMatchObject({ method: "POST", body: formData, cache: "no-store" });
    expect(init.headers).toBeUndefined();
  });
});

describe("Connect API wrapper helpers", () => {
  afterEach(() => {
    vi.resetModules();
    createClientMock.mockReset();
  });

  it("threads PlansService requests and unwraps responses", async () => {
    const plan = { id: "plan-1" };
    const phase = { id: "phase-1" };
    const edge = { fromPlanId: "a", toPlanId: "b" };
    const template = { id: "cli" };
    const client = {
      listPlans: vi.fn().mockResolvedValue({ plans: [plan] }),
      getPlan: vi.fn().mockResolvedValue({ plan }),
      createPlan: vi.fn().mockResolvedValue({ plan }),
      archivePlan: vi.fn().mockResolvedValue({ plan }),
      renderMarkdown: vi.fn().mockResolvedValue({ markdown: "# Plan" }),
      addPhase: vi.fn().mockResolvedValue({ plan }),
      getGraph: vi.fn().mockResolvedValue({ edges: [edge] }),
      listTemplates: vi.fn().mockResolvedValue({ templates: [template] }),
      createFromTemplate: vi.fn().mockResolvedValue({ plan }),
    };
    createClientMock.mockReturnValue(client);

    const plans = await import("./plans");

    await expect(plans.listPlans({ status: PlanStatus.ACTIVE, includeArchived: true })).resolves.toEqual([plan]);
    await expect(plans.getPlan("plan-1")).resolves.toBe(plan);
    await expect(plans.createPlan(plan as never)).resolves.toBe(plan);
    await expect(plans.archivePlan("plan-1")).resolves.toBe(plan);
    await expect(plans.renderPlan("plan-1")).resolves.toBe("# Plan");
    await expect(plans.addPhase("plan-1", phase as never)).resolves.toBe(plan);
    await expect(plans.getGraph("plan-1")).resolves.toEqual([edge]);
    await expect(plans.listTemplates()).resolves.toEqual([template]);
    await expect(plans.createFromTemplate("cli", "New plan", "new-plan")).resolves.toBe(plan);
    await expect(plans.listPlans()).resolves.toEqual([plan]);
    await expect(plans.getGraph()).resolves.toEqual([edge]);
    await expect(plans.createFromTemplate("cli", "Untitled")).resolves.toBe(plan);

    expect(client.listPlans).toHaveBeenCalledWith({ status: PlanStatus.ACTIVE, includeArchived: true });
    expect(client.listPlans).toHaveBeenCalledWith({ status: PlanStatus.UNSPECIFIED, includeArchived: false });
    expect(client.getPlan).toHaveBeenCalledWith({ id: "plan-1" });
    expect(client.addPhase).toHaveBeenCalledWith({ planId: "plan-1", phase });
    expect(client.getGraph).toHaveBeenCalledWith({ planId: "plan-1" });
    expect(client.getGraph).toHaveBeenCalledWith({ planId: "" });
    expect(client.createFromTemplate).toHaveBeenCalledWith({ templateId: "cli", title: "New plan", slug: "new-plan" });
    expect(client.createFromTemplate).toHaveBeenCalledWith({ templateId: "cli", title: "Untitled", slug: "" });
  });

  it("threads AuthoringService requests and unwraps responses", async () => {
    const session = { id: "session-1" };
    const section = { key: "purpose" };
    const step = { stepKind: "purpose" };
    const phase = { id: "phase-1" };
    const violation = { sectionKey: "purpose" };
    const result = { source: "references" };
    const item = { id: "ctx-1" };
    const candidate = { id: "candidate-1" };
    const plan = { id: "plan-1" };
    // Focused response contract: mutations return progress + a summary, never the
    // full session. Full state is read explicitly via getSession.
    const progress = { sessionId: "session-1" };
    const summary = { objectKind: "section" };
    const client = {
      startSession: vi.fn().mockResolvedValue({ session, step }),
      getSession: vi.fn().mockResolvedValue({ session, step }),
      getSection: vi.fn().mockResolvedValue({ section, step }),
      submitSection: vi.fn().mockResolvedValue({ summary, progress, violations: [violation], step }),
      next: vi.fn().mockResolvedValue({ section, complete: true, step }),
      validateStructure: vi.fn().mockResolvedValue({ valid: false, violations: [violation], step }),
      autofill: vi.fn().mockResolvedValue({ results: [result], progress, step }),
      submitRelevantContextItem: vi.fn().mockResolvedValue({ item, summary, progress, violations: [violation], step }),
      listRelevantContext: vi.fn().mockResolvedValue({ items: [item], step }),
      updateRelevantContextItem: vi.fn().mockResolvedValue({ item, summary, progress, violations: [violation], step }),
      removeRelevantContextItem: vi.fn().mockResolvedValue({ summary, progress, violations: [violation], step }),
      discoverContextCandidates: vi.fn().mockResolvedValue({ candidates: [candidate], progress, step }),
      acceptContextCandidate: vi.fn().mockResolvedValue({ candidate, item, summary, progress, violations: [violation], step }),
      rejectContextCandidate: vi.fn().mockResolvedValue({ candidate, progress, step }),
      suggestReferences: vi.fn().mockResolvedValue({ candidates: [candidate], progress, step }),
      listReferenceCandidates: vi.fn().mockResolvedValue({ candidates: [candidate], step }),
      acceptReferenceCandidate: vi.fn().mockResolvedValue({ candidate, summary, progress, violations: [violation], step }),
      rejectReferenceCandidate: vi.fn().mockResolvedValue({ candidate, progress, step }),
      addPhase: vi.fn().mockResolvedValue({ phase, summary, progress, violations: [violation], step }),
      getPhase: vi.fn().mockResolvedValue({ phase, step }),
      submitPhaseField: vi.fn().mockResolvedValue({ phase, summary, progress, violations: [violation], step }),
      nextPhase: vi.fn().mockResolvedValue({ phase, complete: true, step }),
      finalize: vi.fn().mockResolvedValue({ plan, step }),
    };
    createClientMock.mockReturnValue(client);

    const authoring = await import("./authoring");

    await expect(authoring.startSession("Title", "slug", "cli")).resolves.toEqual({ session, step });
    await expect(authoring.getSession("session-1")).resolves.toEqual({ session, step });
    await expect(authoring.getSection("session-1", "purpose")).resolves.toEqual({ section, step });
    await expect(authoring.submitSection("session-1", "purpose", "Body")).resolves.toEqual({ summary, progress, violations: [violation], step });
    await expect(authoring.nextSection("session-1")).resolves.toEqual({ section, complete: true, step });
    await expect(authoring.validateStructure("session-1")).resolves.toEqual({ valid: false, violations: [violation], step });
    await expect(authoring.autofill("session-1", ["references"])).resolves.toEqual({ results: [result], progress, step });
    await expect(authoring.submitRelevantContextItem("session-1", "phase-1", item as never)).resolves.toEqual({ item, summary, progress, violations: [violation], step });
    await expect(authoring.listRelevantContext("session-1", "phase-1")).resolves.toEqual({ items: [item], step });
    await expect(authoring.updateRelevantContextItem("session-1", "phase-1", "ctx-1", item as never)).resolves.toEqual({ item, summary, progress, violations: [violation], step });
    await expect(authoring.removeRelevantContextItem("session-1", "phase-1", "ctx-1")).resolves.toEqual({ summary, progress, violations: [violation], step });
    await expect(authoring.discoverContextCandidates("session-1", ["context"], "architectural")).resolves.toEqual({ candidates: [candidate], progress, step });
    await expect(authoring.acceptContextCandidate("session-1", "candidate-1", "phase-1")).resolves.toEqual({ candidate, item, summary, progress, violations: [violation], step });
    await expect(authoring.rejectContextCandidate("session-1", "candidate-1", "duplicate")).resolves.toEqual({ candidate, progress, step });
    await expect(authoring.suggestReferences("session-1")).resolves.toEqual({ candidates: [candidate], progress, step });
    await expect(authoring.listReferenceCandidates("session-1")).resolves.toEqual({ candidates: [candidate], step });
    await expect(authoring.acceptReferenceCandidate("session-1", "candidate-1")).resolves.toEqual({ candidate, summary, progress, violations: [violation], step });
    await expect(authoring.rejectReferenceCandidate("session-1", "candidate-1", "unrelated")).resolves.toEqual({ candidate, progress, step });
    await expect(authoring.addPhase("session-1", "Title", "Intent")).resolves.toEqual({ phase, summary, progress, violations: [violation], step });
    await expect(authoring.getPhase("session-1", "phase-1")).resolves.toEqual({ phase, step });
    await expect(authoring.submitPhaseField("session-1", "phase-1", "acceptance", "Done")).resolves.toEqual({ phase, summary, progress, violations: [violation], step });
    await expect(authoring.nextPhase("session-1")).resolves.toEqual({ phase, complete: true, step });
    await expect(authoring.finalize("session-1")).resolves.toEqual({ plan, step });
    await expect(authoring.startSession("Bare")).resolves.toEqual({ session, step });
    await expect(authoring.autofill("session-1")).resolves.toEqual({ results: [result], progress, step });

    expect(client.startSession).toHaveBeenCalledWith({ title: "Title", slug: "slug", templateId: "cli" });
    expect(client.startSession).toHaveBeenCalledWith({ title: "Bare", slug: "", templateId: "" });
    expect(client.submitSection).toHaveBeenCalledWith({ sessionId: "session-1", sectionKey: "purpose", content: "Body" });
    expect(client.autofill).toHaveBeenCalledWith({ sessionId: "session-1", sources: ["references"] });
    expect(client.autofill).toHaveBeenCalledWith({ sessionId: "session-1", sources: [] });
    expect(client.submitRelevantContextItem).toHaveBeenCalledWith({ sessionId: "session-1", phaseId: "phase-1", item });
    expect(client.listRelevantContext).toHaveBeenCalledWith({ sessionId: "session-1", phaseId: "phase-1" });
    expect(client.getSession).toHaveBeenCalledWith({ sessionId: "session-1" });
    expect(client.updateRelevantContextItem).toHaveBeenCalledWith({ sessionId: "session-1", phaseId: "phase-1", itemId: "ctx-1", item });
    expect(client.removeRelevantContextItem).toHaveBeenCalledWith({ sessionId: "session-1", phaseId: "phase-1", itemId: "ctx-1" });
    expect(client.discoverContextCandidates).toHaveBeenCalledWith({ sessionId: "session-1", concepts: ["context"], complexity: "architectural" });
    expect(client.acceptContextCandidate).toHaveBeenCalledWith({ sessionId: "session-1", candidateId: "candidate-1", phaseId: "phase-1" });
    expect(client.rejectContextCandidate).toHaveBeenCalledWith({ sessionId: "session-1", candidateId: "candidate-1", reason: "duplicate" });
    expect(client.suggestReferences).toHaveBeenCalledWith({ sessionId: "session-1" });
    expect(client.listReferenceCandidates).toHaveBeenCalledWith({ sessionId: "session-1" });
    expect(client.acceptReferenceCandidate).toHaveBeenCalledWith({ sessionId: "session-1", candidateId: "candidate-1", reference: undefined });
    expect(client.rejectReferenceCandidate).toHaveBeenCalledWith({ sessionId: "session-1", candidateId: "candidate-1", reason: "unrelated" });
    expect(client.addPhase).toHaveBeenCalledWith({ sessionId: "session-1", title: "Title", intent: "Intent" });
    expect(client.getPhase).toHaveBeenCalledWith({ sessionId: "session-1", phaseId: "phase-1" });
    expect(client.submitPhaseField).toHaveBeenCalledWith({ sessionId: "session-1", phaseId: "phase-1", field: "acceptance", content: "Done" });
  });

  it("threads ExecutionService requests and unwraps responses", async () => {
    const execution = { id: "exec-1" };
    const context = { resumePhaseId: "phase-1" };
    const handoff = { id: "handoff-1" };
    const nudge = { kind: "record_finding" };
    const point = { id: "velocity-1" };
    const step = { stepKind: "execution" };
    const client = {
      start: vi.fn().mockResolvedValue({ execution, context, step }),
      getStatus: vi.fn().mockResolvedValue({ execution, context, step }),
      getNext: vi.fn().mockResolvedValue({ context, complete: false, step }),
      getContext: vi.fn().mockResolvedValue({ execution, context, step }),
      resume: vi.fn().mockResolvedValue({ execution, context, step }),
      transitionPhase: vi.fn().mockResolvedValue({ execution, step }),
      complete: vi.fn().mockResolvedValue({ handoff, nudges: [nudge], step }),
      getHandoff: vi.fn().mockResolvedValue({ handoff, step }),
      getVelocity: vi.fn().mockResolvedValue({ points: [point] }),
    };
    createClientMock.mockReturnValue(client);

    const executionApi = await import("./execution");

    await expect(executionApi.startExecution("plan-1", "run-1")).resolves.toEqual({ execution, context, step });
    await expect(executionApi.getStatus("exec-1")).resolves.toEqual({ execution, context, step });
    await expect(executionApi.getNext("exec-1")).resolves.toEqual({ context, complete: false, step });
    await expect(executionApi.getContext("exec-1", "phase-1")).resolves.toEqual({ execution, context, step });
    await expect(executionApi.resumeExecution("plan-1", "phase-1", "run-1")).resolves.toEqual({ execution, context, step });
    await expect(executionApi.transitionPhase("exec-1", "phase-1", PhaseStatus.DONE)).resolves.toEqual({ execution, step });
    await expect(executionApi.completeExecution("exec-1", 10n, 2)).resolves.toEqual({ handoff, nudges: [nudge], step });
    await expect(executionApi.getHandoff("exec-1")).resolves.toEqual({ handoff, step });
    await expect(executionApi.getVelocity("plan-1")).resolves.toEqual([point]);
    await expect(executionApi.startExecution("plan-1")).resolves.toEqual({ execution, context, step });
    await expect(executionApi.completeExecution("exec-1")).resolves.toEqual({ handoff, nudges: [nudge], step });

    expect(client.start).toHaveBeenCalledWith({ planId: "plan-1", runId: "run-1" });
    expect(client.start).toHaveBeenCalledWith({ planId: "plan-1", runId: "" });
    expect(client.getContext).toHaveBeenCalledWith({ executionId: "exec-1", phaseId: "phase-1" });
    expect(client.resume).toHaveBeenCalledWith({ planOrExecution: "plan-1", phaseId: "phase-1", runId: "run-1" });
    expect(client.transitionPhase).toHaveBeenCalledWith({
      executionId: "exec-1",
      phaseId: "phase-1",
      toStatus: PhaseStatus.DONE,
      validationOverride: { reason: "" },
      feedbackOverride: { reason: "" },
    });
    expect(client.complete).toHaveBeenCalledWith({ executionId: "exec-1", tokens: 10n, iterations: 2 });
    expect(client.complete).toHaveBeenCalledWith({ executionId: "exec-1", tokens: 0n, iterations: 0 });
  });

  it("threads LogService requests and unwraps responses", async () => {
    const entry = { id: "log-1" };
    const source = { id: "log-0" };
    const summary = { total: 1 };
    const step = { stepKind: "log" };
    const client = {
      addDecision: vi.fn().mockResolvedValue({ entry, step, deduplicated: false }),
      addFinding: vi.fn().mockResolvedValue({ entry, step, deduplicated: false }),
      addBug: vi.fn().mockResolvedValue({ entry, step, deduplicated: true }),
      addRecord: vi.fn().mockResolvedValue({ entry, step, deduplicated: false }),
      addNote: vi.fn().mockResolvedValue({ entry, step, deduplicated: false }),
      listEntries: vi.fn().mockResolvedValue({ entries: [entry], summary, step }),
      getEntry: vi.fn().mockResolvedValue({ entry, step }),
      updateEntry: vi.fn().mockResolvedValue({ entry, step }),
      promoteEntry: vi.fn().mockResolvedValue({ entry, source, step }),
      syncEntry: vi.fn().mockResolvedValue({ entry, step }),
    };
    createClientMock.mockReturnValue(client);

    const logApi = await import("./log");

    await expect(logApi.addDecision("exec-1", "phase-1", "chose Connect", { detail: "rationale" })).resolves.toEqual({
      entry,
      step,
      deduplicated: false,
    });
    await expect(logApi.addFinding("exec-1", "phase-1", "edge case")).resolves.toEqual({ entry, step, deduplicated: false });
    await expect(
      logApi.addBug("exec-1", "phase-1", "crash", { severity: LogSeverity.HIGH, evidence: ["log.txt"] }),
    ).resolves.toEqual({ entry, step, deduplicated: true });
    await expect(logApi.addRecord("exec-1", "phase-1", "pattern")).resolves.toEqual({ entry, step, deduplicated: false });
    await expect(logApi.addNote("exec-1", "phase-1", "progress")).resolves.toEqual({ entry, step, deduplicated: false });
    await expect(
      logApi.listEntries({ type: LogEntryType.FINDING, triage: FindingTriage.CANDIDATE }),
    ).resolves.toEqual({ entries: [entry], summary, step });
    await expect(logApi.listEntries()).resolves.toEqual({ entries: [entry], summary, step });
    await expect(logApi.getEntry("log-1")).resolves.toEqual({ entry, step });
    await expect(logApi.updateEntry({ id: "log-1", triage: FindingTriage.DISMISSED })).resolves.toEqual({ entry, step });
    await expect(logApi.promoteEntry({ id: "log-1", toType: LogEntryType.BUG_REPORT })).resolves.toEqual({
      entry,
      source,
      step,
    });
    await expect(logApi.syncEntry("log-1")).resolves.toEqual({ entry, step });

    expect(client.addDecision).toHaveBeenCalledWith({
      planOrExecution: "exec-1",
      phaseId: "phase-1",
      title: "chose Connect",
      detail: "rationale",
      evidence: [],
      sourceCommand: "",
      idempotencyKey: "",
      runId: "",
    });
    expect(client.addFinding).toHaveBeenCalledWith({
      planOrExecution: "exec-1",
      phaseId: "phase-1",
      title: "edge case",
      detail: "",
      severity: LogSeverity.UNSPECIFIED,
      evidence: [],
      sourceCommand: "",
      idempotencyKey: "",
      runId: "",
    });
    expect(client.addBug).toHaveBeenCalledWith({
      planOrExecution: "exec-1",
      phaseId: "phase-1",
      title: "crash",
      detail: "",
      severity: LogSeverity.HIGH,
      evidence: ["log.txt"],
      sourceCommand: "",
      idempotencyKey: "",
      runId: "",
    });
    expect(client.listEntries).toHaveBeenCalledWith({
      planOrExecution: "",
      phaseId: "",
      type: LogEntryType.FINDING,
      triage: FindingTriage.CANDIDATE,
      syncStatus: 0,
    });
    expect(client.listEntries).toHaveBeenCalledWith({
      planOrExecution: "",
      phaseId: "",
      type: LogEntryType.UNSPECIFIED,
      triage: FindingTriage.UNSPECIFIED,
      syncStatus: 0,
    });
    expect(client.updateEntry).toHaveBeenCalledWith({
      id: "log-1",
      title: "",
      detail: "",
      severity: LogSeverity.UNSPECIFIED,
      triage: FindingTriage.DISMISSED,
      addEvidence: [],
    });
    expect(client.promoteEntry).toHaveBeenCalledWith({
      id: "log-1",
      toType: LogEntryType.BUG_REPORT,
      title: "",
      detail: "",
      severity: LogSeverity.UNSPECIFIED,
    });
    expect(client.getEntry).toHaveBeenCalledWith({ id: "log-1" });
    expect(client.syncEntry).toHaveBeenCalledWith({ id: "log-1" });
  });

  it("threads ValidationService requests and unwraps responses", async () => {
    const reference = { id: "ref-1" };
    const result = { id: "validation-1" };
    const scope = { commands: ["go test ./..."] };
    const client = {
      resolveReferences: vi.fn().mockResolvedValue({ references: [reference], degraded: true }),
      computeStaleness: vi.fn().mockResolvedValue({ overall: 1, references: [reference], degraded: false }),
      deriveBaselineScope: vi.fn().mockResolvedValue(scope),
      runValidation: vi.fn().mockResolvedValue({ result }),
      verifyDefinitionOfDone: vi.fn().mockResolvedValue({ result, dodMet: true }),
    };
    createClientMock.mockReturnValue(client);

    const validation = await import("./validation");

    await expect(validation.resolveReferences("plan-1", "phase-1")).resolves.toEqual({ references: [reference], degraded: true });
    await expect(validation.computeStaleness("plan-1", "phase-1")).resolves.toEqual({ overall: 1, references: [reference], degraded: false });
    await expect(validation.deriveBaselineScope("plan-1", "phase-1")).resolves.toBe(scope);
    await expect(validation.runValidation("plan-1", "phase-1")).resolves.toBe(result);
    await expect(validation.verifyDefinitionOfDone("plan-1")).resolves.toEqual({ result, dodMet: true });
    await expect(validation.resolveReferences("plan-1")).resolves.toEqual({ references: [reference], degraded: true });
    await expect(validation.computeStaleness("plan-1")).resolves.toEqual({ overall: 1, references: [reference], degraded: false });
    await expect(validation.deriveBaselineScope("plan-1")).resolves.toBe(scope);
    await expect(validation.runValidation("plan-1")).resolves.toBe(result);

    expect(client.resolveReferences).toHaveBeenCalledWith({ planId: "plan-1", phaseId: "phase-1" });
    expect(client.resolveReferences).toHaveBeenCalledWith({ planId: "plan-1", phaseId: "" });
    expect(client.computeStaleness).toHaveBeenCalledWith({ planId: "plan-1", phaseId: "" });
    expect(client.deriveBaselineScope).toHaveBeenCalledWith({ planId: "plan-1", phaseId: "" });
    expect(client.runValidation).toHaveBeenCalledWith({ planId: "plan-1", phaseId: "" });
    expect(client.verifyDefinitionOfDone).toHaveBeenCalledWith({ planId: "plan-1" });
  });
});
