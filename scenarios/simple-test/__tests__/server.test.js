const http = require('http');
const request = require('supertest');
const { createRequestHandler } = require('../server');

describe('Simple Test Server', () => {
  let server;
  let app;

  beforeEach(() => {
    // Create server instance for testing using the actual handler
    app = http.createServer(createRequestHandler());
  });

  afterEach((done) => {
    if (server && server.listening) {
      server.close(done);
    } else {
      done();
    }
  });

  describe('Health Endpoint', () => {
    test('should return 200 status code', async () => {
      const response = await request(app).get('/health');
      expect(response.status).toBe(200);
    });

    test('should return JSON content type', async () => {
      const response = await request(app).get('/health');
      expect(response.headers['content-type']).toContain('application/json');
    });

    test('should return healthy status', async () => {
      const response = await request(app).get('/health');
      expect(response.body).toEqual({
        status: 'healthy',
        service: 'simple-test'
      });
    });

    test('should return proper JSON structure', async () => {
      const response = await request(app).get('/health');
      expect(response.body).toHaveProperty('status');
      expect(response.body).toHaveProperty('service');
    });
  });

  describe('Root Endpoint', () => {
    test('should return 200 status code', async () => {
      const response = await request(app).get('/');
      expect(response.status).toBe(200);
    });

    test('should return text/plain content type', async () => {
      const response = await request(app).get('/');
      expect(response.headers['content-type']).toContain('text/plain');
    });

    test('should return running message', async () => {
      const response = await request(app).get('/');
      expect(response.text).toBe('Simple Test App is running!');
    });
  });

  describe('Unknown Endpoints', () => {
    test('should handle /api endpoint', async () => {
      const response = await request(app).get('/api');
      expect(response.status).toBe(200);
      expect(response.text).toBe('Simple Test App is running!');
    });

    test('should handle /test endpoint', async () => {
      const response = await request(app).get('/test');
      expect(response.status).toBe(200);
      expect(response.text).toBe('Simple Test App is running!');
    });

    test('should handle /status endpoint', async () => {
      const response = await request(app).get('/status');
      expect(response.status).toBe(200);
    });
  });

  describe('HTTP Methods', () => {
    test('should handle POST requests to health', async () => {
      const response = await request(app).post('/health');
      expect(response.status).toBe(200);
      expect(response.body).toEqual({
        status: 'healthy',
        service: 'simple-test'
      });
    });

    test('should handle PUT requests', async () => {
      const response = await request(app).put('/');
      expect(response.status).toBe(200);
    });

    test('should handle DELETE requests', async () => {
      const response = await request(app).delete('/test');
      expect(response.status).toBe(200);
    });
  });

  describe('Server Lifecycle', () => {
    test('should start server on configured port', (done) => {
      const port = 36250;
      server = app.listen(port, () => {
        expect(server.listening).toBe(true);
        expect(server.address().port).toBe(port);
        done();
      });
    });

    test('should use environment variable for port', (done) => {
      const testPort = 36251;
      process.env.API_PORT = testPort;

      server = app.listen(process.env.API_PORT, () => {
        expect(server.address().port).toBe(testPort);
        delete process.env.API_PORT;
        done();
      });
    });

    test('should handle server close gracefully', (done) => {
      server = app.listen(36252, () => {
        server.close(() => {
          expect(server.listening).toBe(false);
          done();
        });
      });
    });
  });

  describe('Error Handling', () => {
    test('should handle successive requests', async () => {
      const response1 = await request(app).get('/health');
      const response2 = await request(app).get('/health');
      const response3 = await request(app).get('/health');

      expect(response1.status).toBe(200);
      expect(response2.status).toBe(200);
      expect(response3.status).toBe(200);
    });

    test('should handle requests with query parameters', async () => {
      const response = await request(app).get('/health?test=1&foo=bar');
      expect(response.status).toBe(200);
    });

    test('should handle requests with headers', async () => {
      const response = await request(app)
        .get('/health')
        .set('X-Custom-Header', 'test-value')
        .set('Authorization', 'Bearer token');

      expect(response.status).toBe(200);
    });
  });

  describe('Response Validation', () => {
    test('should return valid JSON for health endpoint', async () => {
      const response = await request(app).get('/health');
      expect(() => JSON.parse(response.text)).not.toThrow();
    });

    test('should have consistent response format', async () => {
      const response1 = await request(app).get('/health');
      const response2 = await request(app).get('/health');

      expect(response1.body).toEqual(response2.body);
    });

    test('should include service identifier in health response', async () => {
      const response = await request(app).get('/health');
      expect(response.body.service).toBe('simple-test');
    });
  });

  describe('Edge Cases', () => {
    test('should handle empty URL path', async () => {
      const response = await request(app).get('');
      expect(response.status).toBe(200);
    });

    test('should handle deeply nested paths', async () => {
      const response = await request(app).get('/api/v1/test/deep/nested/path');
      expect(response.status).toBe(200);
    });

    test('should handle special characters in URL', async () => {
      const response = await request(app).get('/test%20path');
      expect(response.status).toBe(200);
    });

    test('should handle URL with fragment', async () => {
      const response = await request(app).get('/test#fragment');
      expect(response.status).toBe(200);
    });
  });

  describe('Server Initialization', () => {
    const { startServer } = require('../server');

    test('should start server with custom port', (done) => {
      const testPort = 36255;
      const testServer = startServer(testPort);

      setTimeout(() => {
        expect(testServer.listening).toBe(true);
        expect(testServer.address().port).toBe(testPort);
        testServer.close(done);
      }, 500);
    });
  });
});
