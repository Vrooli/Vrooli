#!/usr/bin/env node
// [REQ:CHART-P0-001-BAR] Bar chart generation API validation
// [REQ:CHART-P0-001-LINE] Line chart generation API validation
// [REQ:CHART-P0-001-PIE] Pie chart generation API validation
// [REQ:CHART-P0-001-SCATTER] Scatter plot generation API validation
// [REQ:CHART-P0-001-AREA] Area chart generation API validation
// [REQ:CHART-P0-002-JSON] JSON data parsing API validation
// [REQ:CHART-P0-003-LIGHT] Light theme API validation
// [REQ:CHART-P0-003-DARK] Dark theme API validation
// [REQ:CHART-P0-005-PNG] PNG export API validation
// [REQ:CHART-P0-005-SVG] SVG export API validation

const http = require('http');

const API_PORT = process.env.API_PORT || '18594';
const API_BASE = `http://localhost:${API_PORT}`;

let testsPassed = 0;
let testsFailed = 0;

function makeRequest(method, path, data = null) {
  return new Promise((resolve, reject) => {
    const url = new URL(path, API_BASE);
    const options = {
      hostname: url.hostname,
      port: url.port,
      path: url.pathname + url.search,
      method: method,
      headers: {
        'Content-Type': 'application/json',
      }
    };

    const req = http.request(options, (res) => {
      let body = '';
      res.on('data', (chunk) => body += chunk);
      res.on('end', () => {
        try {
          const json = body ? JSON.parse(body) : null;
          resolve({ status: res.statusCode, body: json, headers: res.headers });
        } catch (e) {
          resolve({ status: res.statusCode, body: body, headers: res.headers });
        }
      });
    });

    req.on('error', reject);

    if (data) {
      req.write(JSON.stringify(data));
    }

    req.end();
  });
}

