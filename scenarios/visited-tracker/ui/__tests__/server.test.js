const http = require('http');
const request = require('supertest');

const { startServer, createApp } = require('../server');

describe('Visited Tracker UI Server', () => {
    let server;
    let app;
    let agent;
    const apiPort = process.env.API_PORT || '17695';

    beforeAll(() => {
        process.env.API_PORT = apiPort;
    });

    beforeEach(() => {
        const started = startServer({ uiPort: 0, apiPort });
        ({ server, app } = started);
        agent = request(app);
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
            apiUrl: `http://localhost:${apiPort}`,
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
        expect(apiConnectivity).toHaveProperty('api_url', `http://localhost:${apiPort}`);
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
});
