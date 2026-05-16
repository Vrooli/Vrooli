import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { buildQueryString } from "../lib/query-utils";
import type {
  AgentSession,
  AgentSessionAttachment,
  AgentSessionArtifact,
  AgentSessionArtifactType,
  AgentSessionContextRef,
  AgentSessionKind,
  AgentSessionRunEvent,
  AgentSessionStatus,
} from "../types";
import {
  applyAgentSessionProposalResponseSchema,
  cancelAgentSessionResponseSchema,
  continueAgentSessionResponseSchema,
  createAgentSessionResponseSchema,
  deleteAgentSessionResponseSchema,
  getAgentSessionResponseSchema,
  getArtifactsByEntityResponseSchema,
  listAgentSessionEventsResponseSchema,
  listAgentSessionArtifactsResponseSchema,
  listAgentSessionsResponseSchema,
  mapProtoAgentSession,
  mapProtoAgentSessionAttachment,
  mapProtoAgentSessionArtifact,
  mapProtoAgentSessionRunEvent,
  parseProtoResponse,
  refreshAgentSessionResponseSchema,
  requireProtoField,
  startAgentSessionResponseSchema,
  uploadAgentSessionAttachmentsResponseSchema,
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
}

export interface ContinueAgentSessionArgs {
  sessionId: string;
  message: string;
  attachmentIds?: string[];
  contextRefs?: AgentSessionContextRef[];
  autoContextPolicy?: "default" | "none";
}

export interface ApplyAgentSessionProposalResult {
  session: AgentSession;
  artifacts: AgentSessionArtifact[];
}

export interface ListAgentSessionEventsArgs {
  sessionId: string;
  afterSequence?: bigint;
  limit?: number;
}

export interface ListAgentSessionEventsResult {
  events: AgentSessionRunEvent[];
  hasMore: boolean;
  nextAfterSequence: bigint;
}

export interface IAgentSessionService {
  list(filters?: ListAgentSessionsFilters): Promise<AgentSession[]>;
  get(sessionId: string): Promise<AgentSession>;
  create(args: CreateAgentSessionArgs): Promise<AgentSession>;
  start(args: ContinueAgentSessionArgs): Promise<AgentSession>;
  continue(args: ContinueAgentSessionArgs): Promise<AgentSession>;
  uploadAttachments(sessionId: string, files: File[]): Promise<AgentSessionAttachment[]>;
  listEvents(args: ListAgentSessionEventsArgs): Promise<ListAgentSessionEventsResult>;
  refresh(sessionId: string): Promise<AgentSession>;
  cancel(sessionId: string): Promise<AgentSession>;
  delete(sessionId: string): Promise<string>;
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
      });
      const parsed = parseProtoResponse(createAgentSessionResponseSchema, data, "agent session create");
      return mapProtoAgentSession(requireProtoField(parsed.session, "agent session"));
    },

    async start(args: ContinueAgentSessionArgs): Promise<AgentSession> {
      const body: Record<string, unknown> = {
        session_id: args.sessionId,
        message: args.message,
        attachment_ids: args.attachmentIds ?? [],
        context_refs: args.contextRefs ?? [],
      };
      if (args.autoContextPolicy) {
        body.auto_context_policy = args.autoContextPolicy;
      }
      const data = await apiClient.post<unknown>(API_ENDPOINTS.agentSessionStart(args.sessionId), body);
      const parsed = parseProtoResponse(startAgentSessionResponseSchema, data, "agent session start");
      return mapProtoAgentSession(requireProtoField(parsed.session, "agent session"));
    },

    async continue(args: ContinueAgentSessionArgs): Promise<AgentSession> {
      const data = await apiClient.post<unknown>(API_ENDPOINTS.agentSessionContinue(args.sessionId), {
        session_id: args.sessionId,
        message: args.message,
        attachment_ids: args.attachmentIds ?? [],
        context_refs: args.contextRefs ?? [],
      });
      const parsed = parseProtoResponse(continueAgentSessionResponseSchema, data, "agent session continue");
      return mapProtoAgentSession(requireProtoField(parsed.session, "agent session"));
    },

    async uploadAttachments(sessionId: string, files: File[]): Promise<AgentSessionAttachment[]> {
      if (files.length === 0) return [];
      const form = new FormData();
      for (const file of files) {
        form.append("files", file);
      }
      const data = await apiClient.post<unknown>(API_ENDPOINTS.agentSessionAttachments(sessionId), form);
      const parsed = parseProtoResponse(uploadAgentSessionAttachmentsResponseSchema, data, "agent session attachment upload");
      return parsed.attachments.map((attachment) => ({
        ...mapProtoAgentSessionAttachment(attachment),
        url: API_ENDPOINTS.agentSessionAttachment(sessionId, attachment.id),
      }));
    },

    async listEvents(args: ListAgentSessionEventsArgs): Promise<ListAgentSessionEventsResult> {
      const suffix = buildQueryString({
        after_sequence: args.afterSequence?.toString(),
        limit: args.limit,
      });
      const data = await apiClient.get<unknown>(`${API_ENDPOINTS.agentSessionEvents(args.sessionId)}${suffix}`);
      const parsed = parseProtoResponse(listAgentSessionEventsResponseSchema, data, "agent session events");
      return {
        events: parsed.events.map(mapProtoAgentSessionRunEvent),
        hasMore: parsed.hasMore,
        nextAfterSequence: parsed.nextAfterSequence,
      };
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

    async delete(sessionId: string): Promise<string> {
      const data = await apiClient.delete<unknown>(API_ENDPOINTS.agentSessionById(sessionId));
      const parsed = parseProtoResponse(deleteAgentSessionResponseSchema, data, "agent session delete");
      return parsed.sessionId;
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
