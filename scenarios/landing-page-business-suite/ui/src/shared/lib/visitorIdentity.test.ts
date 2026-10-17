import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import { getVisitorId, isValidVisitorId, visitorIdFromCookie } from './visitorIdentity';

beforeEach(() => { localStorage.clear(); document.cookie = 'metrics_visitor_id=; Path=/; Max-Age=0'; });
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); });
describe('shared anonymous visitor identity', () => {
  it.each(['visitor_123_abcd', '8b86bf23-7222-4674-bfe7-a001dbaf79e4', 'a'.repeat(128)])('accepts safe legacy identity %s', value => { expect(isValidVisitorId(value)).toBe(true); });
  it.each(['', ' a', 'a ', 'a/b', 'a\\b', '%61', 'a;b', 'a\n', 'é', 'a'.repeat(129), null])('rejects unsafe identity %s', value => { expect(isValidVisitorId(value)).toBe(false); });
  it('rejects duplicate cookies, encoded values and suffix-name collisions', () => {
    expect(visitorIdFromCookie('other=1; metrics_visitor_id=safe')).toBe('safe');
    expect(visitorIdFromCookie('metrics_visitor_id=a; metrics_visitor_id=a')).toBeUndefined();
    expect(visitorIdFromCookie('metrics_visitor_id=%61')).toBeUndefined();
    expect(visitorIdFromCookie('other_metrics_visitor_id=a')).toBeUndefined();
  });
  it('prefers bootstrap, then cookie, then legacy local storage and persists agreement', () => {
    localStorage.setItem('metrics_visitor_id', 'legacy');
    expect(getVisitorId()).toBe('legacy');
    document.cookie = 'metrics_visitor_id=cookie; Path=/';
    expect(getVisitorId()).toBe('cookie');
    expect(getVisitorId('bootstrap')).toBe('bootstrap');
    expect(localStorage.getItem('metrics_visitor_id')).toBe('bootstrap');
    expect(document.cookie).toContain('metrics_visitor_id=bootstrap');
  });
  it('does not rewrite a matching cookie', () => {
    document.cookie = 'metrics_visitor_id=same; Path=/';
    const write = vi.spyOn(document, 'cookie', 'set');
    expect(getVisitorId('same')).toBe('same'); expect(write).not.toHaveBeenCalled();
  });
  it.each(['http:', 'https:'])('sets the shared lifetime and security attributes on %s', protocol => {
    vi.stubGlobal('location', { protocol });
    const write = vi.spyOn(document, 'cookie', 'set').mockImplementation(() => {});
    getVisitorId('fresh');
    expect(write).toHaveBeenCalledWith(`metrics_visitor_id=fresh; Path=/; SameSite=Lax; Max-Age=31536000${protocol === 'https:' ? '; Secure' : ''}`);
  });
  it('keeps identity stable when cookies and local storage are blocked', () => {
    vi.spyOn(document, 'cookie', 'get').mockImplementation(() => { throw new Error('blocked'); });
    vi.spyOn(document, 'cookie', 'set').mockImplementation(() => { throw new Error('blocked'); });
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('blocked'); });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('blocked'); });
    expect(getVisitorId('in_memory')).toBe('in_memory'); expect(getVisitorId()).toBe('in_memory');
  });
});
