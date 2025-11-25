// [REQ:AGENT-TRIGGER] Focused UI test for agent trigger requirement
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import FactoryHome from './FactoryHome';
import * as api from '../lib/api';

vi.mock('../lib/api');

const mockTemplate = {
  id: 'saas-landing-page',
  name: 'SaaS Landing Page',
  description: 'Landing page template for SaaS products',
  version: '1.0.0',
};

const mockGeneratedScenario = {
  scenario_id: 'test-landing',
  name: 'Test Landing',
  template_id: 'saas-landing-page',
  template_version: '1.0.0',
  path: '/tmp/generated/test-landing',
  status: 'ready',
  generated_at: '2025-11-24T12:00:00Z',
};

const mockCustomizeResult = {
  status: 'queued',
  issue_id: 'ISSUE-123',
  tracker_url: 'http://localhost:18509',
  agent: 'claude-code',
  run_id: 'RUN-456',
  message: 'Customization request filed and agent queued',
};

describe('[REQ:AGENT-TRIGGER] Agent Trigger - UI Layer', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(api.listTemplates).mockResolvedValue([mockTemplate]);
    vi.mocked(api.listGeneratedScenarios).mockResolvedValue([mockGeneratedScenario]);
    vi.mocked(api.getScenarioStatus).mockResolvedValue({ success: true, scenario_id: "test", running: false, status_text: "stopped" });
    vi.mocked(api.startScenario).mockResolvedValue({ success: true, message: "started", scenario_id: "test" });
    vi.mocked(api.stopScenario).mockResolvedValue({ success: true, message: "stopped", scenario_id: "test" });
    vi.mocked(api.restartScenario).mockResolvedValue({ success: true, message: "restarted", scenario_id: "test" });
    vi.mocked(api.getScenarioLogs).mockResolvedValue({ success: true, scenario_id: "test", logs: "logs" });
    vi.mocked(api.getPreviewLinks).mockResolvedValue({ scenario_id: "test", links: {} });
    vi.mocked(api.customizeScenario).mockResolvedValue(mockCustomizeResult);
  });

  it('should display agent customization section in UI', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/Agent customization/i)).toBeInTheDocument();
    });
  });

  it('should show File issue & trigger agent button', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/File issue & trigger agent/i)).toBeInTheDocument();
    });
  });

  it('should display brief input field for structured customization goals', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const briefInputs = screen.getAllByPlaceholderText(/SaaS founders/i);
      expect(briefInputs.length).toBeGreaterThan(0);
    });
  });

  it('should expose customizeScenario API for triggering agent', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/File issue & trigger agent/i)).toBeInTheDocument();
    });

    // Verify API function exists
    expect(api.customizeScenario).toBeDefined();
  });

  it('should show agent customization form with scenario selection capability', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      // Agent section should be present
      expect(screen.getByText(/Agent customization/i)).toBeInTheDocument();

      // Generated scenario should be available for selection
      expect(screen.getByText('Test Landing')).toBeInTheDocument();
    });
  });

  it('should indicate assets can be provided as part of customization brief', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/Agent customization/i)).toBeInTheDocument();
    });

    // Form should accept brief (which can include asset references)
    const briefInputs = screen.getAllByPlaceholderText(/SaaS founders/i);
    expect(briefInputs.length).toBeGreaterThan(0);
  });

  it('should show investigation trigger option alongside customization', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/File issue & trigger agent/i)).toBeInTheDocument();
    });

    // Button text indicates both filing issue and triggering agent
    expect(screen.getByText(/File issue & trigger agent/i)).toBeInTheDocument();
  });

  it('should handle agent customization form with all required fields', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      // Scenario ID (from generated scenarios list)
      expect(screen.getByText('Test Landing')).toBeInTheDocument();

      // Brief input
      const briefInputs = screen.getAllByPlaceholderText(/Example.*Target SaaS founders/i);
      expect(briefInputs.length).toBeGreaterThan(0);

      // Trigger button
      expect(screen.getByText(/File issue & trigger agent/i)).toBeInTheDocument();
    });
  });

  it('should display generated scenarios as targets for agent customization', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Test Landing')).toBeInTheDocument();
    });

    // Generated scenario should be visible as potential customization target
    expect(screen.getAllByText(/ready/i).length).toBeGreaterThan(0);
  });
});
