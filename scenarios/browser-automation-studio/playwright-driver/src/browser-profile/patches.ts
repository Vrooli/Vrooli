/**
 * Browser Signal Patches
 *
 * Individual, composable patch generators for selected browser signals.
 * These code paths do not establish a site-specific detection result.
 * Each patch is a named function that returns a JavaScript snippet (string)
 * to be injected into the browser context.
 *
 * DESIGN DECISION: Patches are separate functions for:
 * 1. Testability - Each patch can be tested independently
 * 2. Discoverability - Find "where we decide to patch X" by function name
 * 3. Extensibility - Add new patches without modifying monolithic function
 * 4. Selective composition - Enable/disable individual patches
 *
 * CHANGE AXIS: New Anti-Detection Patch
 * 1. Create a new patch function in this file
 * 2. Export it from this module
 * 3. Add to PATCH_REGISTRY with its enable condition
 * 4. Write unit test in tests/unit/browser-profile/patches.test.ts
 */

import type { AntiDetectionSettings, FingerprintSettings } from '../types/browser-profile';
import {
  HARDWARE_DEFAULTS,
  WINDOW_CHROME,
  CONNECTION_DEFAULTS,
  COMMON_FONTS,
  WEBGL_DEFAULTS,
  CANVAS_NOISE,
  AUDIO_NOISE,
  DEFAULT_PLUGINS,
  DEFAULT_MIME_TYPES,
  BATTERY_DEFAULTS,
} from './config';

// =============================================================================
// Patch Generator Types
// =============================================================================

/**
 * A patch generator returns JavaScript code to inject into the browser.
 * The code must be valid JavaScript that runs in browser context.
 */
export type PatchGenerator = (
  antiDetection: AntiDetectionSettings,
  fingerprint: FingerprintSettings
) => string;

/**
 * Registry entry for a patch.
 * Binds a patch generator to its enable condition.
 */
export interface PatchRegistryEntry {
  /** Unique identifier for the patch */
  id: string;
  /** Human-readable description */
  description: string;
  /** Function to check if patch should be applied */
  isEnabled: (antiDetection: AntiDetectionSettings) => boolean;
  /** Function to generate the patch code */
  generate: PatchGenerator;
}

// =============================================================================
// Individual Patch Generators
// =============================================================================

/**
 * Adjust the reported navigator.webdriver value and property visibility.
 * This does not establish a detection result or site acceptance.
 */
export function generateWebdriverPatch(): string {
  return `
    // Remove webdriver flag
    Object.defineProperty(navigator, 'webdriver', {
      get: () => false,
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
  `;
}

/**
 * Adjust the reported navigator.plugins and mimeTypes values.
 * This changes selected page-visible properties only; it does not establish
 * browser identity or detection outcomes.
 */
export function generatePluginsPatch(): string {
  const pluginsJson = JSON.stringify(DEFAULT_PLUGINS);
  const mimeTypesJson = JSON.stringify(DEFAULT_MIME_TYPES);

  return `
    // Apply the configured plugin-list values (defaults from browser-profile/config.ts).
    Object.defineProperty(navigator, 'plugins', {
      get: () => {
        const pluginData = ${pluginsJson};
        const plugins = {
          ...pluginData.reduce((acc, p, i) => ({ ...acc, [i]: p }), {}),
          length: pluginData.length,
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

    // Also spoof mimeTypes (defaults from browser-profile/config.ts)
    Object.defineProperty(navigator, 'mimeTypes', {
      get: () => {
        const mimeTypeData = ${mimeTypesJson};
        const mimeTypes = {
          ...mimeTypeData.reduce((acc, m, i) => ({ ...acc, [i]: m }), {}),
          length: mimeTypeData.length,
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
  `;
}

/**
 * Set navigator language values from the configured profile locale.
 */
export function generateLanguagesPatch(fingerprint: FingerprintSettings): string {
  const locale = fingerprint.locale || 'en-US';
  const lang = locale.split('-')[0];

  return `
    // Ensure consistent languages
    Object.defineProperty(navigator, 'languages', {
      get: () => ['${locale}', '${lang}'],
      configurable: true,
    });

    Object.defineProperty(navigator, 'language', {
      get: () => '${locale}',
      configurable: true,
    });
  `;
}

/**
 * Adjust selected WebGL renderer and vendor strings.
 */
