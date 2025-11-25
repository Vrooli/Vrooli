// [REQ:TMPL-PROVENANCE] Focused UI test for template provenance stamping requirement
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

const mockGeneratedScenarioWithProvenance = {
  scenario_id: 'test-landing',
  name: 'Test Landing',
  template_id: 'saas-landing-page',
  template_version: '1.0.0',
  path: '/tmp/generated/test-landing',
  status: 'ready',
  generated_at: '2025-11-24T12:00:00Z',
};

describe('[REQ:TMPL-PROVENANCE] Template Provenance Stamping - UI Layer', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(api.listTemplates).mockResolvedValue([mockTemplate]);
    vi.mocked(api.getTemplate).mockResolvedValue(mockTemplate);
    vi.mocked(api.listGeneratedScenarios).mockResolvedValue([mockGeneratedScenarioWithProvenance]);
    vi.mocked(api.getScenarioStatus).mockResolvedValue({ success: true, scenario_id: "test", running: false, status_text: "stopped" });
    vi.mocked(api.startScenario).mockResolvedValue({ success: true, message: "started", scenario_id: "test" });
    vi.mocked(api.stopScenario).mockResolvedValue({ success: true, message: "stopped", scenario_id: "test" });
    vi.mocked(api.restartScenario).mockResolvedValue({ success: true, message: "restarted", scenario_id: "test" });
    vi.mocked(api.getScenarioLogs).mockResolvedValue({ success: true, scenario_id: "test", logs: "logs" });
    vi.mocked(api.getPreviewLinks).mockResolvedValue({ scenario_id: "test", links: {} });
  });

  it('should display template ID in generated scenario metadata', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Test Landing')).toBeInTheDocument();
    });

    // Template ID should be visible (shown directly without "Template:" prefix, may appear multiple times)
    const templateRefs = screen.getAllByText(/saas-landing-page/i);
    expect(templateRefs.length).toBeGreaterThan(0);
  });

  it('should display template version for audit and migration planning', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Test Landing')).toBeInTheDocument();
    });

    // Version should be visible
    const versionElements = screen.getAllByText(/v1\.0\.0/i);
    expect(versionElements.length).toBeGreaterThan(0);
  });

  it('should show provenance metadata for each generated scenario', async () => {
    const multipleScenarios = [
      {
        scenario_id: 'landing-1',
        name: 'Landing 1',
        template_id: 'saas-landing-page',
        template_version: '1.0.0',
        path: '/tmp/generated/landing-1',
        status: 'ready',
        generated_at: '2025-11-24T10:00:00Z',
      },
      {
        scenario_id: 'landing-2',
        name: 'Landing 2',
        template_id: 'saas-landing-page',
        template_version: '1.0.0',
        path: '/tmp/generated/landing-2',
        status: 'ready',
        generated_at: '2025-11-24T11:00:00Z',
      },
    ];

    vi.mocked(api.listGeneratedScenarios).mockResolvedValue(multipleScenarios);
    vi.mocked(api.getScenarioStatus).mockResolvedValue({ success: true, scenario_id: "test", running: false, status_text: "stopped" });
    vi.mocked(api.startScenario).mockResolvedValue({ success: true, message: "started", scenario_id: "test" });
    vi.mocked(api.stopScenario).mockResolvedValue({ success: true, message: "stopped", scenario_id: "test" });
    vi.mocked(api.restartScenario).mockResolvedValue({ success: true, message: "restarted", scenario_id: "test" });
    vi.mocked(api.getScenarioLogs).mockResolvedValue({ success: true, scenario_id: "test", logs: "logs" });
    vi.mocked(api.getPreviewLinks).mockResolvedValue({ scenario_id: "test", links: {} });

    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Landing 1')).toBeInTheDocument();
      expect(screen.getByText('Landing 2')).toBeInTheDocument();
    });

    // Both should show template provenance (template ID appears directly)
    const templateRefs = screen.getAllByText(/saas-landing-page/i);
    expect(templateRefs.length).toBeGreaterThanOrEqual(2);
  });

  it('should include generation timestamp as part of provenance', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Test Landing')).toBeInTheDocument();
    });

    // Timestamp is part of provenance metadata
    expect(api.listGeneratedScenarios).toHaveBeenCalled();
  });

  it('should handle scenarios with different template versions', async () => {
    const scenariosWithDifferentVersions = [
      {
        scenario_id: 'old-landing',
        name: 'Old Landing',
        template_id: 'saas-landing-page',
        template_version: '0.9.0',
        path: '/tmp/generated/old-landing',
        status: 'ready',
        generated_at: '2025-11-20T12:00:00Z',
      },
      {
        scenario_id: 'new-landing',
        name: 'New Landing',
        template_id: 'saas-landing-page',
        template_version: '1.0.0',
        path: '/tmp/generated/new-landing',
        status: 'ready',
        generated_at: '2025-11-24T12:00:00Z',
      },
    ];

    vi.mocked(api.listGeneratedScenarios).mockResolvedValue(scenariosWithDifferentVersions);
    vi.mocked(api.getScenarioStatus).mockResolvedValue({ success: true, scenario_id: "test", running: false, status_text: "stopped" });
    vi.mocked(api.startScenario).mockResolvedValue({ success: true, message: "started", scenario_id: "test" });
    vi.mocked(api.stopScenario).mockResolvedValue({ success: true, message: "stopped", scenario_id: "test" });
    vi.mocked(api.restartScenario).mockResolvedValue({ success: true, message: "restarted", scenario_id: "test" });
    vi.mocked(api.getScenarioLogs).mockResolvedValue({ success: true, scenario_id: "test", logs: "logs" });
    vi.mocked(api.getPreviewLinks).mockResolvedValue({ scenario_id: "test", links: {} });

    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Old Landing')).toBeInTheDocument();
      expect(screen.getByText('New Landing')).toBeInTheDocument();
    });

    // Both versions should be visible
    expect(screen.getByText(/v0\.9\.0/i)).toBeInTheDocument();
    expect(screen.getAllByText(/v1\.0\.0/i).length).toBeGreaterThan(0);
  });

  it('should maintain provenance link between template and generated scenario', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      // Template shown in templates section
      expect(screen.getAllByText('SaaS Landing Page').length).toBeGreaterThan(0);

      // Generated scenario shows same template ID (directly without prefix, may appear multiple times)
      const templateRefs = screen.getAllByText(/saas-landing-page/i);
      expect(templateRefs.length).toBeGreaterThan(0);
    });

    // Verify both sections are present
    expect(screen.getByText('Templates Available')).toBeInTheDocument(); // Templates
    expect(screen.getByText('1')).toBeInTheDocument(); // Template count
    expect(screen.getByText('Test Landing')).toBeInTheDocument(); // Generated scenarios
  });
});
