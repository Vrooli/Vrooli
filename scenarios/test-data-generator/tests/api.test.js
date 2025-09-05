const request = require('supertest');
const { app } = require('../api/server');

describe('Test Data Generator API', () => {
    describe('Health Check', () => {
        test('GET /health should return healthy status', async () => {
            const response = await request(app)
                .get('/health')
                .expect(200);

            expect(response.body).toMatchObject({
                status: 'healthy',
                service: 'test-data-generator-api'
            });
            expect(response.body.timestamp).toBeDefined();
            expect(response.body.uptime).toBeDefined();
        });
    });

    describe('Data Types', () => {
        test('GET /api/types should return available data types', async () => {
            const response = await request(app)
                .get('/api/types')
                .expect(200);

            expect(response.body.types).toBeDefined();
            expect(Array.isArray(response.body.types)).toBe(true);
            expect(response.body.definitions).toBeDefined();
            expect(response.body.types).toContain('users');
            expect(response.body.types).toContain('companies');
            expect(response.body.types).toContain('products');
        });
    });

    describe('User Data Generation', () => {
        test('POST /api/generate/users should generate user data', async () => {
            const requestBody = {
                count: 5,
                format: 'json',
                fields: ['id', 'name', 'email']
            };

            const response = await request(app)
                .post('/api/generate/users')
                .send(requestBody)
                .expect(200);

            expect(response.body.success).toBe(true);
            expect(response.body.type).toBe('users');
            expect(response.body.count).toBe(5);
            expect(Array.isArray(response.body.data)).toBe(true);
            expect(response.body.data).toHaveLength(5);

            // Check first user has required fields
            const firstUser = response.body.data[0];
            expect(firstUser.id).toBeDefined();
            expect(firstUser.name).toBeDefined();
            expect(firstUser.email).toBeDefined();
        });

        test('POST /api/generate/users with seed should generate consistent data', async () => {
            const requestBody = {
                count: 3,
                format: 'json',
                seed: '12345'
            };

            const response1 = await request(app)
                .post('/api/generate/users')
                .send(requestBody)
                .expect(200);

            const response2 = await request(app)
                .post('/api/generate/users')
                .send(requestBody)
                .expect(200);

            expect(response1.body.data).toEqual(response2.body.data);
        });

        test('POST /api/generate/users with invalid count should return error', async () => {
            const requestBody = {
                count: -1
            };

            await request(app)
                .post('/api/generate/users')
                .send(requestBody)
                .expect(400);
        });

        test('POST /api/generate/users with count over limit should return error', async () => {
            const requestBody = {
                count: 15000
            };

            await request(app)
                .post('/api/generate/users')
                .send(requestBody)
                .expect(400);
        });
    });

    describe('Company Data Generation', () => {
        test('POST /api/generate/companies should generate company data', async () => {
            const requestBody = {
                count: 3,
                format: 'json'
            };

            const response = await request(app)
                .post('/api/generate/companies')
                .send(requestBody)
                .expect(200);

            expect(response.body.success).toBe(true);
            expect(response.body.type).toBe('companies');
            expect(response.body.count).toBe(3);
            expect(Array.isArray(response.body.data)).toBe(true);
            expect(response.body.data).toHaveLength(3);

            const firstCompany = response.body.data[0];
            expect(firstCompany.id).toBeDefined();
            expect(firstCompany.name).toBeDefined();
        });
    });

    describe('Custom Schema Generation', () => {
        test('POST /api/generate/custom should generate data from custom schema', async () => {
            const requestBody = {
                count: 2,
                format: 'json',
                schema: {
                    id: 'uuid',
                    title: 'string',
                    price: 'decimal',
                    active: 'boolean'
                }
            };

            const response = await request(app)
                .post('/api/generate/custom')
                .send(requestBody)
                .expect(200);

            expect(response.body.success).toBe(true);
            expect(response.body.type).toBe('custom');
            expect(response.body.count).toBe(2);
            expect(Array.isArray(response.body.data)).toBe(true);
            expect(response.body.data).toHaveLength(2);

            const firstItem = response.body.data[0];
            expect(firstItem.id).toBeDefined();
            expect(firstItem.title).toBeDefined();
            expect(typeof firstItem.price).toBe('number');
            expect(typeof firstItem.active).toBe('boolean');
        });

        test('POST /api/generate/custom without schema should return error', async () => {
            const requestBody = {
                count: 2,
                format: 'json'
            };

            await request(app)
                .post('/api/generate/custom')
                .send(requestBody)
                .expect(400);
        });
    });

    describe('Format Support', () => {
        test('POST /api/generate/users with XML format', async () => {
            const requestBody = {
                count: 2,
                format: 'xml'
            };

            const response = await request(app)
                .post('/api/generate/users')
                .send(requestBody)
                .expect(200);

            expect(response.body.format).toBe('xml');
            expect(typeof response.body.data).toBe('string');
            expect(response.body.data).toContain('<?xml');
        });

        test('POST /api/generate/users with SQL format', async () => {
            const requestBody = {
                count: 2,
                format: 'sql',
                fields: ['id', 'name', 'email']
            };

            const response = await request(app)
                .post('/api/generate/users')
                .send(requestBody)
                .expect(200);

            expect(response.body.format).toBe('sql');
            expect(typeof response.body.data).toBe('string');
            expect(response.body.data).toContain('INSERT INTO');
            expect(response.body.data).toContain('VALUES');
        });
    });

    describe('Error Handling', () => {
        test('GET /nonexistent should return 404', async () => {
            const response = await request(app)
                .get('/nonexistent')
                .expect(404);

            expect(response.body.success).toBe(false);
            expect(response.body.error).toBe('Endpoint not found');
        });

        test('POST /api/generate/invalid should return 404', async () => {
            await request(app)
                .post('/api/generate/invalid')
                .send({ count: 5 })
                .expect(404);
        });
    });
});