import { describe, expect, it } from 'vitest';
import type { App } from '@/types';
import { buildPreviewSuggestionSections, rankAppsByDiscoveryQuery } from './workspaceDiscovery';

const createApp = (input: Partial<App> & { id: string }): App => ({
  id: input.id,
  name: input.name ?? input.id,
  scenario_name: input.scenario_name ?? input.id,
  path: input.path ?? `/tmp/${input.id}`,
  created_at: input.created_at ?? '2026-02-08T00:00:00Z',
  updated_at: input.updated_at ?? '2026-02-08T00:00:00Z',
  status: input.status ?? 'running',
  port_mappings: input.port_mappings ?? { UI_PORT: 4310 },
  environment: input.environment ?? {},
  config: input.config ?? {},
  description: input.description,
  tags: input.tags,
  view_count: input.view_count,
  last_viewed_at: input.last_viewed_at,
});

describe('rankAppsByDiscoveryQuery', () => {
  it('prioritizes strong scenario-name matches over weaker partial matches', () => {
    const apps = [
      createApp({ id: 'alpha-dashboard', scenario_name: 'alpha-dashboard' }),
      createApp({ id: 'ops-control', scenario_name: 'operations-control' }),
      createApp({ id: 'metrics', scenario_name: 'metrics-station' }),
    ];

    const ranked = rankAppsByDiscoveryQuery(apps, 'alpha');
    expect(ranked[0]?.id).toBe('alpha-dashboard');
    expect(ranked.some((app) => app.id === 'ops-control')).toBe(false);
  });
});

describe('buildPreviewSuggestionSections', () => {
  it('builds grouped sections for recents, running scenarios, and scenario matches', () => {
    const apps = [
      createApp({ id: 'workspace-manager', status: 'running' }),
      createApp({ id: 'billing-ops', status: 'stopped' }),
      createApp({ id: 'app-monitor', status: 'running' }),
    ];

    const sections = buildPreviewSuggestionSections({
      apps,
      history: ['http://localhost:4310/settings', '/apps/workspace-manager/proxy/'],
      query: '',
      referenceUrl: 'http://localhost:3000/apps/app-monitor/proxy/',
    });

    const sectionIds = sections.map((section) => section.id);
    expect(sectionIds).toContain('recent-urls');
    expect(sectionIds).toContain('running-scenarios');
    expect(sectionIds).toContain('scenario-matches');
    const allValues = sections.flatMap((section) => section.items.map((item) => item.value));
    expect(allValues.some((value) => value.includes('/apps/app-monitor/proxy/'))).toBe(false);
  });

  it('filters grouped suggestions by query text', () => {
    const apps = [
      createApp({ id: 'workspace-manager', scenario_name: 'workspace-manager' }),
      createApp({ id: 'billing-ops', scenario_name: 'billing-ops' }),
    ];
    const sections = buildPreviewSuggestionSections({
      apps,
      history: ['http://localhost:4310/workspace', 'http://localhost:4310/billing'],
      query: 'work',
      referenceUrl: 'http://localhost:3000/apps/workspace-manager/proxy/',
    });

    const values = sections.flatMap((section) => section.items.map((item) => item.value));
    expect(values.some((value) => value.includes('billing'))).toBe(false);
    expect(values.some((value) => value.includes('workspace'))).toBe(true);
  });

  it('keeps scenario proxy suggestions on app-monitor origin even when current preview is external', () => {
    const apps = [
      createApp({ id: 'git-control-tower', scenario_name: 'git-control-tower', status: 'running' }),
    ];
    const sections = buildPreviewSuggestionSections({
      apps,
      history: [],
      query: 'git',
      referenceUrl: 'https://reddit.com/r/typescript',
    });

    const values = sections.flatMap((section) => section.items.map((item) => item.value));
    expect(values.some((value) => value.startsWith('https://reddit.com/apps/'))).toBe(false);
    expect(values.some((value) => value.includes('/apps/git-control-tower/proxy/'))).toBe(true);
  });
});
