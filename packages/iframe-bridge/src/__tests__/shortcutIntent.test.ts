import { afterEach, describe, expect, it, vi } from 'vitest';
import { emitShortcutIntent } from '../shortcutIntent.js';

describe('emitShortcutIntent', () => {
  const parentDescriptor = Object.getOwnPropertyDescriptor(window, 'parent');

  afterEach(() => {
    if (parentDescriptor) {
      Object.defineProperty(window, 'parent', parentDescriptor);
    }
  });

  it('returns false outside an iframe', () => {
    expect(emitShortcutIntent({ action: 'host.open-global-switcher' })).toBe(false);
  });

  it('posts the normalized protocol payload to the configured parent origin', () => {
    const postMessage = vi.fn();
    Object.defineProperty(window, 'parent', { configurable: true, value: { postMessage } });

    expect(emitShortcutIntent({ action: ' host.open-global-switcher ', chord: 'Ctrl+K' }, { parentOrigin: 'https://host.test' })).toBe(true);
    expect(postMessage).toHaveBeenCalledWith({
      v: 1,
      t: 'SHORTCUT',
      intent: expect.objectContaining({
        action: 'host.open-global-switcher',
        outcome: 'unhandled',
        chord: 'Ctrl+K',
        source: 'keyboard',
        detail: null,
      }),
    }, 'https://host.test');
  });

  it('rejects an empty action without posting', () => {
    const postMessage = vi.fn();
    Object.defineProperty(window, 'parent', { configurable: true, value: { postMessage } });

    expect(emitShortcutIntent({ action: '   ' })).toBe(false);
    expect(postMessage).not.toHaveBeenCalled();
  });
});
