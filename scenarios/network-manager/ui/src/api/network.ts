import { createClient } from "@connectrpc/connect";

import { AdapterService } from "@vrooli/proto-types/network-manager/v1/adapters/adapters_pb";
import type { Capability, PlatformSummary } from "@vrooli/proto-types/network-manager/v1/adapters/adapters_pb";
import { InventoryService } from "@vrooli/proto-types/network-manager/v1/inventory/inventory_pb";
import type { Device } from "@vrooli/proto-types/network-manager/v1/inventory/inventory_pb";
import { OptimizationService } from "@vrooli/proto-types/network-manager/v1/optimization/optimization_pb";
import type { OptimizationRun } from "@vrooli/proto-types/network-manager/v1/optimization/optimization_pb";
import { PolicyService } from "@vrooli/proto-types/network-manager/v1/policy/policy_pb";
import type { PolicyChange } from "@vrooli/proto-types/network-manager/v1/policy/policy_pb";
import { PrivacyService } from "@vrooli/proto-types/network-manager/v1/privacy/privacy_pb";
import type { RetentionSettings, VisibilitySettings } from "@vrooli/proto-types/network-manager/v1/privacy/privacy_pb";
import { ResolverService } from "@vrooli/proto-types/network-manager/v1/resolver/resolver_pb";
import type { ResolverStatus } from "@vrooli/proto-types/network-manager/v1/resolver/resolver_pb";
import { SnapshotService } from "@vrooli/proto-types/network-manager/v1/snapshot/snapshot_pb";
import type { Snapshot } from "@vrooli/proto-types/network-manager/v1/snapshot/snapshot_pb";

import { transport } from "./client";

const adapterClient = createClient(AdapterService, transport);
const inventoryClient = createClient(InventoryService, transport);
const optimizationClient = createClient(OptimizationService, transport);
const policyClient = createClient(PolicyService, transport);
const privacyClient = createClient(PrivacyService, transport);
const resolverClient = createClient(ResolverService, transport);
const snapshotClient = createClient(SnapshotService, transport);

export interface ControlCenterOverview {
  snapshots: Snapshot[];
  resolverStatus?: ResolverStatus;
  capabilities: Capability[];
  platform?: PlatformSummary;
  devices: Device[];
  retention?: RetentionSettings;
  visibility?: VisibilitySettings;
}

export async function fetchControlCenterOverview(): Promise<ControlCenterOverview> {
  const [
    snapshots,
    resolverStatus,
    capabilities,
    platform,
    devices,
    retention,
    visibility,
  ] = await Promise.all([
    snapshotClient.listSnapshots({}),
    resolverClient.getResolverStatus({}),
    adapterClient.listCapabilities({}),
    adapterClient.getPlatformSummary({}),
    inventoryClient.listDevices({ group: "" }),
    privacyClient.getRetentionSettings({}),
    privacyClient.getVisibilitySettings({}),
  ]);

  return {
    snapshots: snapshots.snapshots,
    resolverStatus: resolverStatus.status,
    capabilities: capabilities.capabilities,
    platform: platform.summary,
    devices: devices.devices,
    retention: retention.settings,
    visibility: visibility.settings,
  };
}

export async function runSnapshot(profile = "home"): Promise<Snapshot | undefined> {
  const resp = await snapshotClient.runSnapshot({ profile, dryRun: false });
  return resp.snapshot;
}

export async function exportSnapshotReport(id: string): Promise<string> {
  const resp = await snapshotClient.exportSnapshotReport({ id, format: "markdown" });
  return resp.report;
}

export async function fetchResolverStatus(): Promise<ResolverStatus | undefined> {
  const resp = await resolverClient.getResolverStatus({});
  return resp.status;
}

export async function previewUpstreams(upstreams: string[]): Promise<string[]> {
  const resp = await resolverClient.updateUpstreams({ upstreams, dryRun: true });
  return resp.changes;
}

export async function previewPolicyChange(input: {
  target: string;
  action: string;
  values: string[];
}): Promise<PolicyChange | undefined> {
  const resp = await policyClient.previewPolicyChange(input);
  return resp.preview;
}

export async function applyPolicyChange(previewId: string): Promise<PolicyChange | undefined> {
  const resp = await policyClient.applyPolicyChange({ previewId, approved: true });
  return resp.change;
}

export async function rollbackPolicyChange(id: string): Promise<PolicyChange | undefined> {
  const resp = await policyClient.rollbackPolicyChange({ id });
  return resp.change;
}

export async function refreshInventory(): Promise<{ devices: Device[]; findings: string[] }> {
  const resp = await inventoryClient.refreshInventory({ dryRun: false });
  return { devices: resp.devices, findings: resp.findings };
}

export async function fetchDevices(group = ""): Promise<Device[]> {
  const resp = await inventoryClient.listDevices({ group });
  return resp.devices;
}

export async function updateDeviceGroup(id: string, group: string): Promise<Device | undefined> {
  const resp = await inventoryClient.updateDeviceGroup({ id, group });
  return resp.device;
}

export async function createOptimizationRun(): Promise<OptimizationRun | undefined> {
  const resp = await optimizationClient.createOptimizationRun({
    scoringProfile: "reliability-first",
    dryRun: false,
  });
  return resp.run;
}

export async function scoreOptimizationRun(runId: string): Promise<OptimizationRun | undefined> {
  const resp = await optimizationClient.scoreCandidates({ runId });
  return resp.run;
}

export async function approveOptimizationCandidate(
  runId: string,
  candidateId: string,
): Promise<OptimizationRun | undefined> {
  const resp = await optimizationClient.approveCandidate({
    runId,
    candidateId,
    approved: true,
  });
  return resp.run;
}

export async function fetchPrivacySettings(): Promise<{
  retention?: RetentionSettings;
  visibility?: VisibilitySettings;
}> {
  const [retention, visibility] = await Promise.all([
    privacyClient.getRetentionSettings({}),
    privacyClient.getVisibilitySettings({}),
  ]);
  return { retention: retention.settings, visibility: visibility.settings };
}

export type {
  Capability,
  Device,
  OptimizationRun,
  PlatformSummary,
  PolicyChange,
  ResolverStatus,
  RetentionSettings,
  Snapshot,
  VisibilitySettings,
};
