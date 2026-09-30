import { describe, expect, it } from 'vitest';
import { safeHref } from './links';
import { scopedPresentationHref } from './publicIntegration';
describe('configured public link guard', () => {
  it('preserves only explicit presentation scope on known marketing routes', () => {
    const scope = { locale: 'fr', variant: 'review' };
    for (const path of ['/', '/apps/example', '/apps/example/download']) expect(scopedPresentationHref(path, '/proxy/', scope)).toBe(`/proxy${path}?locale=fr&variant=review`);
    for (const path of ['/checkout?owner=opaque', '/app/example', 'https://launch.example/?owner=opaque']) expect(scopedPresentationHref(path, '/proxy/', scope)).toBe(path.startsWith('/') ? '/proxy' + path : path);
    expect(scopedPresentationHref('/apps/example', '/proxy/', { locale: 'fr', variant: '' })).toBe('/proxy/apps/example?locale=fr');
    expect(scopedPresentationHref('/apps/example', '/proxy/', { locale: '', variant: '' })).toBe('/proxy/apps/example');
  });
  it.each(['https://user:pass@host.test/path', 'https://user@host.test', '/%5cprivate', '/%00', '/%7f', '/a/../private', '/%2e%2e/private', 'https://host.test/a/%2e%2e/private', '/%252e%252e/private', '/%2fprivate', 'https://host.test/%5cprivate', '/path?token=%0a', '//private.test', 'https:private.test', '/bad%zz'])('rejects %s', value => { expect(safeHref(value)).toBeUndefined(); });
  it.each(['/api/v1/release/image', '/proxy/api/v1/release/image', '#artifact', 'https://cdn.example/art.png', '/checkout?return=%2Fapps%2Fexample', '/art%20label.png'])('retains safe owner URL %s', value => { expect(safeHref(value)).toBe(value); });
});
