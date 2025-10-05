const http = require('http');
const request = require('supertest');
const { createRequestHandler } = require('../server');

describe('Branch Coverage', () => {
  let app;

  beforeEach(() => {
    app = http.createServer(createRequestHandler());
  });

  describe('Request Handler Branches', () => {
    test('should handle /health path', async () => {
      const response = await request(app).get('/health');
      expect(response.status).toBe(200);
      expect(response.headers['content-type']).toContain('application/json');
      expect(response.body.status).toBe('healthy');
    });

    test('should handle non-health paths (root)', async () => {
      const response = await request(app).get('/');
      expect(response.status).toBe(200);
      expect(response.headers['content-type']).toContain('text/plain');
      expect(response.text).toBe('Simple Test App is running!');
    });

    test('should handle non-health paths (other)', async () => {
      const response = await request(app).get('/api');
      expect(response.status).toBe(200);
      expect(response.headers['content-type']).toContain('text/plain');
      expect(response.text).toBe('Simple Test App is running!');
    });

    test('should handle non-health paths (random)', async () => {
      const response = await request(app).get('/random/path/here');
      expect(response.status).toBe(200);
      expect(response.headers['content-type']).toContain('text/plain');
      expect(response.text).toBe('Simple Test App is running!');
    });
  });
});
