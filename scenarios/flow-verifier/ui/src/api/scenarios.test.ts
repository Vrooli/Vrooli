/**
 * scenarios.ts API client tests. After the proto + Connect-RPC cutover
 * these assert that the public types are a faithful projection of the
 * generated proto message shape — no Raw → Public normalisation.
 */
import { describe, expect, it } from "vitest";

import type {
  ScenarioSummary as ProtoScenarioSummary,
  ScenarioDetail as ProtoScenarioDetail,
} from "@vrooli/proto-types/flow-verifier/v1/scenarios/scenarios_pb";

import type { ScenarioDetail, ScenarioSummary } from "./scenarios";

describe("scenarios API public type shape", () => {
  it("ScenarioSummary mirrors the generated proto message field names", () => {
    const proto: ProtoScenarioSummary = {
      $typeName: "vrooli.flow_verifier.v1.scenarios.ScenarioSummary",
      id: "alpha",
      displayName: "Alpha",
      description: "",
      path: "/repo/scenarios/alpha",
      flowCount: 2,
      discoveryError: "",
    };
    const projected: ScenarioSummary = {
      id: proto.id,
      displayName: proto.displayName,
      description: proto.description || undefined,
      path: proto.path,
      flowCount: proto.flowCount,
      discoveryError: proto.discoveryError || undefined,
    };
    expect(projected.id).toBe("alpha");
    expect(projected.flowCount).toBe(2);
  });

  it("ScenarioDetail carries the embedded summary fields plus flows[]", () => {
    const detail: ScenarioDetail = {
      id: "alpha",
      displayName: "Alpha",
      path: "/repo/scenarios/alpha",
      flowCount: 1,
      flows: [
        {
          flowId: "alpha.flow.api",
          contractPath: "scenarios/alpha/flow/flow.json",
          language: "go",
          schemaVersion: 6,
          kind: "temporal",
        },
      ],
    };
    expect(detail.flows).toHaveLength(1);
    expect(detail.flows[0]!.flowId).toBe("alpha.flow.api");
  });

  // Forces a compile-time check that the proto message keys haven't
  // drifted out from under us. If buf regenerates with a renamed field
  // this assignment fails to compile.
  it("ProtoScenarioDetail carries summary + flows fields", () => {
    const _: ProtoScenarioDetail = {
      $typeName: "vrooli.flow_verifier.v1.scenarios.ScenarioDetail",
      summary: undefined,
      flows: [],
    };
    void _;
    expect(true).toBe(true);
  });
});
