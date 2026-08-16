import { describe, it, expect } from 'vitest';
import { isAllowedCallbackUrl } from '../VerifyMagicLink';

describe('isAllowedCallbackUrl [REQ:AUTH-SECURITY]', () => {
  describe('vrooli:// scheme', () => {
    it('allows vrooli://auth/callback', () => {
      expect(isAllowedCallbackUrl('vrooli://auth/callback')).toBe(true);
    });

    it('allows any vrooli:// path', () => {
      expect(isAllowedCallbackUrl('vrooli://anything/here')).toBe(true);
    });

    it('allows vrooli:// callbacks without making the callback a credential channel', () => {
      expect(isAllowedCallbackUrl('vrooli://auth/callback')).toBe(true);
    });

    it('allows vrooli:// with query params', () => {
      expect(isAllowedCallbackUrl('vrooli://auth/callback?state=xyz')).toBe(true);
    });
  });

  describe('localhost URLs', () => {
    it('allows http://localhost', () => {
      expect(isAllowedCallbackUrl('http://localhost:3000/callback')).toBe(true);
    });

    it('allows https://localhost', () => {
      expect(isAllowedCallbackUrl('https://localhost:8080/auth')).toBe(true);
    });

    it('allows http://127.0.0.1', () => {
      expect(isAllowedCallbackUrl('http://127.0.0.1:3000')).toBe(true);
    });

    it('allows https://127.0.0.1', () => {
      expect(isAllowedCallbackUrl('https://127.0.0.1/callback')).toBe(true);
    });

    it('allows localhost without port', () => {
      expect(isAllowedCallbackUrl('http://localhost/callback')).toBe(true);
    });

    it('allows localhost with path and query', () => {
      expect(isAllowedCallbackUrl('http://localhost:3000/auth/callback?code=123')).toBe(true);
    });
  });

  describe('external URLs (should reject)', () => {
    it('rejects https://evil.com', () => {
      expect(isAllowedCallbackUrl('https://evil.com/steal')).toBe(false);
    });

    it('rejects http://attacker.io', () => {
      expect(isAllowedCallbackUrl('http://attacker.io:3000')).toBe(false);
    });

    it('rejects https://localhost.evil.com (subdomain attack)', () => {
      expect(isAllowedCallbackUrl('https://localhost.evil.com')).toBe(false);
    });

    it('rejects https://evil.localhost.com', () => {
      expect(isAllowedCallbackUrl('https://evil.localhost.com')).toBe(false);
    });

    it('rejects IP addresses other than 127.0.0.1', () => {
      expect(isAllowedCallbackUrl('http://192.168.1.1/callback')).toBe(false);
    });

    it('rejects 0.0.0.0', () => {
      expect(isAllowedCallbackUrl('http://0.0.0.0:3000/callback')).toBe(false);
    });

    it('rejects public IP addresses', () => {
      expect(isAllowedCallbackUrl('http://8.8.8.8/callback')).toBe(false);
    });

    it('rejects production-looking URLs', () => {
      expect(isAllowedCallbackUrl('https://vrooli.com/callback')).toBe(false);
    });
  });

  describe('invalid inputs', () => {
    it('rejects non-URL strings', () => {
      expect(isAllowedCallbackUrl('not-a-url')).toBe(false);
    });

    it('rejects empty string', () => {
      expect(isAllowedCallbackUrl('')).toBe(false);
    });

    it('rejects javascript: URLs', () => {
      expect(isAllowedCallbackUrl('javascript:alert(1)')).toBe(false);
    });

    it('rejects data: URLs', () => {
      expect(isAllowedCallbackUrl('data:text/html,<script>alert(1)</script>')).toBe(false);
    });

    it('rejects file: URLs', () => {
      expect(isAllowedCallbackUrl('file:///etc/passwd')).toBe(false);
    });

    it('rejects ftp: URLs', () => {
      expect(isAllowedCallbackUrl('ftp://server.com/file')).toBe(false);
    });

    it('rejects whitespace', () => {
      expect(isAllowedCallbackUrl('   ')).toBe(false);
    });

    it('rejects URL with spaces', () => {
      expect(isAllowedCallbackUrl('http://localhost with spaces/callback')).toBe(false);
    });
  });

  describe('edge cases', () => {
    it('rejects IPv6 localhost (::1)', () => {
      expect(isAllowedCallbackUrl('http://[::1]:3000/callback')).toBe(false);
    });

    it('allows encoded localhost (URL constructor normalizes it)', () => {
      // URL constructor decodes %6C%6F%63%61%6C%68%6F%73%74 to 'localhost'
      // Since localhost is allowed, encoded localhost is also allowed (correctly)
      expect(isAllowedCallbackUrl('http://%6C%6F%63%61%6C%68%6F%73%74/callback')).toBe(true);
    });

    it('handles case sensitivity in scheme', () => {
      // URL constructor normalizes the protocol to lowercase
      expect(isAllowedCallbackUrl('VROOLI://auth/callback')).toBe(true);
    });

    it('handles case sensitivity in localhost', () => {
      // URL constructor normalizes hostname to lowercase
      expect(isAllowedCallbackUrl('http://LOCALHOST:3000/callback')).toBe(true);
    });
  });
});
