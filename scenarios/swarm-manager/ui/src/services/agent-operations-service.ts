/**
 * Agent Operations service — the UI's typed client for the declarative
 * agent-operations operator surface (AgentOperationsService, Connect RPC).
 *
 * Talks Proto + Connect-RPC end-to-end via the generated client (mirroring
 * initiative-mode-service.ts) and projects the wire messages onto the domain
 * types in types/agent-operations.ts. The server is authoritative for every
 * precedence / compatibility / digest decision — this client only renders
 * typed results and never re-implements resolution logic.
 *
 * PutBindingOverride / DeleteBindingOverride / ListBindingOverrides /
 * ListCompatibleModes are part of the interface now so the slice-D override
 * UI only has to add components, not service plumbing.
 */

import { createClient, type Client } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { AgentOperationsService } from "@vrooli/proto-types/swarm-manager/v1/api/agent_operations_pb";
import { API_BASE } from "../lib/api-client";
import type {
  AgentOpsTarget,
  WorkflowBindingOverrideDocument,
  WorkflowCompatibleMode,
  WorkflowExecutionSummary,
  WorkflowMigrationStatus,
  WorkflowProjection,
  WorkflowPutBindingOverrideResult,
  WorkflowResolvedBinding,
} from "../types/agent-operations";
import {
  mapBindingOverrides,
  mapCompatibleModes,
  mapExecutionHistory,
  mapMigrationStatus,
  mapPutBindingOverride,
  mapResolvedBindings,
  mapWorkflowProjection,
  targetKindToProto,
} from "./proto/agent-operations-contracts";

export type AgentOperationsClient = Client<typeof AgentOperationsService>;

export interface PutBindingOverrideArgs {
  owner: AgentOpsTarget;
  operation: string;
  /** Optional exact contract-version pin; empty binds across versions. */
  operationVersion?: string;
  mode: string;
  modeRevision: string;
  /** Write an explicit fail-closed veto instead of a mode selection. */
  disabled?: boolean;
}

export interface IAgentOperationsService {
  /** Canonical workflow projection for a target (found=false → no workflow document). */
  getWorkflowProjection(target: AgentOpsTarget): Promise<WorkflowProjection>;
  /** Immutable execution provenance summaries, newest first. */
  listExecutionHistory(target: AgentOpsTarget, limit?: number): Promise<WorkflowExecutionSummary[]>;
  /** Winning binding + contributing layers per compatible catalog operation. */
  getResolvedBindings(target: AgentOpsTarget): Promise<WorkflowResolvedBinding[]>;
  /** Persisted-state migration status (absent document == not-started). */
  getMigrationStatus(): Promise<WorkflowMigrationStatus>;
  /** Authored modes with server-computed per-operation verdicts for a target. */
  listCompatibleModes(target: AgentOpsTarget, operation?: string): Promise<WorkflowCompatibleMode[]>;
  /** Raw override documents stored at one owner's layer. */
  listBindingOverrides(owner: AgentOpsTarget): Promise<WorkflowBindingOverrideDocument[]>;
  /** Write (idempotent-replace) one binding override at the owner's layer. */
  putBindingOverride(args: PutBindingOverrideArgs): Promise<WorkflowPutBindingOverrideResult>;
  /** Remove the owner's override; found=false when none matched (not an error). */
  deleteBindingOverride(
    owner: AgentOpsTarget,
    operation: string,
    operationVersion?: string,
  ): Promise<{ found: boolean }>;
}

function toSelector(target: AgentOpsTarget) {
  return { kind: targetKindToProto(target.kind), id: target.id };
}

function defaultAgentOperationsClient(): AgentOperationsClient {
  return createClient(AgentOperationsService, createConnectTransport({ baseUrl: API_BASE }));
}

export function createAgentOperationsService(
  client: AgentOperationsClient = defaultAgentOperationsClient(),
): IAgentOperationsService {
  return {
    async getWorkflowProjection(target: AgentOpsTarget): Promise<WorkflowProjection> {
      return mapWorkflowProjection(
        await client.getWorkflowProjection({ target: toSelector(target) }),
      );
    },

    async listExecutionHistory(
      target: AgentOpsTarget,
      limit = 0,
    ): Promise<WorkflowExecutionSummary[]> {
      return mapExecutionHistory(
        await client.listExecutionHistory({ target: toSelector(target), limit }),
      );
    },

    async getResolvedBindings(target: AgentOpsTarget): Promise<WorkflowResolvedBinding[]> {
      return mapResolvedBindings(
        await client.getResolvedBindings({ target: toSelector(target) }),
      );
    },

    async getMigrationStatus(): Promise<WorkflowMigrationStatus> {
      return mapMigrationStatus(await client.getMigrationStatus({}));
    },

    async listCompatibleModes(
      target: AgentOpsTarget,
      operation = "",
    ): Promise<WorkflowCompatibleMode[]> {
      return mapCompatibleModes(
        await client.listCompatibleModes({ target: toSelector(target), operation }),
      );
    },

    async listBindingOverrides(owner: AgentOpsTarget): Promise<WorkflowBindingOverrideDocument[]> {
      return mapBindingOverrides(
        await client.listBindingOverrides({ owner: toSelector(owner) }),
      );
    },

    async putBindingOverride(args: PutBindingOverrideArgs): Promise<WorkflowPutBindingOverrideResult> {
      return mapPutBindingOverride(
        await client.putBindingOverride({
          owner: toSelector(args.owner),
          operation: args.operation,
          operationVersion: args.operationVersion ?? "",
          mode: args.mode,
          modeRevision: args.modeRevision,
          disabled: args.disabled ?? false,
        }),
      );
    },

    async deleteBindingOverride(
      owner: AgentOpsTarget,
      operation: string,
      operationVersion = "",
    ): Promise<{ found: boolean }> {
      const resp = await client.deleteBindingOverride({
        owner: toSelector(owner),
        operation,
        operationVersion,
      });
      return { found: resp.found ?? false };
    },
  };
}

export const agentOperationsService = createAgentOperationsService();
