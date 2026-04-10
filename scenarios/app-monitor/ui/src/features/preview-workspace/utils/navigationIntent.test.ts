import { describe, expect, it } from 'vitest';
import {
  WORKSPACE_INTENT_APP_ID_KEY,
  WORKSPACE_INTENT_MODE_KEY,
  clearWorkspaceIntent,
  readWorkspaceIntent,
} from './navigationIntent';

describe('workspace navigation intent utils', () => {
  it('parses valid replace intent', () => {
    const params = new URLSearchParams({
      [WORKSPACE_INTENT_APP_ID_KEY]: 'scenario-a',
      [WORKSPACE_INTENT_MODE_KEY]: 'replace-focused',
    });

    expect(readWorkspaceIntent(params)).toEqual({
      appId: 'scenario-a',
      mode: 'replace-focused',
    });
  });

  it('returns null for invalid intent', () => {
    const params = new URLSearchParams({
      [WORKSPACE_INTENT_APP_ID_KEY]: 'scenario-a',
      [WORKSPACE_INTENT_MODE_KEY]: 'unsupported',
    });

    expect(readWorkspaceIntent(params)).toBeNull();
  });

  it('clears only workspace intent params', () => {
    const params = new URLSearchParams({
      [WORKSPACE_INTENT_APP_ID_KEY]: 'scenario-a',
      [WORKSPACE_INTENT_MODE_KEY]: 'add-pane',
      segment: 'apps',
    });

    const cleared = clearWorkspaceIntent(params);
    expect(cleared.get(WORKSPACE_INTENT_APP_ID_KEY)).toBeNull();
    expect(cleared.get(WORKSPACE_INTENT_MODE_KEY)).toBeNull();
    expect(cleared.get('segment')).toBe('apps');
  });
});

