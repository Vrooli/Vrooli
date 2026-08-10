import { create } from "zustand";
import {
  agentSessionService,
  type CreateAgentSessionArgs,
  type ContinueAgentSessionArgs,
  type IAgentSessionService,
  type ListAgentSessionEventsArgs,
  type ListAgentSessionEventsResult,
  type ListAgentSessionsFilters,
} from "../services/agent-session-service";
import type { AgentSession, AgentSessionAttachment, AgentSessionArtifact, AgentSessionContextItem } from "../types";
import {
  clearStorage,
  fetchWithRetry,
  loadFromStorage,
  saveToStorage,
  shouldRefetch,
  type LoadStatus,
  type StorePersistConfig,
} from "./store-utils";

const PERSIST_CONFIG: StorePersistConfig = {
  key: "swarm-manager.agent-sessions.v1",
  version: 1,
  maxItems: 100,
};

const ACTIVE_STATUSES = new Set<AgentSession["status"]>([
  "draft",
  "starting",
  "running",
  "waiting_for_user",
  "proposal_ready",
  "applying",
]);

interface AgentSessionStoreState {
  sessions: AgentSession[];
  artifactsByEntity: Record<string, AgentSessionArtifact[]>;
  status: LoadStatus;
  error: Error | null;
  isRefreshing: boolean;
  isMutating: boolean;
  lastFetchedAt: number | null;
  fetchSessions: (filters?: ListAgentSessionsFilters, options?: { force?: boolean }) => Promise<void>;
  loadSession: (sessionId: string) => Promise<AgentSession>;
  loadStartupBrief: (sessionId: string, refresh?: boolean) => Promise<AgentSessionContextItem>;
  createSession: (args: CreateAgentSessionArgs) => Promise<AgentSession>;
  startSession: (args: ContinueAgentSessionArgs) => Promise<AgentSession>;
  continueSession: (args: ContinueAgentSessionArgs) => Promise<AgentSession>;
  uploadSessionAttachments: (sessionId: string, files: File[]) => Promise<AgentSessionAttachment[]>;
  listSessionEvents: (args: ListAgentSessionEventsArgs) => Promise<ListAgentSessionEventsResult>;
  refreshSession: (sessionId: string) => Promise<AgentSession>;
  cancelSession: (sessionId: string) => Promise<AgentSession>;
  deleteSession: (sessionId: string) => Promise<void>;
  applyProposal: (sessionId: string, proposalId: string) => Promise<AgentSessionArtifact[]>;
  loadArtifactsByEntity: (artifactType: AgentSessionArtifact["artifactType"], entityRef: string) => Promise<AgentSessionArtifact[]>;
  reset: () => void;
}

const hydrated = loadFromStorage<AgentSession[]>(PERSIST_CONFIG, []);
let service: IAgentSessionService = agentSessionService;

export const agentSessionStoreInitialState = {
  sessions: hydrated.data,
  artifactsByEntity: {} as Record<string, AgentSessionArtifact[]>,
  status: (hydrated.data.length > 0 ? "success" : "idle") as LoadStatus,
  error: null,
  isRefreshing: false,
  isMutating: false,
  lastFetchedAt: hydrated.lastFetchedAt,
};

export function setAgentSessionStoreService(nextService: IAgentSessionService): void {
  service = nextService;
}

export function resetAgentSessionStoreService(): void {
  service = agentSessionService;
}

