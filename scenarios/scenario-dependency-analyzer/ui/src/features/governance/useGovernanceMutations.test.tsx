import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { createHookWrapper } from "../../test-utils";
import type { ApprovedDependencyRecord } from "../../api/governance";
import { useGovernanceData } from "./useGovernanceData";
import { useGovernanceMutations } from "./useGovernanceMutations";

const mocks = vi.hoisted(() => ({
  validateFleetApprovedDependencies: vi.fn(), listApprovedDependencies: vi.fn(),
  getApprovedDependencyTriage: vi.fn(), listSecurityGovernanceGaps: vi.fn(),
  upsertApprovedDependency: vi.fn(), previewVulnerabilityRemediation: vi.fn(), denyVulnerableDependency: vi.fn()
}));
vi.mock("../../api/governance", () => mocks);

const record = { ecosystem: "npm", packageName: "example", state: "approved" } as ApprovedDependencyRecord;
const vulnerability = { ecosystem: "npm", packageName: "example", vulnerabilityId: "CVE-example", affectedRange: "<2", fixedRange: ">=2", rationale: "Address reviewed vulnerability.", approvedBy: "maintainer" };

describe("governance mutations refresh every dependent view", () => {
  let revision = 0;
  beforeEach(() => {
    vi.clearAllMocks();
    revision = 0;
    mocks.validateFleetApprovedDependencies.mockImplementation(async () => ({ passed: revision > 0 }));
    mocks.listApprovedDependencies.mockImplementation(async () => ({ records: [], guidance: `records-${revision}` }));
    mocks.getApprovedDependencyTriage.mockImplementation(async () => ({ guidance: `triage-${revision}` }));
    mocks.listSecurityGovernanceGaps.mockImplementation(async () => ({ guidance: `gaps-${revision}` }));
    mocks.upsertApprovedDependency.mockImplementation(async (_record, dryRun) => {
      if (!dryRun) revision++;
      return { dryRun };
    });
    mocks.previewVulnerabilityRemediation.mockResolvedValue({ dryRun: true });
    mocks.denyVulnerableDependency.mockImplementation(async (input) => {
      if (input.dryRun === false) revision++;
      return { dryRun: input.dryRun !== false };
    });
  });

  function mountViews() {
    const { wrapper, queryClient } = createHookWrapper();
    const hook = renderHook(() => ({ data: useGovernanceData("advisory"), mutations: useGovernanceMutations() }), { wrapper });
    return { ...hook, queryClient };
  }

  for (const operation of ["apply-decision", "apply-denial"] as const) {
    it(`refreshes fleet, records, triage and security gaps after ${operation}`, async () => {
      const { result, unmount, queryClient } = mountViews();
      await waitFor(() => expect(result.current.data.loading).toBe(false));
      expect(result.current.data.triage?.guidance).toBe("triage-0");
      await act(async () => {
        if (operation === "apply-decision") await result.current.mutations.applyDecision.mutateAsync(record);
        else await result.current.mutations.denyVulnerable.mutateAsync({ ...vulnerability, dryRun: false });
      });
      await waitFor(() => {
        expect(result.current.data.fleet?.passed).toBe(true);
        expect(result.current.data.recordsGuidance).toBe("records-1");
        expect(result.current.data.triage?.guidance).toBe("triage-1");
        expect(result.current.data.securityGaps?.guidance).toBe("gaps-1");
      });
      unmount(); queryClient.clear();
    });
  }

  it("keeps all views unchanged for decision/remediation/denial previews", async () => {
    const { result, unmount, queryClient } = mountViews();
    await waitFor(() => expect(result.current.data.loading).toBe(false));
    await act(async () => {
      await result.current.mutations.previewDecision.mutateAsync(record);
      await result.current.mutations.previewRemediation.mutateAsync(vulnerability);
      await result.current.mutations.denyVulnerable.mutateAsync({ ...vulnerability, dryRun: true });
      await result.current.mutations.denyVulnerable.mutateAsync(vulnerability);
    });
    for (const fetcher of [mocks.validateFleetApprovedDependencies, mocks.listApprovedDependencies, mocks.getApprovedDependencyTriage, mocks.listSecurityGovernanceGaps]) expect(fetcher).toHaveBeenCalledTimes(1);
    expect(result.current.data.fleet?.passed).toBe(false);
    expect(result.current.data.recordsGuidance).toBe("records-0");
    expect(result.current.data.triage?.guidance).toBe("triage-0");
    expect(result.current.data.securityGaps?.guidance).toBe("gaps-0");
    unmount(); queryClient.clear();
  });

  it("retains current data when an applied mutation is rejected", async () => {
    const { result, unmount, queryClient } = mountViews();
    await waitFor(() => expect(result.current.data.loading).toBe(false));
    mocks.upsertApprovedDependency.mockRejectedValueOnce(new Error("governance refused"));
    await act(async () => {
      await expect(result.current.mutations.applyDecision.mutateAsync(record)).rejects.toThrow("governance refused");
    });
    expect(result.current.data.recordsGuidance).toBe("records-0");
    for (const fetcher of [mocks.validateFleetApprovedDependencies, mocks.listApprovedDependencies, mocks.getApprovedDependencyTriage, mocks.listSecurityGovernanceGaps]) expect(fetcher).toHaveBeenCalledTimes(1);
    unmount(); queryClient.clear();
  });
});
