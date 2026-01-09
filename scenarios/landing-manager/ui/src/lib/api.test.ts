import { describe, it, expect, beforeEach, vi } from 'vitest';
import {
  fetchHealth,
  listTemplates,
  getTemplate,
  listGeneratedScenarios,
  generateScenario,
  customizeScenario,
  type Template,
  type GenerationResult,
  type CustomizeResult,
  type GeneratedScenario,
} from './api';

// [REQ:TMPL-AVAILABILITY]
describe('API Client', () => {
  beforeEach(() => {
    // Reset fetch mock before each test
    vi.resetAllMocks();
  });

  describe('fetchHealth', () => {
    it('should fetch health status successfully', async () => {
      const mockHealth = {
        status: 'healthy',
        service: 'landing-manager',
        timestamp: '2025-11-24T12:00:00Z',
      };

      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockHealth),
        } as Response)
      );

      const result = await fetchHealth();
      expect(result).toEqual(mockHealth);
      expect(global.fetch).toHaveBeenCalledTimes(1);
    });

    it('should throw error on failed health check', async () => {
      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: false,
          status: 500,
          text: () => Promise.resolve('Internal Server Error'),
        } as Response)
      );

      await expect(fetchHealth()).rejects.toThrow('API call failed (500): Internal Server Error');
    });
  });

  // [REQ:TMPL-AVAILABILITY]
  describe('listTemplates', () => {
    it('should list available templates', async () => {
      const mockTemplates: Template[] = [
        {
          id: 'saas-landing-page',
          name: 'SaaS Landing Page',
          description: 'Landing page template for SaaS products',
          version: '1.0.0',
        },
      ];

      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockTemplates),
        } as Response)
      );

      const result = await listTemplates();
      expect(result).toEqual(mockTemplates);
      expect(result).toHaveLength(1);
      expect(result[0].id).toBe('saas-landing-page');
    });

    it('should handle empty template list', async () => {
      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.resolve([]),
        } as Response)
      );

      const result = await listTemplates();
      expect(result).toEqual([]);
    });
  });

  // [REQ:TMPL-METADATA]
  describe('getTemplate', () => {
    it('should fetch template details by ID', async () => {
      const mockTemplate: Template = {
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
          { id: 'conversion', name: 'Conversion' },
        ],
        customization_schema: {
          branding: {},
          seo: {},
        },
      };

      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockTemplate),
        } as Response)
      );

      const result = await getTemplate('saas-landing-page');
      expect(result).toEqual(mockTemplate);
      expect(result.sections).toBeDefined();
      expect(result.metrics_hooks).toBeDefined();
    });

    it('should throw error for non-existent template', async () => {
      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: false,
          status: 404,
          text: () => Promise.resolve('Template not found'),
        } as Response)
      );

      await expect(getTemplate('non-existent')).rejects.toThrow('API call failed (404): Template not found');
    });
  });

  // [REQ:TMPL-GENERATION]
  describe('generateScenario', () => {
    it('should generate scenario successfully', async () => {
      const mockResult: GenerationResult = {
        scenario_id: 'test-landing',
        name: 'Test Landing',
        template: 'saas-landing-page',
        path: '/tmp/generated/test-landing',
        status: 'success',
        next_steps: ['Move to /scenarios/test-landing', 'Run make start'],
      };

      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockResult),
        } as Response)
      );

      const result = await generateScenario('saas-landing-page', 'Test Landing', 'test-landing');
      expect(result).toEqual(mockResult);
      expect(result.status).toBe('success');
      expect(result.next_steps).toBeDefined();
    });

    it('should support dry-run mode', async () => {
      const mockResult: GenerationResult = {
        scenario_id: 'test-landing',
        name: 'Test Landing',
        template: 'saas-landing-page',
        path: '/tmp/generated/test-landing',
        status: 'dry-run',
        plan: {
          paths: [
            'api/main.go',
            'ui/src/App.tsx',
            'requirements/index.json',
          ],
        },
      };

      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockResult),
        } as Response)
      );

      const result = await generateScenario('saas-landing-page', 'Test Landing', 'test-landing', { dry_run: true });
      expect(result).toEqual(mockResult);
      expect(result.status).toBe('dry-run');
      expect(result.plan?.paths).toBeDefined();
      expect(result.plan?.paths.length).toBeGreaterThan(0);
    });

    it('should handle generation failures', async () => {
      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: false,
          status: 400,
          text: () => Promise.resolve('Invalid slug format'),
        } as Response)
      );

      await expect(generateScenario('saas-landing-page', 'Test', 'invalid slug')).rejects.toThrow('API call failed (400): Invalid slug format');
    });
  });

  // [REQ:AGENT-TRIGGER]
  describe('customizeScenario', () => {
    it('should trigger agent customization successfully', async () => {
      const mockResult: CustomizeResult = {
        status: 'queued',
        issue_id: 'ISSUE-123',
        tracker_url: 'http://localhost:18509',
        agent: 'claude-code',
        run_id: 'RUN-456',
        message: 'Customization request filed and agent queued',
      };

      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockResult),
        } as Response)
      );

      const result = await customizeScenario(
        'test-landing',
        'Make the hero section more engaging with animations',
        ['assets/logo.svg'],
        true
      );
      expect(result).toEqual(mockResult);
      expect(result.status).toBe('queued');
      expect(result.issue_id).toBeDefined();
      expect(result.agent).toBeDefined();
    });

    it('should handle customization errors', async () => {
      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: false,
          status: 404,
          text: () => Promise.resolve('Scenario not found'),
        } as Response)
      );

      await expect(customizeScenario('non-existent', 'Brief', [])).rejects.toThrow('API call failed (404): Scenario not found');
    });
  });

  describe('listGeneratedScenarios', () => {
    it('should list generated scenarios', async () => {
      const mockScenarios: GeneratedScenario[] = [
        {
          scenario_id: 'test-landing-1',
          name: 'Test Landing 1',
          template_id: 'saas-landing-page',
          template_version: '1.0.0',
          path: '/tmp/generated/test-landing-1',
          status: 'ready',
          generated_at: '2025-11-24T12:00:00Z',
        },
        {
          scenario_id: 'test-landing-2',
          name: 'Test Landing 2',
          template_id: 'saas-landing-page',
          template_version: '1.0.0',
          path: '/tmp/generated/test-landing-2',
          status: 'ready',
          generated_at: '2025-11-24T13:00:00Z',
        },
      ];

      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockScenarios),
        } as Response)
      );

      const result = await listGeneratedScenarios();
      expect(result).toEqual(mockScenarios);
      expect(result).toHaveLength(2);
      expect(result[0].scenario_id).toBe('test-landing-1');
    });

    it('should handle empty generated scenarios list', async () => {
      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.resolve([]),
        } as Response)
      );

      const result = await listGeneratedScenarios();
      expect(result).toEqual([]);
    });
  });
});
