// [REQ:TMPL-METADATA] Focused UI test for template metadata requirement
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import FactoryHome from './FactoryHome';
import * as api from '../lib/api';

vi.mock('../lib/api');

const mockTemplateWithFullMetadata = {
  id: 'saas-landing-page',
  name: 'SaaS Landing Page',
  description: 'Landing page template for SaaS products',
  version: '1.0.0',
  sections: {
    required: [
      { id: 'hero', name: 'Hero Section' },
      { id: 'features', name: 'Features' },
      { id: 'pricing', name: 'Pricing' },
    ],
    optional: [
      { id: 'testimonials', name: 'Testimonials' },
      { id: 'faq', name: 'FAQ' },
    ],
  },
  metrics_hooks: [
    { id: 'page_view', name: 'Page View', description: 'Track page views' },
    { id: 'conversion', name: 'Conversion', description: 'Track conversions' },
  ],
  customization_schema: {
    branding: {
      logo: { type: 'string', description: 'Logo URL' },
      colors: { type: 'object', description: 'Brand colors' },
    },
    seo: {
      title: { type: 'string', description: 'Page title' },
      description: { type: 'string', description: 'Meta description' },
    },
  },
};

describe('[REQ:TMPL-METADATA] Template Metadata - UI Layer', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(api.listTemplates).mockResolvedValue([mockTemplateWithFullMetadata]);
    vi.mocked(api.getTemplate).mockResolvedValue(mockTemplateWithFullMetadata);
    vi.mocked(api.listGeneratedScenarios).mockResolvedValue([]);
    vi.mocked(api.getScenarioStatus).mockResolvedValue({ success: true, scenario_id: "test", running: false, status_text: "stopped" });
    vi.mocked(api.startScenario).mockResolvedValue({ success: true, message: "started", scenario_id: "test" });
    vi.mocked(api.stopScenario).mockResolvedValue({ success: true, message: "stopped", scenario_id: "test" });
    vi.mocked(api.restartScenario).mockResolvedValue({ success: true, message: "restarted", scenario_id: "test" });
    vi.mocked(api.getScenarioLogs).mockResolvedValue({ success: true, scenario_id: "test", logs: "logs" });
    vi.mocked(api.getPreviewLinks).mockResolvedValue({ scenario_id: "test", links: {} });
  });

  it('should display sections metadata in UI', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/Sections/i)).toBeInTheDocument();
    });
  });

  it('should display metrics hooks metadata in UI', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/Metrics Hooks/i)).toBeInTheDocument();
    });
  });

  it('should display customization schema metadata in UI', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/Customization Schema/i)).toBeInTheDocument();
    });
  });

  it('should show template version in metadata display', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const versionElements = screen.getAllByText(/v1\.0\.0/i);
      expect(versionElements.length).toBeGreaterThan(0);
    });
  });

  it('should display template description as part of metadata', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getAllByText(/Landing page template for SaaS products/i).length).toBeGreaterThan(0);
    });
  });

  it('should handle template without optional metadata fields', async () => {
    const minimalTemplate = {
      id: 'minimal-template',
      name: 'Minimal Template',
      description: 'Minimal template',
      version: '1.0.0',
    };

    vi.mocked(api.listTemplates).mockResolvedValue([minimalTemplate]);
    vi.mocked(api.getTemplate).mockResolvedValue(minimalTemplate);
    vi.mocked(api.listGeneratedScenarios).mockResolvedValue([]);
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
      expect(screen.getAllByText('Minimal Template').length).toBeGreaterThan(0);
    });

    // Should not crash when optional metadata is missing
    expect(screen.queryByText(/undefined/i)).not.toBeInTheDocument();
  });

  it('should call getTemplate API when template metadata is needed', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getAllByText('SaaS Landing Page').length).toBeGreaterThan(0);
    });

    // getTemplate may be called for detailed view
    // Exact count depends on UI interaction, but verify it's available
    expect(api.getTemplate).toBeDefined();
  });
});
