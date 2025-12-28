/**
 * Anti-Detection Module
 *
 * Provides browser launch args and init scripts for bypassing bot detection.
 */

import type { BrowserContext } from 'playwright';
import type { AntiDetectionSettings, FingerprintSettings } from '../types/browser-profile';

/**
 * Build Chromium launch arguments for anti-detection.
 */
export function buildAntiDetectionArgs(settings: AntiDetectionSettings): string[] {
  const args: string[] = [];

  if (settings.disable_automation_controlled) {
    args.push('--disable-blink-features=AutomationControlled');
  }

  if (settings.disable_webrtc) {
    // Disable WebRTC to prevent IP leakage
    args.push('--disable-webrtc');
    args.push('--disable-webrtc-encryption');
    args.push('--disable-webrtc-hw-decoding');
    args.push('--disable-webrtc-hw-encoding');
  }

  return args;
}

/**
 * Generate the init script content for anti-detection patches.
 */
export function generateAntiDetectionScript(
  antiDetection: AntiDetectionSettings,
  fingerprint: FingerprintSettings
): string {
  const patches: string[] = [];

  // Remove navigator.webdriver property
  if (antiDetection.patch_navigator_webdriver) {
    patches.push(`
      // Remove webdriver flag
      Object.defineProperty(navigator, 'webdriver', {
        get: () => undefined,
        configurable: true,
      });

      // Also hide it from property enumeration
      const originalGetOwnPropertyDescriptor = Object.getOwnPropertyDescriptor;
      Object.getOwnPropertyDescriptor = function(obj, prop) {
        if (obj === navigator && prop === 'webdriver') {
          return undefined;
        }
        return originalGetOwnPropertyDescriptor.call(this, obj, prop);
      };
    `);
  }

  // Spoof navigator.plugins
  if (antiDetection.patch_navigator_plugins) {
    patches.push(`
      // Spoof plugins to look like a real browser
      Object.defineProperty(navigator, 'plugins', {
        get: () => {
          const plugins = {
            0: { name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer', description: 'Portable Document Format' },
            1: { name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai', description: '' },
            2: { name: 'Native Client', filename: 'internal-nacl-plugin', description: '' },
            length: 3,
            item: function(i) { return this[i] || null; },
            namedItem: function(name) {
              for (let i = 0; i < this.length; i++) {
                if (this[i].name === name) return this[i];
              }
              return null;
            },
            refresh: function() {},
            [Symbol.iterator]: function* () {
              for (let i = 0; i < this.length; i++) yield this[i];
            }
          };
          return plugins;
        },
        configurable: true,
      });

      // Also spoof mimeTypes
      Object.defineProperty(navigator, 'mimeTypes', {
        get: () => {
          const mimeTypes = {
            0: { type: 'application/pdf', description: 'Portable Document Format', suffixes: 'pdf' },
            1: { type: 'text/pdf', description: 'Portable Document Format', suffixes: 'pdf' },
            length: 2,
            item: function(i) { return this[i] || null; },
            namedItem: function(name) {
              for (let i = 0; i < this.length; i++) {
                if (this[i].type === name) return this[i];
              }
              return null;
            },
            [Symbol.iterator]: function* () {
              for (let i = 0; i < this.length; i++) yield this[i];
            }
          };
          return mimeTypes;
        },
        configurable: true,
      });
    `);
  }

  // Ensure consistent navigator.languages
  if (antiDetection.patch_navigator_languages) {
    const locale = fingerprint.locale || 'en-US';
    const lang = locale.split('-')[0];
    patches.push(`
      // Ensure consistent languages
      Object.defineProperty(navigator, 'languages', {
        get: () => ['${locale}', '${lang}'],
        configurable: true,
      });

      Object.defineProperty(navigator, 'language', {
        get: () => '${locale}',
        configurable: true,
      });
    `);
  }

  // Spoof WebGL renderer/vendor
  if (antiDetection.patch_webgl) {
    patches.push(`
      // Spoof WebGL info
      const getParameterOriginal = WebGLRenderingContext.prototype.getParameter;
      WebGLRenderingContext.prototype.getParameter = function(parameter) {
        // UNMASKED_VENDOR_WEBGL
        if (parameter === 37445) {
          return 'Intel Inc.';
        }
        // UNMASKED_RENDERER_WEBGL
        if (parameter === 37446) {
          return 'Intel Iris OpenGL Engine';
        }
        return getParameterOriginal.call(this, parameter);
      };

      // Also patch WebGL2
      if (typeof WebGL2RenderingContext !== 'undefined') {
        const getParameter2Original = WebGL2RenderingContext.prototype.getParameter;
        WebGL2RenderingContext.prototype.getParameter = function(parameter) {
          if (parameter === 37445) {
            return 'Intel Inc.';
          }
          if (parameter === 37446) {
            return 'Intel Iris OpenGL Engine';
          }
          return getParameter2Original.call(this, parameter);
        };
      }
    `);
  }

  // Add noise to canvas fingerprint
  if (antiDetection.patch_canvas) {
    patches.push(`
      // Add subtle noise to canvas to prevent fingerprinting
      const originalToDataURL = HTMLCanvasElement.prototype.toDataURL;
      HTMLCanvasElement.prototype.toDataURL = function(type, quality) {
        const context = this.getContext('2d');
        if (context) {
          const imageData = context.getImageData(0, 0, this.width, this.height);
          const data = imageData.data;
          // Add minimal noise to a few random pixels
          for (let i = 0; i < 10; i++) {
            const idx = Math.floor(Math.random() * (data.length / 4)) * 4;
            data[idx] = data[idx] ^ 1; // Flip least significant bit
          }
          context.putImageData(imageData, 0, 0);
        }
        return originalToDataURL.call(this, type, quality);
      };

      const originalToBlob = HTMLCanvasElement.prototype.toBlob;
      HTMLCanvasElement.prototype.toBlob = function(callback, type, quality) {
        const context = this.getContext('2d');
        if (context) {
          const imageData = context.getImageData(0, 0, this.width, this.height);
          const data = imageData.data;
          for (let i = 0; i < 10; i++) {
            const idx = Math.floor(Math.random() * (data.length / 4)) * 4;
            data[idx] = data[idx] ^ 1;
          }
          context.putImageData(imageData, 0, 0);
        }
        return originalToBlob.call(this, callback, type, quality);
      };
    `);
  }

  // Bypass headless detection
  if (antiDetection.headless_detection_bypass) {
    const hardwareConcurrency = fingerprint.hardware_concurrency || 4;
    const deviceMemory = fingerprint.device_memory || 8;
    patches.push(`
      // Override properties commonly checked for headless detection
      Object.defineProperty(navigator, 'hardwareConcurrency', {
        get: () => ${hardwareConcurrency},
        configurable: true,
      });

      Object.defineProperty(navigator, 'deviceMemory', {
        get: () => ${deviceMemory},
        configurable: true,
      });

      // Ensure window dimensions are realistic
      Object.defineProperty(window, 'outerWidth', {
        get: () => window.innerWidth + 16,
        configurable: true,
      });

      Object.defineProperty(window, 'outerHeight', {
        get: () => window.innerHeight + 88,
        configurable: true,
      });

      // Hide automation flags in chrome object
      if (window.chrome) {
        window.chrome.runtime = window.chrome.runtime || {};
      } else {
        window.chrome = { runtime: {} };
      }

      // Spoof permissions API
      const originalQuery = navigator.permissions?.query;
      if (originalQuery) {
        navigator.permissions.query = function(parameters) {
          if (parameters.name === 'notifications') {
            return Promise.resolve({ state: 'prompt', onchange: null });
          }
          return originalQuery.call(this, parameters);
        };
      }
    `);
  }

  if (patches.length === 0) {
    return '';
  }

  // Wrap in IIFE to avoid polluting global scope
  return `
    (function() {
      'use strict';
      ${patches.join('\n')}
    })();
  `;
}

/**
 * Apply anti-detection patches to a browser context.
 */
export async function applyAntiDetection(
  context: BrowserContext,
  antiDetection: AntiDetectionSettings,
  fingerprint: FingerprintSettings
): Promise<void> {
  const script = generateAntiDetectionScript(antiDetection, fingerprint);

  if (script) {
    await context.addInitScript({ content: script });
  }
}
