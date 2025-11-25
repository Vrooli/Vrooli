const http = require('http');
const request = require('supertest');

const { startServer, createApp } = require('../server');

const LOOPBACK_HOST = process.env.VROOLI_LOOPBACK_HOST || '127.0.0.1';
const LOOPBACK_ORIGIN = `http://${LOOPBACK_HOST}`;

// [REQ:VT-REQ-009] Web Interface Dashboard - UI server functionality tests
describe('Visited Tracker UI Server', () => {
    let server;
    let agent;
    const originalApiPort = process.env.API_PORT;
    const testApiPort = '19999';

    beforeAll(() => {
        process.env.API_PORT = testApiPort;
    });

    afterAll(() => {
        if (typeof originalApiPort === 'undefined') {
            delete process.env.API_PORT;
        } else {
            process.env.API_PORT = originalApiPort;
        }
    });

    beforeEach(() => {
        const app = createApp({ apiPort: testApiPort });
        agent = request(app);
        server = undefined;
    });

    afterEach(async () => {
        if (server) {
            await new Promise((resolve) => server.close(resolve));
            server = undefined;
        }
    });

    test('config endpoint exposes expected metadata', async () => {
        const response = await agent.get('/config');
        expect(response.status).toBe(200);
        expect(response.body).toMatchObject({
            apiUrl: `${LOOPBACK_ORIGIN}:${testApiPort}`,
            directApiBase: `${LOOPBACK_ORIGIN}:${testApiPort}/api/v1`,
            proxyApiUrl: '/proxy',
            proxyApiBase: '/proxy/api/v1',
            proxyHealthUrl: '/proxy/health',
            displayApiUrl: `${LOOPBACK_ORIGIN}:${testApiPort}`,
            version: '1.0.0',
            service: 'visited-tracker'
        });
    });

    test('health endpoint returns structured payload even when API is offline', async () => {
        const response = await agent.get('/health');
        expect(response.status).toBe(200);

        const payload = response.body;
        expect(payload).toHaveProperty('status');
        expect(payload).toHaveProperty('service', 'visited-tracker-ui');
        expect(payload).toHaveProperty('timestamp');
        expect(payload).toHaveProperty('readiness', true);
        expect(payload).toHaveProperty('api_connectivity');

        const apiConnectivity = payload.api_connectivity;
        expect(apiConnectivity).toHaveProperty('api_url', `${LOOPBACK_ORIGIN}:${testApiPort}`);
        expect(apiConnectivity).toHaveProperty('connected');
        expect(apiConnectivity).toHaveProperty('last_check');
        expect(apiConnectivity).toHaveProperty('latency_ms');
    });

    test('health endpoint reports connected status when API responds', async () => {
        const mockApi = http.createServer((_req, res) => {
            res.writeHead(200, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({ status: 'healthy' }));
        });

        const mockPort = await new Promise((resolve) => mockApi.listen(0, () => {
            const address = mockApi.address();
            resolve(typeof address === 'object' && address ? address.port : apiPort);
        }));

        const connectedApp = createApp({ apiPort: mockPort });
        const connectedAgent = request(connectedApp);
        const response = await connectedAgent.get('/health');
        await new Promise((resolve) => mockApi.close(resolve));

        expect(response.status).toBe(200);
        expect(response.body.api_connectivity.connected).toBe(true);
        expect(['healthy', 'degraded']).toContain(response.body.status);
    });

    test('health endpoint reports degraded status when API port missing', async () => {
        const previous = process.env.API_PORT;
        delete process.env.API_PORT;
        const appWithoutConfig = createApp({});
        const response = await request(appWithoutConfig).get('/health');
        if (typeof previous === 'undefined') {
            delete process.env.API_PORT;
        } else {
            process.env.API_PORT = previous;
        }

        expect(response.status).toBe(200);
        expect(response.body.status).toBe('degraded');
        expect(response.body.api_connectivity.error).toMatchObject({
            code: 'MISSING_CONFIG',
            category: 'configuration'
        });
    });

    test('health endpoint surfaces HTTP error responses from API', async () => {
        const mockApi = http.createServer((_req, res) => {
            res.writeHead(503, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({ status: 'unavailable' }));
        });

        const mockPort = await new Promise((resolve) => mockApi.listen(0, () => {
            const address = mockApi.address();
            resolve(typeof address === 'object' && address ? address.port : testApiPort);
        }));

        const degradedApp = createApp({ apiPort: mockPort });
        const response = await request(degradedApp).get('/health');
        await new Promise((resolve) => mockApi.close(resolve));

        expect(response.status).toBe(200);
        expect(response.body.status).toBe('degraded');
        expect(response.body.api_connectivity.error).toMatchObject({
            code: 'HTTP_503',
            category: 'network'
        });
    });

    test('health endpoint handles unreachable API hosts', async () => {
        const unreachablePort = 29999; // no server bound
        const unreachableApp = createApp({ apiPort: unreachablePort });
        const response = await request(unreachableApp).get('/health');

        expect(response.status).toBe(200);
        expect(response.body.api_connectivity.connected).toBe(false);
        expect(['unhealthy', 'degraded']).toContain(response.body.status);
        expect(response.body.api_connectivity.error).toBeTruthy();
    });

    test('serves static files when they exist', async () => {
        const response = await agent.get('/bridge-init.js');
        expect(response.status).toBe(200);
        expect(response.headers['content-type']).toMatch(/javascript|application\/javascript|text\/javascript/);
        expect(response.text).toContain('iframe');
    });

    test('serves docs content for root and namespaced paths', async () => {
        const direct = await agent.get('/docs-content');
        expect(direct.status).toBe(200);
        expect(direct.headers['content-type']).toMatch(/markdown|text\/markdown/);
        expect(direct.text).toContain('# Visited Tracker');

        const namespaced = await agent.get('/assets/sample-scope/docs-content');
        expect(namespaced.status).toBe(200);
        expect(namespaced.text).toContain('# Visited Tracker');
    });

    test('docs content endpoint rejects unsupported methods', async () => {
        const response = await agent.post('/docs-content');
        expect(response.status).toBe(405);
        expect(response.headers['allow']).toBe('GET, HEAD');
    });

    test('root path serves index.html', async () => {
        const response = await agent.get('/');
        expect(response.status).toBe(200);
        expect(response.headers['content-type']).toMatch(/html/);
        expect(response.text).toContain('<!DOCTYPE html>');
    });

    test('startServer launches and can be stopped cleanly', async () => {
        const started = startServer({ uiPort: 0, apiPort: testApiPort });
        server = started.server;
        expect(server.listening).toBe(true);
    });

    test('health endpoint handles API timeout correctly', async () => {
        // Create a mock API server that never responds (simulates timeout)
        const mockApi = http.createServer((req, res) => {
            // Intentionally don't respond to simulate timeout
            // The server just hangs indefinitely
            req.on('close', () => {
                // Clean up when connection is destroyed
                res.end();
            });
        });

        const mockPort = await new Promise((resolve) => mockApi.listen(0, () => {
            const address = mockApi.address();
            resolve(typeof address === 'object' && address ? address.port : testApiPort);
        }));

        const timeoutApp = createApp({ apiPort: mockPort });
        const response = await request(timeoutApp).get('/health').timeout(7000); // Allow extra time for the 5s timeout
        
        await new Promise((resolve) => mockApi.close(resolve));

        expect(response.status).toBe(200);
        expect(response.body.status).toBe('unhealthy');
        expect(response.body.api_connectivity.connected).toBe(false);
        expect(response.body.api_connectivity.error).toMatchObject({
            code: 'TIMEOUT',
            message: 'API health check timed out after 5 seconds',
            category: 'network',
            retryable: true
        });
        expect(response.body.api_connectivity.latency_ms).toBeGreaterThanOrEqual(5000);
    }, 10000); // Increase test timeout to handle the 5s API timeout

    test('health endpoint handles unexpected errors gracefully', async () => {
        // Mock http.request to throw an error only for the health check
        const originalRequest = http.request;
        let callCount = 0;
        
        // Override http.request to throw on the health check call only
        http.request = jest.fn((options, callback) => {
            callCount++;
            // First call is from supertest itself, second is our health check
            if (callCount > 1 && options.path === '/health') {
                throw new Error('Simulated unexpected error');
            }
            return originalRequest(options, callback);
        });

        const errorApp = createApp({ apiPort: testApiPort });
        const response = await request(errorApp).get('/health');

        // Restore original http.request
        http.request = originalRequest;

        expect(response.status).toBe(200);
        expect(response.body.status).toBe('unhealthy');
        expect(response.body.api_connectivity.connected).toBe(false);
        expect(response.body.api_connectivity.error).toMatchObject({
            code: 'UNEXPECTED_ERROR',
            message: 'Unexpected error: Simulated unexpected error',
            category: 'internal',
            retryable: true
        });
    });

    test('health endpoint handles network errors with proper categorization', async () => {
        // Test ENOTFOUND error
        const originalRequest = http.request;
        let callCount = 0;
        
        http.request = jest.fn((options, callback) => {
            callCount++;
            // Only mock the health check call, not supertest's call
            if (callCount > 1 && options.path === '/health') {
                // Create a mock request object with proper event emitter behavior
                const mockReq = {
                    on: jest.fn(function(event, handler) {
                        if (event === 'error') {
                            // Emit error after a tick
                            setImmediate(() => {
                                const error = new Error('getaddrinfo ENOTFOUND localhost');
                                error.code = 'ENOTFOUND';
                                handler(error);
                            });
                        }
                        return this;
                    }),
                    end: jest.fn(),
                    destroy: jest.fn()
                };
                return mockReq;
            }
            return originalRequest(options, callback);
        });

        const notFoundApp = createApp({ apiPort: testApiPort });
        const response = await request(notFoundApp).get('/health');
        
        http.request = originalRequest;

        expect(response.status).toBe(200);
        expect(response.body.status).toBe('degraded');
        expect(response.body.api_connectivity.error).toMatchObject({
            code: 'HOST_NOT_FOUND',
            category: 'configuration',
            retryable: false
        });
    });

    test('health endpoint handles ETIMEOUT error correctly', async () => {
        const originalRequest = http.request;
        let callCount = 0;
        
        http.request = jest.fn((options, callback) => {
            callCount++;
            // Only mock the health check call, not supertest's call
            if (callCount > 1 && options.path === '/health') {
                // Create a mock request object with proper event emitter behavior
                const mockReq = {
                    on: jest.fn(function(event, handler) {
                        if (event === 'error') {
                            // Emit ETIMEOUT error after a tick
                            setImmediate(() => {
                                const error = new Error('connect ETIMEDOUT');
                                error.code = 'ETIMEOUT';
                                handler(error);
                            });
                        }
                        return this;
                    }),
                    end: jest.fn(),
                    destroy: jest.fn()
                };
                return mockReq;
            }
            return originalRequest(options, callback);
        });

        const timeoutApp = createApp({ apiPort: testApiPort });
        const response = await request(timeoutApp).get('/health');
        
        http.request = originalRequest;

        expect(response.status).toBe(200);
        expect(response.body.status).toBe('unhealthy');
        expect(response.body.api_connectivity.error).toMatchObject({
            code: 'TIMEOUT',
            category: 'network',
            retryable: true
        });
    });

    test('startServer logs correct messages to console', async () => {
        const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
        
        const started = startServer({ uiPort: 0, apiPort: testApiPort });
        server = started.server;

        // Wait for server to be listening
        await new Promise(resolve => setTimeout(resolve, 100));

        expect(consoleSpy).toHaveBeenCalledWith(expect.stringMatching(/🚀 Visited Tracker Dashboard running on/));
        expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining(`📊 API: ${LOOPBACK_ORIGIN}:`));
        
        consoleSpy.mockRestore();
    });

    test('proxy forwards API responses for GET requests', async () => {
        let receivedHost = '';
        const mockApi = http.createServer((req, res) => {
            receivedHost = req.headers.host || '';
            if (req.url === '/api/v1/status' && req.method === 'GET') {
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ ok: true, path: req.url, host: receivedHost }));
            } else {
                res.writeHead(404);
                res.end();
            }
        });

        const mockPort = await new Promise((resolve) => mockApi.listen(0, () => {
            const address = mockApi.address();
            resolve(typeof address === 'object' && address ? address.port : testApiPort);
        }));

        const proxiedApp = createApp({ apiPort: mockPort });
        const proxyResponse = await request(proxiedApp).get('/proxy/api/v1/status');

        await new Promise((resolve) => mockApi.close(resolve));

        expect(proxyResponse.status).toBe(200);
        expect(proxyResponse.body).toMatchObject({ ok: true, path: '/api/v1/status' });
        expect(proxyResponse.body.host).toBe(`${LOOPBACK_HOST}:${mockPort}`);
    });

    test('proxy forwards request bodies for non-GET methods', async () => {
        const payload = { message: 'hello proxy' };
        let recordedBody = null;
        const mockApi = http.createServer((req, res) => {
            if (req.url === '/api/v1/echo' && req.method === 'POST') {
                let body = '';
                req.setEncoding('utf8');
                req.on('data', chunk => { body += chunk; });
                req.on('end', () => {
                    recordedBody = body ? JSON.parse(body) : null;
                    res.writeHead(201, { 'Content-Type': 'application/json' });
                    res.end(JSON.stringify({ ok: true, body: recordedBody }));
                });
            } else {
                res.writeHead(404);
                res.end();
            }
        });

        const mockPort = await new Promise((resolve) => mockApi.listen(0, () => {
            const address = mockApi.address();
            resolve(typeof address === 'object' && address ? address.port : testApiPort);
        }));

        const proxiedApp = createApp({ apiPort: mockPort });
        const proxyResponse = await request(proxiedApp)
            .post('/proxy/api/v1/echo')
            .send(payload)
            .set('Content-Type', 'application/json');

        await new Promise((resolve) => mockApi.close(resolve));

        expect(proxyResponse.status).toBe(201);
        expect(proxyResponse.body).toMatchObject({ ok: true, body: payload });
        expect(recordedBody).toEqual(payload);
    });

    test('health endpoint handles request.destroy() during timeout', async () => {
        // Create a mock that simulates the timeout event flow
        const originalRequest = http.request;
        let callCount = 0;
        let mockReq;
        
        http.request = jest.fn((options, callback) => {
            callCount++;
            // Only mock the health check call, not supertest's call
            if (callCount > 1 && options.path === '/health') {
                mockReq = {
                    on: jest.fn(function(event, handler) {
                        if (event === 'timeout') {
                            // Simulate timeout happening immediately
                            setImmediate(() => handler());
                        }
                        return this;
                    }),
                    end: jest.fn(),
                    destroy: jest.fn()
                };
                return mockReq;
            }
            return originalRequest(options, callback);
        });

        const app = createApp({ apiPort: testApiPort });
        const response = await request(app).get('/health');
        
        http.request = originalRequest;

        expect(response.status).toBe(200);
        expect(response.body.status).toBe('unhealthy');
        expect(response.body.api_connectivity.error.code).toBe('TIMEOUT');
        expect(mockReq.destroy).toHaveBeenCalled();
    });
});