export const useAgentSessionStore = create<AgentSessionStoreState>((set, get) => ({
  ...agentSessionStoreInitialState,

  fetchSessions: async (filters, { force = false } = {}): Promise<void> => {
    const { status, sessions, lastFetchedAt } = get();
    if (status === "loading") return;
    if (!shouldRefetch({ lastFetchedAt, hasData: sessions.length > 0, force })) return;

    const filteredRequest = hasSessionListFilters(filters);
    const hasData = sessions.length > 0;
    set({ status: hasData ? "success" : "loading", isRefreshing: hasData, error: null });
    try {
      const result = await fetchWithRetry(() => service.list(filters));
      const now = Date.now();
      set((state) => {
        const nextSessions = filteredRequest && state.sessions.length > 0
          ? mergeSessions(state.sessions, result)
          : sortSessions(result);
        if (!filteredRequest) {
          saveToStorage(PERSIST_CONFIG, result, now);
        }
        return {
          sessions: nextSessions,
          status: "success",
          error: null,
          isRefreshing: false,
          lastFetchedAt: filteredRequest ? state.lastFetchedAt : now,
        };
      });
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to load agent sessions."),
        status: hasData ? "success" : "error",
        isRefreshing: false,
      });
    }
  },

  loadSession: async (sessionId): Promise<AgentSession> => {
    set({ isRefreshing: true, error: null });
    try {
      const session = await service.get(sessionId);
      set((state) => ({
        sessions: upsertSession(state.sessions, session),
        isRefreshing: false,
      }));
      return session;
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to load agent session."),
        isRefreshing: false,
      });
      throw error;
    }
  },

  loadStartupBrief: async (sessionId, refresh = false): Promise<AgentSessionContextItem> => {
    return service.getStartupBrief(sessionId, refresh);
  },

  createSession: async (args): Promise<AgentSession> => {
    set({ isMutating: true, error: null });
    try {
      const session = await service.create(args);
      set((state) => ({
        sessions: upsertSession(state.sessions, session),
        isMutating: false,
      }));
      return session;
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to create agent session."),
        isMutating: false,
      });
      throw error;
    }
  },

  startSession: async (args): Promise<AgentSession> => {
    set({ isMutating: true, error: null });
    try {
      const session = await service.start(args);
      set((state) => ({
        sessions: upsertSession(state.sessions, session),
        isMutating: false,
      }));
      return session;
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to start agent session."),
        isMutating: false,
      });
      throw error;
    }
  },

  continueSession: async (args): Promise<AgentSession> => {
    set({ isMutating: true, error: null });
    try {
      const session = await service.continue(args);
      set((state) => ({
        sessions: upsertSession(state.sessions, session),
        isMutating: false,
      }));
      return session;
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to continue agent session."),
        isMutating: false,
      });
      throw error;
    }
  },

  uploadSessionAttachments: async (sessionId, files): Promise<AgentSessionAttachment[]> => {
    set({ isMutating: true, error: null });
    try {
      const attachments = await service.uploadAttachments(sessionId, files);
      set((state) => ({
        sessions: state.sessions.map((session) =>
          session.id === sessionId
            ? {
                ...session,
                attachments: mergeAttachments(session.attachments ?? [], attachments),
              }
            : session,
        ),
        isMutating: false,
      }));
      return attachments;
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to upload session attachments."),
        isMutating: false,
      });
      throw error;
    }
  },

  listSessionEvents: async (args): Promise<ListAgentSessionEventsResult> => {
    return service.listEvents(args);
  },

  refreshSession: async (sessionId): Promise<AgentSession> => {
    const session = await service.refresh(sessionId);
    // The polling path, so it takes the change-detecting upsert. Mutations
    // below keep the plain one: they always carry a real change, and paying a
    // comparison to discover that would be pure overhead.
    set((state) => ({
      sessions: upsertSessionIfChanged(state.sessions, session),
    }));
    return session;
  },

  cancelSession: async (sessionId): Promise<AgentSession> => {
    set({ isMutating: true, error: null });
    try {
      const session = await service.cancel(sessionId);
      set((state) => ({
        sessions: upsertSession(state.sessions, session),
        isMutating: false,
      }));
      return session;
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to cancel agent session."),
        isMutating: false,
      });
      throw error;
    }
  },

  deleteSession: async (sessionId): Promise<void> => {
    set({ isMutating: true, error: null });
    try {
      const deletedSessionId = await service.delete(sessionId);
      const now = Date.now();
      set((state) => {
        const sessions = state.sessions.filter((session) => session.id !== deletedSessionId);
        saveToStorage(PERSIST_CONFIG, sessions, now);
        return {
          sessions,
          artifactsByEntity: filterArtifactsByDeletedSession(state.artifactsByEntity, deletedSessionId),
          isMutating: false,
          lastFetchedAt: now,
        };
      });
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to delete agent session."),
        isMutating: false,
      });
      throw error;
    }
  },

  applyProposal: async (sessionId, proposalId): Promise<AgentSessionArtifact[]> => {
    set({ isMutating: true, error: null });
    try {
      const result = await service.applyProposal(sessionId, proposalId);
      set((state) => ({
        sessions: upsertSession(state.sessions, result.session),
        isMutating: false,
      }));
      return result.artifacts;
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to apply agent session proposal."),
        isMutating: false,
      });
      throw error;
    }
  },

  loadArtifactsByEntity: async (artifactType, entityRef): Promise<AgentSessionArtifact[]> => {
    const artifacts = await service.getArtifactsByEntity(artifactType, entityRef);
    set((state) => ({
      artifactsByEntity: {
        ...state.artifactsByEntity,
        [artifactEntityKey(artifactType, entityRef)]: artifacts,
      },
    }));
    return artifacts;
  },

  reset: (): void => {
    clearStorage(PERSIST_CONFIG.key);
    set({
      sessions: [],
      artifactsByEntity: {},
      status: "idle",
      error: null,
      isRefreshing: false,
      isMutating: false,
      lastFetchedAt: null,
    });
  },
}));

