import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import type { App } from '@/types';
import type { BrowserTabHistoryRecord } from '@/state/browserTabsStore';
import { AppsSection, WebSection } from './TabSwitcherSections';

const baseApp: App = {
  id: 'app-1',
  name: 'App One',
  scenario_name: 'Scenario Alpha',
  path: '/apps/app-1',
  created_at: '2024-01-01',
  updated_at: '2024-01-02',
  status: 'running',
  port_mappings: {},
  environment: {},
  config: {},
};

const historyEntry: BrowserTabHistoryRecord = {
  id: 'tab-1',
  title: 'Docs',
  url: 'https://example.com/docs',
  createdAt: 1,
  lastActiveAt: 2,
  closedAt: 3,
  screenshotData: null,
  screenshotWidth: null,
  screenshotHeight: null,
  screenshotNote: null,
  faviconUrl: null,
};

describe('TabSwitcherSections', () => {
  it('renders recent apps and empty state messaging', () => {
    render(
      <AppsSection
        showHistory={true}
        recentApps={[baseApp]}
        apps={[]}
        normalizedSearch=""
        sortOption="status"
        sortOptions={[{ value: 'status', label: 'Active first' }]}
        onSortChange={() => undefined}
        onSelect={() => undefined}
        isLoading={false}
        onRetry={() => undefined}
        errorMessage={null}
        isRefreshing={false}
      />
    );

    expect(screen.getByText('Recently opened')).toBeInTheDocument();
    expect(screen.getByText('Scenario Alpha')).toBeInTheDocument();
    expect(screen.getByText('No scenarios match your search.')).toBeInTheDocument();
  });

  it('renders web section empty states when no data', () => {
    render(
      <WebSection
        tabs={[]}
        historyEntries={[]}
        showHistory={false}
        onOpen={() => undefined}
        onClose={() => undefined}
        onReopen={() => undefined}
      />
    );

    expect(screen.getByText('No active web tabs yet.')).toBeInTheDocument();
    expect(screen.getByText('No web browsing history yet.')).toBeInTheDocument();
  });

  it('renders web history entries when available', () => {
    render(
      <WebSection
        tabs={[]}
        historyEntries={[historyEntry]}
        showHistory={true}
        onOpen={() => undefined}
        onClose={() => undefined}
        onReopen={() => undefined}
      />
    );

    expect(screen.getByText('Docs')).toBeInTheDocument();
    expect(screen.getByText('https://example.com/docs')).toBeInTheDocument();
  });
});
