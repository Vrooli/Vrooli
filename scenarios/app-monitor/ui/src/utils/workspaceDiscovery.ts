import type { App } from '@/types';
import { buildPreviewUrl, isAppMonitorScenarioId, parseTimestampValue, resolveAppIdentifier } from '@/utils/appPreview';
import { isAppMonitorProxyPreviewTarget, resolvePreviewUrlCandidate } from '@/utils/previewUrl';

export interface PreviewSuggestionItem {
  id: string;
  label: string;
  value: string;
  detail?: string;
  kind: 'recent-url' | 'scenario' | 'running-scenario';
}

export interface PreviewSuggestionSection {
  id: 'recent-urls' | 'running-scenarios' | 'scenario-matches';
  label: string;
  items: PreviewSuggestionItem[];
}

const tokenize = (value: string): string[] => (
  value
    .trim()
    .toLowerCase()
    .split(/\s+/)
    .filter(Boolean)
);

const normalize = (value: string | null | undefined): string => value?.trim().toLowerCase() ?? '';

const isSubsequence = (needle: string, haystack: string): boolean => {
  if (!needle || needle.length > haystack.length) {
    return false;
  }
  let cursor = 0;
  for (let index = 0; index < haystack.length; index += 1) {
    if (haystack[index] === needle[cursor]) {
      cursor += 1;
      if (cursor >= needle.length) {
        return true;
      }
    }
  }
  return false;
};

const scoreTokenAgainstText = (token: string, text: string): number => {
  if (!token || !text) {
    return -1;
  }
  if (text === token) {
    return 140;
  }
  if (text.startsWith(token)) {
    return 96;
  }
  if (text.includes(` ${token}`) || text.includes(`-${token}`) || text.includes(`_${token}`)) {
    return 78;
  }
  if (text.includes(token)) {
    return 58;
  }
  if (isSubsequence(token, text)) {
    return 36;
  }
  return -1;
};

const scoreTokensAcrossHaystacks = (tokens: string[], haystacks: string[]): number => {
  if (tokens.length === 0) {
    return 0;
  }
  let score = 0;
  for (const token of tokens) {
    let bestForToken = -1;
    for (const haystack of haystacks) {
      const candidate = scoreTokenAgainstText(token, haystack);
      if (candidate > bestForToken) {
        bestForToken = candidate;
      }
    }
    if (bestForToken < 0) {
      return -1;
    }
    score += bestForToken;
  }
  return score;
};

const resolveAppLabel = (app: App): string => app.scenario_name || app.name || app.id || 'Untitled scenario';

const resolveAppHaystacks = (app: App): string[] => {
  const values: string[] = [
    app.scenario_name,
    app.name,
    app.id,
    app.description,
    app.status,
  ]
    .map((entry) => normalize(entry))
    .filter(Boolean);

  for (const tag of app.tags ?? []) {
    const normalized = normalize(tag);
    if (normalized) {
      values.push(normalized);
    }
  }

  const mappings = app.port_mappings ?? {};
  for (const [key, value] of Object.entries(mappings)) {
    const normalizedKey = normalize(key);
    if (normalizedKey) {
      values.push(normalizedKey);
    }
    const valueAsString = normalize(String(value));
    if (valueAsString) {
      values.push(valueAsString);
    }
  }

  return values;
};

const computeScenarioSignalBoost = (app: App): number => {
  let score = 0;
  if (app.status === 'running' || app.status === 'healthy') {
    score += 18;
  } else if (app.status === 'degraded') {
    score += 8;
  }
  const lastViewed = parseTimestampValue(app.last_viewed_at);
  if (lastViewed) {
    const ageMinutes = Math.max(0, (Date.now() - lastViewed) / 60_000);
    if (ageMinutes < 5) {
      score += 24;
    } else if (ageMinutes < 60) {
      score += 14;
    } else if (ageMinutes < 24 * 60) {
      score += 8;
    }
  }
  const views = Number.isFinite(app.view_count) ? Number(app.view_count) : 0;
  if (views > 0) {
    score += Math.min(16, Math.floor(Math.log10(views + 1) * 8));
  }
  return score;
};

export const rankAppsByDiscoveryQuery = (apps: App[], query: string): App[] => {
  const normalizedQuery = normalize(query);
  if (!normalizedQuery) {
    return apps;
  }
  const tokens = tokenize(normalizedQuery);
  return apps
    .map((app) => {
      const haystacks = resolveAppHaystacks(app);
      const fuzzyScore = scoreTokensAcrossHaystacks(tokens, haystacks);
      if (fuzzyScore < 0) {
        return null;
      }
      const signalBoost = computeScenarioSignalBoost(app);
      const updatedAt = parseTimestampValue(app.updated_at) ?? 0;
      return {
        app,
        score: fuzzyScore + signalBoost,
        updatedAt,
      };
    })
    .filter((entry): entry is { app: App; score: number; updatedAt: number } => Boolean(entry))
    .sort((a, b) => {
      if (a.score !== b.score) {
        return b.score - a.score;
      }
      if (a.updatedAt !== b.updatedAt) {
        return b.updatedAt - a.updatedAt;
      }
      return resolveAppLabel(a.app).localeCompare(resolveAppLabel(b.app), undefined, { sensitivity: 'base' });
    })
    .map((entry) => entry.app);
};

