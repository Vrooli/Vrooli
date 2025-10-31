import { useCallback, useMemo, useState } from 'react';
import type { Issue } from '../types/issue';
import type { PriorityFilterValue } from '../types/issueCreation';

interface UseIssueFiltersOptions {
  issues: Issue[];
}

interface UseIssueFiltersResult {
  filteredIssues: Issue[];
  availableApps: string[];
  priorityFilter: PriorityFilterValue;
  handlePriorityFilterChange: (value: PriorityFilterValue) => void;
  appFilter: string;
  handleAppFilterChange: (value: string) => void;
  searchFilter: string;
  handleSearchFilterChange: (value: string) => void;
  activeFilterCount: number;
  syncAppFilterFromParams: (params: URLSearchParams) => void;
}

export function useIssueFilters({ issues }: UseIssueFiltersOptions): UseIssueFiltersResult {
  const [priorityFilter, setPriorityFilter] = useState<PriorityFilterValue>('all');
  const [appFilter, setAppFilter] = useState<string>('all');
  const [searchFilter, setSearchFilter] = useState<string>('');

  const handlePriorityFilterChange = useCallback((value: PriorityFilterValue) => {
    setPriorityFilter(value);
  }, []);

  const handleAppFilterChange = useCallback((value: string) => {
    setAppFilter(value.trim() || 'all');
  }, []);

  const handleSearchFilterChange = useCallback((value: string) => {
    setSearchFilter(value);
  }, []);

  const availableApps = useMemo(() => {
    const apps = new Set<string>();
    issues.forEach((issue) => {
      issue.targets.forEach((target) => {
        apps.add(target.id);
      });
    });
    if (appFilter !== 'all' && appFilter.trim()) {
      apps.add(appFilter.trim());
    }
    return Array.from(apps).sort((first, second) => first.localeCompare(second));
  }, [issues, appFilter]);

  const filteredIssues = useMemo(() => {
    return issues.filter((issue) => {
      const matchesPriority = priorityFilter === 'all' || issue.priority === priorityFilter;
      const matchesApp = appFilter === 'all' || issue.targets.some((target) => target.id === appFilter);

      if (!searchFilter.trim()) {
        return matchesPriority && matchesApp;
      }

      const query = searchFilter.toLowerCase();
      const matchesSearch =
        issue.id.toLowerCase().includes(query) ||
        issue.title.toLowerCase().includes(query) ||
        (issue.description?.toLowerCase().includes(query) ?? false) ||
        issue.assignee.toLowerCase().includes(query) ||
        issue.tags.some((tag) => tag.toLowerCase().includes(query)) ||
        (issue.reporterName?.toLowerCase().includes(query) ?? false) ||
        (issue.reporterEmail?.toLowerCase().includes(query) ?? false);

      return matchesPriority && matchesApp && matchesSearch;
    });
  }, [issues, priorityFilter, appFilter, searchFilter]);

  const activeFilterCount = useMemo(() => {
    let count = 0;
    if (priorityFilter !== 'all') count += 1;
    if (appFilter !== 'all') count += 1;
    if (searchFilter.trim()) count += 1;
    return count;
  }, [priorityFilter, appFilter, searchFilter]);

  const syncAppFilterFromParams = useCallback((params: URLSearchParams) => {
    const appParam = params.get('app_id');
    if (appParam && appParam.trim()) {
      setAppFilter(appParam.trim());
    } else {
      setAppFilter('all');
    }
  }, []);

  return {
    filteredIssues,
    availableApps,
    priorityFilter,
    handlePriorityFilterChange,
    appFilter,
    handleAppFilterChange,
    searchFilter,
    handleSearchFilterChange,
    activeFilterCount,
    syncAppFilterFromParams,
  };
}
