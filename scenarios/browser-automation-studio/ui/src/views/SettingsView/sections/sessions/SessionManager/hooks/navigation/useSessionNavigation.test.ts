import { renderHook, act } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { useSessionNavigation } from './useSessionNavigation';

describe('useSessionNavigation', () => {
  describe('initialization', () => {
    it('initializes with default section (presets) when no initialSection provided', () => {
      const { result } = renderHook(() => useSessionNavigation());

      expect(result.current.activeSection).toBe('presets');
    });

    it('initializes with provided initialSection', () => {
      const { result } = renderHook(() =>
        useSessionNavigation({ initialSection: 'fingerprint' })
      );

      expect(result.current.activeSection).toBe('fingerprint');
    });

    it('initializes with storage section when provided', () => {
      const { result } = renderHook(() =>
        useSessionNavigation({ initialSection: 'cookies' })
      );

      expect(result.current.activeSection).toBe('cookies');
    });
  });

  describe('setActiveSection', () => {
    it('updates active section', () => {
      const { result } = renderHook(() => useSessionNavigation());

      act(() => {
        result.current.setActiveSection('behavior');
      });

      expect(result.current.activeSection).toBe('behavior');
    });

    it('can switch between settings and storage sections', () => {
      const { result } = renderHook(() => useSessionNavigation());

      act(() => {
        result.current.setActiveSection('cookies');
      });
      expect(result.current.activeSection).toBe('cookies');

      act(() => {
        result.current.setActiveSection('fingerprint');
      });
      expect(result.current.activeSection).toBe('fingerprint');
    });
  });

  describe('callback stability', () => {
    it('setActiveSection maintains stable reference', () => {
      const { result, rerender } = renderHook(() => useSessionNavigation());

      const firstRef = result.current.setActiveSection;
      rerender();
      const secondRef = result.current.setActiveSection;

      expect(firstRef).toBe(secondRef);
    });
  });
});
