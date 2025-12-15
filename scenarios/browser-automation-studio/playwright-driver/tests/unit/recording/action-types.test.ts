/**
 * Tests for Recording Action Types (proto-first)
 *
 * These tests ensure we treat proto enums as the source-of-truth and only
 * normalize from browser-provided strings at the boundary.
 */
import {
  SELECTOR_OPTIONAL_ACTIONS,
  actionTypeToString,
  calculateActionConfidence,
  getSupportedActionTypes,
  isSelectorOptional,
  isValidActionType,
  normalizeToProtoActionType,
} from '../../../src/recording/action-types';
import { ActionType } from '../../../src/proto';

describe('Recording Action Types (proto-first)', () => {
  describe('normalizeToProtoActionType', () => {
    it('maps known action strings to proto enums', () => {
      expect(normalizeToProtoActionType('click')).toBe(ActionType.CLICK);
      expect(normalizeToProtoActionType('type')).toBe(ActionType.INPUT);
      expect(normalizeToProtoActionType('scroll')).toBe(ActionType.SCROLL);
      expect(normalizeToProtoActionType('navigate')).toBe(ActionType.NAVIGATE);
      expect(normalizeToProtoActionType('select')).toBe(ActionType.SELECT);
      expect(normalizeToProtoActionType('hover')).toBe(ActionType.HOVER);
      expect(normalizeToProtoActionType('focus')).toBe(ActionType.FOCUS);
      expect(normalizeToProtoActionType('blur')).toBe(ActionType.BLUR);
      expect(normalizeToProtoActionType('keypress')).toBe(ActionType.KEYBOARD);
    });

    it('handles aliases and casing', () => {
      expect(normalizeToProtoActionType('INPUT')).toBe(ActionType.INPUT);
      expect(normalizeToProtoActionType('change')).toBe(ActionType.SELECT);
      expect(normalizeToProtoActionType('keydown')).toBe(ActionType.KEYBOARD);
      expect(normalizeToProtoActionType('keyup')).toBe(ActionType.KEYBOARD);
      expect(normalizeToProtoActionType('dblclick')).toBe(ActionType.CLICK);
    });

    it('defaults to CLICK for unknown action strings', () => {
      expect(normalizeToProtoActionType('unknown')).toBe(ActionType.CLICK);
      expect(normalizeToProtoActionType('')).toBe(ActionType.CLICK);
      expect(normalizeToProtoActionType('gibberish')).toBe(ActionType.CLICK);
    });
  });

  describe('actionTypeToString', () => {
    it('returns stable names for enums', () => {
      expect(actionTypeToString(ActionType.CLICK)).toBe('CLICK');
      expect(actionTypeToString(ActionType.INPUT)).toBe('INPUT');
      expect(actionTypeToString(ActionType.NAVIGATE)).toBe('NAVIGATE');
    });
  });

  describe('isValidActionType', () => {
    it('treats UNSPECIFIED as invalid', () => {
      expect(isValidActionType(ActionType.UNSPECIFIED)).toBe(false);
      expect(isValidActionType(ActionType.CLICK)).toBe(true);
    });
  });

  describe('isSelectorOptional / SELECTOR_OPTIONAL_ACTIONS', () => {
    it('marks known selector-optional actions as optional', () => {
      expect(isSelectorOptional(ActionType.SCROLL)).toBe(true);
      expect(isSelectorOptional(ActionType.NAVIGATE)).toBe(true);
      expect(SELECTOR_OPTIONAL_ACTIONS.has(ActionType.SCROLL)).toBe(true);
      expect(SELECTOR_OPTIONAL_ACTIONS.has(ActionType.NAVIGATE)).toBe(true);
    });

    it('marks click/input as requiring selector', () => {
      expect(isSelectorOptional(ActionType.CLICK)).toBe(false);
      expect(isSelectorOptional(ActionType.INPUT)).toBe(false);
    });
  });

  describe('calculateActionConfidence', () => {
    it('returns 1 for selector-optional actions', () => {
      expect(calculateActionConfidence(ActionType.SCROLL)).toBe(1);
      expect(calculateActionConfidence(ActionType.NAVIGATE)).toBe(1);
    });

    it('returns 0.5 when no selector is provided', () => {
      expect(calculateActionConfidence(ActionType.CLICK)).toBe(0.5);
      expect(calculateActionConfidence(ActionType.CLICK, undefined)).toBe(0.5);
    });

    it('returns 0.5 when selector has no candidates', () => {
      expect(calculateActionConfidence(ActionType.CLICK, { primary: 'div', candidates: [] })).toBe(0.5);
    });

    it('uses candidate confidence when available', () => {
      const selector = {
        primary: '[data-testid="submit"]',
        candidates: [{ type: 'data-testid', value: '[data-testid="submit"]', confidence: 0.98 }],
      };
      expect(calculateActionConfidence(ActionType.CLICK, selector)).toBeGreaterThanOrEqual(0.85);
    });

    it('boosts confidence for stable selector types', () => {
      const selector = {
        primary: '#main-button',
        candidates: [{ type: 'id', value: '#main-button', confidence: 0.7 }],
      };
      expect(calculateActionConfidence(ActionType.CLICK, selector)).toBeGreaterThanOrEqual(0.85);
    });
  });

  describe('getSupportedActionTypes', () => {
    it('includes core action types', () => {
      const supported = new Set(getSupportedActionTypes());
      expect(supported.has(ActionType.NAVIGATE)).toBe(true);
      expect(supported.has(ActionType.CLICK)).toBe(true);
      expect(supported.has(ActionType.INPUT)).toBe(true);
      expect(supported.has(ActionType.SCROLL)).toBe(true);
      expect(supported.has(ActionType.SELECT)).toBe(true);
    });
  });
});
