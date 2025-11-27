import { parseBody } from '../../../src/middleware/body-parser';
import { createMockRequest } from '../../helpers';

describe('Body Parser', () => {
  describe('parseBody', () => {
    it('should parse JSON body', async () => {
      const mockReq = createMockRequest({
        headers: { 'content-type': 'application/json' },
        body: { key: 'value' },
      });

      const result = await parseBody(mockReq);

      expect(result).toEqual({ key: 'value' });
    });

    it('should handle empty body', async () => {
      const mockReq = createMockRequest({
        headers: { 'content-type': 'application/json' },
      });

      const result = await parseBody(mockReq);

      expect(result).toEqual({});
    });

    it('should handle invalid JSON', async () => {
      const mockReq = createMockRequest({
        headers: { 'content-type': 'application/json' },
        body: 'invalid json{',
      });

      await expect(parseBody(mockReq)).rejects.toThrow();
    });

    it('should enforce size limit', async () => {
      const largeBody = { data: 'x'.repeat(20 * 1024 * 1024) }; // 20MB
      const mockReq = createMockRequest({
        headers: { 'content-type': 'application/json' },
        body: largeBody,
      });

      // Should reject bodies larger than 10MB (default limit)
      await expect(parseBody(mockReq)).rejects.toThrow();
    });

    it('should handle string bodies', async () => {
      const mockReq = createMockRequest({
        headers: { 'content-type': 'application/json' },
        body: '{"test": "value"}',
      });

      const result = await parseBody(mockReq);

      expect(result).toEqual({ test: 'value' });
    });
  });
});
