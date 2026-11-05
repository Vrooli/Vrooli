import { describe, expect, it } from "vitest";
import { extractStageResults } from "./build";

describe("extractStageResults", () => {
  it("extracts every typed stage result and non-empty stage log", () => {
    const status = {
      stages: {
        bundle: { details: { kind: { case: "bundle", value: { id: "b" } } } },
        preflight: {
          details: { kind: { case: "preflight", value: { id: "p" } } },
        },
        generate: {
          details: { kind: { case: "generate", value: { id: "g" } } },
        },
        build: { details: { kind: { case: "build", value: { id: "b" } } } },
        smoketest: {
          details: { kind: { case: "smokeTest", value: { id: "s" } } },
        },
        deploy: { details: { kind: { case: "deploy", value: { id: "d" } } } },
        empty: { logs: [] },
        logged: { logs: ["stage complete"] },
      },
    } as never;

    const results = extractStageResults(status);

    expect(results.bundleResult).toEqual({ id: "b" });
    expect(results.preflightResult).toEqual({ id: "p" });
    expect(results.generateResult).toEqual({ id: "g" });
    expect(results.buildResult).toEqual({ id: "b" });
    expect(results.smokeTestResult).toEqual({ id: "s" });
    expect(results.deployResult).toEqual({ id: "d" });
    expect(results.stageLogs).toEqual({ logged: ["stage complete"] });
  });
});
