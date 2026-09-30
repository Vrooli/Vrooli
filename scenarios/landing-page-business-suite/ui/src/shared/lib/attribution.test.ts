import { beforeEach, describe, expect, it } from 'vitest';
import { getAttributionContext, getSessionId, getVisitorId } from './attribution';

beforeEach(() => { sessionStorage.clear(); localStorage.clear(); window.history.replaceState({}, '', '/?utm_source=ad&utm_campaign=launch'); });

describe('attribution', () => {
  it('keeps session identity and first-touch campaign stable', () => {
    const first = getAttributionContext('control');
    window.history.replaceState({}, '', '/pricing?utm_source=other');
    const second = getAttributionContext('variant-a');
    expect(getSessionId()).toBe(first.session_id);
    expect(second.utm_source).toBe('ad');
    expect(second.utm_campaign).toBe('launch');
    expect(second.variant_slug).toBe('variant-a');
  });

  it('keeps the visitor id stable and accepts a server bootstrap id', () => {
    expect(getVisitorId()).toBe(getVisitorId());
    expect(getAttributionContext('control', 'server_visitor').visitor_id).toBe('server_visitor');
  });

  it('falls back to memory when session storage is unavailable', () => {
    Object.defineProperty(window, 'sessionStorage', { configurable: true, get: () => { throw new Error('blocked'); } });
    const first = getAttributionContext('control');
    const second = getAttributionContext('control');
    expect(first.session_id).toBe(second.session_id);
    expect(first.visitor_id).toBeTruthy();
  });
});
