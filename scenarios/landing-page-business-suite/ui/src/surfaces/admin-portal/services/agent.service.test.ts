import { describe, it, expect } from 'vitest';
import {
  DEFAULT_AGENT_FORM,
  validateBrief,
  parseAssets,
  validateAssets,
  isFormValid,
  DEFAULT_SCENARIO_ID,
  type AgentFormState,
} from './agent.service';

describe('agent.service', () => {
  describe('DEFAULT_AGENT_FORM', () => {
    it('has empty brief', () => {
      expect(DEFAULT_AGENT_FORM.brief).toBe('');
    });

    it('has empty assets', () => {
      expect(DEFAULT_AGENT_FORM.assets).toBe('');
    });

    it('has preview enabled by default', () => {
      expect(DEFAULT_AGENT_FORM.preview).toBe(true);
    });
  });

  describe('DEFAULT_SCENARIO_ID', () => {
    it('is landing-page', () => {
      expect(DEFAULT_SCENARIO_ID).toBe('landing-page');
    });
  });

  describe('validateBrief', () => {
    it('returns error for empty brief', () => {
      expect(validateBrief('')).toBe('Please provide a brief for the agent');
    });

    it('returns error for whitespace-only brief', () => {
      expect(validateBrief('   ')).toBe('Please provide a brief for the agent');
    });

    it('returns null for valid brief', () => {
      expect(validateBrief('Make the hero section more compelling')).toBeNull();
    });

    it('returns null for brief with leading/trailing whitespace', () => {
      expect(validateBrief('  Valid brief here  ')).toBeNull();
    });
  });

  describe('parseAssets', () => {
    it('parses newline-separated URLs', () => {
      const input = 'https://example.com/logo.png\nhttps://example.com/hero.jpg';
      const result = parseAssets(input);
      expect(result).toEqual([
        'https://example.com/logo.png',
        'https://example.com/hero.jpg',
      ]);
    });

    it('trims whitespace from URLs', () => {
      const input = '  https://example.com/logo.png  \n  https://example.com/hero.jpg  ';
      const result = parseAssets(input);
      expect(result).toEqual([
        'https://example.com/logo.png',
        'https://example.com/hero.jpg',
      ]);
    });

    it('filters out empty lines', () => {
      const input = 'https://example.com/logo.png\n\n\nhttps://example.com/hero.jpg\n';
      const result = parseAssets(input);
      expect(result).toEqual([
        'https://example.com/logo.png',
        'https://example.com/hero.jpg',
      ]);
    });

    it('returns empty array for empty string', () => {
      expect(parseAssets('')).toEqual([]);
    });

    it('returns empty array for whitespace-only string', () => {
      expect(parseAssets('   \n   \n   ')).toEqual([]);
    });

    it('handles single URL', () => {
      const result = parseAssets('https://example.com/image.png');
      expect(result).toEqual(['https://example.com/image.png']);
    });
  });

  describe('validateAssets', () => {
    it('returns null for empty array', () => {
      expect(validateAssets([])).toBeNull();
    });

    it('returns null for valid URLs', () => {
      const assets = [
        'https://example.com/logo.png',
        'https://cdn.example.org/image.jpg',
        'http://localhost:3000/test.png',
      ];
      expect(validateAssets(assets)).toBeNull();
    });

    it('returns error for invalid URL', () => {
      const assets = ['https://example.com/logo.png', 'not-a-valid-url'];
      const result = validateAssets(assets);
      expect(result).toBe('Invalid URL: not-a-valid-url');
    });

    it('returns error for first invalid URL found', () => {
      const assets = ['invalid1', 'invalid2'];
      const result = validateAssets(assets);
      expect(result).toBe('Invalid URL: invalid1');
    });

    it('accepts various valid URL formats', () => {
      const assets = [
        'https://example.com',
        'https://example.com/path/to/image.png',
        'https://example.com/path?query=value',
        'https://sub.domain.example.com/path',
        'http://localhost:8080/test',
      ];
      expect(validateAssets(assets)).toBeNull();
    });
  });

  describe('isFormValid', () => {
    it('returns false for empty brief', () => {
      const form: AgentFormState = {
        brief: '',
        assets: '',
        preview: true,
      };
      expect(isFormValid(form)).toBe(false);
    });

    it('returns false for whitespace-only brief', () => {
      const form: AgentFormState = {
        brief: '   ',
        assets: '',
        preview: true,
      };
      expect(isFormValid(form)).toBe(false);
    });

    it('returns true for valid brief', () => {
      const form: AgentFormState = {
        brief: 'Make the hero section more compelling',
        assets: '',
        preview: true,
      };
      expect(isFormValid(form)).toBe(true);
    });

    it('returns true with assets', () => {
      const form: AgentFormState = {
        brief: 'Add logo',
        assets: 'https://example.com/logo.png',
        preview: false,
      };
      expect(isFormValid(form)).toBe(true);
    });

    it('allows preview to be false', () => {
      const form: AgentFormState = {
        brief: 'Test brief',
        assets: '',
        preview: false,
      };
      expect(isFormValid(form)).toBe(true);
    });
  });
});
