import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { act } from 'react';
import FactoryHome from './FactoryHome';
import * as api from '../lib/api';

vi.mock('../lib/api');

// Helper to wait for all pending async state updates
async function waitForAsyncUpdates() {
  await act(async () => {
    // Wait for multiple microtask cycles to ensure all effects have settled
    await new Promise(resolve => setTimeout(resolve, 50));
  });
}

const mockTemplates = [
  {
    id: 'saas-landing-page',
    name: 'SaaS Landing Page',
    description: 'Landing page template for SaaS products',
    version: '1.0.0',
    sections: {
      hero: {},
      features: {},
      pricing: {},
    },
    metrics_hooks: [
      { id: 'page_view', name: 'Page View' },
    ],
    customization_schema: {
      branding: {},
    },
  },
];

const mockGeneratedScenarios = [
  {
    scenario_id: 'test-landing',
    name: 'Test Landing',
    template_id: 'saas-landing-page',
    template_version: '1.0.0',
    path: '/tmp/generated/test-landing',
    status: 'ready',
    generated_at: '2025-11-24T12:00:00Z',
  },
];

describe('FactoryHome', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    // Default successful API responses
    vi.mocked(api.listTemplates).mockResolvedValue(mockTemplates);
    vi.mocked(api.getTemplate).mockResolvedValue(mockTemplates[0]);
    vi.mocked(api.listGeneratedScenarios).mockResolvedValue(mockGeneratedScenarios);

    // Mock lifecycle API functions
    vi.mocked(api.getScenarioStatus).mockResolvedValue({
      success: true,
      scenario_id: 'test-landing',
      running: false,
      status_text: 'Status: 🔴 STOPPED',
    });
    vi.mocked(api.startScenario).mockResolvedValue({
      success: true,
      message: 'Scenario started successfully',
      scenario_id: 'test-landing',
    });
    vi.mocked(api.stopScenario).mockResolvedValue({
      success: true,
      message: 'Scenario stopped successfully',
      scenario_id: 'test-landing',
    });
    vi.mocked(api.restartScenario).mockResolvedValue({
      success: true,
      message: 'Scenario restarted successfully',
      scenario_id: 'test-landing',
    });
    vi.mocked(api.getScenarioLogs).mockResolvedValue({
      success: true,
      scenario_id: 'test-landing',
      logs: 'Sample log output',
    });
    vi.mocked(api.getPreviewLinks).mockResolvedValue({
      scenario_id: 'test-landing',
      ui_port: 38611,
      links: {
        public: 'http://localhost:38611/',
        admin: 'http://localhost:38611/admin',
        health: 'http://localhost:38611/health',
      },
    });
  });

  it('should render the factory home page', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/Landing Page Factory/i)).toBeInTheDocument();
      expect(screen.getByText(/Landing Manager · Factory/i)).toBeInTheDocument();
    });

    // Wait for all async state updates to complete (loadStatuses effect)
    await waitFor(() => {
      expect(api.getScenarioStatus).toHaveBeenCalled();
    });

    // Ensure all pending state updates are processed
    await waitForAsyncUpdates();
  });

  it('should load and display templates', async () => {
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
    expect(screen.getByText('Templates Available')).toBeInTheDocument();
    const ones = screen.getAllByText('1');
    expect(ones.length).toBeGreaterThan(0);

    await waitFor(() => {
      expect(api.getScenarioStatus).toHaveBeenCalled();
    });
    await waitForAsyncUpdates();
  });

  it('should display template loading state', async () => {
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
    await waitForAsyncUpdates();
  });

  it('should handle template loading errors', async () => {
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

    await waitFor(() => {
      expect(api.getScenarioStatus).toHaveBeenCalled();
    });
    await waitForAsyncUpdates();
  });

  it('should display template metadata when selected', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const templateElements = screen.getAllByText('SaaS Landing Page');
      expect(templateElements.length).toBeGreaterThan(0);
    });

    const sectionsElements = screen.getAllByText(/Sections/i);
    expect(sectionsElements.length).toBeGreaterThan(0);
    const metricsElements = screen.getAllByText(/Metrics Hooks/i);
    expect(metricsElements.length).toBeGreaterThan(0);
    const customizationElements = screen.getAllByText(/Customization Schema/i);
    expect(customizationElements.length).toBeGreaterThan(0);

    await waitFor(() => {
      expect(api.getScenarioStatus).toHaveBeenCalled();
    });
    await waitForAsyncUpdates();
  });

  it('should load and display generated scenarios', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Test Landing')).toBeInTheDocument();
    });

    expect(api.listGeneratedScenarios).toHaveBeenCalledTimes(1);

    await waitFor(() => {
      expect(api.getScenarioStatus).toHaveBeenCalled();
    });
    await waitForAsyncUpdates();
  });

  it('should handle empty generated scenarios list', async () => {
    vi.mocked(api.listGeneratedScenarios).mockResolvedValue([]);

    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/Ready to Launch Your First Landing Page/i)).toBeInTheDocument();
    });

    await waitFor(() => {
      expect(api.listGeneratedScenarios).toHaveBeenCalled();
    });
    await waitForAsyncUpdates();
  });

  it('should display generation buttons', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/Dry-run \(preview only\)/i)).toBeInTheDocument();
      const generateButtons = screen.getAllByText(/Generate now/i);
      expect(generateButtons.length).toBeGreaterThan(0);
    });

    await waitFor(() => {
      expect(api.getScenarioStatus).toHaveBeenCalled();
    });
    await waitForAsyncUpdates();
  });

  it('should display agent customization section', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const agentCustomizationElements = screen.getAllByText(/Agent customization/i);
      expect(agentCustomizationElements.length).toBeGreaterThan(0);
      expect(screen.getByText(/File issue & trigger agent/i)).toBeInTheDocument();
    });

    await waitFor(() => {
      expect(api.getScenarioStatus).toHaveBeenCalled();
    });
    await waitForAsyncUpdates();
  });

  it('should render name and slug input fields', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const nameInputs = screen.getAllByPlaceholderText(/My Awesome Product/i);
      const slugInputs = screen.getAllByPlaceholderText(/my-awesome-product|my-landing-page/i);

      expect(nameInputs.length).toBeGreaterThan(0);
      expect(slugInputs.length).toBeGreaterThan(0);
    });

    await waitFor(() => {
      expect(api.getScenarioStatus).toHaveBeenCalled();
    });
    await waitForAsyncUpdates();
  });

  it('should handle API errors gracefully', async () => {
    vi.mocked(api.listTemplates).mockRejectedValue(new Error('Network error'));
    vi.mocked(api.listGeneratedScenarios).mockRejectedValue(new Error('Network error'));

    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/Landing Page Factory/i)).toBeInTheDocument();
    });

    await waitFor(() => {
      expect(api.listGeneratedScenarios).toHaveBeenCalled();
    });
    await waitForAsyncUpdates();
  });

  it('should display multiple templates when available', async () => {
    const multipleTemplates = [
      mockTemplates[0],
      {
        id: 'lead-magnet',
        name: 'Lead Magnet',
        description: 'Lead generation template',
        version: '1.0.0',
        sections: { hero: {}, form: {} },
        metrics_hooks: [],
        customization_schema: {},
      },
    ];

    vi.mocked(api.listTemplates).mockResolvedValue(multipleTemplates);

    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(api.listTemplates).toHaveBeenCalled();
    });

    await waitFor(() => {
      const saasElements = screen.getAllByText('SaaS Landing Page');
      expect(saasElements.length).toBeGreaterThan(0);
      expect(screen.getByText('Lead Magnet')).toBeInTheDocument();
    });
    await waitForAsyncUpdates();
  });
});
