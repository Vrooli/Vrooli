/**
 * Unit Tests for Playwright Provider
 *
 * Tests the provider configuration, capabilities, and factory function.
 */

import { describe, beforeEach, afterEach, it, expect, jest } from '@jest/globals';
import {
  createPlaywrightProvider,
  getConfiguredProviderName,
  PROVIDER_CAPABILITIES,
  playwrightProvider,
  logProviderConfig,
} from '../../../src/playwright/provider';
import { RECORDING_TROUBLESHOOTING } from '../../../src/playwright/types';

describe('Playwright Provider', () => {
  const originalEnv = process.env.PLAYWRIGHT_PROVIDER;

  afterEach(() => {
    // Restore original env
    if (originalEnv === undefined) {
      delete process.env.PLAYWRIGHT_PROVIDER;
    } else {
      process.env.PLAYWRIGHT_PROVIDER = originalEnv;
    }
  });

  describe('getConfiguredProviderName', () => {
    it('should return rebrowser-playwright by default', () => {
      delete process.env.PLAYWRIGHT_PROVIDER;
      expect(getConfiguredProviderName()).toBe('rebrowser-playwright');
    });

    it('should return rebrowser-playwright when env is set to rebrowser-playwright', () => {
      process.env.PLAYWRIGHT_PROVIDER = 'rebrowser-playwright';
      expect(getConfiguredProviderName()).toBe('rebrowser-playwright');
    });

    it('should return playwright when env is set to playwright', () => {
      process.env.PLAYWRIGHT_PROVIDER = 'playwright';
      expect(getConfiguredProviderName()).toBe('playwright');
    });

    it('should return default for invalid env value', () => {
      process.env.PLAYWRIGHT_PROVIDER = 'invalid-provider';
      expect(getConfiguredProviderName()).toBe('rebrowser-playwright');
    });
  });

  describe('PROVIDER_CAPABILITIES', () => {
    it('should define capabilities for rebrowser-playwright', () => {
      const caps = PROVIDER_CAPABILITIES['rebrowser-playwright'];
      expect(caps.evaluateIsolated).toBe(true);
      expect(caps.exposeBindingIsolated).toBe(true);
      expect(caps.hasAntiDetection).toBe(true);
    });

    it('should define capabilities for playwright', () => {
      const caps = PROVIDER_CAPABILITIES['playwright'];
      expect(caps.evaluateIsolated).toBe(false);
      expect(caps.exposeBindingIsolated).toBe(false);
      expect(caps.hasAntiDetection).toBe(false);
    });

    it('should have opposite isolation settings between providers', () => {
      // This test documents the key difference that affects recording architecture
      const rebrowser = PROVIDER_CAPABILITIES['rebrowser-playwright'];
      const standard = PROVIDER_CAPABILITIES['playwright'];

      expect(rebrowser.evaluateIsolated).not.toBe(standard.evaluateIsolated);
      expect(rebrowser.exposeBindingIsolated).not.toBe(standard.exposeBindingIsolated);
    });
  });

  describe('createPlaywrightProvider', () => {
    it('should create rebrowser-playwright provider by default', () => {
      delete process.env.PLAYWRIGHT_PROVIDER;
      const provider = createPlaywrightProvider();

      expect(provider.name).toBe('rebrowser-playwright');
      expect(provider.capabilities.evaluateIsolated).toBe(true);
      expect(provider.chromium).toBeDefined();
    });

    it('should create rebrowser-playwright provider when explicitly specified', () => {
      const provider = createPlaywrightProvider('rebrowser-playwright');

      expect(provider.name).toBe('rebrowser-playwright');
      expect(provider.capabilities.evaluateIsolated).toBe(true);
    });

    it('should warn and fallback when playwright is requested but not installed', () => {
      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});

      const provider = createPlaywrightProvider('playwright');

      // Should fallback to rebrowser-playwright since playwright isn't installed
      expect(provider.name).toBe('rebrowser-playwright');
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('Standard playwright requested')
      );

      consoleSpy.mockRestore();
    });
  });

  describe('playwrightProvider singleton', () => {
    it('should be a valid provider instance', () => {
      expect(playwrightProvider.name).toBe('rebrowser-playwright');
      expect(playwrightProvider.chromium).toBeDefined();
      expect(playwrightProvider.capabilities).toBeDefined();
    });

    it('should have launch method on chromium', () => {
      expect(typeof playwrightProvider.chromium.launch).toBe('function');
    });
  });

  describe('logProviderConfig', () => {
    it('should log provider configuration', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation(() => {});

      logProviderConfig();

      expect(consoleSpy).toHaveBeenCalledWith(
        '[playwright-provider] Active configuration:',
        expect.objectContaining({
          name: 'rebrowser-playwright',
          evaluateIsolated: true,
          exposeBindingIsolated: true,
          hasAntiDetection: true,
        })
      );

      consoleSpy.mockRestore();
    });
  });
});

