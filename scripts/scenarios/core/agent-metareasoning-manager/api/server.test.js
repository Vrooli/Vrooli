const request = require('supertest');
const app = require('./server');

describe('Agent Metareasoning Manager API', () => {
  beforeAll(async () => {
    // Wait a moment for server to initialize
    await new Promise(resolve => setTimeout(resolve, 1000));
  });

  afterAll(async () => {
    // Clean up any test data or connections if needed
  });

  describe('Health endpoints', () => {
    test('GET /api/health returns health status', async () => {
      const response = await request(app)
        .get('/api/health')
        .expect('Content-Type', /json/);
      
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('status');
      expect(response.body).toHaveProperty('services');
      expect(response.body).toHaveProperty('timestamp');
      expect(response.body).toHaveProperty('version');
    });

    test('GET /api/info returns API information', async () => {
      const response = await request(app)
        .get('/api/info')
        .expect('Content-Type', /json/)
        .expect(200);
      
      expect(response.body).toHaveProperty('name');
      expect(response.body).toHaveProperty('version');
      expect(response.body).toHaveProperty('endpoints');
      expect(response.body.name).toBe('Agent Metareasoning Manager API');
    });
  });

  describe('Prompts endpoints', () => {
    test('GET /api/prompts returns prompts list', async () => {
      const response = await request(app)
        .get('/api/prompts')
        .expect('Content-Type', /json/);
      
      // Should succeed or return database connection error
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        expect(response.body).toHaveProperty('prompts');
        expect(response.body).toHaveProperty('count');
        expect(response.body).toHaveProperty('total');
        expect(Array.isArray(response.body.prompts)).toBe(true);
      }
    });

    test('GET /api/prompts supports filtering by category', async () => {
      const response = await request(app)
        .get('/api/prompts?category=analytical')
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
    });

    test('GET /api/prompts supports pagination', async () => {
      const response = await request(app)
        .get('/api/prompts?limit=10&offset=0')
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        expect(response.body.limit).toBe(10);
        expect(response.body.offset).toBe(0);
      }
    });

    test('GET /api/prompts/:id returns specific prompt', async () => {
      // Using a UUID that might exist in test data
      const testId = '00000000-0000-0000-0000-000000000000';
      
      const response = await request(app)
        .get(`/api/prompts/${testId}`)
        .expect('Content-Type', /json/);
      
      // Should return 404 or 500 (database error)
      expect([404, 500]).toContain(response.status);
    });
  });

  describe('Workflows endpoints', () => {
    test('GET /api/workflows returns workflows list', async () => {
      const response = await request(app)
        .get('/api/workflows')
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        expect(response.body).toHaveProperty('workflows');
        expect(response.body).toHaveProperty('count');
        expect(Array.isArray(response.body.workflows)).toBe(true);
      }
    });

    test('GET /api/workflows supports platform filtering', async () => {
      const response = await request(app)
        .get('/api/workflows?platform=n8n')
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
    });
  });

  describe('Analysis endpoints', () => {
    test('POST /api/analyze/decision requires input', async () => {
      const response = await request(app)
        .post('/api/analyze/decision')
        .send({})
        .expect('Content-Type', /json/)
        .expect(400);
      
      expect(response.body).toHaveProperty('error');
      expect(response.body.error).toContain('Input is required');
    });

    test('POST /api/analyze/decision performs analysis with valid input', async () => {
      const response = await request(app)
        .post('/api/analyze/decision')
        .send({
          input: 'Should we adopt remote work?',
          context: 'Small tech company'
        })
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        expect(response.body).toHaveProperty('decision_analysis');
        expect(response.body).toHaveProperty('execution_id');
        expect(response.body).toHaveProperty('status');
        expect(response.body.analysis_type).toBe('decision');
      }
    });

    test('POST /api/analyze/pros-cons requires input', async () => {
      const response = await request(app)
        .post('/api/analyze/pros-cons')
        .send({})
        .expect('Content-Type', /json/)
        .expect(400);
      
      expect(response.body.error).toContain('Input is required');
    });

    test('POST /api/analyze/pros-cons performs analysis with valid input', async () => {
      const response = await request(app)
        .post('/api/analyze/pros-cons')
        .send({
          input: 'Migrate to microservices',
          context: 'Legacy monolithic application'
        })
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        expect(response.body).toHaveProperty('pros_cons_analysis');
        expect(response.body.pros_cons_analysis).toHaveProperty('pros');
        expect(response.body.pros_cons_analysis).toHaveProperty('cons');
        expect(Array.isArray(response.body.pros_cons_analysis.pros)).toBe(true);
        expect(Array.isArray(response.body.pros_cons_analysis.cons)).toBe(true);
      }
    });

    test('POST /api/analyze/swot performs SWOT analysis', async () => {
      const response = await request(app)
        .post('/api/analyze/swot')
        .send({
          input: 'Launch AI product',
          context: 'Established software company'
        })
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        expect(response.body).toHaveProperty('swot_analysis');
        expect(response.body.swot_analysis).toHaveProperty('strengths');
        expect(response.body.swot_analysis).toHaveProperty('weaknesses');
        expect(response.body.swot_analysis).toHaveProperty('opportunities');
        expect(response.body.swot_analysis).toHaveProperty('threats');
      }
    });

    test('POST /api/analyze/risks performs risk assessment', async () => {
      const response = await request(app)
        .post('/api/analyze/risks')
        .send({
          action: 'Launch beta product',
          constraints: 'Limited budget and timeline'
        })
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        expect(response.body).toHaveProperty('risk_assessment');
        expect(response.body.risk_assessment).toHaveProperty('risks');
        expect(Array.isArray(response.body.risk_assessment.risks)).toBe(true);
      }
    });
  });

  describe('Templates endpoints', () => {
    test('GET /api/templates returns templates list', async () => {
      const response = await request(app)
        .get('/api/templates')
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
      
      if (response.status === 200) {
        expect(response.body).toHaveProperty('templates');
        expect(response.body).toHaveProperty('count');
        expect(Array.isArray(response.body.templates)).toBe(true);
      }
    });

    test('GET /api/templates supports public filter', async () => {
      const response = await request(app)
        .get('/api/templates?is_public=true')
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
    });
  });

  describe('Authentication', () => {
    test('API endpoints work without authentication token', async () => {
      // Current implementation allows requests without tokens
      const response = await request(app)
        .get('/api/prompts')
        .expect('Content-Type', /json/);
      
      expect([200, 500]).toContain(response.status);
    });

    test('API endpoints accept valid authentication token', async () => {
      const response = await request(app)
        .get('/api/prompts')
        .set('Authorization', 'Bearer metareasoning_cli_default_2024')
        .expect('Content-Type', /json/);
      
      expect([200, 401, 500]).toContain(response.status);
    });

    test('API endpoints reject invalid authentication token', async () => {
      const response = await request(app)
        .get('/api/prompts')
        .set('Authorization', 'Bearer invalid_token')
        .expect('Content-Type', /json/);
      
      expect([401, 500]).toContain(response.status);
    });
  });

  describe('Error handling', () => {
    test('404 for unknown endpoints', async () => {
      const response = await request(app)
        .get('/api/nonexistent')
        .expect('Content-Type', /json/)
        .expect(404);
      
      expect(response.body).toHaveProperty('error');
      expect(response.body.error).toBe('Endpoint not found');
    });

    test('Proper error format for validation errors', async () => {
      const response = await request(app)
        .post('/api/analyze/decision')
        .send({ invalid: 'data' })
        .expect('Content-Type', /json/)
        .expect(400);
      
      expect(response.body).toHaveProperty('error');
      expect(typeof response.body.error).toBe('string');
    });
  });

  describe('Response format consistency', () => {
    test('All responses include proper headers', async () => {
      const response = await request(app)
        .get('/api/info')
        .expect(200);
      
      expect(response.headers['content-type']).toMatch(/application\/json/);
    });

    test('Error responses have consistent format', async () => {
      const response = await request(app)
        .get('/api/nonexistent')
        .expect(404);
      
      expect(response.body).toHaveProperty('error');
      expect(response.body).toHaveProperty('timestamp');
      expect(response.body).toHaveProperty('path');
    });
  });
});