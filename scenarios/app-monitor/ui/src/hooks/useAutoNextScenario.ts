import { logger } from '@/services/logger';
import { useCallback, useState } from 'react';
import type { App } from '@/types';
import {
  normalizeIdentifier,
  parseTimestampValue,
  resolveAppIdentifier,
  isRunningStatus,
  isStoppedStatus,
} from '@/utils/appPreview';
import { useScenarioEngagementStore } from '@/state/scenarioEngagementStore';
import { useScenarioIssuesStore } from '@/state/scenarioIssuesStore';

const RUNNING_BOOST_MS = 20 * 60 * 1000;
const STOPPED_PENALTY_MS = 10 * 60 * 1000;
const PENALTY_STEP_MS = 45 * 60 * 1000;
const AUTO_REPEAT_COOLDOWN_MS = 25 * 60 * 1000;
const ISSUE_BLOCK_PENALTY_MS = 24 * 60 * 60 * 1000;
const STALE_ISSUE_PENALTY_MS = 5 * 60 * 1000;
const LOADING_ISSUE_PENALTY_MS = 2 * 60 * 1000;
const ERROR_ISSUE_PENALTY_MS = 90 * 1000;
const VIEW_COUNT_WEIGHT_MS = 2 * 60 * 1000;
const MAX_CANDIDATES = 24;
const PREPARED_MAX_AGE_MS = 60 * 1000;
const PREPARED_REFRESH_GUARD_MS = 10 * 1000;

interface ScoredCandidate {
  app: App;
  key: string;
  navigationId: string;
  score: number;
}

type PreparedSnapshot =
  | {
    type: 'candidate';
    signature: string;
    preparedAt: number;
    candidate: ScoredCandidate;
    issueKey: string | null;
  }
  | {
    type: 'fallback';
    signature: string;
    preparedAt: number;
    message: string;
    lastCandidateKey: string | null;
  };

let preparedSnapshot: PreparedSnapshot | null = null;

const getPreparedSnapshot = (): PreparedSnapshot | null => preparedSnapshot;
const setPreparedSnapshot = (snapshot: PreparedSnapshot | null) => {
  preparedSnapshot = snapshot;
};

const buildAppsDigest = (apps: App[]): string => (
  apps
    .map((app) => {
      const navigationId = resolveNavigationId(app) ?? app.id ?? '';
      const status = app.status ?? '';
      const updated = app.updated_at ?? '';
      const viewed = Number.isFinite(app.view_count) ? Number(app.view_count) : '';
      return `${navigationId}|${status}|${updated}|${viewed}`;
    })
    .join(';')
);

const createParamsSignature = (
  apps: App[],
  currentKey: string | null,
  maxCandidates?: number,
): string => {
  const digest = buildAppsDigest(apps);
  const normalizedMax = typeof maxCandidates === 'number' && Number.isFinite(maxCandidates)
    ? String(maxCandidates)
    : 'default';
  return `${currentKey ?? 'none'}|${normalizedMax}|${digest}`;
};

interface AutoNextState {
  status: 'idle' | 'running' | 'error';
  message: string | null;
  lastSelectedKey: string | null;
}

export interface AutoNextParams {
  apps: App[];
  currentAppId?: string | null;
  onSelect: (app: App, navigationId: string) => void;
  maxCandidates?: number;
}

const resolveNavigationId = (app: App): string | null => {
  const identifier = resolveAppIdentifier(app) ?? app.id ?? null;
  if (!identifier) {
    return null;
  }
  return identifier.trim();
};

const getIssueStoreKey = (app: App): string | null => normalizeIdentifier(app.id) ?? normalizeIdentifier(resolveAppIdentifier(app));

