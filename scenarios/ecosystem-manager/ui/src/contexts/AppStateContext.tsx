/**
 * App State Context
 * Manages global UI state (filters, column visibility, modals)
 */

import React, { createContext, useContext, useState, useCallback } from 'react';
import type { TaskFilters, TaskStatus, Settings } from '../types/api';

interface ColumnVisibility {
  pending: boolean;
  'in-progress': boolean;
  completed: boolean;
  'completed-finalized': boolean;
  failed: boolean;
  'failed-blocked': boolean;
  archived: boolean;
  review: boolean;
}

interface AppState {
  // Filter state
  filters: TaskFilters;
  setFilters: (filters: TaskFilters) => void;
  updateFilter: <K extends keyof TaskFilters>(key: K, value: TaskFilters[K]) => void;
  clearFilters: () => void;

  // Column visibility (archived column hidden by default)
  columnVisibility: ColumnVisibility;
  setColumnVisibility: (status: TaskStatus, visible: boolean) => void;
  toggleColumnVisibility: (status: TaskStatus) => void;

  // UI state
  isFilterPanelOpen: boolean;
  setFilterPanelOpen: (open: boolean) => void;
  setIsFilterPanelOpen: (open: boolean) => void; // Alias for consistency
  toggleFilterPanel: () => void;

  // Active modal tracking
  activeModal: string | null;
  setActiveModal: (modalId: string | null) => void;
  openModal: (modalId: string) => void;
  closeModal: () => void;

  // Cached settings (for quick access without query)
  cachedSettings: Settings | null;
}

const AppStateContext = createContext<AppState | undefined>(undefined);

// Default column visibility (archived column is hidden by default)
const defaultColumnVisibility: ColumnVisibility = {
  pending: true,
  'in-progress': true,
  completed: true,
  'completed-finalized': true,
  failed: true,
  'failed-blocked': true,
  archived: false, // Hidden by default
  review: true,
};

const defaultFilters: TaskFilters = {
  search: '',
  status: '',
  type: '',
  operation: '',
  priority: '',
};

export function AppStateProvider({ children }: { children: React.ReactNode }) {
  const [filters, setFiltersState] = useState<TaskFilters>(defaultFilters);
  const [columnVisibility, setColumnVisibilityState] = useState<ColumnVisibility>(defaultColumnVisibility);
  const [isFilterPanelOpen, setFilterPanelOpen] = useState(false);
  const [activeModal, setActiveModalState] = useState<string | null>(null);
  const [cachedSettings, setCachedSettings] = useState<Settings | null>(null);

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
  };

  return <AppStateContext.Provider value={value}>{children}</AppStateContext.Provider>;
}

export function useAppState() {
  const context = useContext(AppStateContext);
  if (!context) {
    throw new Error('useAppState must be used within an AppStateProvider');
  }
  return context;
}
