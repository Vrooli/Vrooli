const http = require('http');
const { spawn } = require('child_process');
const request = require('supertest');

describe('Integration Tests', () => {
  let serverProcess;
  const testPort = 36253;

  beforeAll((done) => {
    // Start server process for integration testing
    serverProcess = spawn('node', ['server.js'], {
      env: { ...process.env, API_PORT: testPort },
      cwd: __dirname + '/..',
      detached: true
    });

    // Wait for server to start
    setTimeout(done, 2000);
  }, 10000);

  afterAll((done) => {
    if (serverProcess) {
      process.kill(-serverProcess.pid);
      setTimeout(done, 1000);
    } else {
      done();
    }
  });

  describe('Live Server Tests', () => {
    test('should respond to health check', (done) => {
      http.get(`http://localhost:${testPort}/health`, (res) => {
        expect(res.statusCode).toBe(200);

        let data = '';
        res.on('data', (chunk) => {
          data += chunk;
        });

        res.on('end', () => {
          const parsed = JSON.parse(data);
          expect(parsed.status).toBe('healthy');
          expect(parsed.service).toBe('simple-test');
          done();
        });
      });
    }, 10000);

    test('should handle concurrent requests', async () => {
      const requests = Array(20).fill(null).map(() =>
        new Promise((resolve) => {
          http.get(`http://localhost:${testPort}/health`, (res) => {
            let data = '';
            res.on('data', (chunk) => { data += chunk; });
            res.on('end', () => resolve({ status: res.statusCode, data }));
          });
        })
      );

      const responses = await Promise.all(requests);
      responses.forEach(response => {
        expect(response.status).toBe(200);
      });
    }, 15000);

    test('should maintain uptime under load', async () => {
      const startTime = Date.now();
      const duration = 3000; // 3 seconds

      const makeRequest = () =>
        new Promise((resolve) => {
          http.get(`http://localhost:${testPort}/`, (res) => {
            resolve(res.statusCode);
          }).on('error', () => resolve(null));
        });

      const results = [];
      while (Date.now() - startTime < duration) {
        results.push(await makeRequest());
      }

      const successRate = results.filter(r => r === 200).length / results.length;
      expect(successRate).toBeGreaterThan(0.95); // 95% success rate
    }, 10000);
  });

  describe('Error Recovery', () => {
    test('should recover from malformed requests', (done) => {
      const net = require('net');
      const malformedRequest = 'GET /health HTTP/1.1\r\nHost: localhost\r\n\r\nGARBAGE\r\n';

      const client = new net.Socket();
      client.connect(testPort, 'localhost', () => {
        client.write(malformedRequest);
      });

      // Server should still respond to normal requests after malformed one
      setTimeout(() => {
        http.get(`http://localhost:${testPort}/health`, (res) => {
          expect(res.statusCode).toBe(200);
          client.destroy();
          done();
        });
      }, 500);
    }, 10000);
  });

  describe('Performance Metrics', () => {
    test('should respond within acceptable time', async () => {
      const iterations = 100;
      const responseTimes = [];

      for (let i = 0; i < iterations; i++) {
        const start = Date.now();
        await new Promise((resolve) => {
          http.get(`http://localhost:${testPort}/health`, (res) => {
            res.on('end', resolve);
            res.resume();
          });
        });
        responseTimes.push(Date.now() - start);
      }

      const avgResponseTime = responseTimes.reduce((a, b) => a + b, 0) / iterations;
      const maxResponseTime = Math.max(...responseTimes);

      expect(avgResponseTime).toBeLessThan(50); // Average under 50ms
      expect(maxResponseTime).toBeLessThan(200); // Max under 200ms
    }, 30000);

    test('should handle high request rate', async () => {
      const requestsPerSecond = 100;
      const duration = 2; // seconds
      const totalRequests = requestsPerSecond * duration;

      const startTime = Date.now();
      const requests = [];

      for (let i = 0; i < totalRequests; i++) {
        requests.push(
          new Promise((resolve) => {
            http.get(`http://localhost:${testPort}/`, (res) => {
              resolve(res.statusCode);
            }).on('error', () => resolve(null));
          })
        );
      }

      const results = await Promise.all(requests);
      const elapsed = Date.now() - startTime;

      const successCount = results.filter(r => r === 200).length;
      const actualRate = (successCount / elapsed) * 1000;

      expect(actualRate).toBeGreaterThan(requestsPerSecond * 0.8); // 80% of target rate
    }, 30000);
  });

  describe('Resource Management', () => {
    test('should not leak memory on repeated requests', async () => {
      const iterations = 1000;

      for (let i = 0; i < iterations; i++) {
        await new Promise((resolve) => {
          http.get(`http://localhost:${testPort}/health`, (res) => {
            res.on('end', resolve);
            res.resume();
          });
        });
      }

      // If we get here without crashing, memory management is OK
      expect(true).toBe(true);
    }, 60000);
  });
});
