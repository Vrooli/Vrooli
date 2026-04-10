import type { App, CompleteDiagnostics } from '@/types';

export function createMockApp(overrides: Partial<App> = {}): App {
  return {
    id: 'test-app',
    name: 'Test App',
    scenario_name: 'test-scenario',
    path: '/scenarios/test-app',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    status: 'running',
    port_mappings: { UI_PORT: 3000, API_PORT: 4000 },
    environment: {},
    config: {},
    uptime: '2h 30m',
    type: 'scenario',
    description: 'A test application',
    tags: ['test', 'demo'],
    ...overrides,
  };
}

export function createMockDiagnostics(overrides: Partial<CompleteDiagnostics> = {}): CompleteDiagnostics {
  return {
    app_id: 'test-app',
    scenario: 'test-scenario',
    captured_at: '2024-01-01T00:00:00Z',
    warnings: [],
    severity: 'ok',
    summary: 'All checks passed',
    ...overrides,
  };
}

export function createMockDiagnosticsWithWarnings(count = 3): CompleteDiagnostics {
  return createMockDiagnostics({
    severity: 'warn',
    summary: `${count} issues detected`,
    warnings: Array.from({ length: count }, (_, i) => ({
      source: 'bridge-rules',
      severity: 'warn' as const,
      message: `Warning ${i + 1}: Something needs attention`,
      file_path: `src/file${i}.ts`,
      line: 10 + i,
    })),
  });
}