export const useAutoNextScenario = () => {
  const [state, setState] = useState<AutoNextState>({ status: 'idle', message: null, lastSelectedKey: null });
  const engagement = useScenarioEngagementStore(store => store.records);
  const registerAutoSelection = useScenarioEngagementStore(store => store.registerAutoSelection);
  const issueEntries = useScenarioIssuesStore(store => store.entries);
  const fetchIssues = useScenarioIssuesStore(store => store.fetchIssues);
  interface EvaluateParams {
    apps: App[];
    currentAppId?: string | null;
    maxCandidates?: number;
    now: number;
  }

  type EvaluateResult =
    | { kind: 'candidate'; candidate: ScoredCandidate; issueKey: string | null }
    | { kind: 'fallback'; message: string; lastCandidateKey: string | null }
    | { kind: 'error'; message: string };

  const evaluateAutoNext = useCallback(async ({ apps, currentAppId, maxCandidates, now }: EvaluateParams): Promise<EvaluateResult> => {
    const currentKey = normalizeIdentifier(currentAppId);

    const scored: ScoredCandidate[] = [];

    apps.forEach((app) => {
      const navigationId = resolveNavigationId(app);
      const scenarioKey = normalizeIdentifier(navigationId);
      if (!navigationId || !scenarioKey) {
        return;
      }
      if (currentKey && scenarioKey === currentKey) {
        return;
      }

      const record = engagement[scenarioKey];
      const lastVisited = record?.lastVisitedAt
        ?? parseTimestampValue(app.last_viewed_at)
        ?? parseTimestampValue(app.updated_at)
        ?? parseTimestampValue(app.created_at)
        ?? 0;
      const baseTimestamp = lastVisited
        || parseTimestampValue(app.updated_at)
        || parseTimestampValue(app.created_at)
        || (now - 7 * 24 * 60 * 60 * 1000);

      let score = baseTimestamp;

      if (isRunningStatus(app.status)) {
        score += RUNNING_BOOST_MS;
      } else if (isStoppedStatus(app.status)) {
        score -= STOPPED_PENALTY_MS;
      }

      if (record?.noIssuePenalty) {
        score -= record.noIssuePenalty * PENALTY_STEP_MS;
      }

      if (record?.lastAutoSelectedAt) {
        const sinceAuto = now - record.lastAutoSelectedAt;
        if (sinceAuto < AUTO_REPEAT_COOLDOWN_MS) {
          score -= (AUTO_REPEAT_COOLDOWN_MS - sinceAuto);
        }
      }

      const viewCount = Number.isFinite(app.view_count) ? Number(app.view_count) : 0;
      if (viewCount > 0) {
        score += Math.min(viewCount, 15) * VIEW_COUNT_WEIGHT_MS;
      }

      const issueKey = getIssueStoreKey(app);
      const issueEntry = issueKey ? issueEntries[issueKey] : undefined;
      if (issueEntry) {
        if (issueEntry.status === 'ready') {
          const open = issueEntry.openCount ?? 0;
          const active = issueEntry.activeCount ?? 0;
          if (open + active > 0) {
            score -= ISSUE_BLOCK_PENALTY_MS;
          } else if (issueEntry.stale) {
            score -= STALE_ISSUE_PENALTY_MS;
          }
        } else if (issueEntry.status === 'loading') {
          score -= LOADING_ISSUE_PENALTY_MS;
        } else if (issueEntry.status === 'error') {
          score -= ERROR_ISSUE_PENALTY_MS;
        }
      }

      scored.push({
        app,
        key: scenarioKey,
        navigationId,
        score,
      });
    });

    if (scored.length === 0) {
      return { kind: 'error', message: 'No alternative scenarios to open.' };
    }

    const maxPool = typeof maxCandidates === 'number' && Number.isFinite(maxCandidates)
      ? Math.max(1, Math.floor(maxCandidates))
      : MAX_CANDIDATES;

    const ordered = scored
      .sort((a, b) => b.score - a.score)
      .slice(0, maxPool);

    let fallback: { candidate: ScoredCandidate; reason: 'issues' | 'error' } | null = null;

    for (const candidate of ordered) {
      const issueKey = getIssueStoreKey(candidate.app);
      const existingEntry = issueKey ? issueEntries[issueKey] : undefined;
      const shouldRefresh = !existingEntry
        || existingEntry.status === 'idle'
        || existingEntry.status === 'loading'
        || existingEntry.stale
        || (existingEntry.fetchedAt && (now - existingEntry.fetchedAt > 2 * 60 * 1000));

      let entry = existingEntry;
      if (shouldRefresh) {
        entry = await fetchIssues(candidate.app.id);
      }

      if (entry?.status === 'ready') {
        const open = entry.openCount ?? 0;
        const active = entry.activeCount ?? 0;
        if (open + active > 0) {
          if (!fallback) {
            fallback = { candidate, reason: 'issues' };
          }
          continue;
        }
      }

      if (entry?.status === 'error') {
        if (!fallback) {
          fallback = { candidate, reason: 'error' };
        }
        continue;
      }

      return {
        kind: 'candidate',
        candidate,
        issueKey: issueKey ?? null,
      };
    }

    if (fallback) {
      const message = fallback.reason === 'error'
        ? 'Issue status could not be verified right now. Try again shortly or pick a scenario manually.'
        : 'All suitable scenarios currently have unresolved issues. Clear issues or pick one manually.';
      return {
        kind: 'fallback',
        message,
        lastCandidateKey: fallback.candidate.key,
      };
    }

    return { kind: 'error', message: 'Unable to select a scenario automatically.' };
  }, [engagement, fetchIssues, issueEntries]);

  const autoSelect = useCallback(async ({ apps, currentAppId, onSelect, maxCandidates }: AutoNextParams): Promise<App | null> => {
    if (!apps || apps.length === 0) {
      setState({ status: 'error', message: 'No scenarios available to open.', lastSelectedKey: null });
      return null;
    }

    const currentKey = normalizeIdentifier(currentAppId);
    const signature = createParamsSignature(apps, currentKey ?? null, maxCandidates);
    const now = Date.now();
    const snapshot = getPreparedSnapshot();

    if (snapshot && snapshot.signature === signature && (now - snapshot.preparedAt) <= PREPARED_MAX_AGE_MS) {
      if (snapshot.type === 'candidate') {
        const issueEntry = snapshot.issueKey ? issueEntries[snapshot.issueKey] : undefined;
        const hasBlockingIssues = issueEntry?.status === 'ready'
          && ((issueEntry.openCount ?? 0) + (issueEntry.activeCount ?? 0) > 0);
        const isErrored = issueEntry?.status === 'error';
        if (!hasBlockingIssues && !isErrored) {
          registerAutoSelection(snapshot.candidate.navigationId);
          setState({ status: 'idle', message: null, lastSelectedKey: snapshot.candidate.key });
          setPreparedSnapshot(null);
          onSelect(snapshot.candidate.app, snapshot.candidate.navigationId);
          return snapshot.candidate.app;
        }
      } else if ((now - snapshot.preparedAt) <= PREPARED_REFRESH_GUARD_MS) {
        setState({ status: 'error', message: snapshot.message, lastSelectedKey: snapshot.lastCandidateKey });
        return null;
      }
    }

    setState(prev => ({ ...prev, status: 'running', message: null }));

    try {
      const result = await evaluateAutoNext({ apps, currentAppId, maxCandidates, now: Date.now() });

      if (result.kind === 'candidate') {
        registerAutoSelection(result.candidate.navigationId);
        setPreparedSnapshot(null);
        setState({ status: 'idle', message: null, lastSelectedKey: result.candidate.key });
        onSelect(result.candidate.app, result.candidate.navigationId);
        return result.candidate.app;
      }

      if (result.kind === 'fallback') {
        const fallbackSnapshot: PreparedSnapshot = {
          type: 'fallback',
          signature,
          preparedAt: Date.now(),
          message: result.message,
          lastCandidateKey: result.lastCandidateKey,
        };
        setPreparedSnapshot(fallbackSnapshot);
        setState({ status: 'error', message: result.message, lastSelectedKey: result.lastCandidateKey });
        return null;
      }

      setPreparedSnapshot(null);
      setState({ status: 'error', message: result.message, lastSelectedKey: null });
      return null;
    } catch (error) {
      logger.warn('[autoNextScenario] Failed to select scenario', error);
      setPreparedSnapshot(null);
      setState({ status: 'error', message: 'Something went wrong while finding the next scenario.', lastSelectedKey: null });
      return null;
    }
  }, [evaluateAutoNext, issueEntries, registerAutoSelection]);

  const prepareAutoNext = useCallback(async (
    { apps, currentAppId, maxCandidates }: Omit<AutoNextParams, 'onSelect'>,
  ) => {
    if (!apps || apps.length === 0) {
      setPreparedSnapshot(null);
      return;
    }

    const currentKey = normalizeIdentifier(currentAppId);
    const signature = createParamsSignature(apps, currentKey ?? null, maxCandidates);
    const existing = getPreparedSnapshot();
    const now = Date.now();

    if (existing && existing.signature === signature && (now - existing.preparedAt) < PREPARED_REFRESH_GUARD_MS) {
      return;
    }

    try {
      const result = await evaluateAutoNext({ apps, currentAppId, maxCandidates, now });

      if (result.kind === 'candidate') {
        const snapshot: PreparedSnapshot = {
          type: 'candidate',
          signature,
          preparedAt: Date.now(),
          candidate: result.candidate,
          issueKey: result.issueKey,
        };
        setPreparedSnapshot(snapshot);
      } else if (result.kind === 'fallback') {
        const snapshot: PreparedSnapshot = {
          type: 'fallback',
          signature,
          preparedAt: Date.now(),
          message: result.message,
          lastCandidateKey: result.lastCandidateKey,
        };
        setPreparedSnapshot(snapshot);
      } else {
        setPreparedSnapshot(null);
      }
    } catch (error) {
      logger.warn('[autoNextScenario] Failed to prepare scenario selection', error);
    }
  }, [evaluateAutoNext]);

  const resetAutoNextMessage = useCallback(() => {
    setState(prev => ({ ...prev, status: 'idle', message: null }));
  }, []);

  return {
    autoSelect,
    prepareAutoNext,
    status: state.status,
    message: state.message,
    lastSelectedKey: state.lastSelectedKey,
    resetAutoNextMessage,
  };
};
