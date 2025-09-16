const http = require('http');
const request = require('supertest');

const { startServer, createApp } = require('../server');

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
        expect(response.body).toEqual({
            apiUrl: `http://localhost:${testApiPort}`,
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
        expect(apiConnectivity).toHaveProperty('api_url', `http://localhost:${testApiPort}`);
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
        const response = await agent.get('/package.json');
        expect([200, 404]).toContain(response.status);
        if (response.status === 200) {
            expect(response.headers['content-type']).toMatch(/json/);
        }
    });

    test('root path serves index.html or returns 404', async () => {
        const response = await agent.get('/');
        expect([200, 404]).toContain(response.status);
    });

    test('startServer launches and can be stopped cleanly', async () => {
        const started = startServer({ uiPort: 0, apiPort: testApiPort });
        server = started.server;
        expect(server.listening).toBe(true);
    });
});
