/**
 * Storage shim kept in a small entry so applications can patch sandboxed
 * storage before loading modules that access localStorage at import time.
 */
export interface StorageShimEntry {
  prop: 'localStorage' | 'sessionStorage';
  patched: boolean;
  reason: string;
}

declare global {
  interface Window {
    __VROOLI_UI_SMOKE_STORAGE_PATCH__?: StorageShimEntry[];
  }
}


/**
 * Creates an in-memory Storage implementation that mimics localStorage/sessionStorage.
 * Implements the full Storage interface including length, key(), and indexed access.
 */
function createMemoryStorage(): Storage {
  const data = new Map<string, string>();

  // Use a Proxy to handle indexed access (storage['key'] and storage[0])
  const handler: ProxyHandler<Storage> = {
    get(target, prop) {
      if (prop === 'length') {
        return data.size;
      }
      if (prop === 'key') {
        return (index: number): string | null => {
          const keys = Array.from(data.keys());
          return keys[index] ?? null;
        };
      }
      if (prop === 'getItem') {
        return (key: string): string | null => data.get(key) ?? null;
      }
      if (prop === 'setItem') {
        return (key: string, value: string): void => {
          data.set(key, String(value));
        };
      }
      if (prop === 'removeItem') {
        return (key: string): void => {
          data.delete(key);
        };
      }
      if (prop === 'clear') {
        return (): void => {
          data.clear();
        };
      }
      // Handle numeric index access (storage[0])
      if (typeof prop === 'string' && /^\d+$/.test(prop)) {
        const keys = Array.from(data.keys());
        return keys[parseInt(prop, 10)] ?? undefined;
      }
      // Handle string key access (storage['myKey'])
      if (typeof prop === 'string') {
        return data.get(prop) ?? undefined;
      }
      return undefined;
    },
    set(target, prop, value) {
      if (typeof prop === 'string' && !['length', 'key', 'getItem', 'setItem', 'removeItem', 'clear'].includes(prop)) {
        data.set(prop, String(value));
        return true;
      }
      return false;
    },
    deleteProperty(target, prop) {
      if (typeof prop === 'string') {
        data.delete(prop);
        return true;
      }
      return false;
    },
    has(target, prop) {
      if (['length', 'key', 'getItem', 'setItem', 'removeItem', 'clear'].includes(String(prop))) {
        return true;
      }
      return data.has(String(prop));
    },
    ownKeys() {
      return Array.from(data.keys());
    },
    getOwnPropertyDescriptor(target, prop) {
      if (data.has(String(prop))) {
        return {
          value: data.get(String(prop)),
          writable: true,
          enumerable: true,
          configurable: true,
        };
      }
      return undefined;
    },
  };

  // Create base object with required methods (will be intercepted by proxy)
  const baseStorage: Storage = {
    length: 0,
    key: () => null,
    getItem: () => null,
    setItem: () => {},
    removeItem: () => {},
    clear: () => {},
  };

  return new Proxy(baseStorage, handler);
}

/**
 * Tests if a storage type is accessible and functional.
 */
function testStorageAccess(type: 'localStorage' | 'sessionStorage'): { accessible: boolean; error?: string } {
  try {
    // Even accessing the property can throw in sandboxed contexts
    const storage = type === 'localStorage' ? window.localStorage : window.sessionStorage;
    if (!storage) {
      return { accessible: false, error: 'storage-undefined' };
    }
    const testKey = '__vrooli_storage_test__';
    storage.setItem(testKey, 'test');
    const retrieved = storage.getItem(testKey);
    storage.removeItem(testKey);
    if (retrieved !== 'test') {
      return { accessible: false, error: 'storage-corrupted' };
    }
    return { accessible: true };
  } catch (e) {
    return {
      accessible: false,
      error: e instanceof Error ? e.message : String(e),
    };
  }
}

/**
 * Attempts to shim a specific storage type with an in-memory implementation.
 */
function shimStorageType(type: 'localStorage' | 'sessionStorage'): StorageShimEntry {
  const testResult = testStorageAccess(type);

  if (testResult.accessible) {
    return { prop: type, patched: false, reason: 'native-storage-accessible' };
  }

  try {
    const memoryStorage = createMemoryStorage();
    Object.defineProperty(window, type, {
      value: memoryStorage,
      writable: true,
      configurable: true,
    });

    // Verify the shim works
    const verifyResult = testStorageAccess(type);
    if (verifyResult.accessible) {
      return { prop: type, patched: true, reason: 'shimmed-successfully' };
    } else {
      return {
        prop: type,
        patched: false,
        reason: `shim-verification-failed: ${verifyResult.error}`,
      };
    }
  } catch (e) {
    return {
      prop: type,
      patched: false,
      reason: `shim-failed: ${e instanceof Error ? e.message : String(e)}`,
    };
  }
}

/**
 * Shims localStorage and sessionStorage with in-memory implementations if they are
 * blocked (common in sandboxed/headless browser containers).
 *
 * This function should be called as early as possible in your application,
 * before any code that might access localStorage or sessionStorage.
 *
 * Results are stored in window.__VROOLI_UI_SMOKE_STORAGE_PATCH__ for inspection
 * by UI smoke tests.
 *
 * @returns Array of shim results for each storage type
 *
 * @example
 * ```typescript
 * import { shimStorage } from '@vrooli/iframe-bridge';
 *
 * // Call at the very top of your entry point, before other imports if possible
 * shimStorage();
 * ```
 */
export function shimStorage(): StorageShimEntry[] {
  if (typeof window === 'undefined') {
    return [];
  }

  // Don't re-shim if already done (idempotent)
  if (window.__VROOLI_UI_SMOKE_STORAGE_PATCH__) {
    return window.__VROOLI_UI_SMOKE_STORAGE_PATCH__;
  }

  const results: StorageShimEntry[] = [
    shimStorageType('localStorage'),
    shimStorageType('sessionStorage'),
  ];

  // Record results for smoke test inspection
  window.__VROOLI_UI_SMOKE_STORAGE_PATCH__ = results;

  return results;
}

