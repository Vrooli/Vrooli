/**
 * Tests for Recording Action Types
 *
 * These tests verify the centralized action type handling works correctly.
 * They serve as regression guards for the primary change axis in recording.
 */
import {
  ACTION_TYPES,
  normalizeActionType,
  toRecordedActionKind,
  isValidActionType,
  isSelectorOptional,
  buildTypedActionPayload,
  calculateActionConfidence,
} from '../../../src/recording/action-types';
import type { RawBrowserEvent } from '../../../src/recording/types';

describe('Recording Action Types', () => {
  describe('ACTION_TYPES', () => {
    it('should contain all expected action types', () => {
      expect(ACTION_TYPES).toContain('click');
      expect(ACTION_TYPES).toContain('type');
      expect(ACTION_TYPES).toContain('scroll');
      expect(ACTION_TYPES).toContain('navigate');
      expect(ACTION_TYPES).toContain('select');
      expect(ACTION_TYPES).toContain('hover');
      expect(ACTION_TYPES).toContain('focus');
      expect(ACTION_TYPES).toContain('blur');
      expect(ACTION_TYPES).toContain('keypress');
    });
  });

  describe('normalizeActionType', () => {
    it('should normalize known action types', () => {
      expect(normalizeActionType('click')).toBe('click');
      expect(normalizeActionType('CLICK')).toBe('click');
      expect(normalizeActionType('Click')).toBe('click');
      expect(normalizeActionType('type')).toBe('type');
      expect(normalizeActionType('navigate')).toBe('navigate');
    });

    it('should handle aliases correctly', () => {
      expect(normalizeActionType('input')).toBe('type');
      expect(normalizeActionType('INPUT')).toBe('type');
      expect(normalizeActionType('change')).toBe('select');
      expect(normalizeActionType('keydown')).toBe('keypress');
      expect(normalizeActionType('keyup')).toBe('keypress');
    });

    it('should default to click for unknown types', () => {
      expect(normalizeActionType('unknown')).toBe('click');
      expect(normalizeActionType('')).toBe('click');
      expect(normalizeActionType('gibberish')).toBe('click');
    });
  });

  describe('toRecordedActionKind', () => {
    it('should map action types to proto kinds', () => {
      expect(toRecordedActionKind('navigate')).toBe('RECORDED_ACTION_TYPE_NAVIGATE');
      expect(toRecordedActionKind('click')).toBe('RECORDED_ACTION_TYPE_CLICK');
      expect(toRecordedActionKind('hover')).toBe('RECORDED_ACTION_TYPE_CLICK');
      expect(toRecordedActionKind('focus')).toBe('RECORDED_ACTION_TYPE_CLICK');
      expect(toRecordedActionKind('blur')).toBe('RECORDED_ACTION_TYPE_CLICK');
      expect(toRecordedActionKind('type')).toBe('RECORDED_ACTION_TYPE_INPUT');
      expect(toRecordedActionKind('keypress')).toBe('RECORDED_ACTION_TYPE_INPUT');
      expect(toRecordedActionKind('select')).toBe('RECORDED_ACTION_TYPE_INPUT');
      expect(toRecordedActionKind('scroll')).toBe('RECORDED_ACTION_TYPE_UNSPECIFIED');
    });
  });

  describe('isValidActionType', () => {
    it('should return true for valid action types', () => {
      expect(isValidActionType('click')).toBe(true);
      expect(isValidActionType('type')).toBe(true);
      expect(isValidActionType('navigate')).toBe(true);
    });

    it('should return false for invalid action types', () => {
      expect(isValidActionType('invalid')).toBe(false);
      expect(isValidActionType('')).toBe(false);
      expect(isValidActionType('CLICK')).toBe(false); // Case-sensitive
    });
  });

  describe('isSelectorOptional', () => {
    it('should return true for scroll and navigate', () => {
      expect(isSelectorOptional('scroll')).toBe(true);
      expect(isSelectorOptional('navigate')).toBe(true);
    });

    it('should return false for click, type, and other actions', () => {
      expect(isSelectorOptional('click')).toBe(false);
      expect(isSelectorOptional('type')).toBe(false);
      expect(isSelectorOptional('hover')).toBe(false);
      expect(isSelectorOptional('focus')).toBe(false);
    });
  });

  describe('buildTypedActionPayload', () => {
    it('should build navigate payload', () => {
      const raw: RawBrowserEvent = {
        actionType: 'navigate',
        timestamp: Date.now(),
        selector: { primary: '', candidates: [] },
        elementMeta: { tagName: 'body', isVisible: true, isEnabled: true },
        url: 'https://example.com',
        payload: { targetUrl: 'https://example.com/page' },
      };

      const result = buildTypedActionPayload('RECORDED_ACTION_TYPE_NAVIGATE', raw);
      expect(result).toHaveProperty('navigate');
      expect((result as { navigate: { url: string } }).navigate.url).toBe('https://example.com/page');
    });

    it('should build click payload', () => {
      const raw: RawBrowserEvent = {
        actionType: 'click',
        timestamp: Date.now(),
        selector: { primary: 'button.submit', candidates: [] },
        elementMeta: { tagName: 'button', isVisible: true, isEnabled: true },
        url: 'https://example.com',
        payload: { button: 'left', clickCount: 1 },
      };

      const result = buildTypedActionPayload('RECORDED_ACTION_TYPE_CLICK', raw);
      expect(result).toHaveProperty('click');
      expect((result as { click: { selector?: string } }).click.selector).toBe('button.submit');
    });

    it('should build input payload', () => {
      const raw: RawBrowserEvent = {
        actionType: 'type',
        timestamp: Date.now(),
        selector: { primary: 'input#email', candidates: [] },
        elementMeta: { tagName: 'input', isVisible: true, isEnabled: true },
        url: 'https://example.com',
        payload: { text: 'user@example.com' },
      };

      const result = buildTypedActionPayload('RECORDED_ACTION_TYPE_INPUT', raw);
      expect(result).toHaveProperty('input');
      expect((result as { input: { value: string } }).input.value).toBe('user@example.com');
    });

    it('should return undefined for unknown kinds', () => {
      const raw: RawBrowserEvent = {
        actionType: 'unknown',
        timestamp: Date.now(),
        selector: { primary: '', candidates: [] },
        elementMeta: { tagName: 'div', isVisible: true, isEnabled: true },
        url: 'https://example.com',
      };

      const result = buildTypedActionPayload('RECORDED_ACTION_TYPE_UNSPECIFIED', raw);
      expect(result).toBeUndefined();
    });
  });

  describe('calculateActionConfidence', () => {
    it('should return 1 for selector-optional actions', () => {
      expect(calculateActionConfidence('scroll')).toBe(1);
      expect(calculateActionConfidence('navigate')).toBe(1);
    });

    it('should return 0.5 when no selector is provided', () => {
      expect(calculateActionConfidence('click')).toBe(0.5);
      expect(calculateActionConfidence('click', undefined)).toBe(0.5);
    });

    it('should return 0.5 when selector has no candidates', () => {
      expect(calculateActionConfidence('click', { primary: 'div', candidates: [] })).toBe(0.5);
    });

    it('should use candidate confidence when available', () => {
      const selector = {
        primary: '[data-testid="submit"]',
        candidates: [{ type: 'data-testid', value: '[data-testid="submit"]', confidence: 0.98 }],
      };
      expect(calculateActionConfidence('click', selector)).toBeGreaterThanOrEqual(0.85);
    });

    it('should boost confidence for stable selector types', () => {
      const selector = {
        primary: '#main-button',
        candidates: [{ type: 'id', value: '#main-button', confidence: 0.7 }],
      };
      // Strong types (id, data-testid, aria, data-attr) get boosted to at least 0.85
      expect(calculateActionConfidence('click', selector)).toBeGreaterThanOrEqual(0.85);
    });
  });
});
