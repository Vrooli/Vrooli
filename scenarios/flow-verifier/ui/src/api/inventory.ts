// API client for inventory: flows, flow detail, runs, run detail, and
// kicking off a verify pipeline run. Thin wrapper over generated Connect
// clients; the public types below shadow the proto-generated message
// shapes (matching field names) so existing consumers keep compiling.
import { create } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";

import { transport } from "./client";

import { FlowsService } from "@vrooli/proto-types/flow-verifier/v1/flows/flows_pb";
import { RunsService } from "@vrooli/proto-types/flow-verifier/v1/runs/runs_pb";
import {
  VerificationsService,
  VerificationMode,
  StartVerificationRequestSchema,
} from "@vrooli/proto-types/flow-verifier/v1/verifications/verifications_pb";
import {
  RunStatus,
  RunMode,
  FailureReason as ProtoFailureReason,
  type Run as ProtoRun,
} from "@vrooli/proto-types/flow-verifier/v1/runs/runs_pb";
import type {
  FlowSummary as ProtoFlowSummary,
  FlowDetail as ProtoFlowDetail,
  FlowState as ProtoFlowState,
  FlowEvent as ProtoFlowEvent,
  FlowTransition as ProtoFlowTransition,
  FlowTrace as ProtoFlowTrace,
  FlowTraceStep as ProtoFlowTraceStep,
  FlowInvariant as ProtoFlowInvariant,
  FlowModel as ProtoFlowModel,
  FlowRuntime as ProtoFlowRuntime,
} from "@vrooli/proto-types/flow-verifier/v1/flows/flows_pb";

export const flowsClient = createClient(FlowsService, transport);
export const runsClient = createClient(RunsService, transport);
export const verificationsClient = createClient(VerificationsService, transport);

export type FailureReason =
  | ""
  | "missing_artifacts"
  | "stale_artifacts"
  | "counterexample"
  | "lint"
  | "quint_failure"
  | "io";

export type FlowSummary = {
  flowId: string;
  contractPath: string;
  language: string;
  schemaVersion: number;
  scenarioId?: string;
  kind: string;
};

export type RunRow = {
  id: string;
  flowId: string;
  flowPath: string;
  root: string;
  mode: "run" | "check";
  status: "passed" | "failed" | "error";
  errorMessage?: string;
  failureReason?: FailureReason;
  missingArtifacts?: string[];
  output?: string;
  counterexample?: string;
  startedAt: string;
  finishedAt: string;
  durationMs: number;
};

export type VerifyResponse = {
  status: "passed" | "failed";
  error?: string;
  runs: RunRow[];
};

export type FlowState = {
  id: string;
  quint: string;
  initial?: boolean;
  terminal?: boolean;
};
export type FlowEvent = { id: string; quint: string };
export type FlowTransition = {
  from: string;
  event: string;
  to: string;
  wantError: boolean;
};
export type FlowTraceStep = { event: string; want: string; wantError: boolean };
export type FlowTrace = { name: string; initial: string; steps: FlowTraceStep[] };
export type FlowInvariant = {
  id: string;
  quint: string;
  description: string;
  expression?: string;
};
export type FlowModelVerify = { invariants: string[] };
export type FlowModel = {
  module: string;
  seed: string;
  maxSteps: number;
  traceCount: number;
  verify: FlowModelVerify;
};
export type FlowGoRuntime = {
  package: string;
  statusType: string;
  eventType: string;
  constantPrefix: string;
};
export type FlowTypeScriptRuntime = {
  statusType: string;
  eventType: string;
  statusesConst: string;
  eventsConst: string;
  formalExpectationConst: string;
  stateUnionType?: string;
  eventUnionType?: string;
};
export type FlowRuntime = {
  go?: FlowGoRuntime;
  typescript?: FlowTypeScriptRuntime;
  sideEffects?: string[];
  staleCompletion?: string;
};
export type FlowDetail = {
  flowId: string;
  kind: string;
  domain?: string;
  description?: string;
  contractPath: string;
  language: string;
  schemaVersion: number;
  initialState: string;
  states: FlowState[];
  events: FlowEvent[];
  transitions: FlowTransition[];
  traces: FlowTrace[];
  invariants: FlowInvariant[];
  model: FlowModel;
  runtime: FlowRuntime;
  report: string;
};

export async function fetchFlows(
  opts: { scenarioId?: string; root?: string } = {},
): Promise<FlowSummary[]> {
  const resp = await flowsClient.listFlows({ root: opts.root ?? "", flowId: "" });
  let rows = resp.flows.map(flowSummaryFromProto);
  if (opts.scenarioId) {
    rows = rows.map((r) => ({ ...r, scenarioId: r.scenarioId || opts.scenarioId }));
  }
  return rows;
}