const dedupeNormalized = (values: string[], limit: number): string[] => {
  const seen = new Set<string>();
  const next: string[] = [];
  for (const value of values) {
    const normalized = normalize(value);
    if (!normalized || seen.has(normalized)) {
      continue;
    }
    seen.add(normalized);
    next.push(value);
    if (next.length >= limit) {
      break;
    }
  }
  return next;
};

const resolveSuggestionCandidate = (
  value: string,
  referenceUrl: string | null,
): string => {
  const trimmed = value.trim();
  if (!trimmed) {
    return trimmed;
  }
  if (trimmed.startsWith('/')) {
    return resolvePreviewUrlCandidate(trimmed, null) ?? trimmed;
  }
  return resolvePreviewUrlCandidate(trimmed, referenceUrl) ?? trimmed;
};

const normalizeHistoryUrls = (history: string[], referenceUrl: string | null, query: string): string[] => {
  const normalizedQuery = normalize(query);
  const normalized = history
    .map((entry) => resolveSuggestionCandidate(entry, referenceUrl))
    .filter((entry) => entry.length > 0)
    .filter((entry) => !isAppMonitorProxyPreviewTarget(entry))
    .reverse();

  const filtered = normalizedQuery
    ? normalized.filter((entry) => normalize(entry).includes(normalizedQuery))
    : normalized;

  return dedupeNormalized(filtered, 8);
};

const normalizePreviewableApps = (apps: App[]): App[] => (
  apps.filter((app) => {
    const identifier = resolveAppIdentifier(app);
    return !isAppMonitorScenarioId(identifier) && !isAppMonitorScenarioId(app.id);
  })
);

export interface BuildPreviewSuggestionSectionsOptions {
  apps: App[];
  history: string[];
  query: string;
  referenceUrl: string | null;
}

export const buildPreviewSuggestionSections = ({
  apps,
  history,
  query,
  referenceUrl,
}: BuildPreviewSuggestionSectionsOptions): PreviewSuggestionSection[] => {
  const previewableApps = normalizePreviewableApps(apps);
  const recentUrls = normalizeHistoryUrls(history, referenceUrl, query).map((entry) => ({
    id: `recent:${entry}`,
    label: entry,
    value: entry,
    kind: 'recent-url' as const,
  }));

  // Rank all apps once, then partition into running vs non-running in a single pass
  const ranked = query ? rankAppsByDiscoveryQuery(previewableApps, query) : previewableApps;
  const runningSuggestions: PreviewSuggestionItem[] = [];
  const scenarioSuggestions: PreviewSuggestionItem[] = [];
  const runningValues = new Set<string>();

  for (const app of ranked) {
    const previewUrl = buildPreviewUrl(app);
    if (!previewUrl) {
      continue;
    }
    const previewTarget = resolveSuggestionCandidate(previewUrl, referenceUrl);
    const label = resolveAppLabel(app);
    const isRunning = app.status === 'running' || app.status === 'healthy';

    if (isRunning && runningSuggestions.length < 6) {
      runningSuggestions.push({
        id: `running:${resolveAppIdentifier(app) ?? app.id}`,
        label,
        detail: 'Running',
        value: previewTarget,
        kind: 'running-scenario',
      });
      runningValues.add(previewTarget);
    }

    if (scenarioSuggestions.length < 10) {
      scenarioSuggestions.push({
        id: `scenario:${resolveAppIdentifier(app) ?? app.id}`,
        label,
        detail: app.status ? app.status.toUpperCase() : undefined,
        value: previewTarget,
        kind: isRunning ? 'running-scenario' : 'scenario',
      });
    }

    if (runningSuggestions.length >= 6 && scenarioSuggestions.length >= 10) {
      break;
    }
  }

  // Filter out running items from the scenario section to avoid duplicates
  const dedupedScenarioSuggestions = scenarioSuggestions.filter(
    (item) => !runningValues.has(item.value),
  );

  const sections: PreviewSuggestionSection[] = [];
  if (recentUrls.length > 0) {
    sections.push({
      id: 'recent-urls',
      label: 'Recent URLs',
      items: recentUrls,
    });
  }
  if (runningSuggestions.length > 0) {
    sections.push({
      id: 'running-scenarios',
      label: 'Running scenarios',
      items: runningSuggestions,
    });
  }
  if (dedupedScenarioSuggestions.length > 0) {
    sections.push({
      id: 'scenario-matches',
      label: query.trim().length > 0 ? 'Scenario matches' : 'Scenarios',
      items: dedupedScenarioSuggestions,
    });
  }
  return sections;
};
