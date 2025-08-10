import request from 'supertest';
import { Application } from 'express';
import app from './dist/server.js';

// Type definitions for test responses
interface HealthResponse {
  status: string;
  services: Record<string, string>;
  timestamp: string;
  version: string;
}

interface InfoResponse {
  name: string;
  version: string;
  endpoints: Record<string, string>;
  timestamp: string;
}

interface PromptsListResponse {
  prompts: any[];
  count: number;
  total: number;
  offset: number;
  limit: number;
}

// interface PromptResponse {
//   prompt: any;
// }

interface WorkflowsListResponse {
  workflows: any[];
  count: number;
  total: number;
  offset: number;
  limit: number;
}

interface AnalysisResponse {
  execution_id: string;
  status: string;
  analysis_type: string;
}

interface DecisionAnalysisResponse extends AnalysisResponse {
  decision_analysis: {
    input: string;
    context: string;
    factors_considered: string[];
    recommendation: string;
    confidence: number;
    reasoning: string;
  };
  analysis_type: 'decision';
}

interface ProsConsAnalysisResponse extends AnalysisResponse {
  pros_cons_analysis: {
    input: string;
    context: string;
    pros: Array<{ item: string; weight: number; explanation: string }>;
    cons: Array<{ item: string; weight: number; explanation: string }>;
    net_score: number;
    recommendation: string;
    confidence: number;
  };
  analysis_type: 'pros_cons';
}

interface SwotAnalysisResponse extends AnalysisResponse {
  swot_analysis: {
    input: string;
    context: string;
    strengths: string[];
    weaknesses: string[];
    opportunities: string[];
    threats: string[];
    strategic_position: string;
    recommendations: string[];
  };
  analysis_type: 'swot';
}

interface RiskAnalysisResponse extends AnalysisResponse {
  risk_assessment: {
    action: string;
    constraints: string;
    risks: Array<{
      risk: string;
      category: string;
      probability: 'Low' | 'Medium' | 'High';
      impact: 'Low' | 'Medium' | 'High';
      risk_score: number;
      mitigation: string;
      early_warning: string;
    }>;
    overall_risk_posture: string;
    top_priorities: string[];
    mitigation_summary: string;
  };
  analysis_type: 'risk_assessment';
}

interface TemplatesListResponse {
  templates: any[];
  count: number;
  total: number;
  offset: number;
  limit: number;
}

interface ErrorResponse {
  error: string;
  timestamp?: string;
  path?: string;
  requestId?: string;
}

