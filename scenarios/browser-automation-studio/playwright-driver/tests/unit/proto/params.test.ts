import { create } from '@bufbuild/protobuf';
import {
  ActionDefinitionSchema,
  ActionType,
  ClickParamsSchema,
  CookieOperation,
  CookieOptionsSchema,
  CookieSameSite,
  CookieStorageParamsSchema,
  DeviceOrientation,
  GestureParamsSchema,
  GestureType,
  KeyboardModifier,
  MouseButton,
  NavigateParamsSchema,
  NavigateWaitEvent,
  RotateParamsSchema,
  SelectParamsSchema,
  StorageType,
  WaitParamsSchema,
  WaitState,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import {
  getClickParams,
  getCookieStorageParams,
  getGestureParams,
  getInputParams,
  getNavigateParams,
  getRotateParams,
  getSelectParams,
  getWaitParams,
} from '../../../src/proto/instruction';

describe('proto action parameter conversions', () => {
  it('formats click enums using the generated params oneof case', () => {
    const action = create(ActionDefinitionSchema, {
      type: ActionType.CLICK,
      params: {
        case: 'click',
        value: create(ClickParamsSchema, {
          selector: '#save',
          button: MouseButton.RIGHT,
          modifiers: [KeyboardModifier.CTRL],
        }),
      },
    });

    expect(getClickParams(action)).toMatchObject({
      selector: '#save',
      button: 'right',
      modifiers: ['Control'],
    });
  });

  it('reads SELECT from the generated selectOption params oneof case', () => {
    const action = create(ActionDefinitionSchema, {
      type: ActionType.SELECT,
      params: {
        case: 'selectOption',
        value: create(SelectParamsSchema, {
          selector: '#country',
          selectBy: { case: 'value', value: 'ca' },
        }),
      },
    });

    expect(getSelectParams(action)).toMatchObject({ selector: '#country', value: 'ca' });
  });

  it('returns undefined when a typed accessor receives another action type', () => {
    const action = create(ActionDefinitionSchema, {
      type: ActionType.NAVIGATE,
      params: {
        case: 'navigate',
        value: create(NavigateParamsSchema, { url: 'https://example.com' }),
      },
    });

    expect(getInputParams(action)).toBeUndefined();
  });

  it('preserves the generated wait condition oneof and enum spelling', () => {
    const action = create(ActionDefinitionSchema, {
      type: ActionType.WAIT,
      params: {
        case: 'wait',
        value: create(WaitParamsSchema, {
          waitFor: { case: 'durationMs', value: 250 },
          state: WaitState.ATTACHED,
        }),
      },
    });

    expect(getWaitParams(action)).toMatchObject({ durationMs: 250, state: 'attached' });
  });

  it('formats generated navigation enum values for the driver', () => {
    const action = create(ActionDefinitionSchema, {
      type: ActionType.NAVIGATE,
      params: {
        case: 'navigate',
        value: create(NavigateParamsSchema, {
          url: 'https://example.com',
          waitUntil: NavigateWaitEvent.DOMCONTENTLOADED,
        }),
      },
    });

    expect(getNavigateParams(action)?.waitUntil).toBe('domcontentloaded');
  });

  it('preserves handler spellings for cookie and storage enum values', () => {
    const params = create(CookieStorageParamsSchema, {
      operation: CookieOperation.SET,
      storageType: StorageType.SESSION_STORAGE,
      cookieOptions: create(CookieOptionsSchema, { sameSite: CookieSameSite.STRICT }),
    });
    const action = create(ActionDefinitionSchema, {
      type: ActionType.COOKIE_STORAGE,
      params: { case: 'cookieStorage', value: params },
    });

    expect(getCookieStorageParams(action)).toMatchObject({
      operation: 'set',
      storageType: 'sessionStorage',
      cookieOptions: { sameSite: 'Strict' },
    });
  });

  it('preserves lower camel enum values for gesture types', () => {
    const action = create(ActionDefinitionSchema, {
      type: ActionType.GESTURE,
      params: {
        case: 'gesture',
        value: create(GestureParamsSchema, { gestureType: GestureType.DOUBLE_TAP }),
      },
    });

    expect(getGestureParams(action)?.gestureType).toBe('doubleTap');
  });

  it('uses the existing portrait fallback for an unspecified orientation', () => {
    const action = create(ActionDefinitionSchema, {
      type: ActionType.ROTATE,
      params: {
        case: 'rotate',
        value: create(RotateParamsSchema, { orientation: DeviceOrientation.UNSPECIFIED }),
      },
    });

    expect(getRotateParams(action)?.orientation).toBe('portrait');
  });
});
