import { describe, expect, it } from 'vitest';
import {
  formatPreviewUrlForDisplay,
  isAppMonitorProxyPreviewTarget,
  isBlockedHostEmbedPreviewTarget,
  normalizeScenarioNavigationInput,
  parseScenarioProxyPreviewTarget,
  resolvePreviewUrlCandidate,
} from './previewUrl';

describe('resolvePreviewUrlCandidate', () => {
  it('resolves scenario proxy paths against the current origin', () => {
    expect(resolvePreviewUrlCandidate('/apps/git-control-tower/proxy/')).toBe(
      'http://localhost:3000/apps/git-control-tower/proxy/',
    );
  });

  it('accepts localhost targets without protocol', () => {
    expect(resolvePreviewUrlCandidate('localhost:5173')).toBe('http://localhost:5173/');
  });

  it('resolves root-relative paths against the active preview reference when provided', () => {
    expect(resolvePreviewUrlCandidate(
      '/apps/git-control-tower/proxy/?path=README.md',
      'https://app-monitor.itsagitime.com/apps/workspace',
    )).toBe('https://app-monitor.itsagitime.com/apps/git-control-tower/proxy/?path=README.md');
  });
});

describe('parseScenarioProxyPreviewTarget', () => {
  it('parses absolute scenario proxy URLs into compact labels', () => {
    expect(parseScenarioProxyPreviewTarget(
      'https://app-monitor.itsagitime.com/apps/prompt-manager/proxy/?skill=scientific-debugging',
    )).toEqual({
      scenarioIdentifier: 'prompt-manager',
      query: '?skill=scientific-debugging',
      displayLabel: 'prompt-manager:?skill=scientific-debugging',
    });
  });
});

describe('formatPreviewUrlForDisplay', () => {
  it('formats scenario proxy paths using scenario identifier + query', () => {
    expect(formatPreviewUrlForDisplay('/apps/git-control-tower/proxy/')).toBe('git-control-tower');
  });

  it('preserves non-scenario URLs', () => {
    expect(formatPreviewUrlForDisplay('https://example.com/docs')).toBe('https://example.com/docs');
  });
});

describe('normalizeScenarioNavigationInput', () => {
  it('rewrites bare scenario identifier to proxy path', () => {
    expect(normalizeScenarioNavigationInput('agent-inbox')).toBe('/apps/agent-inbox/proxy/');
  });

  it('rewrites /apps/<scenario> to proxy path', () => {
    expect(normalizeScenarioNavigationInput('/apps/git-control-tower')).toBe('/apps/git-control-tower/proxy/');
  });

  it('does not rewrite reserved shell routes', () => {
    expect(normalizeScenarioNavigationInput('/apps/workspace')).toBeNull();
  });

  it('does not rewrite generic relative paths', () => {
    expect(normalizeScenarioNavigationInput('settings')).toBeNull();
  });
});

describe('isBlockedHostEmbedPreviewTarget', () => {
  it('blocks app-monitor workspace URL on host origin', () => {
    expect(isBlockedHostEmbedPreviewTarget('http://localhost:3000/apps/workspace', 'http://localhost:3000')).toBe(true);
  });

  it('allows scenario proxy URL on host origin', () => {
    expect(isBlockedHostEmbedPreviewTarget('http://localhost:3000/apps/git-control-tower/proxy/', 'http://localhost:3000')).toBe(false);
  });

  it('allows same path on non-host origin', () => {
    expect(isBlockedHostEmbedPreviewTarget('http://localhost:4310/apps/workspace', 'http://localhost:3000')).toBe(false);
  });
});

describe('isAppMonitorProxyPreviewTarget', () => {
  it('detects app-monitor proxy URLs', () => {
    expect(isAppMonitorProxyPreviewTarget('/apps/app-monitor/proxy/')).toBe(true);
  });

  it('ignores non app-monitor scenario proxy URLs', () => {
    expect(isAppMonitorProxyPreviewTarget('/apps/git-control-tower/proxy/')).toBe(false);
  });
});
