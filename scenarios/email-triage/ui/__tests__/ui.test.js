/**
 * Email Triage UI Automation Tests
 *
 * Tests core UI functionality:
 * - Page loads correctly
 * - Tab navigation works
 * - Forms render properly
 * - API connectivity UI elements present
 */

const http = require('http');

describe('Email Triage UI', () => {
    const UI_PORT = process.env.UI_PORT || '36114';
    const API_PORT = process.env.API_PORT || '19525';
    const UI_URL = `http://localhost:${UI_PORT}`;

    // Helper to make HTTP GET requests
    function httpGet(url) {
        return new Promise((resolve, reject) => {
            http.get(url, (res) => {
                let data = '';
                res.on('data', (chunk) => { data += chunk; });
                res.on('end', () => resolve({ status: res.statusCode, data }));
            }).on('error', reject);
        });
    }

    describe('Page Loading', () => {
        test('UI server responds with HTML', async () => {
            const response = await httpGet(UI_URL);
            expect(response.status).toBe(200);
            expect(response.data).toContain('<!DOCTYPE html>');
            expect(response.data).toContain('Email Triage');
        });

        test('index.html contains application title', async () => {
            const response = await httpGet(UI_URL);
            expect(response.data).toContain('<title>Email Triage - AI-Powered Email Management</title>');
        });

        test('UI loads main JavaScript file', async () => {
            const response = await httpGet(UI_URL);
            expect(response.data).toContain('src/main.js');
        });
    });

    describe('Core UI Elements', () => {
        test('Dashboard has all required tabs', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            // Check for all 5 tabs
            expect(html).toContain('Dashboard');
            expect(html).toContain('Email Accounts');
            expect(html).toContain('Triage Rules');
            expect(html).toContain('Search');
            expect(html).toContain('Settings');
        });

        test('Dashboard has stat cards', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain('Emails Processed');
            expect(html).toContain('Active Rules');
            expect(html).toContain('Actions Automated');
            expect(html).toContain('Avg Process Time');
        });

        test('Email accounts tab has add account form', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain('+ Add Email Account');
            expect(html).toContain('id="email-address"');
            expect(html).toContain('id="email-password"');
        });

        test('Rules tab has rule creation form', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain('+ Create New Rule');
            expect(html).toContain('id="rule-description"');
            expect(html).toContain('Generate Rule with AI');
        });

        test('Search tab has semantic search interface', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain('Semantic Email Search');
            expect(html).toContain('id="search-query"');
            expect(html).toContain('Search by meaning');
        });

        test('Settings tab shows API configuration', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain('System Status');
            expect(html).toContain('API Base URL');
            expect(html).toContain('Plan Type');
        });
    });

    describe('Business Model Elements', () => {
        test('UI displays pricing tiers', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain('Free');
            expect(html).toContain('Pro');
            expect(html).toContain('Business');
            expect(html).toContain('$29');
            expect(html).toContain('$99');
        });

        test('UI shows revenue potential messaging', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain('$20K-50K');
            expect(html).toContain('Multi-tenant SaaS');
        });
    });

    describe('Accessibility', () => {
        test('HTML has proper language attribute', async () => {
            const response = await httpGet(UI_URL);
            expect(response.data).toContain('<html lang="en">');
        });

        test('Page has proper viewport meta tag', async () => {
            const response = await httpGet(UI_URL);
            expect(response.data).toContain('<meta name="viewport"');
        });

        test('Form inputs have labels', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain('<label class="form-label">Email Address</label>');
            expect(html).toContain('<label class="form-label">Password</label>');
        });

        test('Page has semantic HTML structure', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain('<h1>');
            expect(html).toContain('<h2>');
            expect(html).toContain('<button');
        });
    });

    describe('JavaScript Functionality', () => {
        test('main.js is accessible', async () => {
            const response = await httpGet(`${UI_URL}/src/main.js`);
            expect(response.status).toBe(200);
            expect(response.data).toContain('getApiBaseUrl');
        });

        test('main.js contains API integration functions', async () => {
            const response = await httpGet(`${UI_URL}/src/main.js`);
            const js = response.data;

            expect(js).toContain('apiRequest');
            expect(js).toContain('API_BASE_URL');
        });

        test('main.js has tab navigation logic', async () => {
            const response = await httpGet(`${UI_URL}/src/main.js`);
            const js = response.data;

            expect(js).toContain('showTab');
            expect(js).toContain('currentTab');
        });

        test('main.js has form handlers', async () => {
            const response = await httpGet(`${UI_URL}/src/main.js`);
            const js = response.data;

            expect(js).toContain('addEmailAccount');
            expect(js).toContain('createTriageRule');
            expect(js).toContain('searchEmails');
        });
    });

    describe('UI Health Check', () => {
        test('health endpoint exists and responds', async () => {
            const response = await httpGet(`${UI_URL}/health`);
            expect(response.status).toBe(200);
        });

        test('health endpoint returns valid JSON or OK status', async () => {
            const response = await httpGet(`${UI_URL}/health`);
            // Can be either JSON with status or simple OK text
            const isValidHealth = response.data.includes('ok') ||
                                  response.data.includes('healthy') ||
                                  response.data.includes('"status"');
            expect(isValidHealth).toBe(true);
        });
    });

    describe('Responsive Design', () => {
        test('CSS includes media queries for mobile', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain('@media (max-width: 768px)');
        });

        test('CSS has responsive grid layout', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain('grid-template-columns');
            expect(html).toContain('flex');
        });
    });

    describe('Visual Design', () => {
        test('UI uses purple gradient branding', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain('#667eea');
            expect(html).toContain('#764ba2');
            expect(html).toContain('linear-gradient');
        });

        test('UI has professional styling', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain('border-radius');
            expect(html).toContain('box-shadow');
            expect(html).toContain('transition');
        });
    });

    describe('Integration with Vrooli', () => {
        test('UI includes iframe-bridge integration', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain('iframe-bridge');
            expect(html).toContain('initIframeBridgeChild');
        });

        test('iframe-bridge has proper configuration', async () => {
            const response = await httpGet(UI_URL);
            const html = response.data;

            expect(html).toContain("appId: 'email-triage'");
            expect(html).toContain('parentOrigin');
        });
    });
});
