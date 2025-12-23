import { describe, expect, it } from 'vitest';
import { resolveBackgroundDecor } from '../catalog';

describe('resolveBackgroundDecor', () => {
  it('builds gradient background decor', () => {
    const decor = resolveBackgroundDecor(
      {
        type: 'gradient',
        value: {
          type: 'linear',
          angle: 90,
          stops: [
            { color: '#111111', position: 0 },
            { color: '#222222', position: 100 },
          ],
        },
      },
      'aurora',
    );

    expect(decor.containerStyle?.backgroundImage).toContain('linear-gradient');
  });

  it('builds image background decor', () => {
    const decor = resolveBackgroundDecor(
      {
        type: 'image',
        url: 'https://example.com/background.jpg',
        fit: 'contain',
      },
      'aurora',
    );

    expect(decor.image?.url).toBe('https://example.com/background.jpg');
    expect(decor.image?.fit).toBe('contain');
  });
});
