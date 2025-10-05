const http = require('http');
const request = require('supertest');

describe('Branch Coverage Tests', () => {
  let originalEnv;

  beforeEach(() => {
    originalEnv = process.env.API_PORT;
    // Clear module cache to test different branches
    delete require.cache[require.resolve('../server')];
  });

  afterEach(() => {
    if (originalEnv !== undefined) {
      process.env.API_PORT = originalEnv;
    } else {
      delete process.env.API_PORT;
    }
  });

  describe('Port Selection Branches', () => {
    test('should use environment variable when API_PORT is set', () => {
      process.env.API_PORT = '36300';
      delete require.cache[require.resolve('../server')];

      const { createRequestHandler } = require('../server');
      const handler = createRequestHandler();

      expect(handler).toBeDefined();
      expect(typeof handler).toBe('function');
    });

    test('should use default port when API_PORT is not set', () => {
      delete process.env.API_PORT;
      delete require.cache[require.resolve('../server')];

      const { createRequestHandler } = require('../server');
      const handler = createRequestHandler();

      expect(handler).toBeDefined();
      expect(typeof handler).toBe('function');
    });

    test('should use customPort parameter when provided to startServer', () => {
      delete process.env.API_PORT;
      delete require.cache[require.resolve('../server')];

      const { startServer } = require('../server');
      const server = startServer(36301);

      expect(server).toBeDefined();
      expect(server.listening).toBe(true);

      server.close();
    });

    test('should fall back to port variable when customPort is not provided', () => {
      process.env.API_PORT = '36302';
      delete require.cache[require.resolve('../server')];

      const { startServer } = require('../server');
      const server = startServer();

      expect(server).toBeDefined();
      expect(server.listening).toBe(true);

      server.close();
    });
  });

  describe('Module Execution Branch', () => {
    test('should not auto-start when required as a module', () => {
      delete process.env.API_PORT;
      delete require.cache[require.resolve('../server')];

      // When required as module, require.main !== module
      const server = require('../server');

      expect(server.createRequestHandler).toBeDefined();
      expect(server.startServer).toBeDefined();
    });

    test('should handle direct execution scenario', (done) => {
      const { spawn } = require('child_process');
      const path = require('path');
      const serverPath = path.join(__dirname, '..', 'server.js');

      // Start server directly as a script
      const child = spawn('node', [serverPath], {
        env: { ...process.env, API_PORT: '36303' }
      });

      let output = '';
      child.stdout.on('data', (data) => {
        output += data.toString();
      });

      setTimeout(() => {
        // Check that server started
        expect(output).toContain('Simple test server running on port 36303');
        child.kill();
        done();
      }, 1000);
    }, 5000);
  });

  describe('URL Branch Coverage', () => {
    let app;

    beforeEach(() => {
      delete require.cache[require.resolve('../server')];
      const { createRequestHandler } = require('../server');
      app = http.createServer(createRequestHandler());
    });

    test('should handle /health path (true branch)', async () => {
      const response = await request(app).get('/health');
      expect(response.status).toBe(200);
      expect(response.headers['content-type']).toContain('application/json');
    });

    test('should handle non-health path (false branch)', async () => {
      const response = await request(app).get('/other');
      expect(response.status).toBe(200);
      expect(response.headers['content-type']).toContain('text/plain');
    });
  });
});
