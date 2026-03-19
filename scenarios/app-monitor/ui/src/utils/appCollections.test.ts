import { describe, expect, it } from 'vitest';
import { buildRecentApps } from './appCollections';
import type { App } from '@/types';

const makeApp = (overrides: Partial<App> & { id: string }): App => ({
  name: overrides.id,
  scenario_name: overrides.id,
  path: '',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  status: 'running',
  port_mappings: {},
  environment: {},
  config: {},
  ...overrides,
});

describe('buildRecentApps', () => {
  it('returns apps with last_viewed_at', () => {
    const apps = [
      makeApp({ id: 'viewed', last_viewed_at: '2026-03-01T00:00:00Z', view_count: 1 }),
      makeApp({ id: 'not-viewed' }),
    ];
    const result = buildRecentApps(apps);
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe('viewed');
  });

  it('returns apps with view_count > 0 even without last_viewed_at', () => {
    const apps = [
      makeApp({ id: 'counted', view_count: 3 }),
      makeApp({ id: 'not-counted', view_count: 0 }),
    ];
    const result = buildRecentApps(apps);
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe('counted');
  });

  it('excludes apps matching excludeIdentifiers', () => {
    const apps = [
      makeApp({ id: 'app-a', last_viewed_at: '2026-03-01T00:00:00Z', view_count: 1 }),
      makeApp({ id: 'app-b', last_viewed_at: '2026-03-02T00:00:00Z', view_count: 1 }),
    ];
    const result = buildRecentApps(apps, { excludeIdentifiers: ['app-a'] });
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe('app-b');
  });

  it('respects limit', () => {
    const apps = [
      makeApp({ id: 'a', last_viewed_at: '2026-03-01T00:00:00Z', view_count: 1 }),
      makeApp({ id: 'b', last_viewed_at: '2026-03-02T00:00:00Z', view_count: 1 }),
      makeApp({ id: 'c', last_viewed_at: '2026-03-03T00:00:00Z', view_count: 1 }),
    ];
    const result = buildRecentApps(apps, { limit: 2 });
    expect(result).toHaveLength(2);
    expect(result[0].id).toBe('c');
    expect(result[1].id).toBe('b');
  });

  it('sorts by last_viewed_at descending, then view_count', () => {
    const apps = [
      makeApp({ id: 'old', last_viewed_at: '2026-01-01T00:00:00Z', view_count: 10 }),
      makeApp({ id: 'new', last_viewed_at: '2026-03-01T00:00:00Z', view_count: 1 }),
    ];
    const result = buildRecentApps(apps);
    expect(result[0].id).toBe('new');
    expect(result[1].id).toBe('old');
  });

  it('returns empty array when no apps have view history', () => {
    const apps = [
      makeApp({ id: 'a' }),
      makeApp({ id: 'b' }),
    ];
    const result = buildRecentApps(apps);
    expect(result).toHaveLength(0);
  });
});
