/**
 * App State Context
 * Manages global UI state (filters, column visibility, modals)
 */

import React, { useState, useCallback, useEffect } from 'react';
import type { TaskFilters, TaskStatus, Settings } from '../types/api';
import { useSettings } from '../hooks/useSettings';
import { AppStateContext, type AppState, type ColumnVisibility } from './AppStateContext.value';

// Default column visibility (archived column is hidden by default)
const defaultColumnVisibility: ColumnVisibility = {
  pending: true,
  'in-progress': true,
  completed: true,
  'completed-finalized': true,
  failed: true,
  'failed-blocked': true,
  archived: false, // Hidden by default
};

const defaultFilters: TaskFilters = {
  search: '',
  status: '',
  type: '',
  operation: '',
  priority: '',
  sort: 'updated_desc',
};

export function AppStateProvider({ children }: { children: React.ReactNode }) {
  const [filters, setFiltersState] = useState<TaskFilters>(defaultFilters);
  const [columnVisibility, setColumnVisibilityState] = useState<ColumnVisibility>(defaultColumnVisibility);
  const [isFilterPanelOpen, setFilterPanelOpen] = useState(false);
  const [activeModal, setActiveModalState] = useState<string | null>(null);
  const [cachedSettings, setCachedSettings] = useState<Settings | null>(null);
  const { data: settingsData } = useSettings();

  const setFilters = useCallback((newFilters: TaskFilters) => {
    setFiltersState(newFilters);
  }, []);

  const updateFilter = useCallback(<K extends keyof TaskFilters>(key: K, value: TaskFilters[K]) => {
    setFiltersState(prev => ({ ...prev, [key]: value }));
  }, []);

  const clearFilters = useCallback(() => {
    setFiltersState(defaultFilters);
  }, []);

  const setColumnVisibility = useCallback((status: TaskStatus, visible: boolean) => {
    setColumnVisibilityState(prev => ({ ...prev, [status]: visible }));
  }, []);

  const toggleColumnVisibility = useCallback((status: TaskStatus) => {
    setColumnVisibilityState(prev => ({ ...prev, [status]: !prev[status] }));
  }, []);

  const toggleFilterPanel = useCallback(() => {
    setFilterPanelOpen(prev => !prev);
  }, []);

  const setActiveModal = useCallback((modalId: string | null) => {
    setActiveModalState(modalId);
  }, []);

  const openModal = useCallback((modalId: string) => {
    setActiveModalState(modalId);
  }, []);

  const closeModal = useCallback(() => {
    setActiveModalState(null);
  }, []);

  // Keep cached settings in sync with the API response
  useEffect(() => {
    if (settingsData?.settings) {
      setCachedSettings(settingsData.settings);
    }
  }, [settingsData]);

  const value: AppState = {
    filters,
    setFilters,
    updateFilter,
    clearFilters,
    columnVisibility,
    setColumnVisibility,
    toggleColumnVisibility,
    isFilterPanelOpen,
    setFilterPanelOpen,
    setIsFilterPanelOpen: setFilterPanelOpen, // Alias
    toggleFilterPanel,
    activeModal,
    setActiveModal,
    openModal,
    closeModal,
    cachedSettings,
    setCachedSettings,
  };

  return <AppStateContext.Provider value={value}>{children}</AppStateContext.Provider>;
}