export async function fetchFlowDetail(
  flowId: string,
  opts: { scenarioId?: string; root?: string } = {},
): Promise<FlowDetail> {
  void opts.scenarioId;
  const resp = await flowsClient.getFlow({ flowId, root: opts.root ?? "" });
  if (!resp.flow) throw new Error("server returned no flow");
  return flowDetailFromProto(resp.flow);
}

export async function fetchRuns(
  opts: { flowId?: string; limit?: number } = {},
): Promise<RunRow[]> {
  const resp = await runsClient.listRuns({
    flowId: opts.flowId ?? "",
    limit: opts.limit ?? 0,
  });
  return resp.runs.map(runFromProto);
}

export type NavigationStudioRoute = {
  id: string;
  path: string;
  page: string;
  requires: string;
  parents: string[];
};
export type NavigationStudioPresentation = { in: string; label: string; testId: string };
export type NavigationStudioAffordance = {
  id: string;
  to: string;
  showWhen: string;
  sideEffect: string;
  presentations: NavigationStudioPresentation[];
};
export type NavigationStudioContainer = {
  id: string;
  kind: string;
  showWhen: string;
  disclosure: string;
  hostRoutes: string[];
};
export type NavigationStudioContext = {
  name: string;
  kind: "enum" | "bool" | string;
  values: string[];
  defaultValue: string;
};
export type NavigationStudioInvariant = { id: string; passed: boolean; message: string };
export type NavigationStudioDescriptor = {
  renderer: string;
  routes: NavigationStudioRoute[];
  affordances: NavigationStudioAffordance[];
  containers: NavigationStudioContainer[];
  contexts: NavigationStudioContext[];
  invariants: NavigationStudioInvariant[];
};

export async function fetchNavigationStudio(
  flowId: string,
  opts: { root?: string } = {},
): Promise<NavigationStudioDescriptor> {
  const resp = await flowsClient.getNavigationStudio({ flowId, root: opts.root ?? "" });
  const d = resp.descriptor;
  if (!d) throw new Error("server returned no descriptor");
  return {
    renderer: d.renderer,
    routes: d.routes.map((r) => ({
      id: r.id,
      path: r.path,
      page: r.page,
      requires: r.requires,
      parents: [...r.parents],
    })),
    affordances: d.affordances.map((a) => ({
      id: a.id,
      to: a.to,
      showWhen: a.showWhen,
      sideEffect: a.sideEffect,
      presentations: a.presentations.map((p) => ({ in: p.in, label: p.label, testId: p.testId })),
    })),
    containers: d.containers.map((c) => ({
      id: c.id,
      kind: c.kind,
      showWhen: c.showWhen,
      disclosure: c.disclosure,
      hostRoutes: [...c.hostRoutes],
    })),
    contexts: d.contexts.map((c) => ({
      name: c.name,
      kind: c.kind,
      values: [...c.values],
      defaultValue: c.defaultValue,
    })),
    invariants: d.invariants.map((i) => ({ id: i.id, passed: i.passed, message: i.message })),
  };
}

export async function fetchRun(runId: string): Promise<RunRow> {
  const resp = await runsClient.getRun({ id: runId });
  if (!resp.run) throw new Error("server returned no run");
  return runFromProto(resp.run);
}

export async function verifyFlow(root: string, flowId?: string): Promise<VerifyResponse> {
  const req = create(StartVerificationRequestSchema, {
    root: root || ".",
    flowId: flowId ?? "",
    mode: VerificationMode.CHECK,
  });
  const resp = await verificationsClient.startVerification(req);
  return {
    status: resp.status === "passed" ? "passed" : "failed",
    error: resp.errorMessage || undefined,
    runs: resp.runs.map(runFromProto),
  };
}

export function flowSummaryFromProto(p: ProtoFlowSummary): FlowSummary {
  return {
    flowId: p.flowId,
    contractPath: p.contractPath,
    language: p.language,
    schemaVersion: p.schemaVersion,
    scenarioId: p.scenarioId || undefined,
    kind: p.kind || "temporal",
  };
}

