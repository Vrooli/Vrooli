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

export interface ExpandableCardState {
  isExpanded: boolean;
  loading: boolean;
  error?: string;
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

export interface ScriptEditorData {
  id: string;
  name: string;
  description: string;
  category: string;
  code: string;
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

export interface MetricThresholds {
  cpu: {
    warning: number;
    critical: number;
  };
  memory: {
    warning: number;
    critical: number;
  };
  tcp: {
    warning: number;
    critical: number;
  };
  gpu?: {
    warning: number;
    critical: number;
  };
  fileDescriptors: {
    warning: number;
    critical: number;
  };
}

export interface WebSocketMessage {
  type: 'metrics' | 'alert' | 'investigation' | 'system_status';
  data: any;
  timestamp: string;
}

// Theme-related types for Matrix/Cyberpunk styling
export interface ThemeColors {
  primary: string;
  secondary: string;
  accent: string;
  background: string;
  surface: string;
  text: string;
  success: string;
  warning: string;
  error: string;
  info: string;
}

export type ComponentSize = 'small' | 'medium' | 'large';

export type ComponentVariant = 'primary' | 'secondary' | 'success' | 'warning' | 'error';

// Performance monitoring
export interface PerformanceMetrics {
  renderTime: number;
  apiLatency: number;
  memoryUsage: number;
  componentCount: number;
}
