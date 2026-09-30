import { describe, expect, it } from "vitest";
import { fromJson, toJsonString } from "@bufbuild/protobuf";
import { DeploymentSchema } from "@vrooli/proto-types/scenario-to-cloud/v1/deployments/deployments_pb";
import { ErrorSchema } from "@vrooli/proto-types/scenario-to-cloud/v1/errors/errors_pb";
import fixture from "./fixtures/deployment.v1.json";
import { ApiError, ErrorCodes, parseApiError } from "./apiErrors";

// [REQ:STC-P0-015] The UI parses the same fixture the Go round-trip test
// verifies, through the generated schema, so both clients read one contract.
describe("scenario-to-cloud deployment contract", () => {
  it("parses the shared Deployment fixture through the generated schema", () => {
    const deployment = fromJson(DeploymentSchema, fixture as never, { ignoreUnknownFields: false });

    expect(deployment.ref?.id).toBe("0f0e1d2c-3b4a-4596-8778-99aabbccddee");
    expect(deployment.ref?.scenarioId).toBe("demo-app");
    expect(deployment.ref?.environment).toBe("production");
    expect(deployment.ref?.target?.machineId).toBe("machine-7");
    expect(deployment.ref?.target?.enrollmentGeneration).toBe(2n);
    expect(deployment.ref?.target?.transport).toBe("bridge");
    expect(deployment.ref?.target?.locator?.host).toBe("203.0.113.10");
    expect(deployment.fence).toBe(4n);
    expect(deployment.status).toBe("deployed");
    expect(deployment.domain).toBe("demo.example");
    expect(deployment.createdAt?.seconds).toBe(BigInt(Date.UTC(2026, 8, 9, 10) / 1000));

    // Writing back with proto names reproduces the wire shape byte for byte
    // in content (key order aside), so the UI can echo identities to the API.
    const echoed = JSON.parse(toJsonString(DeploymentSchema, deployment, { useProtoFieldName: true }));
    expect(echoed).toEqual(fixture);
  });
});

// [REQ:STC-P0-017] Every client consumes the same stable code.
describe("typed API errors", () => {
  it("decodes the typed envelope with code, retryability and next action", () => {
    const body = JSON.stringify({
      error: {
        code: "deployment_selector_ambiguous",
        message: "Selector matches more than one deployment",
        retryable: false,
        next_action: { owner: "scenario-to-cloud", kind: "selector", reference: "id", label: "Select by deployment id" },
        details: { candidates: [{ id: "a" }, { id: "b" }] },
      },
    });
    const err = parseApiError(409, body);
    expect(err).toBeInstanceOf(ApiError);
    expect(err.code).toBe(ErrorCodes.deploymentSelectorAmbiguous);
    expect(err.status).toBe(409);
    expect(err.retryable).toBe(false);
    expect(err.nextAction?.reference).toBe("id");
    const echoed = JSON.parse(toJsonString(ErrorSchema, err.typed, { useProtoFieldName: true }));
    expect(echoed.details.candidates).toEqual([{ id: "a" }, { id: "b" }]);
    expect(echoed.next_action.owner).toBe("scenario-to-cloud");
  });

  it("falls back to a status-derived stable code for untyped bodies", () => {
    const err = parseApiError(404, "<html>gateway</html>");
    expect(err.code).toBe(ErrorCodes.deploymentNotFound);
    expect(err.message).toContain("gateway");
    expect(parseApiError(502, "").code).toBe(ErrorCodes.internal);
  });
});
