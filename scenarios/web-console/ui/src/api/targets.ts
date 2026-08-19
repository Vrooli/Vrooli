import { createClient } from "@connectrpc/connect";
import {
  TargetCatalogService,
} from "@vrooli/proto-types/web-console/v1/targets/targets_pb";
import { CatalogState } from "@vrooli/proto-types/web-console/v1/targets/targets_pb";
import { TargetState, type Target } from "@vrooli/proto-types/web-console/v1/shared/target_pb";

import { transport } from "./client";

export const targetCatalogClient = createClient(TargetCatalogService, transport);

export type TargetCatalogStatus = "ready" | "configured-empty" | "unconfigured" | "registry-error";
export type TerminalTargetState = "dispatchable" | "offline" | "needs-update" | "unavailable" | "unconfigured";

export interface TargetReadinessFact {
  key: string;
  label: string;
  passed: boolean;
  detail: string;
}

export interface TerminalTarget {
  id: string;
  kind: "local" | "bridge-node" | "ssh" | "attached";
  label: string;
  os?: string;
  arch?: string;
  node_id?: string;
  revision?: string;
  status?: string;
  online?: boolean;
  last_seen_at?: string;
  available: boolean;
  readiness?: TargetReadinessFact[];
  failure_rung?: string;
  state?: TerminalTargetState;
  recovery_action?: string;
  survives_restart?: boolean;
}

export interface TargetCatalog {
  status: TargetCatalogStatus;
  targets: TerminalTarget[];
  message?: string;
  recovery_action?: string;
}

function catalogStatus(state: CatalogState): TargetCatalogStatus {
  switch (state) {
    case CatalogState.CONFIGURED_EMPTY:
      return "configured-empty";
    case CatalogState.UNCONFIGURED:
      return "unconfigured";
    case CatalogState.REGISTRY_ERROR:
      return "registry-error";
    default:
      return "ready";
  }
}

function targetState(target: Target): TerminalTargetState {
  if (target.dispatchable) return "dispatchable";
  switch (target.state) {
    case TargetState.OFFLINE:
      return "offline";
    case TargetState.NEEDS_UPDATE:
      return "needs-update";
    default:
      return target.failureRung.toLowerCase().includes("credential") ? "unconfigured" : "unavailable";
  }
}

function targetKind(kind: string): TerminalTarget["kind"] {
  switch (kind) {
    case "local":
    case "bridge-node":
    case "ssh":
    case "attached":
      return kind;
    default:
      return "bridge-node";
  }
}

function timestampString(timestamp: Target["lastSeenAt"]): string | undefined {
  if (!timestamp) return undefined;
  const seconds = Number(timestamp.seconds);
  if (!Number.isFinite(seconds) || seconds <= 0) return undefined;
  return new Date(seconds * 1000).toISOString();
}

export function decodeTarget(target: Target): TerminalTarget {
  return {
    id: target.id,
    kind: targetKind(target.kind),
    label: target.label || target.id,
    os: target.os,
    arch: target.arch,
    node_id: target.nodeId || undefined,
    revision: target.revision || undefined,
    status: target.status,
    online: target.online,
    last_seen_at: timestampString(target.lastSeenAt),
    available: target.dispatchable,
    readiness: target.readiness.map((fact) => ({
      key: fact.key,
      label: fact.label,
      passed: fact.passed,
      detail: fact.detail,
    })),
    failure_rung: target.failureRung || undefined,
    state: targetState(target),
    recovery_action: target.recoveryAction || undefined,
    survives_restart: target.survivesRestart,
  };
}

export async function listTargetCatalog(): Promise<TargetCatalog> {
  const response = await targetCatalogClient.list({});
  return {
    status: catalogStatus(response.state),
    targets: response.targets.map(decodeTarget),
    message: response.message || undefined,
    recovery_action: response.recoveryAction || undefined,
  };
}

export async function getTarget(id: string): Promise<TerminalTarget> {
  const response = await targetCatalogClient.get({ id });
  if (!response.target) throw new Error(`Target catalog returned no target for ${id}`);
  return decodeTarget(response.target);
}
