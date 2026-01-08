import { renderHook, act } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { useProfileSettings } from './useProfileSettings';
import type { BrowserProfile } from '@/domains/recording/types/types';

describe('useProfileSettings', () => {
  const mockInitialProfile: BrowserProfile = {
    preset: 'balanced',
    fingerprint: {
      viewport_width: 1920,
      viewport_height: 1080,
      locale: 'en-US',
    },
    behavior: {
      typing_delay_min: 30,
      typing_delay_max: 80,
    },
    anti_detection: {
      disable_automation_controlled: true,
      patch_navigator_webdriver: true,
    },
    proxy: {
      enabled: false,
    },
    extra_headers: {
      'X-Custom-Header': 'test-value',
    },
  };

  describe('initialization', () => {
    it('initializes with default values when no initialProfile', () => {
      const { result } = renderHook(() => useProfileSettings());

      expect(result.current.preset).toBe('none');
      expect(result.current.fingerprint).toEqual({});
      expect(result.current.behavior).toEqual({});
      expect(result.current.antiDetection).toEqual({});
      expect(result.current.proxy).toEqual({});
      expect(result.current.extraHeaders).toEqual({});
    });

    it('initializes from initialProfile', () => {
      const { result } = renderHook(() =>
        useProfileSettings({ initialProfile: mockInitialProfile })
      );

      expect(result.current.preset).toBe('balanced');
      expect(result.current.fingerprint).toEqual(mockInitialProfile.fingerprint);
      expect(result.current.behavior).toEqual(mockInitialProfile.behavior);
      expect(result.current.antiDetection).toEqual(mockInitialProfile.anti_detection);
      expect(result.current.proxy).toEqual(mockInitialProfile.proxy);
      expect(result.current.extraHeaders).toEqual(mockInitialProfile.extra_headers);
    });
  });

  describe('preset application', () => {
    it('applies stealth preset correctly', () => {
      const { result } = renderHook(() => useProfileSettings());

      act(() => {
        result.current.applyPreset('stealth');
      });

      expect(result.current.preset).toBe('stealth');
      expect(result.current.behavior.typing_delay_min).toBe(50);
      expect(result.current.behavior.typing_delay_max).toBe(150);
      expect(result.current.behavior.mouse_movement_style).toBe('natural');
      expect(result.current.antiDetection.disable_automation_controlled).toBe(true);
      expect(result.current.antiDetection.patch_canvas).toBe(true);
    });

    it('applies fast preset correctly', () => {
      const { result } = renderHook(() => useProfileSettings());

      act(() => {
        result.current.applyPreset('fast');
      });

      expect(result.current.preset).toBe('fast');
      expect(result.current.behavior.typing_delay_min).toBe(10);
      expect(result.current.behavior.mouse_movement_style).toBe('linear');
      expect(result.current.antiDetection.patch_canvas).toBe(false);
    });

    it('merges preset with existing values', () => {
      const { result } = renderHook(() => useProfileSettings());

      // Set a custom value first
      act(() => {
        result.current.updateBehavior('scroll_style', 'stepped');
      });

      // Apply preset - should merge, not replace entirely
      act(() => {
        result.current.applyPreset('balanced');
      });

      // Preset value should override
      expect(result.current.behavior.scroll_style).toBe('smooth');
      // Preset's typing values should be set
      expect(result.current.behavior.typing_delay_min).toBe(30);
    });
  });

  describe('individual updates', () => {
    it('updates fingerprint property', () => {
      const { result } = renderHook(() => useProfileSettings());

      act(() => {
        result.current.updateFingerprint('viewport_width', 1280);
      });

      expect(result.current.fingerprint.viewport_width).toBe(1280);
    });

    it('updates behavior property', () => {
      const { result } = renderHook(() => useProfileSettings());

      act(() => {
        result.current.updateBehavior('typing_delay_min', 100);
      });

      expect(result.current.behavior.typing_delay_min).toBe(100);
    });

    it('updates antiDetection property', () => {
      const { result } = renderHook(() => useProfileSettings());

      act(() => {
        result.current.updateAntiDetection('disable_webrtc', true);
      });

      expect(result.current.antiDetection.disable_webrtc).toBe(true);
    });

    it('updates proxy property', () => {
      const { result } = renderHook(() => useProfileSettings());

      act(() => {
        result.current.updateProxy('enabled', true);
        result.current.updateProxy('server', 'http://proxy.example.com:8080');
      });

      expect(result.current.proxy.enabled).toBe(true);
      expect(result.current.proxy.server).toBe('http://proxy.example.com:8080');
    });

    it('preserves existing properties when updating', () => {
      const { result } = renderHook(() => useProfileSettings());

      act(() => {
        result.current.updateFingerprint('viewport_width', 1920);
        result.current.updateFingerprint('viewport_height', 1080);
      });

      expect(result.current.fingerprint.viewport_width).toBe(1920);
      expect(result.current.fingerprint.viewport_height).toBe(1080);
    });
  });

  describe('extra headers', () => {
    it('adds header', () => {
      const { result } = renderHook(() => useProfileSettings());

      act(() => {
        result.current.addExtraHeader('X-Test', 'value');
      });

      expect(result.current.extraHeaders['X-Test']).toBe('value');
    });

    it('updates header key and value', () => {
      const { result } = renderHook(() => useProfileSettings());

      act(() => {
        result.current.addExtraHeader('X-Old', 'old-value');
      });

      act(() => {
        result.current.updateExtraHeader('X-Old', 'X-New', 'new-value');
      });

      expect(result.current.extraHeaders['X-Old']).toBeUndefined();
      expect(result.current.extraHeaders['X-New']).toBe('new-value');
    });

    it('removes header', () => {
      const { result } = renderHook(() => useProfileSettings());

      act(() => {
        result.current.addExtraHeader('X-Test', 'value');
        result.current.addExtraHeader('X-Keep', 'keep');
      });

      act(() => {
        result.current.removeExtraHeader('X-Test');
      });

      expect(result.current.extraHeaders['X-Test']).toBeUndefined();
      expect(result.current.extraHeaders['X-Keep']).toBe('keep');
    });
  });

  describe('change detection', () => {
    it('reports no changes initially', () => {
      const { result } = renderHook(() =>
        useProfileSettings({ initialProfile: mockInitialProfile })
      );

      expect(result.current.hasChanges).toBe(false);
    });

    it('reports changes after preset update', () => {
      const { result } = renderHook(() =>
        useProfileSettings({ initialProfile: mockInitialProfile })
      );

      act(() => {
        result.current.applyPreset('stealth');
      });

      expect(result.current.hasChanges).toBe(true);
    });

    it('reports changes after fingerprint update', () => {
      const { result } = renderHook(() =>
        useProfileSettings({ initialProfile: mockInitialProfile })
      );

      act(() => {
        result.current.updateFingerprint('viewport_width', 1280);
      });

      expect(result.current.hasChanges).toBe(true);
    });

    it('reports changes after adding extra header', () => {
      const { result } = renderHook(() =>
        useProfileSettings({ initialProfile: mockInitialProfile })
      );

      act(() => {
        result.current.addExtraHeader('X-New', 'value');
      });

      expect(result.current.hasChanges).toBe(true);
    });

    it('reports no changes after reset', () => {
      const { result } = renderHook(() =>
        useProfileSettings({ initialProfile: mockInitialProfile })
      );

      act(() => {
        result.current.applyPreset('stealth');
        result.current.updateFingerprint('viewport_width', 1280);
      });

      expect(result.current.hasChanges).toBe(true);

      act(() => {
        result.current.resetToInitial();
      });

      expect(result.current.hasChanges).toBe(false);
    });

    it('reports no changes when no initial profile and defaults unchanged', () => {
      const { result } = renderHook(() => useProfileSettings());

      expect(result.current.hasChanges).toBe(false);
    });

    it('reports changes when no initial profile and values changed', () => {
      const { result } = renderHook(() => useProfileSettings());

      act(() => {
        result.current.applyPreset('stealth');
      });

      expect(result.current.hasChanges).toBe(true);
    });
  });

  describe('getCurrentProfile', () => {
    it('returns current profile state', () => {
      const { result } = renderHook(() => useProfileSettings());

      act(() => {
        result.current.applyPreset('balanced');
        result.current.updateFingerprint('viewport_width', 1920);
        result.current.addExtraHeader('X-Test', 'value');
      });

      const profile = result.current.getCurrentProfile();

      expect(profile.preset).toBe('balanced');
      expect(profile.fingerprint?.viewport_width).toBe(1920);
      expect(profile.extra_headers?.['X-Test']).toBe('value');
    });

    it('omits empty objects from profile', () => {
      const { result } = renderHook(() => useProfileSettings());

      const profile = result.current.getCurrentProfile();

      // Empty objects should be undefined, not {}
      expect(profile.fingerprint).toBeUndefined();
      expect(profile.behavior).toBeUndefined();
      expect(profile.anti_detection).toBeUndefined();
      expect(profile.proxy).toBeUndefined();
      expect(profile.extra_headers).toBeUndefined();
    });
  });

  describe('callback stability', () => {
    it('maintains stable callback references', () => {
      const { result, rerender } = renderHook(() => useProfileSettings());

      const refs = {
        applyPreset: result.current.applyPreset,
        updateFingerprint: result.current.updateFingerprint,
        updateBehavior: result.current.updateBehavior,
        addExtraHeader: result.current.addExtraHeader,
        resetToInitial: result.current.resetToInitial,
      };

      rerender();

      expect(result.current.applyPreset).toBe(refs.applyPreset);
      expect(result.current.updateFingerprint).toBe(refs.updateFingerprint);
      expect(result.current.updateBehavior).toBe(refs.updateBehavior);
      expect(result.current.addExtraHeader).toBe(refs.addExtraHeader);
      // resetToInitial depends on initialProfile, so it may change
    });
  });
});
