/**
 * useExecutionFilters Hook
 *
 * Manages filter state for the execution picker in the New Export flow.
 * Provides filters for workflow, date range, status, search text, and exportability.
 */

import { useState, useCallback, useMemo } from 'react';
import type { ExecutionWithExportability } from '@/domains/executions/store';

export type StatusFilter = 'all' | 'completed' | 'failed';

export interface ExecutionFilters {
  workflowId: string | null;
  workflowName: string;
  dateFrom: Date | null;
  dateTo: Date | null;
  status: StatusFilter;
  searchText: string;
  showNonExportable: boolean;
}

const DEFAULT_FILTERS: ExecutionFilters = {
  workflowId: null,
  workflowName: '',
  dateFrom: null,
  dateTo: null,
  status: 'all',
  searchText: '',
  showNonExportable: false,
};

export interface UseExecutionFiltersReturn {
  filters: ExecutionFilters;
  setWorkflowFilter: (id: string | null, name: string) => void;
  setDateRange: (from: Date | null, to: Date | null) => void;
  setStatusFilter: (status: StatusFilter) => void;
  setSearchText: (text: string) => void;
  toggleShowNonExportable: () => void;
  resetFilters: () => void;
  filterExecutions: (executions: ExecutionWithExportability[]) => ExecutionWithExportability[];
  hasActiveFilters: boolean;
}

export function useExecutionFilters(): UseExecutionFiltersReturn {
  const [filters, setFilters] = useState<ExecutionFilters>(DEFAULT_FILTERS);

  const setWorkflowFilter = useCallback((id: string | null, name: string) => {
    setFilters((prev) => ({ ...prev, workflowId: id, workflowName: name }));
  }, []);

  const setDateRange = useCallback((from: Date | null, to: Date | null) => {
    setFilters((prev) => ({ ...prev, dateFrom: from, dateTo: to }));
  }, []);

  const setStatusFilter = useCallback((status: StatusFilter) => {
    setFilters((prev) => ({ ...prev, status }));
  }, []);

  const setSearchText = useCallback((text: string) => {
    setFilters((prev) => ({ ...prev, searchText: text }));
  }, []);

  const toggleShowNonExportable = useCallback(() => {
    setFilters((prev) => ({ ...prev, showNonExportable: !prev.showNonExportable }));
  }, []);

  const resetFilters = useCallback(() => {
    setFilters(DEFAULT_FILTERS);
  }, []);

  const filterExecutions = useCallback(
    (executions: ExecutionWithExportability[]): ExecutionWithExportability[] => {
      return executions.filter((execution) => {
        // Filter by workflow
        if (filters.workflowId && execution.workflowId !== filters.workflowId) {
          return false;
        }

        // Filter by date range
        if (filters.dateFrom && execution.startedAt < filters.dateFrom) {
          return false;
        }
        if (filters.dateTo) {
          // Include the entire "to" day
          const endOfDay = new Date(filters.dateTo);
          endOfDay.setHours(23, 59, 59, 999);
          if (execution.startedAt > endOfDay) {
            return false;
          }
        }

        // Filter by status
        if (filters.status !== 'all') {
          if (filters.status === 'completed' && execution.status !== 'completed') {
            return false;
          }
          if (filters.status === 'failed' && execution.status !== 'failed' && execution.status !== 'cancelled') {
            return false;
          }
        }

        // Filter by search text (search in workflow name if available)
        if (filters.searchText.trim()) {
          const searchLower = filters.searchText.toLowerCase();
          const matchesId = execution.id.toLowerCase().includes(searchLower);
          const matchesWorkflow = execution.workflowId.toLowerCase().includes(searchLower);
          if (!matchesId && !matchesWorkflow) {
            return false;
          }
        }

        // Filter by exportability
        if (!filters.showNonExportable && !execution.exportability?.isExportable) {
          return false;
        }

        return true;
      });
    },
    [filters]
  );

  const hasActiveFilters = useMemo(() => {
    return (
      filters.workflowId !== null ||
      filters.dateFrom !== null ||
      filters.dateTo !== null ||
      filters.status !== 'all' ||
      filters.searchText.trim() !== '' ||
      filters.showNonExportable
    );
  }, [filters]);

  return {
    filters,
    setWorkflowFilter,
    setDateRange,
    setStatusFilter,
    setSearchText,
    toggleShowNonExportable,
    resetFilters,
    filterExecutions,
    hasActiveFilters,
  };
}
