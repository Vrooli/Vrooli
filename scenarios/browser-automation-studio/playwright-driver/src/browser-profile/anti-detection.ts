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

  // Spoof AudioContext to prevent audio fingerprinting
  if (antiDetection.patch_audio_context) {
    patches.push(`
      // Add noise to AnalyserNode frequency data
      const originalGetByteFrequencyData = AnalyserNode.prototype.getByteFrequencyData;
      AnalyserNode.prototype.getByteFrequencyData = function(array) {
        originalGetByteFrequencyData.call(this, array);
        // Add small noise to ~5% of values
        for (let i = 0; i < array.length; i++) {
          if (Math.random() < 0.05) {
            array[i] = Math.max(0, Math.min(255, array[i] + Math.floor(Math.random() * 3) - 1));
          }
        }
      };

      const originalGetFloatFrequencyData = AnalyserNode.prototype.getFloatFrequencyData;
      AnalyserNode.prototype.getFloatFrequencyData = function(array) {
        originalGetFloatFrequencyData.call(this, array);
        for (let i = 0; i < array.length; i++) {
          if (Math.random() < 0.05) {
            array[i] += (Math.random() - 0.5) * 0.1;
          }
        }
      };

      // Add noise to time domain data
      const originalGetByteTimeDomainData = AnalyserNode.prototype.getByteTimeDomainData;
      AnalyserNode.prototype.getByteTimeDomainData = function(array) {
        originalGetByteTimeDomainData.call(this, array);
        for (let i = 0; i < array.length; i++) {
          if (Math.random() < 0.05) {
            array[i] = Math.max(0, Math.min(255, array[i] + Math.floor(Math.random() * 3) - 1));
          }
        }
      };

      const originalGetFloatTimeDomainData = AnalyserNode.prototype.getFloatTimeDomainData;
      AnalyserNode.prototype.getFloatTimeDomainData = function(array) {
        originalGetFloatTimeDomainData.call(this, array);
        for (let i = 0; i < array.length; i++) {
          if (Math.random() < 0.05) {
            array[i] += (Math.random() - 0.5) * 0.0001;
          }
        }
      };

      // Add noise to AudioBuffer channel data reads
      const originalGetChannelData = AudioBuffer.prototype.getChannelData;
      AudioBuffer.prototype.getChannelData = function(channel) {
        const data = originalGetChannelData.call(this, channel);
        // Only modify on first access per buffer
        if (!this._noised) {
          this._noised = true;
          for (let i = 0; i < data.length; i++) {
            if (Math.random() < 0.001) { // Very sparse noise for audio buffers
              data[i] += (Math.random() - 0.5) * 0.0001;
            }
          }
        }
        return data;
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

  // Spoof font fingerprinting
  if (antiDetection.patch_fonts) {
    patches.push(`
      // Override font detection to return common fonts only
      const commonFonts = new Set([
        'Arial', 'Arial Black', 'Comic Sans MS', 'Courier New', 'Georgia',
        'Helvetica', 'Impact', 'Lucida Console', 'Lucida Sans Unicode',
        'Palatino Linotype', 'Tahoma', 'Times New Roman', 'Trebuchet MS', 'Verdana'
      ]);

      // Patch document.fonts.check to only confirm common fonts
      if (document.fonts && document.fonts.check) {
        const originalCheck = document.fonts.check.bind(document.fonts);
        document.fonts.check = function(font, text) {
          const match = font.match(/(?:^|\\s)([\\w\\s-]+)(?:,|$)/);
          if (match) {
            const fontName = match[1].trim().replace(/['"]/g, '');
            if (!commonFonts.has(fontName)) {
              return false;
            }
          }
          return originalCheck(font, text);
        };
      }
    `);
  }

  // Spoof screen properties
  if (antiDetection.patch_screen_properties) {
    const width = fingerprint.viewport_width || 1920;
    const height = fingerprint.viewport_height || 1080;
    patches.push(`
      // Spoof screen dimensions to match viewport
      Object.defineProperty(screen, 'width', { get: () => ${width}, configurable: true });
      Object.defineProperty(screen, 'height', { get: () => ${height}, configurable: true });
      Object.defineProperty(screen, 'availWidth', { get: () => ${width}, configurable: true });
      Object.defineProperty(screen, 'availHeight', { get: () => ${height - 40}, configurable: true });
      Object.defineProperty(screen, 'colorDepth', { get: () => 24, configurable: true });
      Object.defineProperty(screen, 'pixelDepth', { get: () => 24, configurable: true });
    `);
  }

  // Spoof battery API
  if (antiDetection.patch_battery_api) {
    patches.push(`
      // Return realistic battery status (fully charged, plugged in)
      if (navigator.getBattery) {
        navigator.getBattery = function() {
          return Promise.resolve({
            charging: true,
            chargingTime: 0,
            dischargingTime: Infinity,
            level: 1.0,
            onchargingchange: null,
            onchargingtimechange: null,
            ondischargingtimechange: null,
            onlevelchange: null,
            addEventListener: function() {},
            removeEventListener: function() {},
            dispatchEvent: function() { return true; }
          });
        };
      }
    `);
  }

  // Spoof connection API
  if (antiDetection.patch_connection_api) {
    patches.push(`
      // Spoof network connection to appear as typical residential WiFi
      if (navigator.connection) {
        Object.defineProperty(navigator, 'connection', {
          get: () => ({
            effectiveType: '4g',
            type: 'wifi',
            downlink: 10,
            rtt: 50,
            saveData: false,
            onchange: null,
            addEventListener: function() {},
            removeEventListener: function() {},
            dispatchEvent: function() { return true; }
          }),
          configurable: true
        });
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
