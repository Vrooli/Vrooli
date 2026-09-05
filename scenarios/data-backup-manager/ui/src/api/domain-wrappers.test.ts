import { afterEach, describe, expect, it, vi } from "vitest";

import { AuditStatus, auditsClient, getAudit, isTerminalAudit, listAudits, runSnapshotAudit } from "./audits";
import { acceptDefaultTargets, coverageClient, getCoverageReport } from "./coverage";
import {
	analyzeDestination,
	createDestination,
	deleteDestination,
	destinationsClient,
	executeDestinationPreparation,
	getDestination,
	getDestinationUsage,
	listDestinations,
	planDestinationPreparation,
	updateDestination,
} from "./destinations";
import { discoveryClient, dismissSuggestion, listDestinationSuggestions, listTargetSuggestions } from "./discovery";
import { drillsClient, listDrills, runDrill } from "./drills";
import { createPlan, deletePlan, getPlan, listPlans, plansClient, updatePlan } from "./plans";
import { getRestore, listRestores, restoreTarget, restoresClient, verifyTarget } from "./restores";
import {
	browseSnapshot,
	getRun,
	getRunStats,
	listRuns,
	listTargetStatus,
	runsClient,
	triggerRun,
} from "./runs";
import { deregisterTarget, getTarget, listTargets, registerTarget, targetsClient } from "./targets";

const response = (value: unknown) => value as never;

afterEach(() => vi.restoreAllMocks());