export function selectActiveAgentSessions(state: AgentSessionStoreState): AgentSession[] {
  return state.sessions.filter((session) => ACTIVE_STATUSES.has(session.status));
}

export function isActiveAgentSession(session: AgentSession): boolean {
  return ACTIVE_STATUSES.has(session.status);
}

export function artifactEntityKey(artifactType: AgentSessionArtifact["artifactType"], entityRef: string): string {
  return `${artifactType}:${entityRef}`;
}

function upsertSession(sessions: AgentSession[], session: AgentSession): AgentSession[] {
  return sortSessions([session, ...sessions.filter((entry) => entry.id !== session.id)]);
}

/**
 * Upsert that keeps the existing array — and the existing session object —
 * when the incoming copy is identical.
 *
 * The detail page polls every 3s for as long as a session is running. Most of
 * those responses are byte-for-byte what we already hold, but replacing the
 * object anyway re-rendered the entire page: new session object → new messages
 * array → every bubble re-renders → every message re-parses its markdown, and
 * (before the renderer was made identity-stable) every mermaid diagram
 * remounted and re-laid-out. Returning the previous state lets Zustand's
 * reference equality stop the update at the store instead.
 *
 * Compared by serialisation rather than field-by-field on purpose: both sides
 * come from the same service deserialiser, so key order matches, and a
 * hand-written comparison would silently go stale the next time the session
 * shape grows a field. The cost is one pass over a payload we just parsed —
 * far below the render it avoids.
 */
function upsertSessionIfChanged(sessions: AgentSession[], session: AgentSession): AgentSession[] {
  const existing = sessions.find((entry) => entry.id === session.id);
  if (existing && JSON.stringify(existing) === JSON.stringify(session)) return sessions;
  return upsertSession(sessions, session);
}

function mergeSessions(existing: AgentSession[], incoming: AgentSession[]): AgentSession[] {
  const byID = new Map<string, AgentSession>();
  for (const session of existing) {
    byID.set(session.id, session);
  }
  for (const session of incoming) {
    byID.set(session.id, session);
  }
  return sortSessions(Array.from(byID.values()));
}

function hasSessionListFilters(filters?: ListAgentSessionsFilters): boolean {
  return !!(filters?.kind || filters?.status || filters?.activeOnly || (filters?.limit && filters.limit > 0));
}

function sortSessions(sessions: AgentSession[]): AgentSession[] {
  return [...sessions].sort((a, b) => {
    const first = new Date(a.updatedAt).getTime();
    const second = new Date(b.updatedAt).getTime();
    if (first === second) return b.id.localeCompare(a.id);
    return second - first;
  });
}

function mergeAttachments(existing: AgentSessionAttachment[] = [], incoming: AgentSessionAttachment[]): AgentSessionAttachment[] {
  const byId = new Map<string, AgentSessionAttachment>();
  for (const attachment of existing) {
    byId.set(attachment.id, attachment);
  }
  for (const attachment of incoming) {
    byId.set(attachment.id, attachment);
  }
  return Array.from(byId.values());
}

function filterArtifactsByDeletedSession(
  artifactsByEntity: Record<string, AgentSessionArtifact[]>,
  sessionId: string,
): Record<string, AgentSessionArtifact[]> {
  const next: Record<string, AgentSessionArtifact[]> = {};
  for (const [key, artifacts] of Object.entries(artifactsByEntity)) {
    const remaining = artifacts.filter((artifact) => artifact.sessionId !== sessionId);
    if (remaining.length > 0) {
      next[key] = remaining;
    }
  }
  return next;
}
