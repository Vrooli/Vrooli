import { createContext } from 'react';
import type React from 'react';
import type { Settings, TaskFilters, TaskStatus } from '../types/api';

export interface ColumnVisibility {
  pending: boolean;
  'in-progress': boolean;
  completed: boolean;
  'completed-finalized': boolean;
  failed: boolean;
  'failed-blocked': boolean;
  archived: boolean;
}

export interface AppState {
  filters: TaskFilters;
  setFilters: (filters: TaskFilters) => void;
  updateFilter: <K extends keyof TaskFilters>(key: K, value: TaskFilters[K]) => void;
  clearFilters: () => void;
  columnVisibility: ColumnVisibility;
  setColumnVisibility: (status: TaskStatus, visible: boolean) => void;
  toggleColumnVisibility: (status: TaskStatus) => void;
  isFilterPanelOpen: boolean;
  setFilterPanelOpen: (open: boolean) => void;
  setIsFilterPanelOpen: (open: boolean) => void;
  toggleFilterPanel: () => void;
  activeModal: string | null;
  setActiveModal: (modalId: string | null) => void;
  openModal: (modalId: string) => void;
  closeModal: () => void;
  cachedSettings: Settings | null;
  setCachedSettings: React.Dispatch<React.SetStateAction<Settings | null>>;
}

export const AppStateContext = createContext<AppState | undefined>(undefined);
