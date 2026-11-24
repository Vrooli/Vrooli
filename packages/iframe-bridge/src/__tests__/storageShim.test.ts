import { afterEach, describe, expect, it } from 'vitest';
import { shimStorage } from '../storageShim.js';

const localDescriptor = Object.getOwnPropertyDescriptor(window, 'localStorage');
const sessionDescriptor = Object.getOwnPropertyDescriptor(window, 'sessionStorage');

afterEach(() => {
  if (localDescriptor) Object.defineProperty(window, 'localStorage', localDescriptor);
  if (sessionDescriptor) Object.defineProperty(window, 'sessionStorage', sessionDescriptor);
  delete window.__VROOLI_UI_SMOKE_STORAGE_PATCH__;
});

describe('shimStorage', () => {
  it('leaves accessible native storage in place and records an idempotent result', () => {
    const results = shimStorage();
    expect(results).toHaveLength(2);
    expect(results.every((result) => !result.patched && result.reason === 'native-storage-accessible')).toBe(true);
    expect(shimStorage()).toBe(results);
  });

  it('patches blocked storage before application modules access it', () => {
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      get: () => {
        throw new DOMException('blocked', 'SecurityError');
      },
    });

    const results = shimStorage();
    const local = results.find((result) => result.prop === 'localStorage');
    expect(local?.patched).toBe(true);
    window.localStorage.setItem('early', 'ready');
    expect(window.localStorage.getItem('early')).toBe('ready');
  });
});
