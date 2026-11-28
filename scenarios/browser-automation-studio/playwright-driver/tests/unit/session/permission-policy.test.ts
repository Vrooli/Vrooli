import { ActionType } from '../../../src/proto';
import { authorizeInstruction, authorizeOperation, authorizeUnreceiptedOperation } from '../../../src/session/permission-policy';
import type { BrowserPermissionProfile } from '../../../src/types/session';

function instruction(type: ActionType, params: Record<string, unknown> = {}) {
  return { index: 0, nodeId: 'n1', action: { type, ...params } } as any;
}

describe('driver second-layer browser permission policy', () => {
  const scoped: BrowserPermissionProfile = {
    effect: 'act', approval: 'none', control: true, view: true,
    url_allow: ['https://*.example.test/app/*'],
    url_deny: ['https://private.example.test/*'],
  };

  it('requires an allowed target URL for navigation and denies broader scopes', () => {
    const navigate = (url: string) => instruction(ActionType.NAVIGATE, { params: { case: 'navigate', value: { url } } });
    expect(authorizeInstruction(scoped, navigate('https://shop.example.test/app/cart'), 'about:blank').allowed).toBe(true);
    expect(authorizeInstruction(scoped, navigate('https://shop.example.test/private'), 'about:blank').allowed).toBe(false);
    expect(authorizeInstruction(scoped, navigate('https://private.example.test/app/cart'), 'about:blank').allowed).toBe(false);
  });

  it('requires view/control and applies budgets to the selected session', () => {
    const observe = { ...scoped, effect: 'observe' as const, control: false };
    expect(authorizeInstruction(observe, instruction(ActionType.CLICK), 'https://shop.example.test/app/').allowed).toBe(false);
    expect(authorizeInstruction({ ...scoped, control: false }, instruction(ActionType.CLICK), 'https://shop.example.test/app/').allowed).toBe(false);
    expect(authorizeInstruction({ ...scoped, view: false }, instruction(ActionType.SCREENSHOT), 'https://shop.example.test/app/').allowed).toBe(false);
    expect(authorizeInstruction({ ...scoped, max_actions: 1 }, instruction(ActionType.ASSERT), 'https://shop.example.test/app/', Date.now(), undefined, 1).allowed).toBe(false);
  });

  it('keeps history reads read-only but requires explicit history and URL scope grants', () => {
    const observe = { ...scoped, effect: 'observe' as const, control: false, history_read: true };
    expect(authorizeOperation(observe, 'history_read', 'https://shop.example.test/app/orders').allowed).toBe(true);
    expect(authorizeOperation({ ...observe, history_read: false }, 'history_read', 'https://shop.example.test/app/orders').allowed).toBe(false);
    expect(authorizeOperation(observe, 'history_read', 'https://outside.test/orders').allowed).toBe(false);
  });

  it('shares API approval tiers across driver routes without a receipt verifier', () => {
    const click = instruction(ActionType.CLICK);
    const input = instruction(ActionType.INPUT);
    const safetyPass = { ...scoped, approval: 'safety_pass' as const };
    const everyAction = { ...scoped, approval: 'every_action' as const };
    const currentURL = 'https://shop.example.test/app/orders';

    // The API requires no receipt for an ordinary click under safety_pass, so
    // the driver may retain that legitimate task path.
    expect(authorizeInstruction(safetyPass, click, currentURL).allowed).toBe(true);
    // API-classified high-impact input and every_action mutations require an
    // owner receipt the independent driver route cannot verify.
    expect(authorizeInstruction(safetyPass, input, currentURL)).toMatchObject({ allowed: false });
    expect(authorizeInstruction(everyAction, click, currentURL)).toMatchObject({ allowed: false });
    // The same operation mapping is used by recording/lifecycle routes.
    expect(authorizeUnreceiptedOperation(safetyPass, 'navigate', currentURL, 'https://shop.example.test/app/next').allowed).toBe(true);
    expect(authorizeUnreceiptedOperation(safetyPass, 'recording_start', currentURL).allowed).toBe(false);
    expect(authorizeUnreceiptedOperation(safetyPass, 'reset', currentURL).allowed).toBe(false);
    expect(authorizeUnreceiptedOperation({ ...scoped, approval: 'safety_pass' }, 'audio_playback_restart', currentURL).allowed).toBe(false);
  });
});
