// [REQ:TMPL-AVAILABILITY] Focused UI test for template availability requirement
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import FactoryHome from './FactoryHome';
import * as api from '../lib/api';

vi.mock('../lib/api');

const mockTemplates = [
  {
    id: 'saas-landing-page',
    name: 'SaaS Landing Page',
    description: 'Landing page template for SaaS products',
    version: '1.0.0',
  },
];

describe('[REQ:TMPL-AVAILABILITY] Template Availability - UI Layer', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(api.listTemplates).mockResolvedValue(mockTemplates);
    vi.mocked(api.getTemplate).mockResolvedValue(mockTemplates[0]);
    vi.mocked(api.listGeneratedScenarios).mockResolvedValue([]);
    vi.mocked(api.getScenarioStatus).mockResolvedValue({ success: true, scenario_id: "test", running: false, status_text: "stopped" });
    vi.mocked(api.startScenario).mockResolvedValue({ success: true, message: "started", scenario_id: "test" });
    vi.mocked(api.stopScenario).mockResolvedValue({ success: true, message: "stopped", scenario_id: "test" });
    vi.mocked(api.restartScenario).mockResolvedValue({ success: true, message: "restarted", scenario_id: "test" });
    vi.mocked(api.getScenarioLogs).mockResolvedValue({ success: true, scenario_id: "test", logs: "logs" });
    vi.mocked(api.getPreviewLinks).mockResolvedValue({ scenario_id: "test", links: {} });
  });

  it('should expose saas-landing-page template via UI listing', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const templateElements = screen.getAllByText('SaaS Landing Page');
      expect(templateElements.length).toBeGreaterThan(0);
    });

    expect(api.listTemplates).toHaveBeenCalledTimes(1);
  });

  it('should display template count in UI', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Templates Available')).toBeInTheDocument();
      expect(screen.getAllByText('1').length).toBeGreaterThan(0);
    });
  });

  it('should show template details when available', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const templateElements = screen.getAllByText('SaaS Landing Page');
      expect(templateElements.length).toBeGreaterThan(0);
      expect(screen.getAllByText(/Landing page template for SaaS products/i).length).toBeGreaterThan(0);
    });
  });

  it('should handle empty template list gracefully', async () => {
    vi.mocked(api.listTemplates).mockResolvedValue([]);
    vi.mocked(api.getTemplate).mockResolvedValue(null as any);

    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Templates Available')).toBeInTheDocument();
      expect(screen.getByText('0')).toBeInTheDocument();
    });
  });

  it('should display loading state while fetching templates', async () => {
    vi.mocked(api.listTemplates).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByLabelText(/Loading templates/i)).toBeInTheDocument();
    });
  });

  it('should handle template loading errors with user-friendly message', async () => {
    const errorMessage = 'Failed to load templates';
    vi.mocked(api.listTemplates).mockRejectedValue(new Error(errorMessage));

    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(errorMessage)).toBeInTheDocument();
    });
  });

  it('should call listTemplates API exactly once on mount', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(api.listTemplates).toHaveBeenCalledTimes(1);
    });
  });
});
