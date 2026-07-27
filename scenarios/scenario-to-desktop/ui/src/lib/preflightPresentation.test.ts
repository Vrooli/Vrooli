import { describe, expect, it } from "vitest";
import { CheckStatus, PreflightCheckStep } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";
import { presentPreflight } from "./preflightPresentation";

describe("presentPreflight", () => {
  it("presents typed preflight evidence without leaking protobuf representation details", () => {
    const result = presentPreflight({
      ports: [{ serviceId: "api", name: "http", port: 15200 }],
      checks: [{ id: "runtime", step: PreflightCheckStep.RUNTIME, name: "Runtime", status: CheckStatus.PASSED, detail: "ready" }],
      errors: [{ message: "missing optional asset" }],
      serviceFingerprints: [],
      logTails: [],
    } as never);

    expect(result.ports).toEqual({ api: { http: 15200 } });
    expect(result.checks).toEqual([{ id: "runtime", step: "runtime", name: "Runtime", status: "pass", detail: "ready" }]);
    expect(result.errors).toEqual(["missing optional asset"]);
  });

  it("preserves optional timestamps and translates unknown enum values safely", () => {
    const result = presentPreflight({
      ports: [],
      checks: [{ id: "unknown", step: PreflightCheckStep.UNSPECIFIED, name: "Unknown", status: CheckStatus.UNSPECIFIED }],
      errors: [],
      serviceFingerprints: [{ serviceId: "shell", binarySizeBytes: 12n }],
      logTails: [],
      ready: { ready: true, details: [], waitedSeconds: 3, snapshotAt: { seconds: 2n, nanos: 0 } },
    } as never);

    expect(result.ready).toMatchObject({ ready: true, waited_seconds: 3, snapshot_at: "1970-01-01T00:00:02.000Z" });
    expect(result.checks[0]).toMatchObject({ step: "", status: "warning" });
    expect(result.fingerprints[0]).toMatchObject({ service_id: "shell", binary_size_bytes: 12 });
  });
});
