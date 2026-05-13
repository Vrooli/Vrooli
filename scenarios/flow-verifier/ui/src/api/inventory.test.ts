/**
 * inventory.ts API client tests. After the proto + Connect-RPC cutover
 * these assert that the public types are faithful projections of the
 * generated proto message shape, and that the runFromProto /
 * flowSummaryFromProto helpers map enum values correctly.
 */
import { describe, expect, it } from "vitest";

import {
  RunStatus,
  RunMode,
  FailureReason as ProtoFailureReason,
  type Run as ProtoRun,
} from "@vrooli/proto-types/flow-verifier/v1/runs/runs_pb";
import type { FlowSummary as ProtoFlowSummary } from "@vrooli/proto-types/flow-verifier/v1/flows/flows_pb";

import { flowSummaryFromProto, runFromProto, type FlowSummary, type RunRow } from "./inventory";

describe("inventory API public type shape", () => {
  it("flowSummaryFromProto preserves every field exactly", () => {
    const proto: ProtoFlowSummary = {
      $typeName: "vrooli.flow_verifier.v1.flows.FlowSummary",
      flowId: "alpha.flow.api",
      contractPath: "scenarios/alpha/flow/flow.json",
      language: "go",
      schemaVersion: 6,
      scenarioId: "alpha",
      kind: "temporal",
    };
    const got: FlowSummary = flowSummaryFromProto(proto);
    expect(got).toEqual({
      flowId: "alpha.flow.api",
      contractPath: "scenarios/alpha/flow/flow.json",
      language: "go",
      schemaVersion: 6,
      scenarioId: "alpha",
    });
  });

  it("flowSummaryFromProto drops empty scenarioId to undefined", () => {
    const proto: ProtoFlowSummary = {
      $typeName: "vrooli.flow_verifier.v1.flows.FlowSummary",
      flowId: "x",
      contractPath: "y",
      language: "go",
      schemaVersion: 6,
      scenarioId: "",
      kind: "temporal",
    };
    expect(flowSummaryFromProto(proto).scenarioId).toBeUndefined();
  });

  it("runFromProto maps RunStatus and RunMode enums to lowercase strings", () => {
    const proto: ProtoRun = {
      $typeName: "vrooli.flow_verifier.v1.runs.Run",
      id: "r1",
      flowId: "f1",
      flowPath: "p",
      root: "/r",
      sourceSha256: "",
      modelSha256: "",
      genSha256: "",
      mode: RunMode.RUN,
      status: RunStatus.PASSED,
      counterexample: "",
      errorMessage: "",
      failureReason: ProtoFailureReason.UNSPECIFIED,
      missingArtifacts: [],
      output: "",
      startedAt: undefined,
      finishedAt: undefined,
      durationMs: BigInt(42),
    };
    const got: RunRow = runFromProto(proto);
    expect(got.status).toBe("passed");
    expect(got.mode).toBe("run");
    expect(got.durationMs).toBe(42);
    expect(got.failureReason).toBe("");
  });

  it("runFromProto translates FailureReason.MISSING_ARTIFACTS to the typed string", () => {
    const proto: ProtoRun = {
      $typeName: "vrooli.flow_verifier.v1.runs.Run",
      id: "r2",
      flowId: "f1",
      flowPath: "p",
      root: "/r",
      sourceSha256: "",
      modelSha256: "",
      genSha256: "",
      mode: RunMode.CHECK,
      status: RunStatus.FAILED,
      counterexample: "",
      errorMessage: "",
      failureReason: ProtoFailureReason.MISSING_ARTIFACTS,
      missingArtifacts: ["runtime.go"],
      output: "",
      startedAt: undefined,
      finishedAt: undefined,
      durationMs: BigInt(0),
    };
    const got = runFromProto(proto);
    expect(got.status).toBe("failed");
    expect(got.failureReason).toBe("missing_artifacts");
    expect(got.missingArtifacts).toEqual(["runtime.go"]);
  });
});