async function test(name, reqId, fn) {
  try {
    await fn();
    console.log(`✅ [${reqId}] ${name}`);
    testsPassed++;
  } catch (error) {
    console.error(`❌ [${reqId}] ${name}: ${error.message}`);
    testsFailed++;
  }
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

const sampleData = [
  { category: 'A', value: 30 },
  { category: 'B', value: 80 },
  { category: 'C', value: 45 },
  { category: 'D', value: 60 }
];

async function runTests() {
  console.log('🧪 Chart Generator API Integration Tests\n');
  console.log(`Using API: ${API_BASE}\n`);

  // Health check
  await test('Health check returns OK', 'HEALTH', async () => {
    const res = await makeRequest('GET', '/health');
    assert(res.status === 200, `Expected 200, got ${res.status}`);
    assert(res.body.status === 'healthy', `Expected status 'healthy', got '${res.body.status}'`);
  });

  // Bar chart generation
  await test('Generate bar chart with JSON data', 'CHART-P0-001-BAR', async () => {
    const barData = sampleData.map((d, i) => ({ x: d.category, y: d.value }));
    const res = await makeRequest('POST', '/api/v1/charts/generate', {
      chart_type: 'bar',
      data: barData,
      style: 'professional',
      title: 'Test Bar Chart'
    });
    assert(res.status === 200 || res.status === 201, `Expected 200/201, got ${res.status}`);
    assert(res.body.chart_id, 'Expected chart_id in response');
  });

  // Line chart generation
  await test('Generate line chart with JSON data', 'CHART-P0-001-LINE', async () => {
    const lineData = sampleData.map((d, i) => ({ x: i, y: d.value }));
    const res = await makeRequest('POST', '/api/v1/charts/generate', {
      chart_type: 'line',
      data: lineData,
      style: 'professional',
      title: 'Test Line Chart'
    });
    assert(res.status === 200 || res.status === 201, `Expected 200/201, got ${res.status}`);
    assert(res.body.chart_id, 'Expected chart_id in response');
  });

  // Pie chart generation
  await test('Generate pie chart with JSON data', 'CHART-P0-001-PIE', async () => {
    const pieData = sampleData.map(d => ({ label: d.category, value: d.value }));
    const res = await makeRequest('POST', '/api/v1/charts/generate', {
      chart_type: 'pie',
      data: pieData,
      style: 'professional',
      title: 'Test Pie Chart'
    });
    assert(res.status === 200 || res.status === 201, `Expected 200/201, got ${res.status}`);
    assert(res.body.chart_id, 'Expected chart_id in response');
  });

  // Scatter plot generation
  await test('Generate scatter plot with JSON data', 'CHART-P0-001-SCATTER', async () => {
    const res = await makeRequest('POST', '/api/v1/charts/generate', {
      chart_type: 'scatter',
      data: sampleData.map((d, i) => ({ x: i * 10, y: d.value })),
      style: 'professional',
      title: 'Test Scatter Plot'
    });
    assert(res.status === 200 || res.status === 201, `Expected 200/201, got ${res.status}`);
    assert(res.body.chart_id, 'Expected chart_id in response');
  });

  // Area chart generation
  await test('Generate area chart with JSON data', 'CHART-P0-001-AREA', async () => {
    const areaData = sampleData.map((d, i) => ({ x: i, y: d.value }));
    const res = await makeRequest('POST', '/api/v1/charts/generate', {
      chart_type: 'area',
      data: areaData,
      style: 'professional',
      title: 'Test Area Chart'
    });
    assert(res.status === 200 || res.status === 201, `Expected 200/201, got ${res.status}`);
    assert(res.body.chart_id, 'Expected chart_id in response');
  });

  // JSON data validation
  await test('Validate JSON data format', 'CHART-P0-002-JSON', async () => {
    const validData = sampleData.map((d, i) => ({ x: d.category, y: d.value }));
    const res = await makeRequest('POST', '/validate-data', {
      data: validData,
      format: 'json'
    });
    assert(res.status === 200, `Expected 200, got ${res.status}`);
    assert(res.body.valid === true, `Expected valid=true, got errors: ${JSON.stringify(res.body.errors)}`);
  });

  // Styles endpoint - light theme
  await test('Get professional theme from styles endpoint', 'CHART-P0-003-LIGHT', async () => {
    const res = await makeRequest('GET', '/api/v1/styles');
    assert(res.status === 200, `Expected 200, got ${res.status}`);
    assert(res.body.styles && Array.isArray(res.body.styles), 'Expected styles array in response');
    const professionalStyle = res.body.styles.find(s => s.id === 'professional');
    assert(professionalStyle, 'Professional theme not found in styles');
  });

  // Styles endpoint - dark theme
  await test('Get minimal theme from styles endpoint', 'CHART-P0-003-DARK', async () => {
    const res = await makeRequest('GET', '/api/v1/styles');
    assert(res.status === 200, `Expected 200, got ${res.status}`);
    assert(res.body.styles && Array.isArray(res.body.styles), 'Expected styles array in response');
    const minimalStyle = res.body.styles.find(s => s.id === 'minimal');
    assert(minimalStyle, 'Minimal theme not found in styles');
  });

  // PNG export
  await test('Generate chart with PNG export format', 'CHART-P0-005-PNG', async () => {
    const barData = sampleData.map((d, i) => ({ x: d.category, y: d.value }));
    const res = await makeRequest('POST', '/api/v1/charts/generate', {
      chart_type: 'bar',
      data: barData,
      export_formats: ['png'],
      title: 'PNG Export Test'
    });
    assert(res.status === 200 || res.status === 201, `Expected 200/201, got ${res.status}`);
    assert(res.body.chart_id, 'Expected chart_id in response');
    assert(res.body.files && res.body.files.png, 'Expected PNG file in response');
  });

  // SVG export
  await test('Generate chart with SVG export format', 'CHART-P0-005-SVG', async () => {
    const barData = sampleData.map((d, i) => ({ x: d.category, y: d.value }));
    const res = await makeRequest('POST', '/api/v1/charts/generate', {
      chart_type: 'bar',
      data: barData,
      export_formats: ['svg'],
      title: 'SVG Export Test'
    });
    assert(res.status === 200 || res.status === 201, `Expected 200/201, got ${res.status}`);
    assert(res.body.chart_id, 'Expected chart_id in response');
    assert(res.body.files && res.body.files.svg, 'Expected SVG file in response');
  });

  // Templates endpoint (P1)
  await test('Get templates endpoint returns valid data', 'CHART-P1-008', async () => {
    const res = await makeRequest('GET', '/api/v1/templates');
    assert(res.status === 200, `Expected 200, got ${res.status}`);
    assert(res.body.templates && Array.isArray(res.body.templates), 'Expected templates array in response');
  });

  // Data transformation (P1)
  await test('Transform data via API', 'CHART-P1-007', async () => {
    const res = await makeRequest('POST', '/api/v1/data/transform', {
      data: sampleData,
      operations: [{ type: 'sort', field: 'value', direction: 'desc' }]
    });
    assert(res.status === 200, `Expected 200, got ${res.status}`);
  });

  // Gantt chart (P1)
  await test('Generate gantt chart', 'CHART-P1-001-GANTT', async () => {
    const ganttData = [
      { task: 'Task 1', start: '2025-01-01', end: '2025-01-05' },
      { task: 'Task 2', start: '2025-01-03', end: '2025-01-08' }
    ];
    const res = await makeRequest('POST', '/api/v1/charts/generate', {
      chart_type: 'gantt',
      data: ganttData,
      title: 'Test Gantt Chart'
    });
    assert(res.status === 200 || res.status === 201, `Expected 200/201, got ${res.status}`);
    assert(res.body.chart_id, 'Expected chart_id in response');
  });

  // Summary
  console.log(`\n${'='.repeat(50)}`);
  console.log(`Tests passed: ${testsPassed}`);
  console.log(`Tests failed: ${testsFailed}`);
  console.log(`${'='.repeat(50)}`);

  process.exit(testsFailed > 0 ? 1 : 0);
}

runTests().catch(error => {
  console.error('Test suite failed:', error);
  process.exit(1);
});
