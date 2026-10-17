import { describe, expect, it } from 'vitest';
import {
  ACTION_TYPES,
  actionTypeToNodeType,
  buildActionDefinition,
  getActionParamsFieldName,
  nodeTypeToActionType,
} from '../domains/workflows/utils/normalizers';

describe('action type conversion', () => {
  it('round-trips every generated proto action through the node vocabulary', () => {
    for (const actionType of Object.values(ACTION_TYPES)) {
      if (actionType === ACTION_TYPES.UNSPECIFIED) continue;
      expect(nodeTypeToActionType(actionTypeToNodeType(actionType))).toBe(actionType);
    }
  });

  it('keeps unknown and unspecified values on the unknown fallback', () => {
    expect(actionTypeToNodeType(ACTION_TYPES.UNSPECIFIED)).toBe('unknown');
    expect(actionTypeToNodeType('ACTION_TYPE_FUTURE' as never)).toBe('unknown');
  });

  it('builds params under the proto JSON field selected by the generated oneof', () => {
    const action = buildActionDefinition('select', {
      selector: '#country',
      value: 'US',
      label: 'Choose country',
    });

    expect(action.type).toBe(ACTION_TYPES.SELECT);
    expect(action.selectOption).toMatchObject({ selector: '#country', value: 'US' });
    expect(action.metadata).toEqual({ label: 'Choose country' });
  });

  it.each([
    ['goto', ACTION_TYPES.NAVIGATE],
    ['fill', ACTION_TYPES.INPUT],
    ['use_variable', ACTION_TYPES.SET_VARIABLE],
    ['drag-drop', ACTION_TYPES.DRAG_DROP],
    ['set_cookie', ACTION_TYPES.COOKIE_STORAGE],
  ])('preserves the %s legacy node alias', (nodeType, actionType) => {
    expect(nodeTypeToActionType(nodeType)).toBe(actionType);
  });
});


describe('generated action oneof field lookup', () => {
  it.each([
    [ACTION_TYPES.SELECT, 'selectOption'],
    [ACTION_TYPES.UPLOAD_FILE, 'uploadFile'],
    [ACTION_TYPES.SET_VARIABLE, 'setVariable'],
    [ACTION_TYPES.NETWORK_MOCK, 'networkMock'],
  ])('maps %s to its generated params field %s', (actionType, fieldName) => {
    expect(getActionParamsFieldName(actionType)).toBe(fieldName);
  });

  it('returns null for the unspecified action', () => {
    expect(getActionParamsFieldName(ACTION_TYPES.UNSPECIFIED)).toBeNull();
  });
});