export function generateWebGLPatch(): string {
  return `
    // Apply configured WebGL values (defaults from browser-profile/config.ts)
    const getParameterOriginal = WebGLRenderingContext.prototype.getParameter;
    WebGLRenderingContext.prototype.getParameter = function(parameter) {
      // UNMASKED_VENDOR_WEBGL
      if (parameter === 37445) {
        return '${WEBGL_DEFAULTS.vendor}';
      }
      // UNMASKED_RENDERER_WEBGL
      if (parameter === 37446) {
        return '${WEBGL_DEFAULTS.renderer}';
      }
      return getParameterOriginal.call(this, parameter);
    };

    // Also patch WebGL2
    if (typeof WebGL2RenderingContext !== 'undefined') {
      const getParameter2Original = WebGL2RenderingContext.prototype.getParameter;
      WebGL2RenderingContext.prototype.getParameter = function(parameter) {
        if (parameter === 37445) {
          return '${WEBGL_DEFAULTS.vendor}';
        }
        if (parameter === 37446) {
          return '${WEBGL_DEFAULTS.renderer}';
        }
        return getParameter2Original.call(this, parameter);
      };
    }
  `;
}

/**
 * Apply a small randomized change to selected canvas outputs.
 */
export function generateCanvasPatch(): string {
  return `
    // Add subtle noise to canvas to prevent fingerprinting (CANVAS_NOISE from config.ts)
    const canvasPixelsToModify = ${CANVAS_NOISE.pixelsToModify};
    const originalToDataURL = HTMLCanvasElement.prototype.toDataURL;
    HTMLCanvasElement.prototype.toDataURL = function(type, quality) {
      const context = this.getContext('2d');
      if (context) {
        const imageData = context.getImageData(0, 0, this.width, this.height);
        const data = imageData.data;
        // Add minimal noise to random pixels (count from config)
        for (let i = 0; i < canvasPixelsToModify; i++) {
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
        for (let i = 0; i < canvasPixelsToModify; i++) {
          const idx = Math.floor(Math.random() * (data.length / 4)) * 4;
          data[idx] = data[idx] ^ 1;
        }
        context.putImageData(imageData, 0, 0);
      }
      return originalToBlob.call(this, callback, type, quality);
    };
  `;
}

/**
 * Apply small randomized changes to selected audio outputs.
 */
export function generateAudioContextPatch(): string {
  return `
    // Add noise to AnalyserNode frequency data (AUDIO_NOISE from config.ts)
    const audioSampleProb = ${AUDIO_NOISE.sampleModifyProbability};
    const audioFloatAmplitude = ${AUDIO_NOISE.floatNoiseAmplitude};
    const audioBufferAmplitude = ${AUDIO_NOISE.bufferNoiseAmplitude};
    const audioBufferProb = ${AUDIO_NOISE.bufferModifyProbability};

    const originalGetByteFrequencyData = AnalyserNode.prototype.getByteFrequencyData;
    AnalyserNode.prototype.getByteFrequencyData = function(array) {
      originalGetByteFrequencyData.call(this, array);
      for (let i = 0; i < array.length; i++) {
        if (Math.random() < audioSampleProb) {
          array[i] = Math.max(0, Math.min(255, array[i] + Math.floor(Math.random() * 3) - 1));
        }
      }
    };

    const originalGetFloatFrequencyData = AnalyserNode.prototype.getFloatFrequencyData;
    AnalyserNode.prototype.getFloatFrequencyData = function(array) {
      originalGetFloatFrequencyData.call(this, array);
      for (let i = 0; i < array.length; i++) {
        if (Math.random() < audioSampleProb) {
          array[i] += (Math.random() - 0.5) * audioFloatAmplitude;
        }
      }
    };

    // Add noise to time domain data
    const originalGetByteTimeDomainData = AnalyserNode.prototype.getByteTimeDomainData;
    AnalyserNode.prototype.getByteTimeDomainData = function(array) {
      originalGetByteTimeDomainData.call(this, array);
      for (let i = 0; i < array.length; i++) {
        if (Math.random() < audioSampleProb) {
          array[i] = Math.max(0, Math.min(255, array[i] + Math.floor(Math.random() * 3) - 1));
        }
      }
    };

    const originalGetFloatTimeDomainData = AnalyserNode.prototype.getFloatTimeDomainData;
    AnalyserNode.prototype.getFloatTimeDomainData = function(array) {
      originalGetFloatTimeDomainData.call(this, array);
      for (let i = 0; i < array.length; i++) {
        if (Math.random() < audioSampleProb) {
          array[i] += (Math.random() - 0.5) * audioBufferAmplitude;
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
          if (Math.random() < audioBufferProb) { // Very sparse noise for audio buffers
            data[i] += (Math.random() - 0.5) * audioBufferAmplitude;
          }
        }
      }
      return data;
    };
  `;
}

