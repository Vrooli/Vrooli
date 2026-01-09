import { create } from 'zustand';
import { normalizeIdentifier } from '@/utils/appPreview';

const MAX_NO_ISSUE_PENALTY = 6;

export interface ScenarioEngagementRecord {
  lastVisitedAt: number;
  lastAutoSelectedAt: number | null;
  lastIssueCreatedAt: number | null;
  lastLeaveWithoutIssueAt: number | null;
  noIssuePenalty: number;
  totalVisits: number;
  autoVisits: number;
}

interface ScenarioSessionState {
  startedAt: number;
  viaAutoNext: boolean;
  issueCreated: boolean;
}

interface ScenarioEngagementState {
  records: Record<string, ScenarioEngagementRecord>;
  sessions: Record<string, ScenarioSessionState>;
  activeKey: string | null;
  beginSession: (identifier: string, options?: { viaAutoNext?: boolean; timestamp?: number }) => void;
  endSession: (identifier: string, options?: { timestamp?: number }) => void;
  markIssueCreated: (identifier: string, timestamp?: number) => void;
  registerAutoSelection: (identifier: string, timestamp?: number) => void;
}

const createDefaultRecord = (timestamp: number): ScenarioEngagementRecord => ({
  lastVisitedAt: timestamp,
  lastAutoSelectedAt: null,
  lastIssueCreatedAt: null,
  lastLeaveWithoutIssueAt: null,
  noIssuePenalty: 0,
  totalVisits: 0,
  autoVisits: 0,
});

const ensureKey = (identifier: string | null | undefined): string | null => normalizeIdentifier(identifier);

const finalizeRecord = (
  record: ScenarioEngagementRecord,
  session: ScenarioSessionState,
  timestamp: number,
): ScenarioEngagementRecord => {
  const next: ScenarioEngagementRecord = {
    ...record,
    lastVisitedAt: Math.max(record.lastVisitedAt, timestamp),
  };

  if (session.issueCreated) {
    next.noIssuePenalty = 0;
  } else {
    const penalty = Math.min(record.noIssuePenalty + 1, MAX_NO_ISSUE_PENALTY);
    next.noIssuePenalty = penalty;
    next.lastLeaveWithoutIssueAt = timestamp;
  }

  return next;
};

export const useScenarioEngagementStore = create<ScenarioEngagementState>((set) => ({
  records: {},
  sessions: {},
  activeKey: null,

  beginSession: (identifier, options) => {
    const key = ensureKey(identifier);
    if (!key) {
      return;
    }

    const timestamp = options?.timestamp ?? Date.now();

    set((state) => {
      const nextRecords = { ...state.records };
      const nextSessions = { ...state.sessions };
      const nextActiveKey = key;

      if (state.activeKey && state.activeKey !== key) {
        const previousKey = state.activeKey;
        const prevSession = state.sessions[previousKey];
        if (prevSession) {
          const prevRecord = nextRecords[previousKey] ?? createDefaultRecord(prevSession.startedAt);
          nextRecords[previousKey] = finalizeRecord(prevRecord, prevSession, timestamp);
          delete nextSessions[previousKey];
        }
      }

      const existingRecord = nextRecords[key] ?? createDefaultRecord(timestamp);
      const sessionExists = Boolean(nextSessions[key]);

      nextRecords[key] = {
        ...existingRecord,
        lastVisitedAt: Math.max(existingRecord.lastVisitedAt, timestamp),
        totalVisits: existingRecord.totalVisits + (sessionExists ? 0 : 1),
      };

      nextSessions[key] = {
        startedAt: timestamp,
        viaAutoNext: Boolean(options?.viaAutoNext),
        issueCreated: nextSessions[key]?.issueCreated ?? false,
      };

      return {
        records: nextRecords,
        sessions: nextSessions,
        activeKey: nextActiveKey,
      };
    });
  },

  endSession: (identifier, options) => {
    const key = ensureKey(identifier);
    if (!key) {
      return;
    }

    const timestamp = options?.timestamp ?? Date.now();

    set((state) => {
      const session = state.sessions[key];
      if (!session) {
        return state.activeKey === key
          ? { ...state, activeKey: null }
          : state;
      }

      const record = state.records[key] ?? createDefaultRecord(session.startedAt);
      const nextRecord = finalizeRecord(record, session, timestamp);
      const restSessions = { ...state.sessions };
      delete restSessions[key];
      const nextActiveKey = state.activeKey === key ? null : state.activeKey;

      return {
        records: {
          ...state.records,
          [key]: nextRecord,
        },
        sessions: restSessions,
        activeKey: nextActiveKey,
      };
    });
  },

  markIssueCreated: (identifier, timestamp) => {
    const key = ensureKey(identifier);
    if (!key) {
      return;
    }

    const resolvedTimestamp = timestamp ?? Date.now();

    set((state) => {
      const existingRecord = state.records[key] ?? createDefaultRecord(resolvedTimestamp);
      const nextRecord: ScenarioEngagementRecord = {
        ...existingRecord,
        lastVisitedAt: Math.max(existingRecord.lastVisitedAt, resolvedTimestamp),
        lastIssueCreatedAt: resolvedTimestamp,
        noIssuePenalty: 0,
      };

      if (existingRecord.lastLeaveWithoutIssueAt && existingRecord.lastLeaveWithoutIssueAt > resolvedTimestamp) {
        nextRecord.lastLeaveWithoutIssueAt = existingRecord.lastLeaveWithoutIssueAt;
      }

      const session = state.sessions[key];
      const nextSessions = session
        ? {
          ...state.sessions,
          [key]: {
            ...session,
            issueCreated: true,
          },
        }
        : state.sessions;

      return {
        records: {
          ...state.records,
          [key]: nextRecord,
        },
        sessions: nextSessions,
      };
    });
  },

  registerAutoSelection: (identifier, timestamp) => {
    const key = ensureKey(identifier);
    if (!key) {
      return;
    }

    const resolvedTimestamp = timestamp ?? Date.now();

    set((state) => {
      const existingRecord = state.records[key] ?? createDefaultRecord(resolvedTimestamp);
      const nextRecord: ScenarioEngagementRecord = {
        ...existingRecord,
        lastVisitedAt: Math.max(existingRecord.lastVisitedAt, resolvedTimestamp),
        lastAutoSelectedAt: resolvedTimestamp,
        autoVisits: existingRecord.autoVisits + 1,
      };

      const session = state.sessions[key];
      const nextSessions = session
        ? {
          ...state.sessions,
          [key]: {
            ...session,
            viaAutoNext: true,
          },
        }
        : state.sessions;

      return {
        records: {
          ...state.records,
          [key]: nextRecord,
        },
        sessions: nextSessions,
      };
    });
  },
}));

export const scenarioEngagementGetState = () => useScenarioEngagementStore.getState();
export type ScenarioEngagementStateSnapshot = ReturnType<typeof scenarioEngagementGetState>;
