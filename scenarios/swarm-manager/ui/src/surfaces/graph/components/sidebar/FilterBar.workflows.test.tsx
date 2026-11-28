import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { FilterBar } from './FilterBar';
import { useSidebarState } from './useSidebarState';
import type { SidebarTab } from './types';

function Harness({ tab = 'backlog' }: { tab?: SidebarTab }) {
  const [state, dispatch] = useSidebarState(tab);
  return <>
    {(['backlog', 'captures', 'goals', 'executions', 'scenarios', 'sessions'] as const).map((name) =>
      <button key={name} onClick={() => dispatch({ type: 'SET_TAB', tab: name })}>Open {name}</button>)}
    <FilterBar activeTab={state.activeTab} backlogFilters={state.filters.backlog}
      captureFilters={state.filters.captures} executionFilters={state.filters.executions}
      scenarioFilters={state.filters.scenarios} sessionFilters={state.filters.sessions}
      sort={state.sorts[state.activeTab]} dispatch={dispatch} />
    <output data-testid="sidebar-state">{JSON.stringify(state)}</output>
  </>;
}
function state() { return JSON.parse(screen.getByTestId('sidebar-state').textContent!); }
function click(name: string) { fireEvent.click(screen.getByRole('button', { name })); }
beforeEach(() => localStorage.clear());
afterEach(() => { cleanup(); vi.restoreAllMocks(); localStorage.clear(); });

describe('sidebar filter controls with the actual reducer and persistence', () => {
  it('persists disclosure and backlog filters, removes selected values and clears the active tab', async () => {
    const view = render(<Harness />);
    expect(screen.queryByTestId('filter-bar-content')).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId('filter-bar-toggle'));
    click('ready'); click('fix');
    fireEvent.click(screen.getByLabelText('Show Archived'));
    expect(state().filters.backlog).toMatchObject({ statuses: ['ready'], kinds: ['fix'], showArchived: true });
    click('ready');
    expect(state().filters.backlog.statuses).toEqual([]);
    view.unmount();
    render(<Harness />);
    expect(screen.getByTestId('filter-bar-content')).toBeInTheDocument();
    expect(state().filters.backlog.kinds).toEqual(['fix']);
    expect(screen.getByLabelText('Show Archived')).toBeChecked();
    click('Clear');
    expect(state().filters.backlog).toMatchObject({ statuses: [], kinds: [], showArchived: false });
    await waitFor(() => expect(JSON.parse(localStorage.getItem('swarm-manager.sidebar.state.v1')!).filters.backlog.kinds).toEqual([]));
  });

  it.each(['backlog', 'captures', 'goals', 'executions', 'scenarios', 'sessions'] as const)
    ('changes and reverses %s sorting, then restores that tab defaults', (tab) => {
      localStorage.setItem('swarm-manager.section.sidebar-filters', '1');
      render(<Harness tab={tab} />);
      const initial = state().sorts[tab];
      const label = initial.field === 'alphabetical' ? 'Recent' : 'A-Z';
      const field = label === 'Recent' ? 'recency' : 'alphabetical';
      click(label);
      expect(state().sorts[tab].field).toBe(field);
      const direction = state().sorts[tab].direction;
      click(label);
      expect(state().sorts[tab].direction).toBe(direction === 'asc' ? 'desc' : 'asc');
      click('Clear');
      expect(state().sorts[tab]).toEqual(initial);
      expect(screen.queryByRole('button', { name: 'Clear' })).not.toBeInTheDocument();
    });

  it('keeps capture and execution selections independent when changing tabs', () => {
    localStorage.setItem('swarm-manager.section.sidebar-filters', '1');
    render(<Harness tab="captures" />);
    click('failed');
    expect(state().filters.captures.statuses).toEqual(['failed']);
    click('Open executions'); click('running'); click('manual');
    expect(state().filters.executions).toEqual({ statuses: ['running'], modes: ['manual'] });
    click('manual');
    expect(state().filters.executions.modes).toEqual([]);
    click('Clear');
    expect(state().filters.executions.statuses).toEqual([]);
    click('Open captures');
    expect(state().filters.captures.statuses).toEqual(['failed']);
    click('Clear');
    expect(state().filters.captures.statuses).toEqual([]);
  });

  it('filters session kind, status, active state and artifact affordances, with reversible selection', () => {
    localStorage.setItem('swarm-manager.section.sidebar-filters', '1');
    render(<Harness tab="sessions" />);
    click('proposal ready'); click('Plan work');
    fireEvent.click(screen.getByLabelText('Active only'));
    fireEvent.click(screen.getByLabelText('Has proposals'));
    fireEvent.click(screen.getByLabelText('Has applied artifacts'));
    expect(state().filters.sessions).toEqual({ statuses: ['proposal_ready'], kinds: ['meta_orchestration'], activeOnly: true, hasProposals: true, hasAppliedArtifacts: true });
    click('Plan work'); click('proposal ready');
    expect(state().filters.sessions.kinds).toEqual([]);
    expect(state().filters.sessions.statuses).toEqual([]);
    click('Clear');
    expect(state().filters.sessions).toEqual({ statuses: [], kinds: [], activeOnly: false, hasProposals: false, hasAppliedArtifacts: false });
  });

  it('combines scenario lifecycle, evidence and remediation without changing other tab filters', () => {
    localStorage.setItem('swarm-manager.section.sidebar-filters', '1');
    render(<Harness tab="scenarios" />);
    click('running'); click('stale'); click('suggested');
    expect(state().filters.scenarios).toEqual({ lifecycle: ['running'], evidenceStates: ['stale'], remediationStates: ['suggested'] });
    click('stale');
    expect(state().filters.scenarios.evidenceStates).toEqual([]);
    click('Open backlog'); click('failed');
    click('Open scenarios'); click('Clear');
    expect(state().filters.scenarios).toEqual({ lifecycle: [], evidenceStates: [], remediationStates: [] });
    expect(state().filters.backlog.statuses).toEqual(['failed']);
  });

  it('recovers safely from invalid persistence and retains in-session filtering when storage is unavailable', () => {
    localStorage.setItem('swarm-manager.sidebar.state.v1', '{broken');
    render(<Harness tab="sessions" />);
    expect(state().activeTab).toBe('sessions');
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new DOMException('blocked', 'SecurityError'); });
    fireEvent.click(screen.getByTestId('filter-bar-toggle'));
    click('Workflow authoring');
    expect(state().filters.sessions.kinds).toEqual(['workflow_authoring']);
  });
});
