/**
 * Storage factory and exports
 *
 * Detects the runtime environment and provides the appropriate storage implementation.
 */

import type { AssetStorage } from './types';
import { IndexedDbAssetStorage } from './indexedDbStorage';

// Singleton instance
let storageInstance: AssetStorage | null = null;

/**
 * Detected runtime environment
 */
export type RuntimeEnvironment = 'electron' | 'capacitor' | 'web';

/**
 * Detect the current runtime environment
 *
 * Priority:
 * 1. Electron (window.desktop from scenario-to-desktop preload)
 * 2. Capacitor (future mobile support)
 * 3. Web/PWA (default fallback)
 */
export function detectEnvironment(): RuntimeEnvironment {
  if (typeof window !== 'undefined') {
    // Check for scenario-to-desktop bridge
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    if ((window as any).desktop?.storage) {
      return 'electron';
    }
    // Check for Capacitor native bridge (future)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    if ((window as any).Capacitor?.isNativePlatform?.()) {
      return 'capacitor';
    }
  }
  return 'web';
}

/**
 * Get the asset storage instance for the current environment
 *
 * Uses lazy initialization and returns a singleton.
 */
export async function getAssetStorage(): Promise<AssetStorage> {
  if (storageInstance) return storageInstance;

  const env = detectEnvironment();

  switch (env) {
    case 'electron': {
      // Dynamic import to avoid bundling desktop code in web builds
      const { DesktopAssetStorage } = await import('./desktopStorage');
      storageInstance = new DesktopAssetStorage();
      break;
    }

    case 'capacitor':
      // Future: CapacitorAssetStorage
      // For now, fall through to IndexedDB
      console.info('Capacitor detected, using IndexedDB storage (native storage not yet implemented)');
      storageInstance = new IndexedDbAssetStorage();
      break;

    case 'web':
    default:
      storageInstance = new IndexedDbAssetStorage();
      break;
  }

  return storageInstance;
}

/**
 * Clear the storage singleton (mainly for testing)
 */
export function resetStorageInstance(): void {
  storageInstance = null;
}

// Re-export types
export * from './types';
export { IndexedDbAssetStorage } from './indexedDbStorage';
