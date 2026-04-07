import { describe, it, expect, afterEach } from 'vitest';
import {
  DEFAULT_SPATIAL_FOCUS_CSS,
  injectSpatialStyles,
  removeSpatialStyles,
} from '../spatialNavStyles.js';

describe('spatialNavStyles', () => {
  afterEach(() => {
    // Clean up any injected styles.
    document.querySelectorAll('style[data-spatial-styles]').forEach((el) => el.remove());
  });

  it('injects a <style> element with data-spatial-styles attribute', () => {
    const style = injectSpatialStyles();
    expect(style).toBeInstanceOf(HTMLStyleElement);
    expect(style.getAttribute('data-spatial-styles')).toBe('');
    expect(style.textContent).toBe(DEFAULT_SPATIAL_FOCUS_CSS);
    expect(document.head.contains(style)).toBe(true);
  });

  it('returns the same element on duplicate calls (no duplicates)', () => {
    const style1 = injectSpatialStyles();
    const style2 = injectSpatialStyles();
    expect(style1).toBe(style2);

    const all = document.querySelectorAll('style[data-spatial-styles]');
    expect(all.length).toBe(1);
  });

  it('removeSpatialStyles removes the element from the DOM', () => {
    const style = injectSpatialStyles();
    expect(document.head.contains(style)).toBe(true);

    removeSpatialStyles(style);
    expect(document.head.contains(style)).toBe(false);
  });

  it('DEFAULT_SPATIAL_FOCUS_CSS contains cursor:none and focus ring', () => {
    expect(DEFAULT_SPATIAL_FOCUS_CSS).toContain('cursor: none');
    expect(DEFAULT_SPATIAL_FOCUS_CSS).toContain('data-spatial-focus');
    expect(DEFAULT_SPATIAL_FOCUS_CSS).toContain('outline');
  });
});
