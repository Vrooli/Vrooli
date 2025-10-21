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

interface ScoredCandidate {
  app: App;
  key: string;
  navigationId: string;
  score: number;
}

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

  const autoSelect = useCallback(async ({ apps, currentAppId, onSelect, maxCandidates }: AutoNextParams): Promise<App | null> => {
    if (!apps || apps.length === 0) {
      setState({ status: 'error', message: 'No scenarios available to open.', lastSelectedKey: null });
      return null;
    }

    setState(prev => ({ ...prev, status: 'running', message: null }));

    try {
      const now = Date.now();
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
        setState({ status: 'error', message: 'No alternative scenarios to open.', lastSelectedKey: null });
        return null;
      }

      const maxPool = maxCandidates && Number.isFinite(maxCandidates)
        ? Math.max(1, Math.floor(maxCandidates))
        : MAX_CANDIDATES;

      const ordered = scored
        .sort((a, b) => b.score - a.score)
        .slice(0, maxPool);

      let fallback: { candidate: ScoredCandidate; reason: 'issues' | 'error' } | null = null;

      for (const candidate of ordered) {
        const issueKey = getIssueStoreKey(candidate.app);
        const existingEntry = issueKey ? issueEntries[issueKey] : undefined;
        const needsRefresh = !existingEntry
          || existingEntry.status === 'idle'
          || existingEntry.status === 'loading'
          || existingEntry.stale
          || (existingEntry.fetchedAt && (Date.now() - existingEntry.fetchedAt > 2 * 60 * 1000));

        let entry = existingEntry;
        if (needsRefresh) {
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

        registerAutoSelection(candidate.navigationId);
        setState({ status: 'idle', message: null, lastSelectedKey: candidate.key });
        onSelect(candidate.app, candidate.navigationId);
        return candidate.app;
      }

      if (fallback) {
        const message = fallback.reason === 'error'
          ? 'Issue status could not be verified right now. Try again shortly or pick a scenario manually.'
          : 'All suitable scenarios currently have unresolved issues. Clear issues or pick one manually.';
        setState({
          status: 'error',
          message,
          lastSelectedKey: fallback.candidate.key,
        });
        return null;
      }

      setState({ status: 'error', message: 'Unable to select a scenario automatically.', lastSelectedKey: null });
      return null;
    } catch (error) {
      console.warn('[autoNextScenario] Failed to select scenario', error);
      setState({ status: 'error', message: 'Something went wrong while finding the next scenario.', lastSelectedKey: null });
      return null;
    }
  }, [engagement, fetchIssues, issueEntries, registerAutoSelection]);

  const resetAutoNextMessage = useCallback(() => {
    setState(prev => ({ ...prev, status: 'idle', message: null }));
  }, []);

  return {
    autoSelect,
    status: state.status,
    message: state.message,
    lastSelectedKey: state.lastSelectedKey,
    resetAutoNextMessage,
  };
};
