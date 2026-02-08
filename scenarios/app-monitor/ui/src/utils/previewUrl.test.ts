import { describe, expect, it } from 'vitest';
import {
  formatPreviewUrlForDisplay,
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
