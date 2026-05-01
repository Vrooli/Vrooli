import { create } from "zustand";
import {
  agentSessionService,
  type CreateAgentSessionArgs,
  type ContinueAgentSessionArgs,
  type IAgentSessionService,
  type ListAgentSessionsFilters,
} from "../services/agent-session-service";
import type { AgentSession, AgentSessionArtifact } from "../types";
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
  "starting",
  "running",
  "waiting_for_user",
  "proposal_ready",
  "applying",
]);

interface AgentSessionStoreState {
  sessions: AgentSession[];
  activeSession: AgentSession | null;
  artifactsByEntity: Record<string, AgentSessionArtifact[]>;
  status: LoadStatus;
  error: Error | null;
  isRefreshing: boolean;
  isMutating: boolean;
  lastFetchedAt: number | null;
  fetchSessions: (filters?: ListAgentSessionsFilters, options?: { force?: boolean }) => Promise<void>;
  openSession: (sessionId: string) => Promise<void>;
  setActiveSession: (session: AgentSession | null) => void;
  createSession: (args: CreateAgentSessionArgs) => Promise<AgentSession>;
  continueSession: (args: ContinueAgentSessionArgs) => Promise<AgentSession>;
  refreshSession: (sessionId: string) => Promise<AgentSession>;
  cancelSession: (sessionId: string) => Promise<AgentSession>;
  applyProposal: (sessionId: string, proposalId: string) => Promise<AgentSessionArtifact[]>;
  loadArtifactsByEntity: (artifactType: AgentSessionArtifact["artifactType"], entityRef: string) => Promise<AgentSessionArtifact[]>;
  reset: () => void;
}

const hydrated = loadFromStorage<AgentSession[]>(PERSIST_CONFIG, []);
let service: IAgentSessionService = agentSessionService;

export const agentSessionStoreInitialState = {
  sessions: hydrated.data,
  activeSession: null,
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
    const { status, sessions, lastFetchedAt, isRefreshing } = get();
    if (status === "loading" || isRefreshing) return;
    if (!shouldRefetch({ lastFetchedAt, hasData: sessions.length > 0, force })) return;

    const hasData = sessions.length > 0;
    set({ status: hasData ? "success" : "loading", isRefreshing: hasData, error: null });
    try {
      const result = await fetchWithRetry(() => service.list(filters));
      const now = Date.now();
      set({
        sessions: sortSessions(result),
        status: "success",
        error: null,
        isRefreshing: false,
        lastFetchedAt: now,
      });
      saveToStorage(PERSIST_CONFIG, result, now);
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to load agent sessions."),
        status: hasData ? "success" : "error",
        isRefreshing: false,
      });
    }
  },

  openSession: async (sessionId): Promise<void> => {
    if (!sessionId) return;
    set({ isRefreshing: true, error: null });
    try {
      const session = await service.get(sessionId);
      set((state) => ({
        activeSession: session,
        sessions: upsertSession(state.sessions, session),
        isRefreshing: false,
      }));
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to load agent session."),
        isRefreshing: false,
      });
      throw error;
    }
  },

  setActiveSession: (session): void => {
    set((state) => ({
      activeSession: session,
      sessions: session ? upsertSession(state.sessions, session) : state.sessions,
    }));
  },

  createSession: async (args): Promise<AgentSession> => {
    set({ isMutating: true, error: null });
    try {
      const session = await service.create(args);
      set((state) => ({
        activeSession: session,
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

  continueSession: async (args): Promise<AgentSession> => {
    set({ isMutating: true, error: null });
    try {
      const session = await service.continue(args);
      set((state) => ({
        activeSession: session,
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

  refreshSession: async (sessionId): Promise<AgentSession> => {
    const session = await service.refresh(sessionId);
    set((state) => ({
      activeSession: state.activeSession?.id === session.id ? session : state.activeSession,
      sessions: upsertSession(state.sessions, session),
    }));
    return session;
  },

  cancelSession: async (sessionId): Promise<AgentSession> => {
    set({ isMutating: true, error: null });
    try {
      const session = await service.cancel(sessionId);
      set((state) => ({
        activeSession: state.activeSession?.id === session.id ? session : state.activeSession,
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

  applyProposal: async (sessionId, proposalId): Promise<AgentSessionArtifact[]> => {
    set({ isMutating: true, error: null });
    try {
      const result = await service.applyProposal(sessionId, proposalId);
      set((state) => ({
        activeSession: result.session,
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
      activeSession: null,
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

function sortSessions(sessions: AgentSession[]): AgentSession[] {
  return [...sessions].sort((a, b) => {
    const first = new Date(a.updatedAt).getTime();
    const second = new Date(b.updatedAt).getTime();
    if (first === second) return b.id.localeCompare(a.id);
    return second - first;
  });
}
