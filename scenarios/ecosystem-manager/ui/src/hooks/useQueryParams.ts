import { useEffect, useCallback } from 'react';
import type { TaskFilters } from '../types/api';
import { isOperationType, isPriority, isTaskSort, isTaskType } from '../types/guards';

/**
 * Syncs filter state with URL query parameters
 */
export function useQueryParams(
  filters: TaskFilters,
  setFilters: (filters: Partial<TaskFilters>) => void
) {
  // Read filters from URL on mount
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const urlFilters: Partial<TaskFilters> = {};

    const search = params.get('search');
    const type = params.get('type');
    const operation = params.get('operation');
    const priority = params.get('priority');
    const sort = params.get('sort');

    if (search) urlFilters.search = search;
    if (isTaskType(type)) urlFilters.type = type;
    if (isOperationType(operation)) urlFilters.operation = operation;
    if (isPriority(priority)) urlFilters.priority = priority;
    if (isTaskSort(sort)) urlFilters.sort = sort;

    if (Object.keys(urlFilters).length > 0) {
      setFilters(urlFilters);
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Update URL when filters change
  useEffect(() => {
    const params = new URLSearchParams();

    if (filters.search) params.set('search', filters.search);
    if (filters.type) params.set('type', filters.type);
    if (filters.operation) params.set('operation', filters.operation);
    if (filters.priority) params.set('priority', filters.priority);
    if (filters.sort) params.set('sort', filters.sort);

    const newUrl = params.toString()
      ? `${window.location.pathname}?${params.toString()}`
      : window.location.pathname;

    window.history.replaceState({}, '', newUrl);
  }, [filters]);

  const clearFilters = useCallback(() => {
    setFilters({
      search: '',
      type: '',
      operation: '',
      priority: '',
      sort: 'updated_desc',
    });
  }, [setFilters]);

  return { clearFilters };
}
