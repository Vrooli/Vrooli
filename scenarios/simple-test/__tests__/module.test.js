describe('Server Module', () => {
  test('should export createRequestHandler function', () => {
    const { createRequestHandler } = require('../server');
    expect(typeof createRequestHandler).toBe('function');
  });

  test('should export startServer function', () => {
    const { startServer } = require('../server');
    expect(typeof startServer).toBe('function');
  });

  test('createRequestHandler should return a function', () => {
    const { createRequestHandler } = require('../server');
    const handler = createRequestHandler();
    expect(typeof handler).toBe('function');
  });

  test('handler should accept req and res parameters', () => {
    const { createRequestHandler } = require('../server');
    const handler = createRequestHandler();
    expect(handler.length).toBe(2); // Should accept 2 parameters
  });

  test('startServer should return a server instance', () => {
    const { startServer } = require('../server');
    const http = require('http');

    // Start server on unique port for this test only
    const uniquePort = 36299;
    const server = startServer(uniquePort);

    expect(server).toBeDefined();
    expect(server).toBeInstanceOf(http.Server);

    // Clean up
    server.close();
  });

  test('createRequestHandler should be usable with http.createServer', () => {
    const { createRequestHandler } = require('../server');
    const http = require('http');

    const handler = createRequestHandler();
    const server = http.createServer(handler);

    expect(server).toBeInstanceOf(http.Server);

    // Clean up without starting
    server.close();
  });
});
