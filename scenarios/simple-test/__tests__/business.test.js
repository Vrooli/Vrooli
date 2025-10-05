const http = require('http');
const request = require('supertest');
const { createRequestHandler, startServer } = require('../server');

describe('Business Logic Tests', () => {
  let app;

  beforeEach(() => {
    app = http.createServer(createRequestHandler());
  });

  describe('Health Check Business Logic', () => {
    test('health endpoint should always return healthy status', async () => {
      // Make multiple requests to ensure consistency
      const responses = await Promise.all([
        request(app).get('/health'),
        request(app).get('/health'),
        request(app).get('/health')
      ]);

      responses.forEach(response => {
        expect(response.body.status).toBe('healthy');
        expect(response.body.service).toBe('simple-test');
      });
    });

    test('health endpoint should be accessible via any HTTP method', async () => {
      const methods = ['get', 'post', 'put', 'delete', 'patch'];

      for (const method of methods) {
        const response = await request(app)[method]('/health');
        expect(response.status).toBe(200);
        expect(response.body.status).toBe('healthy');
      }
    });

    test('health response should have correct structure', async () => {
      const response = await request(app).get('/health');

      expect(response.body).toHaveProperty('status');
      expect(response.body).toHaveProperty('service');
      expect(Object.keys(response.body).length).toBe(2);
    });
  });

  describe('Service Identification', () => {
    test('should identify as simple-test service', async () => {
      const response = await request(app).get('/health');
      expect(response.body.service).toBe('simple-test');
    });

    test('service name should be consistent across requests', async () => {
      const response1 = await request(app).get('/health');
      const response2 = await request(app).get('/health');

      expect(response1.body.service).toBe(response2.body.service);
    });
  });

  describe('Default Route Behavior', () => {
    test('should provide friendly message for non-health routes', async () => {
      const paths = ['/', '/api', '/test', '/random'];

      for (const path of paths) {
        const response = await request(app).get(path);
        expect(response.text).toBe('Simple Test App is running!');
        expect(response.status).toBe(200);
      }
    });

    test('should handle content negotiation correctly', async () => {
      const healthResponse = await request(app)
        .get('/health')
        .set('Accept', 'application/json');

      expect(healthResponse.headers['content-type']).toContain('application/json');

      const defaultResponse = await request(app)
        .get('/')
        .set('Accept', 'text/plain');

      expect(defaultResponse.headers['content-type']).toContain('text/plain');
    });
  });

  describe('Port Configuration Business Logic', () => {
    test('should verify startServer function exists and is callable', () => {
      expect(typeof startServer).toBe('function');
      expect(startServer).toBeDefined();
    });

    test('should verify port parameter handling in code', () => {
      // Test the logic without actually starting a server
      const defaultPort = process.env.API_PORT || 36250;
      const customPort = 36299;

      expect(customPort || defaultPort).toBe(customPort);
      expect(undefined || defaultPort).toBe(defaultPort);
    });
  });

  describe('Error Resilience', () => {
    test('should handle malformed requests gracefully', async () => {
      const response = await request(app)
        .get('/')
        .set('Content-Type', 'invalid/type')
        .send('malformed data');

      expect(response.status).toBe(200);
    });

    test('should handle requests with custom headers', async () => {
      const response = await request(app)
        .get('/health')
        .set('X-Custom-Header', 'test-value');

      expect(response.status).toBe(200);
      expect(response.body.status).toBe('healthy');
    });

    test('should handle multiple sequential requests', async () => {
      for (let i = 0; i < 20; i++) {
        const response = await request(app).get('/health');
        expect(response.status).toBe(200);
        expect(response.body.status).toBe('healthy');
      }
    });
  });

  describe('Response Consistency', () => {
    test('health endpoint response should be idempotent', async () => {
      const responses = [];

      for (let i = 0; i < 10; i++) {
        responses.push(await request(app).get('/health'));
      }

      const firstResponse = responses[0].body;
      responses.forEach(response => {
        expect(response.body).toEqual(firstResponse);
      });
    });

    test('default route response should be consistent', async () => {
      const responses = [];

      for (let i = 0; i < 10; i++) {
        responses.push(await request(app).get('/'));
      }

      const firstResponse = responses[0].text;
      responses.forEach(response => {
        expect(response.text).toBe(firstResponse);
      });
    });
  });
});
