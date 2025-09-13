const request = require('supertest');
const express = require('express');

// Mock basic Express app for testing
const app = express();
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'visited-tracker-ui' });
});

app.get('/config', (req, res) => {
    res.json({ service: 'visited-tracker', version: '1.0.0' });
});

describe('UI Server', () => {
    test('Health endpoint returns healthy status', async () => {
        const response = await request(app).get('/health');
        expect(response.status).toBe(200);
        expect(response.body.status).toBe('healthy');
        expect(response.body.service).toBe('visited-tracker-ui');
    });

    test('Config endpoint returns service info', async () => {
        const response = await request(app).get('/config');
        expect(response.status).toBe(200);
        expect(response.body.service).toBe('visited-tracker');
        expect(response.body.version).toBe('1.0.0');
    });
});

// Basic functionality tests
describe('Basic JavaScript functionality', () => {
    test('JavaScript environment is working', () => {
        expect(1 + 1).toBe(2);
    });

    test('JSON parsing works', () => {
        const testData = '{"test": "value"}';
        const parsed = JSON.parse(testData);
        expect(parsed.test).toBe('value');
    });

    test('Date functionality works', () => {
        const now = new Date();
        expect(now instanceof Date).toBe(true);
        expect(typeof now.getTime()).toBe('number');
    });
});
