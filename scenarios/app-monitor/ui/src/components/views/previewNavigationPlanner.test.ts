import { describe, expect, it } from 'vitest';
import {
  createPreviewNavigationPlan,
  PREVIEW_NAV_BLOCKED_HOST_MESSAGE,
  PREVIEW_NAV_INVALID_MESSAGE,
} from './previewNavigationPlanner';

describe('createPreviewNavigationPlan', () => {
  const base = {
    navigationReference: 'http://localhost:3000/apps/git-control-tower/proxy/',
    hostOrigin: 'http://localhost:3000',
    bridgeSupported: false,
    childOrigin: null as string | null,
  };

  it('returns empty plan for blank input', () => {
    const plan = createPreviewNavigationPlan({
      ...base,
      rawValue: '   ',
    });

    expect(plan).toEqual({
      kind: 'empty',
      nextInput: '',
    });
  });

  it('returns invalid plan when candidate URL cannot be resolved', () => {
    const plan = createPreviewNavigationPlan({
      ...base,
      rawValue: 'http://[::1',
      navigationReference: null,
    });

    expect(plan).toEqual({
      kind: 'invalid',
      nextInput: 'http://[::1',
      message: PREVIEW_NAV_INVALID_MESSAGE,
    });
  });

  it('normalizes bare scenario identifiers and routes locally when bridge is disabled', () => {
    const plan = createPreviewNavigationPlan({
      ...base,
      rawValue: 'agent-inbox',
    });

    expect(plan).toEqual({
      kind: 'local-go',
      nextInput: '/apps/agent-inbox/proxy/',
      resolvedTarget: 'http://localhost:3000/apps/agent-inbox/proxy/',
    });
  });

  it('blocks app-monitor shell targets', () => {
    const plan = createPreviewNavigationPlan({
      ...base,
      rawValue: '/apps/workspace',
    });

    expect(plan).toEqual({
      kind: 'blocked-host',
      nextInput: '/apps/workspace',
      resolvedTarget: 'http://localhost:3000/apps/workspace',
      message: PREVIEW_NAV_BLOCKED_HOST_MESSAGE,
    });
  });

  it('plans bridge-go when target stays within the same scenario proxy scope', () => {
    const plan = createPreviewNavigationPlan({
      ...base,
      rawValue: '/apps/git-control-tower/proxy/?path=README.md',
      bridgeSupported: true,
      childOrigin: 'http://localhost:3000',
    });

    expect(plan).toEqual({
      kind: 'bridge-go',
      nextInput: '/apps/git-control-tower/proxy/?path=README.md',
      resolvedTarget: 'http://localhost:3000/apps/git-control-tower/proxy/?path=README.md',
    });
  });

  it('falls back to local-go when switching scenario proxy scope', () => {
    const plan = createPreviewNavigationPlan({
      ...base,
      rawValue: '/apps/agent-inbox/proxy/',
      bridgeSupported: true,
      childOrigin: 'http://localhost:3000',
    });

    expect(plan).toEqual({
      kind: 'local-go',
      nextInput: '/apps/agent-inbox/proxy/',
      resolvedTarget: 'http://localhost:3000/apps/agent-inbox/proxy/',
    });
  });

  it('falls back to local-go when staying in same scenario but changing proxy path', () => {
    const plan = createPreviewNavigationPlan({
      ...base,
      rawValue: '/apps/git-control-tower/proxy/settings',
      bridgeSupported: true,
      childOrigin: 'http://localhost:3000',
    });

    expect(plan).toEqual({
      kind: 'local-go',
      nextInput: '/apps/git-control-tower/proxy/settings',
      resolvedTarget: 'http://localhost:3000/apps/git-control-tower/proxy/settings',
    });
  });
});
