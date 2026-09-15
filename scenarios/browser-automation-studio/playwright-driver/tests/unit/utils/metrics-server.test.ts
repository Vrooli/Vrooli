import http from 'http';
import type { AddressInfo } from 'net';
import { closeMetricsServer, createMetricsServer } from '../../../src/utils/metrics-server';
import { metrics } from '../../../src/utils/metrics';

function request(server: http.Server, path: string): Promise<{ statusCode: number; body: string; contentType?: string }> {
  const address = server.address() as AddressInfo | null;
  if (!address) {
    throw new Error('server is not listening');
  }

  return new Promise((resolve, reject) => {
    const req = http.request(
      {
        host: '127.0.0.1',
        port: address.port,
        path,
        method: 'GET',
      },
      (res) => {
        const chunks: Buffer[] = [];
        res.on('data', (chunk: Buffer) => chunks.push(chunk));
        res.on('end', () => {
          resolve({
            statusCode: res.statusCode ?? 0,
            body: Buffer.concat(chunks).toString('utf8'),
            contentType: res.headers['content-type'],
          });
        });
      }
    );
    req.on('error', reject);
    req.end();
  });
}

describe('metrics server', () => {
  const servers: http.Server[] = [];

  afterEach(async () => {
    jest.restoreAllMocks();
    await Promise.all(servers.splice(0).map((server) => closeMetricsServer(server)));
  });

  it('serves Prometheus metrics with the registry content type', async () => {
    const server = await createMetricsServer(0);
    servers.push(server);

    const response = await request(server, '/metrics');

    expect(response.statusCode).toBe(200);
    expect(response.contentType).toContain('text/plain');
    expect(response.body).toContain('playwright_driver_sessions');
  });

  it('returns 404 for non-metrics paths', async () => {
    const server = await createMetricsServer(0);
    servers.push(server);

    const response = await request(server, '/health');

    expect(response.statusCode).toBe(404);
    expect(response.body).toBe('Not found');
  });

  it('returns 500 when metrics collection fails', async () => {
    jest.spyOn(metrics, 'getMetrics').mockRejectedValueOnce(new Error('registry unavailable'));
    const server = await createMetricsServer(0);
    servers.push(server);

    const response = await request(server, '/metrics');

    expect(response.statusCode).toBe(500);
    expect(response.body).toBe('Failed to collect metrics');
  });

  it('rejects with a clear error when the port is already in use', async () => {
    const first = await createMetricsServer(0);
    servers.push(first);
    const address = first.address() as AddressInfo;

    await expect(createMetricsServer(address.port)).rejects.toThrow(
      `Metrics port ${address.port} is already in use`
    );
  });
});