/**
 * Adjust selected properties commonly associated with headless configurations.
 *
 * The function retains its legacy name. It adjusts selected properties and
 * does not make the browser equivalent to a headed browser or guarantee site
 * acceptance.
 */
export function generateHeadlessBypassPatch(fingerprint: FingerprintSettings): string {
  const hardwareConcurrency = fingerprint.hardware_concurrency || HARDWARE_DEFAULTS.hardwareConcurrency;
  const deviceMemory = fingerprint.device_memory || HARDWARE_DEFAULTS.deviceMemory;

  return `
    // Override properties commonly checked for headless detection (HARDWARE_DEFAULTS from config.ts)
    Object.defineProperty(navigator, 'hardwareConcurrency', {
      get: () => ${hardwareConcurrency},
      configurable: true,
    });

    Object.defineProperty(navigator, 'deviceMemory', {
      get: () => ${deviceMemory},
      configurable: true,
    });

    // Apply configured window dimensions (WINDOW_CHROME from config.ts)
    Object.defineProperty(window, 'outerWidth', {
      get: () => window.innerWidth + ${WINDOW_CHROME.widthDelta},
      configurable: true,
    });

    Object.defineProperty(window, 'outerHeight', {
      get: () => window.innerHeight + ${WINDOW_CHROME.heightDelta},
      configurable: true,
    });

    // Hide automation flags in chrome object
    if (window.chrome) {
      window.chrome.runtime = window.chrome.runtime || {};
    } else {
      window.chrome = { runtime: {} };
    }

    // Adjust the notifications permission response
    const originalQuery = navigator.permissions?.query;
    if (originalQuery) {
      navigator.permissions.query = function(parameters) {
        if (parameters.name === 'notifications') {
          return Promise.resolve({ state: 'prompt', onchange: null });
        }
        return originalQuery.call(this, parameters);
      };
    }
  `;
}

/**
 * Restrict positive document.fonts.check results to configured common fonts.
 */
export function generateFontsPatch(): string {
  const fontsArray = Array.from(COMMON_FONTS);

  return `
    // Apply configured common-font list (COMMON_FONTS from config.ts)
    const commonFonts = new Set(${JSON.stringify(fontsArray)});

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
  `;
}

/**
 * Adjust selected screen dimensions to match the configured viewport.
 * This does not establish that the viewport or device identity matches a
 * physical device or a site's expectations.
 */
export function generateScreenPropertiesPatch(fingerprint: FingerprintSettings): string {
  const width = fingerprint.viewport_width || HARDWARE_DEFAULTS.viewportWidth;
  const height = fingerprint.viewport_height || HARDWARE_DEFAULTS.viewportHeight;
  const availHeight = height - HARDWARE_DEFAULTS.taskbarHeight;

  return `
    // Apply configured screen dimensions (HARDWARE_DEFAULTS from config.ts)
    Object.defineProperty(screen, 'width', { get: () => ${width}, configurable: true });
    Object.defineProperty(screen, 'height', { get: () => ${height}, configurable: true });
    Object.defineProperty(screen, 'availWidth', { get: () => ${width}, configurable: true });
    Object.defineProperty(screen, 'availHeight', { get: () => ${availHeight}, configurable: true });
    Object.defineProperty(screen, 'colorDepth', { get: () => ${HARDWARE_DEFAULTS.colorDepth}, configurable: true });
    Object.defineProperty(screen, 'pixelDepth', { get: () => ${HARDWARE_DEFAULTS.pixelDepth}, configurable: true });
  `;
}

/**
 * Return configured battery status when the Battery API is available.
 */
export function generateBatteryPatch(): string {
  return `
    // Return configured battery status (BATTERY_DEFAULTS from config.ts)
    if (navigator.getBattery) {
      navigator.getBattery = function() {
        return Promise.resolve({
          charging: ${BATTERY_DEFAULTS.charging},
          chargingTime: ${BATTERY_DEFAULTS.chargingTime},
          dischargingTime: ${BATTERY_DEFAULTS.dischargingTime},
          level: ${BATTERY_DEFAULTS.level},
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
  `;
}

/**
 * Adjust selected navigator.connection values; this does not change the
 * actual network path or indicate a residential connection.
 */
