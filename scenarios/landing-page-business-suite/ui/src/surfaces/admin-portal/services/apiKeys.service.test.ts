import { describe, it, expect } from 'vitest';
import type { APIKey } from '../../../shared/api';
import {
  DEFAULT_NEW_KEY_FORM,
  validateNewKeyForm,
  getProviderLabel,
  getProviderDescription,
  getAvailableProviders,
  hasAvailableProviders,
  removeTestResult,
  addTestResult,
  PROVIDER_OPTIONS,
  type NewKeyFormState,
  type KeyTestResult,
} from './apiKeys.service';

const createMockKey = (overrides: Partial<APIKey> = {}): APIKey => ({
  id: '1',
  provider: 'anthropic',
  key_hint: 'sk-...abc',
  is_active: true,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  ...overrides,
});

describe('apiKeys.service', () => {
  describe('DEFAULT_NEW_KEY_FORM', () => {
    it('has empty fields', () => {
      expect(DEFAULT_NEW_KEY_FORM.provider).toBe('');
      expect(DEFAULT_NEW_KEY_FORM.keyValue).toBe('');
    });
  });

  describe('PROVIDER_OPTIONS', () => {
    it('has options', () => {
      expect(PROVIDER_OPTIONS.length).toBeGreaterThan(0);
    });

    it('each option has value, label, and description', () => {
      PROVIDER_OPTIONS.forEach((option) => {
        expect(option.value).toBeTruthy();
        expect(option.label).toBeTruthy();
        expect(option.description).toBeTruthy();
      });
    });
  });

  describe('validateNewKeyForm', () => {
    it('returns error for empty provider', () => {
      const form: NewKeyFormState = {
        provider: '',
        keyValue: 'sk-test-key',
      };
      expect(validateNewKeyForm(form)).toBe('Provider is required');
    });

    it('returns error for empty key value', () => {
      const form: NewKeyFormState = {
        provider: 'anthropic',
        keyValue: '',
      };
      expect(validateNewKeyForm(form)).toBe('API key is required');
    });

    it('returns null for valid form', () => {
      const form: NewKeyFormState = {
        provider: 'anthropic',
        keyValue: 'sk-test-key',
      };
      expect(validateNewKeyForm(form)).toBeNull();
    });
  });

  describe('getProviderLabel', () => {
    it('returns label for known provider', () => {
      const firstProvider = PROVIDER_OPTIONS[0];
      expect(getProviderLabel(firstProvider.value)).toBe(firstProvider.label);
    });

    it('returns provider string for unknown provider', () => {
      expect(getProviderLabel('unknown-provider')).toBe('unknown-provider');
    });
  });

  describe('getProviderDescription', () => {
    it('returns description for known provider', () => {
      const firstProvider = PROVIDER_OPTIONS[0];
      expect(getProviderDescription(firstProvider.value)).toBe(firstProvider.description);
    });

    it('returns empty string for unknown provider', () => {
      expect(getProviderDescription('unknown-provider')).toBe('');
    });
  });

  describe('getAvailableProviders', () => {
    it('returns all providers when none configured', () => {
      const result = getAvailableProviders([]);
      expect(result.length).toBe(PROVIDER_OPTIONS.length);
    });

    it('excludes configured providers', () => {
      const configured = [createMockKey({ provider: PROVIDER_OPTIONS[0].value })];
      const result = getAvailableProviders(configured);
      expect(result.length).toBe(PROVIDER_OPTIONS.length - 1);
      expect(result.find((p) => p.value === PROVIDER_OPTIONS[0].value)).toBeUndefined();
    });

    it('returns empty when all configured', () => {
      const configured = PROVIDER_OPTIONS.map((p, i) =>
        createMockKey({ id: String(i), provider: p.value })
      );
      const result = getAvailableProviders(configured);
      expect(result.length).toBe(0);
    });
  });

  describe('hasAvailableProviders', () => {
    it('returns true when providers available', () => {
      expect(hasAvailableProviders([])).toBe(true);
    });

    it('returns false when all configured', () => {
      const configured = PROVIDER_OPTIONS.map((p, i) =>
        createMockKey({ id: String(i), provider: p.value })
      );
      expect(hasAvailableProviders(configured)).toBe(false);
    });
  });

  describe('removeTestResult', () => {
    it('removes result for provider', () => {
      const results: Record<string, KeyTestResult> = {
        anthropic: { success: true, message: 'OK' },
        openai: { success: true, message: 'OK' },
      };
      const next = removeTestResult(results, 'anthropic');
      expect(next.anthropic).toBeUndefined();
      expect(next.openai).toBeDefined();
    });

    it('does not mutate original', () => {
      const results: Record<string, KeyTestResult> = {
        anthropic: { success: true, message: 'OK' },
      };
      removeTestResult(results, 'anthropic');
      expect(results.anthropic).toBeDefined();
    });

    it('handles missing provider', () => {
      const results: Record<string, KeyTestResult> = {
        anthropic: { success: true, message: 'OK' },
      };
      const next = removeTestResult(results, 'openai');
      expect(Object.keys(next).length).toBe(1);
    });
  });

  describe('addTestResult', () => {
    it('adds new result', () => {
      const results: Record<string, KeyTestResult> = {};
      const next = addTestResult(results, 'anthropic', { success: true, message: 'OK' });
      expect(next.anthropic).toEqual({ success: true, message: 'OK' });
    });

    it('updates existing result', () => {
      const results: Record<string, KeyTestResult> = {
        anthropic: { success: false, message: 'Failed' },
      };
      const next = addTestResult(results, 'anthropic', { success: true, message: 'OK' });
      expect(next.anthropic).toEqual({ success: true, message: 'OK' });
    });

    it('does not mutate original', () => {
      const results: Record<string, KeyTestResult> = {};
      addTestResult(results, 'anthropic', { success: true, message: 'OK' });
      expect(results.anthropic).toBeUndefined();
    });
  });
});
