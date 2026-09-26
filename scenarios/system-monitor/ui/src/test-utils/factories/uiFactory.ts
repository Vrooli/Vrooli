import type { Alert, DashboardState, MetricHistory } from '../../types/ui'

export const createAlert = (overrides: Partial<Alert> = {}): Alert => ({
  id: 'alert-1',
  timestamp: '2026-01-01T00:00:00Z',
  severity: 'medium',
  category: 'system',
  message: 'alert message',
  resolved: false,
  ...overrides,
})

export const createDashboardState = (overrides: Partial<DashboardState> = {}): DashboardState => ({
  lastUpdate: '2026-01-01T00:00:00Z',
  expandedCards: new Set(),
  expandedPanels: new Set(),
  terminalVisible: false,
  unreadErrorCount: 0,
  alerts: [],
  ...overrides,
})

export const createMetricHistory = (overrides: Partial<MetricHistory> = {}): MetricHistory => ({
  windowSeconds: 60,
  sampleIntervalSeconds: 5,
  cpu: [],
  memory: [],
  swap: [],
  network: [],
  ...overrides,
})
