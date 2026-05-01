import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { buildQueryString } from "../lib/query-utils";
import type {
  AgentSession,
  AgentSessionArtifact,
  AgentSessionArtifactType,
  AgentSessionKind,
  AgentSessionStatus,
} from "../types";
import {
  applyAgentSessionProposalResponseSchema,
  cancelAgentSessionResponseSchema,
  continueAgentSessionResponseSchema,
  createAgentSessionResponseSchema,
  getAgentSessionResponseSchema,
  getArtifactsByEntityResponseSchema,
  listAgentSessionArtifactsResponseSchema,
  listAgentSessionsResponseSchema,
  mapProtoAgentSession,
  mapProtoAgentSessionArtifact,
  parseProtoResponse,
  refreshAgentSessionResponseSchema,
  requireProtoField,
} from "./proto-contracts";

export interface ListAgentSessionsFilters {
  kind?: AgentSessionKind;
  status?: AgentSessionStatus;
  activeOnly?: boolean;
  limit?: number;
}

export interface CreateAgentSessionArgs {
  kind: AgentSessionKind;
  title: string;
  initialMessage: string;
  initiative?: string;
}

export interface ContinueAgentSessionArgs {
  sessionId: string;
  message: string;
  attachmentIds?: string[];
}

export interface ApplyAgentSessionProposalResult {
  session: AgentSession;
  artifacts: AgentSessionArtifact[];
}

export interface IAgentSessionService {
  list(filters?: ListAgentSessionsFilters): Promise<AgentSession[]>;
  get(sessionId: string): Promise<AgentSession>;
  create(args: CreateAgentSessionArgs): Promise<AgentSession>;
  continue(args: ContinueAgentSessionArgs): Promise<AgentSession>;
  refresh(sessionId: string): Promise<AgentSession>;
  cancel(sessionId: string): Promise<AgentSession>;
  applyProposal(sessionId: string, proposalId: string): Promise<ApplyAgentSessionProposalResult>;
  listArtifacts(sessionId: string): Promise<AgentSessionArtifact[]>;
  getArtifactsByEntity(artifactType: AgentSessionArtifactType, entityRef: string): Promise<AgentSessionArtifact[]>;
}

export function createAgentSessionService(apiClient: IApiClient = defaultApiClient): IAgentSessionService {
  return {
    async list(filters?: ListAgentSessionsFilters): Promise<AgentSession[]> {
      const suffix = buildQueryString({
        kind: filters?.kind,
        status: filters?.status,
        active_only: filters?.activeOnly,
        limit: filters?.limit,
      });
      const data = await apiClient.get<unknown>(`${API_ENDPOINTS.agentSessions}${suffix}`);
      const parsed = parseProtoResponse(listAgentSessionsResponseSchema, data, "agent sessions");
      return parsed.sessions.map(mapProtoAgentSession);
    },

    async get(sessionId: string): Promise<AgentSession> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.agentSessionById(sessionId));
      const parsed = parseProtoResponse(getAgentSessionResponseSchema, data, "agent session");
      return mapProtoAgentSession(requireProtoField(parsed.session, "agent session"));
    },

    async create(args: CreateAgentSessionArgs): Promise<AgentSession> {
      const data = await apiClient.post<unknown>(API_ENDPOINTS.agentSessions, {
        kind: args.kind,
        title: args.title,
        initial_message: args.initialMessage,
        ...(args.initiative ? { initiative: args.initiative } : {}),
      });
      const parsed = parseProtoResponse(createAgentSessionResponseSchema, data, "agent session create");
      return mapProtoAgentSession(requireProtoField(parsed.session, "agent session"));
    },

    async continue(args: ContinueAgentSessionArgs): Promise<AgentSession> {
      const data = await apiClient.post<unknown>(API_ENDPOINTS.agentSessionContinue(args.sessionId), {
        session_id: args.sessionId,
        message: args.message,
        attachment_ids: args.attachmentIds ?? [],
      });
      const parsed = parseProtoResponse(continueAgentSessionResponseSchema, data, "agent session continue");
      return mapProtoAgentSession(requireProtoField(parsed.session, "agent session"));
    },

    async refresh(sessionId: string): Promise<AgentSession> {
      const data = await apiClient.post<unknown>(API_ENDPOINTS.agentSessionRefresh(sessionId), {});
      const parsed = parseProtoResponse(refreshAgentSessionResponseSchema, data, "agent session refresh");
      return mapProtoAgentSession(requireProtoField(parsed.session, "agent session"));
    },

    async cancel(sessionId: string): Promise<AgentSession> {
      const data = await apiClient.post<unknown>(API_ENDPOINTS.agentSessionCancel(sessionId), {});
      const parsed = parseProtoResponse(cancelAgentSessionResponseSchema, data, "agent session cancel");
      return mapProtoAgentSession(requireProtoField(parsed.session, "agent session"));
    },

    async applyProposal(sessionId: string, proposalId: string): Promise<ApplyAgentSessionProposalResult> {
      const data = await apiClient.post<unknown>(
        API_ENDPOINTS.agentSessionApplyProposal(sessionId, proposalId),
        { session_id: sessionId, proposal_id: proposalId }
      );
      const parsed = parseProtoResponse(
        applyAgentSessionProposalResponseSchema,
        data,
        "agent session proposal apply"
      );
      return {
        session: mapProtoAgentSession(requireProtoField(parsed.session, "agent session")),
        artifacts: parsed.artifacts.map(mapProtoAgentSessionArtifact),
      };
    },

    async listArtifacts(sessionId: string): Promise<AgentSessionArtifact[]> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.agentSessionArtifacts(sessionId));
      const parsed = parseProtoResponse(listAgentSessionArtifactsResponseSchema, data, "agent session artifacts");
      return parsed.artifacts.map(mapProtoAgentSessionArtifact);
    },

    async getArtifactsByEntity(
      artifactType: AgentSessionArtifactType,
      entityRef: string
    ): Promise<AgentSessionArtifact[]> {
      const suffix = buildQueryString({
        artifact_type: artifactType,
        entity_ref: entityRef,
      });
      const data = await apiClient.get<unknown>(`${API_ENDPOINTS.agentSessionArtifactsByEntity}${suffix}`);
      const parsed = parseProtoResponse(getArtifactsByEntityResponseSchema, data, "agent session artifacts by entity");
      return parsed.artifacts.map(mapProtoAgentSessionArtifact);
    },
  };
}

export const agentSessionService = createAgentSessionService();
