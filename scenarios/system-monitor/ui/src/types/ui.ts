// UI-specific types for React components
import type { InvestigationScript, ScriptExecution, DiskInfo, StorageIOInfo, GPUMetrics } from './api';

export type AlertSeverity = 'low' | 'medium' | 'high' | 'critical';

export type CardType = 'cpu' | 'memory' | 'network' | 'disk' | 'gpu';

export type PanelType = 'process' | 'infrastructure';

export interface Alert {
  id: string;
  timestamp: string;
  severity: AlertSeverity;
  category: string;
  message: string;
  resolved: boolean;
  details?: string;
}

export interface TerminalLine {
  id: string;
  timestamp: string;
  message: string;
  type: 'info' | 'success' | 'warning' | 'error';
}

export interface DashboardState {
  isOnline: boolean;
  lastUpdate: string;
  expandedCards: Set<CardType>;
  expandedPanels: Set<PanelType>;
  terminalVisible: boolean;
  unreadErrorCount: number;
  alerts: Alert[];
}

export interface ModalState {
  reportModal: {
    isOpen: boolean;
    reportId?: string;
    loading: boolean;
  };
  scriptEditor: {
    isOpen: boolean;
    scriptId?: string;
    mode: 'create' | 'edit' | 'view';
    script?: InvestigationScript;
    scriptContent?: string;
  };
  scriptResults: {
    isOpen: boolean;
    executionId?: string;
    scriptId?: string;
    execution?: ScriptExecution;
  };
}

export interface ChartDataPoint {
  timestamp: string;
  value: number;
  label?: string;
}

export interface DiskCardDetails {
  diskUsage: DiskInfo;
  storageIO?: StorageIOInfo;
  lastUpdated?: string;
}

export interface GPUCardDetails {
  metrics: GPUMetrics;
  lastUpdated?: string;
}

export interface MetricHistory {
  windowSeconds: number;
  sampleIntervalSeconds: number;
  cpu: ChartDataPoint[];
  memory: ChartDataPoint[];
  network: ChartDataPoint[];
  gpu?: ChartDataPoint[];
  diskUsage?: ChartDataPoint[];
  diskRead?: ChartDataPoint[];
  diskWrite?: ChartDataPoint[];
}

export interface InvestigationAgentState {
  id: string;
  status: string;
  startTime: string;
  autoFix: boolean;
  operationMode?: string;
  model?: string;
  resource?: string;
  progress?: number;
  riskLevel?: string;
  note?: string;
  label?: string;
  anomalyId?: string;
  details?: Record<string, unknown>;
  lastUpdated?: string;
  completedAt?: string;
  error?: string;
}