describe("Connect domain wrappers", () => {
  it("maps audit, coverage, discovery, and target responses", async () => {
    vi.spyOn(auditsClient, "listAudits").mockResolvedValue(response({ audits: [] }));
    vi.spyOn(auditsClient, "getAudit").mockResolvedValue(response({ audit: undefined }));
    vi.spyOn(auditsClient, "runSnapshotAudit").mockResolvedValue(response({ audit: undefined }));
    vi.spyOn(coverageClient, "getCoverageReport").mockResolvedValue(response({ report: undefined }));
    vi.spyOn(coverageClient, "acceptDefaultTargets").mockResolvedValue(response({}));
    vi.spyOn(discoveryClient, "listTargetSuggestions").mockResolvedValue(response({ suggestions: [] }));
    vi.spyOn(discoveryClient, "listDestinationSuggestions").mockResolvedValue(response({ suggestions: [] }));
    vi.spyOn(discoveryClient, "dismissSuggestion").mockResolvedValue(response({ dismissed: true }));
    vi.spyOn(drillsClient, "listDrills").mockResolvedValue(response({ drills: [] }));
    vi.spyOn(drillsClient, "runDrill").mockResolvedValue(response({ drill: undefined }));
    vi.spyOn(targetsClient, "listTargets").mockResolvedValue(response({ targets: [] }));
    vi.spyOn(targetsClient, "getTarget").mockResolvedValue(response({ target: undefined }));
    vi.spyOn(targetsClient, "registerTarget").mockResolvedValue(response({ target: undefined }));
    vi.spyOn(targetsClient, "deregisterTarget").mockResolvedValue(response({ removed: true }));

    await expect(listAudits()).resolves.toEqual([]);
    await expect(getAudit("audit-1")).resolves.toBeUndefined();
    await expect(runSnapshotAudit({ targetId: "t", destinationId: "d", snapshotId: "s" })).resolves.toBeUndefined();
    await expect(runSnapshotAudit({ targetId: "t", destinationId: "d", snapshotId: "s", includeContentHash: false, includeSqliteChecks: false })).resolves.toBeUndefined();
    await expect(getCoverageReport()).resolves.toBeUndefined();
    await expect(acceptDefaultTargets()).resolves.toEqual({});
    await expect(acceptDefaultTargets({ includeSensitive: true, dryRun: true })).resolves.toEqual({});
    await expect(listTargetSuggestions()).resolves.toEqual([]);
    await expect(listDestinationSuggestions()).resolves.toEqual([]);
    await expect(dismissSuggestion("suggestion-1")).resolves.toBe(true);
    await expect(listDrills()).resolves.toEqual([]);
    await expect(listDrills("plan-1", "target-1")).resolves.toEqual([]);
    await expect(runDrill("plan-1")).resolves.toBeUndefined();
    await expect(runDrill("plan-1", "target-1", "dest-1", "idempotency-1")).resolves.toBeUndefined();
    await expect(listTargets()).resolves.toEqual([]);
    await expect(getTarget("target-1")).resolves.toBeUndefined();
    await expect(registerTarget({ owner: "owner", name: "name", sourceKind: 0, locator: "loc", critical: false })).resolves.toBeUndefined();
    await expect(deregisterTarget("owner", "name")).resolves.toBe(true);
    expect(isTerminalAudit(AuditStatus.COMPLETED)).toBe(true);
    expect(isTerminalAudit(AuditStatus.UNSPECIFIED)).toBe(false);
  });

  it("maps destination, plan, restore, and run responses", async () => {
    vi.spyOn(destinationsClient, "listDestinations").mockResolvedValue(response({ destinations: [] }));
    vi.spyOn(destinationsClient, "getDestination").mockResolvedValue(response({ destination: undefined }));
    vi.spyOn(destinationsClient, "createDestination").mockResolvedValue(response({ destination: undefined }));
    vi.spyOn(destinationsClient, "updateDestination").mockResolvedValue(response({ destination: undefined }));
    vi.spyOn(destinationsClient, "deleteDestination").mockResolvedValue(response({}));
    vi.spyOn(destinationsClient, "getDestinationUsage").mockResolvedValue(response({}));
    vi.spyOn(destinationsClient, "analyzeDestination").mockResolvedValue(response({ report: undefined }));
    vi.spyOn(destinationsClient, "planDestinationPreparation").mockResolvedValue(response({ plan: undefined }));
    vi.spyOn(destinationsClient, "executeDestinationPreparation").mockResolvedValue(response({}));
    vi.spyOn(plansClient, "listPlans").mockResolvedValue(response({ plans: [] }));
    vi.spyOn(plansClient, "getPlan").mockResolvedValue(response({ plan: undefined }));
    vi.spyOn(plansClient, "createPlan").mockResolvedValue(response({ plan: undefined }));
    vi.spyOn(plansClient, "updatePlan").mockResolvedValue(response({ plan: undefined }));
    vi.spyOn(plansClient, "deletePlan").mockResolvedValue(response({}));
    vi.spyOn(restoresClient, "listRestores").mockResolvedValue(response({ restores: [] }));
    vi.spyOn(restoresClient, "getRestore").mockResolvedValue(response({ restore: undefined }));
    vi.spyOn(restoresClient, "verifyTarget").mockResolvedValue(response({ restore: undefined }));
    vi.spyOn(restoresClient, "restoreTarget").mockResolvedValue(response({ restore: undefined }));
    vi.spyOn(runsClient, "listRuns").mockResolvedValue(response({ runs: [] }));
    vi.spyOn(runsClient, "getRun").mockResolvedValue(response({ run: undefined }));
    vi.spyOn(runsClient, "triggerRun").mockResolvedValue(response({ run: undefined }));
    vi.spyOn(runsClient, "listTargetStatus").mockResolvedValue(response({ statuses: [] }));
    vi.spyOn(runsClient, "browseSnapshot").mockResolvedValue(response({ entries: [] }));
    vi.spyOn(runsClient, "getRunStats").mockResolvedValue(response({ stats: undefined }));

    await expect(listDestinations()).resolves.toEqual([]);
    await expect(getDestination("d")).resolves.toBeUndefined();
    await expect(createDestination({ name: "d", backendKind: 0, location: "l", capBytes: 0n, capPolicy: 0 })).resolves.toBeUndefined();
    await expect(updateDestination({ id: "d", capBytes: 0n, capPolicy: 0 })).resolves.toBeUndefined();
    await expect(deleteDestination("d", false)).resolves.toBeUndefined();
    await expect(getDestinationUsage("d")).resolves.toEqual({});
    await expect(analyzeDestination({ location: "l" })).resolves.toBeUndefined();
    await expect(analyzeDestination({ location: "l", proposedSubdir: "sub", selectedTargetBytes: 10n, retentionCopies: 2, crossPlatformRequired: true })).resolves.toBeUndefined();
    await expect(planDestinationPreparation({} as never)).resolves.toBeUndefined();
    await expect(executeDestinationPreparation({} as never)).resolves.toEqual({});
    await expect(listPlans()).resolves.toEqual([]);
    await expect(getPlan("p")).resolves.toBeUndefined();
    await expect(createPlan({ name: "p", targetIds: [], destinationIds: [], schedule: "", keepLatest: 1, enabled: true })).resolves.toBeUndefined();
    await expect(createPlan({ name: "p", targetIds: [], destinationIds: [], schedule: "", keepLatest: 1, enabled: true, allowIncompleteCoverage: true })).resolves.toBeUndefined();
    await expect(updatePlan("p", { name: "p", targetIds: [], destinationIds: [], schedule: "", keepLatest: 1, enabled: true })).resolves.toBeUndefined();
    await expect(deletePlan("p")).resolves.toBeUndefined();
    await expect(listRestores()).resolves.toEqual([]);
    await expect(getRestore("r")).resolves.toBeUndefined();
    await expect(verifyTarget("t", "d", "s")).resolves.toBeUndefined();
    await expect(restoreTarget("t", "d", "s", "/tmp")).resolves.toBeUndefined();
    await expect(listRuns()).resolves.toEqual([]);
    await expect(getRun("r")).resolves.toBeUndefined();
    await expect(triggerRun("p")).resolves.toBeUndefined();
    await expect(listTargetStatus()).resolves.toEqual([]);
    await expect(browseSnapshot("d", "s", "nested")).resolves.toEqual([]);
    await expect(getRunStats()).resolves.toBeUndefined();
  });
});