export function generateConnectionPatch(): string {
  return `
    // Apply configured connection values (CONNECTION_DEFAULTS from config.ts)
    if (navigator.connection) {
      Object.defineProperty(navigator, 'connection', {
        get: () => ({
          effectiveType: '${CONNECTION_DEFAULTS.effectiveType}',
          type: '${CONNECTION_DEFAULTS.type}',
          downlink: ${CONNECTION_DEFAULTS.downlink},
          rtt: ${CONNECTION_DEFAULTS.rtt},
          saveData: ${CONNECTION_DEFAULTS.saveData},
          onchange: null,
          addEventListener: function() {},
          removeEventListener: function() {},
          dispatchEvent: function() { return true; }
        }),
        configurable: true
      });
    }
  `;
}

// =============================================================================
// Patch Registry
// =============================================================================

/**
 * Registry of all available patches.
 *
 * CHANGE AXIS: To add a new patch:
 * 1. Create the patch generator function above
 * 2. Add entry to this registry with enable condition
 *
 * The registry pattern allows:
 * - Easy iteration over all patches
 * - Consistent enable/disable logic
 * - Future: selective patch application based on target site
 */
export const PATCH_REGISTRY: PatchRegistryEntry[] = [
  {
    id: 'webdriver',
    description: 'Adjust navigator.webdriver value and visibility',
    isEnabled: (a) => a.patch_navigator_webdriver === true,
    generate: () => generateWebdriverPatch(),
  },
  {
    id: 'plugins',
    description: 'Adjust navigator.plugins and mimeTypes values',
    isEnabled: (a) => a.patch_navigator_plugins === true,
    generate: () => generatePluginsPatch(),
  },
  {
    id: 'languages',
    description: 'Set navigator.languages from profile locale',
    isEnabled: (a) => a.patch_navigator_languages === true,
    generate: (_, f) => generateLanguagesPatch(f),
  },
  {
    id: 'webgl',
    description: 'Adjust selected WebGL renderer and vendor values',
    isEnabled: (a) => a.patch_webgl === true,
    generate: () => generateWebGLPatch(),
  },
  {
    id: 'canvas',
    description: 'Adjust selected canvas outputs',
    isEnabled: (a) => a.patch_canvas === true,
    generate: () => generateCanvasPatch(),
  },
  {
    id: 'audio',
    description: 'Adjust selected AudioContext outputs',
    isEnabled: (a) => a.patch_audio_context === true,
    generate: () => generateAudioContextPatch(),
  },
  {
    id: 'headless',
    description: 'Adjust selected properties for headless browser contexts',
    isEnabled: (a) => a.headless_detection_bypass === true,
    generate: (_, f) => generateHeadlessBypassPatch(f),
  },
  {
    id: 'fonts',
    description: 'Adjust the fonts exposed to page scripts',
    isEnabled: (a) => a.patch_fonts === true,
    generate: () => generateFontsPatch(),
  },
  {
    id: 'screen',
    description: 'Adjust selected screen properties',
    isEnabled: (a) => a.patch_screen_properties === true,
    generate: (_, f) => generateScreenPropertiesPatch(f),
  },
  {
    id: 'battery',
    description: 'Return configured battery status',
    isEnabled: (a) => a.patch_battery_api === true,
    generate: () => generateBatteryPatch(),
  },
  {
    id: 'connection',
    description: 'Adjust selected navigator.connection values',
    isEnabled: (a) => a.patch_connection_api === true,
    generate: () => generateConnectionPatch(),
  },
];

/**
 * Get list of patches that will be applied for given settings.
 * Useful for logging and debugging.
 */
export function getEnabledPatches(antiDetection: AntiDetectionSettings): string[] {
  return PATCH_REGISTRY
    .filter((entry) => entry.isEnabled(antiDetection))
    .map((entry) => entry.id);
}

/**
 * Compose enabled patches into a single script.
 *
 * DECISION: Compose patches at runtime rather than build time.
 * This allows dynamic patch selection based on settings.
 */
export function composePatches(
  antiDetection: AntiDetectionSettings,
  fingerprint: FingerprintSettings,
  options: { deterministicAudio?: boolean } = {},
): string {
  const patches: string[] = [];

  for (const entry of PATCH_REGISTRY) {
    if (entry.id === 'audio' && options.deterministicAudio) continue;
    if (entry.isEnabled(antiDetection)) {
      patches.push(entry.generate(antiDetection, fingerprint));
    }
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
