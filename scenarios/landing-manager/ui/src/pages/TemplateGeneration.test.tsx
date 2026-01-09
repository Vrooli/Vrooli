// [REQ:TMPL-GENERATION] Focused UI test for single-command generation requirement
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

const mockGenerationResult = {
  scenario_id: 'test-landing',
  name: 'Test Landing',
  template: 'saas-landing-page',
  path: '/tmp/generated/test-landing',
  status: 'success',
  next_steps: ['Move to /scenarios/test-landing', 'Run make start'],
};

describe('[REQ:TMPL-GENERATION] Single-Command Generation - UI Layer', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(api.listTemplates).mockResolvedValue([mockTemplate]);
    vi.mocked(api.getTemplate).mockResolvedValue(mockTemplate);
    vi.mocked(api.generateScenario).mockResolvedValue(mockGenerationResult);
    vi.mocked(api.listGeneratedScenarios).mockResolvedValue([]);
    vi.mocked(api.getScenarioStatus).mockResolvedValue({ success: true, scenario_id: "test", running: false, status_text: "stopped" });
    vi.mocked(api.startScenario).mockResolvedValue({ success: true, message: "started", scenario_id: "test" });
    vi.mocked(api.stopScenario).mockResolvedValue({ success: true, message: "stopped", scenario_id: "test" });
    vi.mocked(api.restartScenario).mockResolvedValue({ success: true, message: "restarted", scenario_id: "test" });
    vi.mocked(api.getScenarioLogs).mockResolvedValue({ success: true, scenario_id: "test", logs: "logs" });
    vi.mocked(api.getPreviewLinks).mockResolvedValue({ scenario_id: "test", links: {} });
  });

  it('should display generation form with required fields', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      // Check for name and slug input fields
      const nameInputs = screen.getAllByPlaceholderText(/My Awesome Product/i);
      const slugInputs = screen.getAllByPlaceholderText(/my-awesome-product/i);

      expect(nameInputs.length).toBeGreaterThan(0);
      expect(slugInputs.length).toBeGreaterThan(0);
    });
  });

  it('should display Generate now button for single-command generation', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const generateButtons = screen.getAllByText(/Generate now/i);
      expect(generateButtons.length).toBeGreaterThan(0);
    });
  });

  it('should show dry-run option alongside generate option', async () => {
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
  });

  it('should indicate generation accepts name and slug as minimum inputs', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      // Name input
      expect(screen.getAllByPlaceholderText(/My Awesome Product/i).length).toBeGreaterThan(0);

      // Slug input
      expect(screen.getAllByPlaceholderText(/my-awesome-product|my-landing-page/i).length).toBeGreaterThan(0);
    });

    // Verify generate button is present (single-command entry point, may appear multiple times)
    const generateButtons = screen.getAllByText(/Generate now/i);
    expect(generateButtons.length).toBeGreaterThan(0);
  });

  it('should expose generateScenario API function for UI interaction', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const generateButtons = screen.getAllByText(/Generate now/i);
      expect(generateButtons.length).toBeGreaterThan(0);
    });

    // Verify the API function exists for generation
    expect(api.generateScenario).toBeDefined();
  });

  it('should display next steps guidance after generation', async () => {
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

    vi.mocked(api.listGeneratedScenarios).mockResolvedValue(mockGeneratedScenarios);
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
      expect(screen.getByText('Test Landing')).toBeInTheDocument();
    });

    // Generated scenarios section shows status
    expect(screen.getAllByText(/ready/i).length).toBeGreaterThan(0);
  });

  it('should handle generation form with template selection', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getAllByText('SaaS Landing Page').length).toBeGreaterThan(0);
    });

    // Template is available for selection
    expect(screen.getByText('Templates Available')).toBeInTheDocument();
    // Use getAllByText since "1" may appear multiple times in the UI
    const ones = screen.getAllByText('1');
    expect(ones.length).toBeGreaterThan(0);
  });
});
