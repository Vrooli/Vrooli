import { parseJsonBody } from '../../../src/middleware/body-parser';
import { createMockHttpRequest, createTestConfig } from '../../helpers';

describe('Body Parser', () => {
  const config = createTestConfig();

  describe('parseJsonBody', () => {
    it('should parse JSON body', async () => {
      const mockReq = createMockHttpRequest({
        headers: { 'content-type': 'application/json' },
        body: { key: 'value' },
      });

      const result = await parseJsonBody(mockReq, config);

      expect(result).toEqual({ key: 'value' });
    });

    it('should handle empty body', async () => {
      const mockReq = createMockHttpRequest({
        headers: { 'content-type': 'application/json' },
      });

      const result = await parseJsonBody(mockReq, config);

      expect(result).toEqual({});
    });

    it('should handle invalid JSON', async () => {
      const mockReq = createMockHttpRequest({
        headers: { 'content-type': 'application/json' },
        body: 'invalid json{',
      });

      await expect(parseJsonBody(mockReq, config)).rejects.toThrow('Invalid JSON');
    });

    it('should enforce size limit', async () => {
      // Create config with very small size limit for testing
      const smallConfig = createTestConfig({
        server: { port: 39400, host: '127.0.0.1', requestTimeout: 300000, maxRequestSize: 100 },
      });
      const largeBody = { data: 'x'.repeat(200) }; // Larger than 100 bytes
      const mockReq = createMockHttpRequest({
        headers: { 'content-type': 'application/json' },
        body: largeBody,
      });

      // Should reject bodies larger than the configured limit
      await expect(parseJsonBody(mockReq, smallConfig)).rejects.toThrow('too large');
    });

    it('should handle string bodies', async () => {
      const mockReq = createMockHttpRequest({
        headers: { 'content-type': 'application/json' },
        body: '{"test": "value"}',
      });

      const result = await parseJsonBody(mockReq, config);

      expect(result).toEqual({ test: 'value' });
    });
  });
});
