import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  FindingTriage,
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
    const violation = { sectionKey: "purpose" };
    const result = { source: "references" };
    const plan = { id: "plan-1" };
    const client = {
      startSession: vi.fn().mockResolvedValue({ session }),
      getSection: vi.fn().mockResolvedValue({ section }),
      submitSection: vi.fn().mockResolvedValue({ session, violations: [violation] }),
      next: vi.fn().mockResolvedValue({ section, complete: true }),
      validateStructure: vi.fn().mockResolvedValue({ valid: false, violations: [violation] }),
      autofill: vi.fn().mockResolvedValue({ session, results: [result] }),
      finalize: vi.fn().mockResolvedValue({ plan }),
    };
    createClientMock.mockReturnValue(client);

    const authoring = await import("./authoring");

    await expect(authoring.startSession("Title", "slug", "cli")).resolves.toBe(session);
    await expect(authoring.getSection("session-1", "purpose")).resolves.toBe(section);
    await expect(authoring.submitSection("session-1", "purpose", "Body")).resolves.toEqual({ session, violations: [violation] });
    await expect(authoring.nextSection("session-1")).resolves.toEqual({ section, complete: true });
    await expect(authoring.validateStructure("session-1")).resolves.toEqual({ valid: false, violations: [violation] });
    await expect(authoring.autofill("session-1", ["references"])).resolves.toEqual({ session, results: [result] });
    await expect(authoring.finalize("session-1")).resolves.toBe(plan);
    await expect(authoring.startSession("Bare")).resolves.toBe(session);
    await expect(authoring.autofill("session-1")).resolves.toEqual({ session, results: [result] });

    expect(client.startSession).toHaveBeenCalledWith({ title: "Title", slug: "slug", templateId: "cli" });
    expect(client.startSession).toHaveBeenCalledWith({ title: "Bare", slug: "", templateId: "" });
    expect(client.submitSection).toHaveBeenCalledWith({ sessionId: "session-1", sectionKey: "purpose", content: "Body" });
    expect(client.autofill).toHaveBeenCalledWith({ sessionId: "session-1", sources: ["references"] });
    expect(client.autofill).toHaveBeenCalledWith({ sessionId: "session-1", sources: [] });
  });

  it("threads ExecutionService requests and unwraps responses", async () => {
    const execution = { id: "exec-1" };
    const context = { resumePhaseId: "phase-1" };
    const decision = { id: "decision-1" };
    const finding = { id: "finding-1" };
    const handoff = { id: "handoff-1" };
    const nudge = { kind: "record_finding" };
    const point = { id: "velocity-1" };
    const client = {
      start: vi.fn().mockResolvedValue({ execution }),
      getStatus: vi.fn().mockResolvedValue({ execution, context }),
      getNext: vi.fn().mockResolvedValue({ context, complete: false }),
      transitionPhase: vi.fn().mockResolvedValue({ execution }),
      recordDecision: vi.fn().mockResolvedValue({ decision }),
      recordFinding: vi.fn().mockResolvedValue({ finding }),
      complete: vi.fn().mockResolvedValue({ handoff, nudges: [nudge] }),
      getHandoff: vi.fn().mockResolvedValue({ handoff }),
      listCandidateFindings: vi.fn().mockResolvedValue({ findings: [finding] }),
      triageFinding: vi.fn().mockResolvedValue({ finding }),
      getVelocity: vi.fn().mockResolvedValue({ points: [point] }),
    };
    createClientMock.mockReturnValue(client);

    const executionApi = await import("./execution");

    await expect(executionApi.startExecution("plan-1", "run-1")).resolves.toBe(execution);
    await expect(executionApi.getStatus("exec-1")).resolves.toEqual({ execution, context });
    await expect(executionApi.getNext("exec-1")).resolves.toEqual({ context, complete: false });
    await expect(executionApi.transitionPhase("exec-1", "phase-1", PhaseStatus.DONE)).resolves.toEqual({ execution });
    await expect(executionApi.recordDecision("exec-1", "phase-1", "summary", "detail")).resolves.toBe(decision);
    await expect(executionApi.recordFinding("exec-1", "phase-1", "title", "detail")).resolves.toBe(finding);
    await expect(executionApi.completeExecution("exec-1", 10n, 2)).resolves.toEqual({ handoff, nudges: [nudge] });
    await expect(executionApi.getHandoff("exec-1")).resolves.toBe(handoff);
    await expect(executionApi.listCandidateFindings("exec-1")).resolves.toEqual([finding]);
    await expect(executionApi.triageFinding("finding-1", FindingTriage.PROMOTED)).resolves.toBe(finding);
    await expect(executionApi.getVelocity("plan-1")).resolves.toEqual([point]);
    await expect(executionApi.startExecution("plan-1")).resolves.toBe(execution);
    await expect(executionApi.recordDecision("exec-1", "phase-1", "summary")).resolves.toBe(decision);
    await expect(executionApi.recordFinding("exec-1", "phase-1", "title")).resolves.toBe(finding);
    await expect(executionApi.completeExecution("exec-1")).resolves.toEqual({ handoff, nudges: [nudge] });
    await expect(executionApi.listCandidateFindings()).resolves.toEqual([finding]);

    expect(client.start).toHaveBeenCalledWith({ planId: "plan-1", runId: "run-1" });
    expect(client.start).toHaveBeenCalledWith({ planId: "plan-1", runId: "" });
    expect(client.recordDecision).toHaveBeenCalledWith({ executionId: "exec-1", phaseId: "phase-1", summary: "summary", detail: "" });
    expect(client.recordFinding).toHaveBeenCalledWith({ executionId: "exec-1", phaseId: "phase-1", title: "title", detail: "" });
    expect(client.complete).toHaveBeenCalledWith({ executionId: "exec-1", tokens: 10n, iterations: 2 });
    expect(client.complete).toHaveBeenCalledWith({ executionId: "exec-1", tokens: 0n, iterations: 0 });
    expect(client.listCandidateFindings).toHaveBeenCalledWith({ executionId: "" });
    expect(client.triageFinding).toHaveBeenCalledWith({ findingId: "finding-1", triage: 2 });
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
