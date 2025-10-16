import { describe, it, expect } from 'vitest';
import { act } from 'react';
import { renderHook } from '../../test-utils/renderHook';
import { useIssueFilters } from '../useIssueFilters';
import { SampleData, type Issue } from '../../data/sampleData';

describe('useIssueFilters', () => {
  const sampleIssues: Issue[] = SampleData.issues.slice(0, 4);

  it('exposes initial issues and app options', () => {
    const { result } = renderHook(() => useIssueFilters({ issues: sampleIssues }));

    expect(result.current.filteredIssues).toHaveLength(sampleIssues.length);
    expect(result.current.availableApps.length).toBeGreaterThan(0);
    expect(result.current.activeFilterCount).toBe(0);
  });

  it('applies priority filter', () => {
    const { result } = renderHook(() => useIssueFilters({ issues: sampleIssues }));

    act(() => {
      result.current.handlePriorityFilterChange('High');
    });

    expect(result.current.priorityFilter).toBe('High');
    expect(result.current.filteredIssues.every((issue) => issue.priority === 'High')).toBe(true);
    expect(result.current.activeFilterCount).toBe(1);
  });

  it('matches free-text search across fields', () => {
    const { result } = renderHook(() => useIssueFilters({ issues: sampleIssues }));

    const target = sampleIssues[0];

    act(() => {
      result.current.handleSearchFilterChange(target.title.slice(0, 4));
    });

    expect(result.current.filteredIssues.map((issue) => issue.id)).toContain(target.id);
    expect(result.current.activeFilterCount).toBeGreaterThanOrEqual(1);
  });

  it('syncs app filter from URL parameters', () => {
    const { result } = renderHook(() => useIssueFilters({ issues: sampleIssues }));

    const params = new URLSearchParams();
    params.set('app_id', sampleIssues[1].app);

    act(() => {
      result.current.syncAppFilterFromParams(params);
    });

    expect(result.current.appFilter).toBe(sampleIssues[1].app);
    expect(result.current.activeFilterCount).toBeGreaterThanOrEqual(1);
  });

  it('retains selected app filter when no issues currently match', () => {
    const { result, rerender } = renderHook(() => useIssueFilters({ issues: sampleIssues }));

    const targetApp = sampleIssues[0].app;

    act(() => {
      result.current.handleAppFilterChange(targetApp);
    });

    expect(result.current.appFilter).toBe(targetApp);
    expect(result.current.filteredIssues.every((issue) => issue.app === targetApp)).toBe(true);

    const issuesWithoutTarget = sampleIssues.filter((issue) => issue.app !== targetApp);

    rerender(() => useIssueFilters({ issues: issuesWithoutTarget }));

    expect(result.current.appFilter).toBe(targetApp);
    expect(result.current.filteredIssues).toHaveLength(0);
  });
});
