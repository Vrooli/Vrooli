// [REQ:TMPL-MULTIPLE] Focused UI test for multiple templates requirement
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import FactoryHome from './FactoryHome';
import * as api from '../lib/api';

vi.mock('../lib/api');

const mockMultipleTemplates = [
  {
    id: 'saas-landing-page',
    name: 'SaaS Landing Page',
    description: 'Landing page template for SaaS products',
    version: '1.0.0',
  },
  {
    id: 'lead-magnet',
    name: 'Lead Magnet Landing',
    description: 'Landing page optimized for lead generation',
    version: '1.0.0',
  },
];

describe('[REQ:TMPL-MULTIPLE] Multiple Templates - UI Layer', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(api.listTemplates).mockResolvedValue(mockMultipleTemplates);
    vi.mocked(api.getTemplate).mockImplementation((id: string) => {
      const template = mockMultipleTemplates.find(t => t.id === id);
      return Promise.resolve(template || mockMultipleTemplates[0]);
    });
    vi.mocked(api.listGeneratedScenarios).mockResolvedValue([]);
    vi.mocked(api.getScenarioStatus).mockResolvedValue({ success: true, scenario_id: "test", running: false, status_text: "stopped" });
    vi.mocked(api.startScenario).mockResolvedValue({ success: true, message: "started", scenario_id: "test" });
    vi.mocked(api.stopScenario).mockResolvedValue({ success: true, message: "stopped", scenario_id: "test" });
    vi.mocked(api.restartScenario).mockResolvedValue({ success: true, message: "restarted", scenario_id: "test" });
    vi.mocked(api.getScenarioLogs).mockResolvedValue({ success: true, scenario_id: "test", logs: "logs" });
    vi.mocked(api.getPreviewLinks).mockResolvedValue({ scenario_id: "test", links: {} });
  });

  it('should list multiple templates in the UI', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    // Wait for templates to load - check for 2 available
    await waitFor(() => {
      expect(screen.getByText('Templates Available')).toBeInTheDocument();
      expect(screen.getByText('2')).toBeInTheDocument();
    }, { timeout: 3000 });

    // Verify both templates are shown in template cards (multiple instances exist due to selected template display)
    expect(screen.getAllByText('SaaS Landing Page').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Lead Magnet Landing').length).toBeGreaterThan(0);
  });

  it('should display template grid with multiple template cards', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const saasCard = screen.getByTestId('template-card-saas-landing-page');
      const leadMagnetCard = screen.getByTestId('template-card-lead-magnet');

      expect(saasCard).toBeInTheDocument();
      expect(leadMagnetCard).toBeInTheDocument();
    });
  });

  it('should show correct details for each template', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    // Wait for template catalog to be populated
    await waitFor(() => {
      const catalog = screen.getByTestId('template-catalog');
      expect(catalog.textContent).toContain('SaaS Landing Page');
      expect(catalog.textContent).toContain('Lead Magnet Landing');
    }, { timeout: 3000 });

    // Verify both template cards are present using testids
    expect(screen.getByTestId('template-card-saas-landing-page')).toBeInTheDocument();
    expect(screen.getByTestId('template-card-lead-magnet')).toBeInTheDocument();
  });

  it('should allow selection between multiple templates', async () => {
    const { container } = render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    // Wait for both template cards to be rendered
    await waitFor(() => {
      expect(screen.getByTestId('template-card-saas-landing-page')).toBeInTheDocument();
      expect(screen.getByTestId('template-card-lead-magnet')).toBeInTheDocument();
    }, { timeout: 5000 });

    // Both templates should be clickable and visible
    const saasCard = screen.getByTestId('template-card-saas-landing-page');
    const leadMagnetCard = screen.getByTestId('template-card-lead-magnet');

    expect(saasCard).toBeInTheDocument();
    expect(leadMagnetCard).toBeInTheDocument();

    // First template should be selected by default
    expect(saasCard.className).toContain('border-emerald-400/60');
  });

  it('should display version for each template', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const saasVersion = screen.getByTestId('template-version-saas-landing-page');
      const leadMagnetVersion = screen.getByTestId('template-version-lead-magnet');

      expect(saasVersion).toHaveTextContent('v1.0.0');
      expect(leadMagnetVersion).toHaveTextContent('v1.0.0');
    });
  });

  it('should handle templates with different versions', async () => {
    const templatesWithDifferentVersions = [
      { ...mockMultipleTemplates[0], version: '1.0.0' },
      { ...mockMultipleTemplates[1], version: '2.1.0' },
    ];
    vi.mocked(api.listTemplates).mockResolvedValue(templatesWithDifferentVersions);

    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const saasVersion = screen.getByTestId('template-version-saas-landing-page');
      const leadMagnetVersion = screen.getByTestId('template-version-lead-magnet');

      expect(saasVersion).toHaveTextContent('v1.0.0');
      expect(leadMagnetVersion).toHaveTextContent('v2.1.0');
    });
  });
});
