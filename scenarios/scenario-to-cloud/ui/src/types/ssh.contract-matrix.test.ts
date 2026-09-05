import { describe, expect, it } from "vitest";
import matrix from "../../../contracts/ssh-contract-matrix.json";
import {
  SSH_API_OUTCOME_STATUSES,
  SSH_COPY_KEY_API_STATUSES,
  SSH_ERROR_HINTS,
  SSH_TEST_API_STATUSES,
} from "./ssh";

type MatrixEndpoint = {
  request_required_fields?: string[];
  response_required_fields?: string[];
  response_statuses?: string[];
};

type MatrixShape = {
  outcome_statuses: string[];
  endpoints: Record<string, MatrixEndpoint>;
};

function toSorted(values: readonly string[]): string[] {
  return [...values].sort();
}

describe("ssh contract matrix", () => {
  const contract = matrix as MatrixShape;

  it("matches outcome status vocabulary", () => {
    expect(toSorted(SSH_API_OUTCOME_STATUSES)).toEqual(toSorted(contract.outcome_statuses));
  });

  it("matches /ssh/test statuses", () => {
    const endpoint = contract.endpoints["/api/v1/ssh/test"];
    if (!endpoint) throw new Error("missing /ssh/test endpoint contract");
    expect(toSorted(SSH_TEST_API_STATUSES)).toEqual(toSorted(endpoint.response_statuses ?? []));
  });

  it("matches /ssh/copy-key statuses", () => {
    const endpoint = contract.endpoints["/api/v1/ssh/copy-key"];
    if (!endpoint) throw new Error("missing /ssh/copy-key endpoint contract");
    expect(toSorted(SSH_COPY_KEY_API_STATUSES)).toEqual(toSorted(endpoint.response_statuses ?? []));
  });

  it("has UI error hints for all /ssh/test failure statuses", () => {
    const failureStatuses = SSH_TEST_API_STATUSES.filter((status) => status !== "success");
    for (const status of failureStatuses) {
      expect(SSH_ERROR_HINTS).toHaveProperty(status);
    }
  });
});