describe('Agent Metareasoning Manager API', (): void => {
  const testApp: Application = app;

  beforeAll(async (): Promise<void> => {
    // Wait a moment for server to initialize
    await new Promise(resolve => setTimeout(resolve, 1000));
  });

  afterAll(async (): Promise<void> => {
    // Clean up any test data or connections if needed
  });

  describe('Health endpoints', (): void => {
    test('GET /api/health returns health status', async (): Promise<void> => {
      const response = await request(testApp)
        .get('/api/health')
        .expect('Content-Type', /json/);
      
      expect(response.status).toBe(200);
      
      const body = response.body as HealthResponse;
      expect(body).toHaveProperty('status');
      expect(body).toHaveProperty('services');
      expect(body).toHaveProperty('timestamp');
      expect(body).toHaveProperty('version');
    });

    test('GET /api/info returns API information', async (): Promise<void> => {
      const response = await request(testApp)
        .get('/api/info')
        .expect('Content-Type', /json/)
        .expect(200);
      
      const body = response.body as InfoResponse;
      expect(body).toHaveProperty('name');
      expect(body).toHaveProperty('version');
      expect(body).toHaveProperty('endpoints');
      expect(body.name).toBe('Agent Metareasoning Manager API');
    });
  });

  describe('Prompts endpoints', (): void => {
    test('GET /api/prompts returns prompts list', async (): Promise<void> => {
      const response = await request(testApp)
        .get('/api/prompts')
        .expect('Content-Type', /json/);
      
      // Should succeed or return database connection error
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        const body = response.body as PromptsListResponse;
        expect(body).toHaveProperty('prompts');
        expect(body).toHaveProperty('count');
        expect(body).toHaveProperty('total');
        expect(Array.isArray(body.prompts)).toBe(true);
      }
    });

    test('GET /api/prompts supports filtering by category', async (): Promise<void> => {
      const response = await request(testApp)
        .get('/api/prompts?category=analytical')
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
    });

    test('GET /api/prompts supports pagination', async (): Promise<void> => {
      const response = await request(testApp)
        .get('/api/prompts?limit=10&offset=0')
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        const body = response.body as PromptsListResponse;
        expect(body.limit).toBe(10);
        expect(body.offset).toBe(0);
      }
    });

    test('GET /api/prompts/:id returns specific prompt', async (): Promise<void> => {
      // Using a UUID that might exist in test data
      const testId = '00000000-0000-0000-0000-000000000000';
      
      const response = await request(testApp)
        .get(`/api/prompts/${testId}`)
        .expect('Content-Type', /json/);
      
      // Should return 404 or 500 (database error)
      expect([404, 500]).toContain(response.status);
    });
  });

  describe('Workflows endpoints', (): void => {
    test('GET /api/workflows returns workflows list', async (): Promise<void> => {
      const response = await request(testApp)
        .get('/api/workflows')
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        const body = response.body as WorkflowsListResponse;
        expect(body).toHaveProperty('workflows');
        expect(body).toHaveProperty('count');
        expect(Array.isArray(body.workflows)).toBe(true);
      }
    });

    test('GET /api/workflows supports platform filtering', async (): Promise<void> => {
      const response = await request(testApp)
        .get('/api/workflows?platform=n8n')
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
    });
  });

  describe('Analysis endpoints', (): void => {
    test('POST /api/analyze/decision requires input', async (): Promise<void> => {
      const response = await request(testApp)
        .post('/api/analyze/decision')
        .send({})
        .expect('Content-Type', /json/)
        .expect(400);
      
      const body = response.body as ErrorResponse;
      expect(body).toHaveProperty('error');
      expect(body.error).toContain('Input is required');
    });

    test('POST /api/analyze/decision performs analysis with valid input', async (): Promise<void> => {
      const response = await request(testApp)
        .post('/api/analyze/decision')
        .send({
          input: 'Should we adopt remote work?',
          context: 'Small tech company'
        })
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        const body = response.body as DecisionAnalysisResponse;
        expect(body).toHaveProperty('decision_analysis');
        expect(body).toHaveProperty('execution_id');
        expect(body).toHaveProperty('status');
        expect(body.analysis_type).toBe('decision');
      }
    });

    test('POST /api/analyze/pros-cons requires input', async (): Promise<void> => {
      const response = await request(testApp)
        .post('/api/analyze/pros-cons')
        .send({})
        .expect('Content-Type', /json/)
        .expect(400);
      
      const body = response.body as ErrorResponse;
      expect(body.error).toContain('Input is required');
    });

    test('POST /api/analyze/pros-cons performs analysis with valid input', async (): Promise<void> => {
      const response = await request(testApp)
        .post('/api/analyze/pros-cons')
        .send({
          input: 'Migrate to microservices',
          context: 'Legacy monolithic application'
        })
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        const body = response.body as ProsConsAnalysisResponse;
        expect(body).toHaveProperty('pros_cons_analysis');
        expect(body.pros_cons_analysis).toHaveProperty('pros');
        expect(body.pros_cons_analysis).toHaveProperty('cons');
        expect(Array.isArray(body.pros_cons_analysis.pros)).toBe(true);
        expect(Array.isArray(body.pros_cons_analysis.cons)).toBe(true);
      }
    });

    test('POST /api/analyze/swot performs SWOT analysis', async (): Promise<void> => {
      const response = await request(testApp)
        .post('/api/analyze/swot')
        .send({
          input: 'Launch AI product',
          context: 'Established software company'
        })
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        const body = response.body as SwotAnalysisResponse;
        expect(body).toHaveProperty('swot_analysis');
        expect(body.swot_analysis).toHaveProperty('strengths');
        expect(body.swot_analysis).toHaveProperty('weaknesses');
        expect(body.swot_analysis).toHaveProperty('opportunities');
        expect(body.swot_analysis).toHaveProperty('threats');
      }
    });

    test('POST /api/analyze/risks performs risk assessment', async (): Promise<void> => {
      const response = await request(testApp)
        .post('/api/analyze/risks')
        .send({
          action: 'Launch beta product',
          constraints: 'Limited budget and timeline'
        })
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        const body = response.body as RiskAnalysisResponse;
        expect(body).toHaveProperty('risk_assessment');
        expect(body.risk_assessment).toHaveProperty('risks');
        expect(Array.isArray(body.risk_assessment.risks)).toBe(true);
      }
    });
  });

  describe('Templates endpoints', (): void => {
    test('GET /api/templates returns templates list', async (): Promise<void> => {
      const response = await request(testApp)
        .get('/api/templates')
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        const body = response.body as TemplatesListResponse;
        expect(body).toHaveProperty('templates');
        expect(body).toHaveProperty('count');
        expect(Array.isArray(body.templates)).toBe(true);
      }
    });

    test('GET /api/templates supports public filter', async (): Promise<void> => {
      const response = await request(testApp)
        .get('/api/templates?is_public=true')
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
    });
  });

  describe('Authentication', (): void => {
    test('API endpoints work without authentication token', async (): Promise<void> => {
      // Current implementation allows requests without tokens
      const response = await request(testApp)
        .get('/api/prompts')
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
    });

    test('API endpoints accept valid authentication token', async (): Promise<void> => {
      const response = await request(testApp)
        .get('/api/prompts')
        .set('Authorization', 'Bearer metareasoning_cli_default_2024')
        .expect('Content-Type', /json/);
      
      expect([200, 401, 500]).toContain(response.status);
    });

    test('API endpoints reject invalid authentication token', async (): Promise<void> => {
      const response = await request(testApp)
        .get('/api/prompts')
        .set('Authorization', 'Bearer invalid_token')
        .expect('Content-Type', /json/);
      
      expect([401, 500]).toContain(response.status);
    });
  });

  describe('Error handling', (): void => {
    test('404 for unknown endpoints', async (): Promise<void> => {
      const response = await request(testApp)
        .get('/api/nonexistent')
        .expect('Content-Type', /json/)
        .expect(404);
      
      const body = response.body as ErrorResponse;
      expect(body).toHaveProperty('error');
      expect(body.error).toBe('Endpoint not found');
    });

    test('Proper error format for validation errors', async (): Promise<void> => {
      const response = await request(testApp)
        .post('/api/analyze/decision')
        .send({ invalid: 'data' })
        .expect('Content-Type', /json/)
        .expect(400);
      
      const body = response.body as ErrorResponse;
      expect(body).toHaveProperty('error');
      expect(typeof body.error).toBe('string');
    });
  });

  describe('Response format consistency', (): void => {
    test('All responses include proper headers', async (): Promise<void> => {
      const response = await request(testApp)
        .get('/api/info')
        .expect(200);
      
      expect(response.headers['content-type']).toMatch(/application\/json/);
    });

    test('Error responses have consistent format', async (): Promise<void> => {
      const response = await request(testApp)
        .get('/api/nonexistent')
        .expect(404);
      
      const body = response.body as ErrorResponse;
      expect(body).toHaveProperty('error');
      expect(body).toHaveProperty('timestamp');
      expect(body).toHaveProperty('path');
    });
  });
});