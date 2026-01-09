import { describe, it, expect } from 'vitest';
import {
  MIN_VIEWPORT_DIMENSION,
  MAX_VIEWPORT_DIMENSION,
  isPlainObject,
  clampDimension,
  parseViewportDimension,
  parseViewportPreset,
  sanitizeViewportSettings,
  extractExecutionViewport,
} from './viewport';

describe('viewport utilities', () => {
  describe('isPlainObject', () => {
    it('returns true for plain objects', () => {
      expect(isPlainObject({})).toBe(true);
      expect(isPlainObject({ a: 1 })).toBe(true);
      expect(isPlainObject({ nested: { value: true } })).toBe(true);
    });

    it('returns false for arrays', () => {
      expect(isPlainObject([])).toBe(false);
      expect(isPlainObject([1, 2, 3])).toBe(false);
    });

    it('returns false for null and undefined', () => {
      expect(isPlainObject(null)).toBe(false);
      expect(isPlainObject(undefined)).toBe(false);
    });

    it('returns false for primitives', () => {
      expect(isPlainObject('string')).toBe(false);
      expect(isPlainObject(123)).toBe(false);
      expect(isPlainObject(true)).toBe(false);
    });
  });

  describe('clampDimension', () => {
    it('returns MIN_VIEWPORT_DIMENSION for non-finite values', () => {
      expect(clampDimension(NaN)).toBe(MIN_VIEWPORT_DIMENSION);
      expect(clampDimension(Infinity)).toBe(MIN_VIEWPORT_DIMENSION);
      expect(clampDimension(-Infinity)).toBe(MIN_VIEWPORT_DIMENSION);
    });

    it('clamps values below minimum', () => {
      expect(clampDimension(0)).toBe(MIN_VIEWPORT_DIMENSION);
      expect(clampDimension(50)).toBe(MIN_VIEWPORT_DIMENSION);
      expect(clampDimension(199)).toBe(MIN_VIEWPORT_DIMENSION);
    });

    it('clamps values above maximum', () => {
      expect(clampDimension(10001)).toBe(MAX_VIEWPORT_DIMENSION);
      expect(clampDimension(20000)).toBe(MAX_VIEWPORT_DIMENSION);
    });

    it('rounds values to nearest integer', () => {
      expect(clampDimension(500.4)).toBe(500);
      expect(clampDimension(500.6)).toBe(501);
    });

    it('passes through valid values', () => {
      expect(clampDimension(200)).toBe(200);
      expect(clampDimension(1920)).toBe(1920);
      expect(clampDimension(10000)).toBe(10000);
    });
  });

  describe('parseViewportDimension', () => {
    it('parses valid numbers', () => {
      expect(parseViewportDimension(1920)).toBe(1920);
      expect(parseViewportDimension(1080)).toBe(1080);
    });

    it('returns null for zero or negative numbers', () => {
      expect(parseViewportDimension(0)).toBe(null);
      expect(parseViewportDimension(-100)).toBe(null);
    });

    it('parses valid string numbers', () => {
      expect(parseViewportDimension('1920')).toBe(1920);
      expect(parseViewportDimension('  1080  ')).toBe(1080);
    });

    it('returns null for empty strings', () => {
      expect(parseViewportDimension('')).toBe(null);
      expect(parseViewportDimension('   ')).toBe(null);
    });

    it('returns null for non-numeric strings', () => {
      expect(parseViewportDimension('abc')).toBe(null);
    });

    it('parses leading numbers from mixed strings (parseInt behavior)', () => {
      // parseInt('12px', 10) returns 12, which gets clamped to MIN
      expect(parseViewportDimension('12px')).toBe(MIN_VIEWPORT_DIMENSION);
      expect(parseViewportDimension('1920px')).toBe(1920);
    });

    it('returns null for other types', () => {
      expect(parseViewportDimension(null)).toBe(null);
      expect(parseViewportDimension(undefined)).toBe(null);
      expect(parseViewportDimension({})).toBe(null);
      expect(parseViewportDimension([])).toBe(null);
    });

    it('clamps parsed values', () => {
      expect(parseViewportDimension(50)).toBe(MIN_VIEWPORT_DIMENSION);
      expect(parseViewportDimension(20000)).toBe(MAX_VIEWPORT_DIMENSION);
    });
  });

  describe('parseViewportPreset', () => {
    it('parses valid presets', () => {
      expect(parseViewportPreset('desktop')).toBe('desktop');
      expect(parseViewportPreset('mobile')).toBe('mobile');
      expect(parseViewportPreset('custom')).toBe('custom');
    });

    it('handles case insensitivity', () => {
      expect(parseViewportPreset('DESKTOP')).toBe('desktop');
      expect(parseViewportPreset('Mobile')).toBe('mobile');
      expect(parseViewportPreset('CUSTOM')).toBe('custom');
    });

    it('trims whitespace', () => {
      expect(parseViewportPreset('  desktop  ')).toBe('desktop');
    });

    it('returns undefined for invalid presets', () => {
      expect(parseViewportPreset('tablet')).toBe(undefined);
      expect(parseViewportPreset('invalid')).toBe(undefined);
      expect(parseViewportPreset('')).toBe(undefined);
    });

    it('returns undefined for non-strings', () => {
      expect(parseViewportPreset(123)).toBe(undefined);
      expect(parseViewportPreset(null)).toBe(undefined);
      expect(parseViewportPreset(undefined)).toBe(undefined);
      expect(parseViewportPreset({})).toBe(undefined);
    });
  });

  describe('sanitizeViewportSettings', () => {
    it('returns undefined for null/undefined input', () => {
      expect(sanitizeViewportSettings(null)).toBe(undefined);
      expect(sanitizeViewportSettings(undefined)).toBe(undefined);
    });

    it('returns undefined for invalid dimensions', () => {
      expect(sanitizeViewportSettings({ width: 0, height: 1080 })).toBe(undefined);
      expect(sanitizeViewportSettings({ width: 1920, height: 0 })).toBe(undefined);
      expect(sanitizeViewportSettings({ width: -100, height: -100 })).toBe(undefined);
    });

    it('sanitizes valid settings without preset', () => {
      const result = sanitizeViewportSettings({ width: 1920, height: 1080 });
      expect(result).toEqual({ width: 1920, height: 1080 });
    });

    it('sanitizes valid settings with preset', () => {
      const result = sanitizeViewportSettings({ width: 1920, height: 1080, preset: 'desktop' });
      expect(result).toEqual({ width: 1920, height: 1080, preset: 'desktop' });
    });

    it('clamps dimensions', () => {
      const result = sanitizeViewportSettings({ width: 50, height: 20000 });
      expect(result).toEqual({ width: MIN_VIEWPORT_DIMENSION, height: MAX_VIEWPORT_DIMENSION });
    });

    it('ignores invalid preset', () => {
      const result = sanitizeViewportSettings({ width: 1920, height: 1080, preset: 'invalid' as any });
      expect(result).toEqual({ width: 1920, height: 1080 });
    });
  });

  describe('extractExecutionViewport', () => {
    it('returns undefined for non-object input', () => {
      expect(extractExecutionViewport(null)).toBe(undefined);
      expect(extractExecutionViewport(undefined)).toBe(undefined);
      expect(extractExecutionViewport('string')).toBe(undefined);
      expect(extractExecutionViewport(123)).toBe(undefined);
    });

    it('returns undefined when settings is missing', () => {
      expect(extractExecutionViewport({})).toBe(undefined);
      expect(extractExecutionViewport({ other: 'value' })).toBe(undefined);
    });

    it('returns undefined when settings is not an object', () => {
      expect(extractExecutionViewport({ settings: 'string' })).toBe(undefined);
      expect(extractExecutionViewport({ settings: 123 })).toBe(undefined);
    });

    it('extracts from executionViewport', () => {
      const definition = {
        settings: {
          executionViewport: { width: 1920, height: 1080 },
        },
      };
      expect(extractExecutionViewport(definition)).toEqual({ width: 1920, height: 1080 });
    });

    it('extracts from viewport (fallback)', () => {
      const definition = {
        settings: {
          viewport: { width: 1920, height: 1080 },
        },
      };
      expect(extractExecutionViewport(definition)).toEqual({ width: 1920, height: 1080 });
    });

    it('prefers executionViewport over viewport', () => {
      const definition = {
        settings: {
          executionViewport: { width: 1920, height: 1080 },
          viewport: { width: 800, height: 600 },
        },
      };
      expect(extractExecutionViewport(definition)).toEqual({ width: 1920, height: 1080 });
    });

    it('extracts preset from preset field', () => {
      const definition = {
        settings: {
          executionViewport: { width: 1920, height: 1080, preset: 'desktop' },
        },
      };
      expect(extractExecutionViewport(definition)).toEqual({ width: 1920, height: 1080, preset: 'desktop' });
    });

    it('extracts preset from mode field (fallback)', () => {
      const definition = {
        settings: {
          executionViewport: { width: 375, height: 667, mode: 'mobile' },
        },
      };
      expect(extractExecutionViewport(definition)).toEqual({ width: 375, height: 667, preset: 'mobile' });
    });

    it('returns undefined for invalid viewport dimensions', () => {
      const definition = {
        settings: {
          executionViewport: { width: 0, height: 1080 },
        },
      };
      expect(extractExecutionViewport(definition)).toBe(undefined);
    });
  });
});