describe('RECORDING_TROUBLESHOOTING', () => {
  it('should have noEvents troubleshooting entry', () => {
    expect(RECORDING_TROUBLESHOOTING.noEvents).toBeDefined();
    expect(RECORDING_TROUBLESHOOTING.noEvents.symptom).toBe('No events captured during recording');
    expect(RECORDING_TROUBLESHOOTING.noEvents.checkList.length).toBeGreaterThan(0);
  });

  it('should have noNavigationEvents troubleshooting entry', () => {
    expect(RECORDING_TROUBLESHOOTING.noNavigationEvents).toBeDefined();
    expect(RECORDING_TROUBLESHOOTING.noNavigationEvents.symptom).toContain('pushState');
    expect(RECORDING_TROUBLESHOOTING.noNavigationEvents.checkList.length).toBeGreaterThan(0);
  });

  it('should have duplicateEvents troubleshooting entry', () => {
    expect(RECORDING_TROUBLESHOOTING.duplicateEvents).toBeDefined();
    expect(RECORDING_TROUBLESHOOTING.duplicateEvents.symptom).toContain('multiple identical events');
  });

  it('should have delayedEvents troubleshooting entry', () => {
    expect(RECORDING_TROUBLESHOOTING.delayedEvents).toBeDefined();
    expect(RECORDING_TROUBLESHOOTING.delayedEvents.symptom).toContain('late or in batches');
  });

  it('should provide actionable checklists', () => {
    // Every troubleshooting entry should have at least 2 checklist items
    for (const [key, entry] of Object.entries(RECORDING_TROUBLESHOOTING)) {
      expect(entry.checkList.length).toBeGreaterThanOrEqual(2);
      for (const item of entry.checkList) {
        // Each item should be a meaningful sentence
        expect(item.length).toBeGreaterThan(20);
      }
    }
  });
});

describe('Provider Architecture Documentation', () => {
  // These tests serve as documentation of the architectural differences

  it('documents that rebrowser-playwright requires HTML injection for scripts', () => {
    const caps = PROVIDER_CAPABILITIES['rebrowser-playwright'];

    // When evaluateIsolated is true, scripts run in ISOLATED context
    // This means addInitScript() won't work for History API wrapping
    // Must use HTML injection via route interception instead
    expect(caps.evaluateIsolated).toBe(true);
  });

  it('documents that rebrowser-playwright requires fetch for event communication', () => {
    const caps = PROVIDER_CAPABILITIES['rebrowser-playwright'];

    // When exposeBindingIsolated is true, bindings only work from ISOLATED context
    // Scripts in MAIN context (injected HTML) cannot call bindings
    // Must use fetch() to a route-intercepted URL instead
    expect(caps.exposeBindingIsolated).toBe(true);
  });

  it('documents that standard playwright allows simpler architecture', () => {
    const caps = PROVIDER_CAPABILITIES['playwright'];

    // When evaluateIsolated is false, addInitScript() works in MAIN context
    // When exposeBindingIsolated is false, bindings work from any context
    // This allows simpler recording architecture without route interception
    expect(caps.evaluateIsolated).toBe(false);
    expect(caps.exposeBindingIsolated).toBe(false);
  });

  it('documents anti-detection trade-offs', () => {
    // rebrowser-playwright has anti-detection, standard does not
    // This is the main reason to use rebrowser-playwright for production
    expect(PROVIDER_CAPABILITIES['rebrowser-playwright'].hasAntiDetection).toBe(true);
    expect(PROVIDER_CAPABILITIES['playwright'].hasAntiDetection).toBe(false);
  });
});
