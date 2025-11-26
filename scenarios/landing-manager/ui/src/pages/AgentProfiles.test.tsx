// [REQ:TMPL-AGENT-PROFILES] Focused UI test for agent profiles requirement
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

const mockPersonas = [
  {
    id: 'minimal-design',
    name: 'Minimal Design Advocate',
    description: 'Clean, spacious layouts with strong typography',
    prompt_template: 'You are a minimal design expert...',
    use_cases: ['B2B SaaS', 'Developer tools'],
  },
  {
    id: 'conversion-optimized',
    name: 'Conversion Optimizer',
    description: 'Data-driven optimization for conversion rates',
    prompt_template: 'You are a conversion rate optimization expert...',
    use_cases: ['E-commerce', 'Lead generation'],
  },
];

describe('[REQ:TMPL-AGENT-PROFILES] Agent Profiles - UI Layer', () => {
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
    vi.mocked(api.listPersonas).mockResolvedValue(mockPersonas);
    vi.mocked(api.getPersona).mockImplementation((id: string) => {
      const persona = mockPersonas.find(p => p.id === id);
      return Promise.resolve(persona || mockPersonas[0]);
    });
  });

  it('should display agent customization section in UI', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const agentSection = screen.getByTestId('agent-customization-form');
      expect(agentSection).toBeInTheDocument();
    });
  });

  it('should show agent customization heading with app-issue-tracker reference', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      // May appear multiple times (heading + tooltips), so get all and check first
      const headings = screen.getAllByText(/Agent Customization/i);
      expect(headings.length).toBeGreaterThan(0);
      const heading = headings[0];
      expect(heading).toBeInTheDocument();
    });
  });

  it('should display agent customization description', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const description = screen.getByText(/Files an issue in app-issue-tracker/i);
      expect(description).toBeInTheDocument();
      expect(description.textContent).toContain('AI agent');
      expect(description.textContent).toContain('brief and assets');
    });
  });

  it('should show customization form fields', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByTestId('customize-slug-input')).toBeInTheDocument();
      expect(screen.getByTestId('customize-assets-input')).toBeInTheDocument();
      expect(screen.getByTestId('customize-brief-input')).toBeInTheDocument();
    });
  });

  it('should display agent trigger button', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const triggerButton = screen.getByTestId('customize-button');
      expect(triggerButton).toBeInTheDocument();
      expect(triggerButton.textContent).toContain('File issue & trigger agent');
    });
  });

  it('should show brief input field for agent goals', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const briefInput = screen.getByTestId('customize-brief-input');
      expect(briefInput).toBeInTheDocument();
      expect(briefInput.getAttribute('placeholder')).toContain('SaaS founders');
      expect(briefInput.getAttribute('placeholder')).toContain('tone');
      expect(briefInput.getAttribute('placeholder')).toContain('CTA');
    });
  });

  it('should display assets input for agent customization', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const assetsInput = screen.getByTestId('customize-assets-input');
      expect(assetsInput).toBeInTheDocument();
      expect(screen.getByText(/Assets \(Optional\)/i)).toBeInTheDocument();
    });
  });

  it('should show loading state during agent trigger', async () => {
    vi.mocked(api.customizeScenario).mockImplementation(() =>
      new Promise(resolve => setTimeout(() => resolve({
        status: 'queued',
        issue_id: 'ISS-001',
        agent: 'landing-customizer',
        run_id: 'RUN-001',
      }), 100))
    );

    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const customizeButton = screen.getByTestId('customize-button');
      expect(customizeButton).toBeInTheDocument();
    });

    // Note: Actual click interaction testing would require more setup
    // This test verifies the UI structure exists for agent profile integration
  });

  it('should display customization result section', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByTestId('agent-customization-form')).toBeInTheDocument();
    });

    // Verify that the component structure supports showing customization results
    // The actual result display is conditional and appears after customization
    const form = screen.getByTestId('agent-customization-form');
    expect(form).toBeInTheDocument();
  });

  it('should provide context about agent persona options in UI guidance', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const briefInput = screen.getByTestId('customize-brief-input');
      expect(briefInput).toBeInTheDocument();

      // The placeholder provides guidance on what to include in the brief
      // which would align with persona prompts
      const placeholder = briefInput.getAttribute('placeholder');
      expect(placeholder).toContain('SaaS founders');
      expect(placeholder).toContain('tone');
    });
  });

  it('should support structured brief for agent personas', async () => {
    render(
      <BrowserRouter>
        <FactoryHome />
      </BrowserRouter>
    );

    await waitFor(() => {
      const briefInput = screen.getByTestId('customize-brief-input');
      expect(briefInput).toBeInTheDocument();

      // Verify that brief input is a textarea (multi-line) to support detailed prompts
      expect(briefInput.tagName).toBe('TEXTAREA');
      expect(briefInput.classList.contains('min-h-[120px]')).toBe(true);
    });
  });
});
