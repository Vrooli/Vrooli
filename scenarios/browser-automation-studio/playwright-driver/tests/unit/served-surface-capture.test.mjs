import test from 'node:test';
import assert from 'node:assert/strict';
import { parseCaptureManifest, parseViewport } from '../../scripts/served-surface-capture.mjs';

test('capture manifest accepts named, synthetic-only presentation states', () => {
  const parsed = parseCaptureManifest({ states: [
    { id: 'home', actions: [] },
    { id: 'group-menu', actions: [{ kind: 'right-click-text', name: 'Workbench' }] },
    { id: 'messages', actions: [{ kind: 'click', role: 'button', name: 'Switch to messages view' }, { kind: 'click', role: 'textbox', name: 'Search 2 messages' }] },
  ] });
  assert.equal(parsed.states.length, 3);
});

test('capture manifest rejects malformed or effectful actions', () => {
  assert.throws(() => parseCaptureManifest({ states: [] }), /non-empty states/);
  assert.throws(() => parseCaptureManifest({ states: [{ id: 'Bad ID', actions: [] }] }), /invalid state id/);
  assert.throws(() => parseCaptureManifest({ states: [{ id: 'home', actions: [{ kind: 'click', role: 'button', name: 'Save' }] }] }), /effectful action/);
  assert.throws(() => parseCaptureManifest({ states: [{ id: 'home', actions: [{ kind: 'click', role: 'button', name: 'Open' }, { kind: 'click', role: 'button', name: 'Close group' }] }] }), /effectful action/);
  assert.throws(() => parseCaptureManifest({ states: [{ id: 'home', actions: [{ kind: 'click', role: 'text', name: 'Workbench' }] }] }), /supported semantic role/);
});

test('viewport parser bounds desktop and phone review sizes', () => {
  assert.deepEqual(parseViewport('1440x900'), { width: 1440, height: 900 });
  assert.deepEqual(parseViewport('390x844'), { width: 390, height: 844 });
  for (const value of ['bad', '3000x900', '390x2400']) assert.throws(() => parseViewport(value));
});

test('capture manifest permits only allowlisted view-only test ids', () => {
  assert.equal(parseCaptureManifest({ states: [{ id: 'settings', actions: [{ kind: 'click-testid', name: 'Settings', testId: 'nav-settings' }, { kind: 'click-testid', name: 'Manage Groups', testId: 'group-ctx-manage-groups' }, { kind: 'click-testid', name: 'Toggle group display', testId: 'group-ctx-toggle-collapse' }, { kind: 'click-testid', name: 'Open AI Command', testId: 'toolbar-ai' }, { kind: 'click-testid', name: 'Read synthetic message', testId: 'msg-open-reader-synthetic-event-0002' }, { kind: 'click-testid', name: 'Enter message selection', testId: 'msg-export-enter' }, { kind: 'click-testid', name: 'Review selected messages', testId: 'msg-export-continue' }, { kind: 'click-testid', name: 'Open message navigator', testId: 'messages-search-field' }, { kind: 'click-testid', name: 'Open synthetic group palette', testId: 'manage-groups-recolor-synthetic-group-0001' }, { kind: 'right-click-testid', name: 'Open synthetic message action menu', testId: 'msg-card-synthetic-event-0002' }, { kind: 'click-testid', name: 'select synthetic pane', testId: 'sidebar-session-synthetic-session-0002' }] }] }).states.length, 1);
  assert.throws(() => parseCaptureManifest({ states: [{ id: 'unsafe', actions: [{ kind: 'click-testid', name: 'continue export', testId: 'msg-export-continue' }] }] }), /effectful action|allowlist/);
});