function flowDetailFromProto(p: ProtoFlowDetail): FlowDetail {
  return {
    flowId: p.flowId,
    kind: p.kind || "temporal",
    domain: p.domain || undefined,
    description: p.description || undefined,
    contractPath: p.contractPath,
    language: p.language,
    schemaVersion: p.schemaVersion,
    initialState: p.initialState,
    states: p.states.map(stateFromProto),
    events: p.events.map(eventFromProto),
    transitions: p.transitions.map(transitionFromProto),
    traces: p.traces.map(traceFromProto),
    invariants: p.invariants.map(invariantFromProto),
    model: modelFromProto(p.model),
    runtime: runtimeFromProto(p.runtime),
    report: p.report,
  };
}

function stateFromProto(s: ProtoFlowState): FlowState {
  return {
    id: s.id,
    quint: s.quint,
    initial: s.initial || undefined,
    terminal: s.terminal || undefined,
  };
}
function eventFromProto(e: ProtoFlowEvent): FlowEvent {
  return { id: e.id, quint: e.quint };
}
function transitionFromProto(t: ProtoFlowTransition): FlowTransition {
  return {
    from: t.from[0] ?? "",
    event: t.event[0] ?? "",
    to: t.to,
    wantError: t.wantError,
  };
}
function traceFromProto(t: ProtoFlowTrace): FlowTrace {
  return {
    name: t.name,
    initial: t.initial,
    steps: t.steps.map(stepFromProto),
  };
}
function stepFromProto(s: ProtoFlowTraceStep): FlowTraceStep {
  return { event: s.event, want: s.want, wantError: s.wantError };
}
function invariantFromProto(inv: ProtoFlowInvariant): FlowInvariant {
  return {
    id: inv.id,
    quint: inv.quint,
    description: inv.description,
    expression: inv.expression || undefined,
  };
}
function modelFromProto(m: ProtoFlowModel | undefined): FlowModel {
  return {
    module: m?.module ?? "",
    seed: m?.seed ?? "",
    maxSteps: m?.maxSteps ?? 0,
    traceCount: m?.traceCount ?? 0,
    verify: { invariants: m?.verify?.invariants ?? [] },
  };
}
function runtimeFromProto(r: ProtoFlowRuntime | undefined): FlowRuntime {
  if (!r) return { sideEffects: [] };
  return {
    go: r.go
      ? {
          package: r.go.package,
          statusType: r.go.statusType,
          eventType: r.go.eventType,
          constantPrefix: r.go.constantPrefix,
        }
      : undefined,
    typescript: r.typescript
      ? {
          statusType: r.typescript.statusType,
          eventType: r.typescript.eventType,
          statusesConst: r.typescript.statusesConst,
          eventsConst: r.typescript.eventsConst,
          formalExpectationConst: r.typescript.formalExpectationConst,
          stateUnionType: r.typescript.stateUnionType || undefined,
          eventUnionType: r.typescript.eventUnionType || undefined,
        }
      : undefined,
    sideEffects: r.sideEffects,
    staleCompletion: r.staleCompletion || undefined,
  };
}

export function runFromProto(r: ProtoRun): RunRow {
  return {
    id: r.id,
    flowId: r.flowId,
    flowPath: r.flowPath,
    root: r.root,
    mode: r.mode === RunMode.RUN ? "run" : "check",
    status: runStatusToString(r.status),
    errorMessage: r.errorMessage || undefined,
    failureReason: failureReasonToString(r.failureReason),
    missingArtifacts: r.missingArtifacts.length > 0 ? r.missingArtifacts : undefined,
    output: r.output || undefined,
    counterexample: r.counterexample || undefined,
    startedAt: r.startedAt ? new Date(Number(r.startedAt.seconds) * 1000).toISOString() : "",
    finishedAt: r.finishedAt ? new Date(Number(r.finishedAt.seconds) * 1000).toISOString() : "",
    durationMs: Number(r.durationMs),
  };
}

function runStatusToString(s: RunStatus): "passed" | "failed" | "error" {
  switch (s) {
    case RunStatus.PASSED:
      return "passed";
    case RunStatus.FAILED:
      return "failed";
    case RunStatus.ERROR:
      return "error";
  }
  return "error";
}

function failureReasonToString(r: ProtoFailureReason): FailureReason {
  switch (r) {
    case ProtoFailureReason.MISSING_ARTIFACTS:
      return "missing_artifacts";
    case ProtoFailureReason.STALE_ARTIFACTS:
      return "stale_artifacts";
    case ProtoFailureReason.COUNTEREXAMPLE:
      return "counterexample";
    case ProtoFailureReason.LINT:
      return "lint";
    case ProtoFailureReason.QUINT_FAILURE:
      return "quint_failure";
    case ProtoFailureReason.IO:
      return "io";
  }
  return "";
}
