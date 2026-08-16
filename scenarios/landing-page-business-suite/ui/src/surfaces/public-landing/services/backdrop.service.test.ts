import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest';
import type { BackdropReference } from './backdrop.service';

type ResolveBackdropReference = (
  reference: Pick<BackdropReference, 'id'>,
  fetcher?: typeof fetch,
) => Promise<BackdropReference | null>;

let resolveBackdropReference: ResolveBackdropReference;

beforeAll(async () => {
  vi.stubEnv('VITE_BACKDROP_STUDIO_URL', 'https://backdrop.example.test/');
  vi.resetModules();
  ({ resolveBackdropReference } = await import('./backdrop.service'));
});

afterAll(() => vi.unstubAllEnvs());

describe('resolveBackdropReference', () => {
  it('rejects missing IDs before making a request', async () => {
    const fetcher = vi.fn();

    await expect(resolveBackdropReference({ id: '' }, fetcher)).resolves.toBeNull();
    expect(fetcher).not.toHaveBeenCalled();
  });

  it('rejects unsuccessful and malformed release responses', async () => {
    await expect(resolveBackdropReference({ id: 'missing' }, vi.fn().mockResolvedValue({ ok: false }))).resolves.toBeNull();
    await expect(resolveBackdropReference({ id: 'malformed' }, vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ id: 'wrong-shape' }),
    }))).resolves.toBeNull();
  });

  it('maps valid release metadata and filters invalid reserved regions', async () => {
    const fetcher = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        id: 'released-hero',
        uri: '/assets/hero.png',
        placement: 'full_bleed',
        altText: 'Founder workspace',
        reservedRegions: [
          { x: 0, y: 0, width: 0.5, height: 0.5, kind: 'copy' },
          { x: 'invalid', y: 0, width: 1, height: 1 },
        ],
      }),
    });

    await expect(resolveBackdropReference({ id: 'released-hero' }, fetcher)).resolves.toEqual({
      id: 'released-hero',
      uri: '/assets/hero.png',
      url: 'https://backdrop.example.test/assets/hero.png',
      placement: 'full_bleed',
      alt_text: 'Founder workspace',
      reserved_regions: [{ x: 0, y: 0, width: 0.5, height: 0.5, kind: 'copy' }],
    });
    expect(fetcher).toHaveBeenCalledWith(
      'https://backdrop.example.test/vrooli.backdrop_studio.v1.release.ReleaseService/GetReference',
      expect.objectContaining({ method: 'POST' }),
    );
  });

  it('preserves absolute asset URLs and optional metadata when present', async () => {
    const fetcher = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ id: 'absolute', uri: 'https://cdn.example.test/hero.png', reservedRegions: null }),
    });

    await expect(resolveBackdropReference({ id: 'absolute' }, fetcher)).resolves.toMatchObject({
      id: 'absolute',
      uri: 'https://cdn.example.test/hero.png',
      url: 'https://cdn.example.test/hero.png',
      reserved_regions: [],
    });
  });
});
