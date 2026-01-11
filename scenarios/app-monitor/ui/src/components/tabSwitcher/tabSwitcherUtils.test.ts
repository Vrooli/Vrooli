import { describe, expect, it } from 'vitest';
import type { BrowserTabRecord } from '@/state/browserTabsStore';
import type { Resource } from '@/types';
import {
  matchesResourceSearch,
  matchesWebTabSearch,
  normalizeSearchValue,
  parseWebTabInput,
} from './tabSwitcherUtils';

const baseResource: Resource = {
  id: 'redis',
  name: 'Redis Cache',
  type: 'cache',
  status: 'online',
  description: 'In-memory store',
};

const baseTab: BrowserTabRecord = {
  id: 'tab-1',
  title: 'Documentation',
  url: 'https://example.com/docs',
  createdAt: 1,
  lastActiveAt: 2,
  screenshotData: null,
  screenshotWidth: null,
  screenshotHeight: null,
  screenshotNote: null,
  faviconUrl: null,
};

describe('tabSwitcherUtils', () => {
  it('normalizes search input', () => {
    expect(normalizeSearchValue('  HeLLo ')).toBe('hello');
  });

  it('matches resource search on name, type, and description', () => {
    expect(matchesResourceSearch(baseResource, 'redis')).toBe(true);
    expect(matchesResourceSearch(baseResource, 'cache')).toBe(true);
    expect(matchesResourceSearch(baseResource, 'memory')).toBe(true);
    expect(matchesResourceSearch(baseResource, 'queue')).toBe(false);
  });

  it('matches web tab search on title or url', () => {
    expect(matchesWebTabSearch(baseTab, 'doc')).toBe(true);
    expect(matchesWebTabSearch(baseTab, 'example')).toBe(true);
    expect(matchesWebTabSearch(baseTab, 'unknown')).toBe(false);
  });

  it('parses web tab input and normalizes host', () => {
    const parsed = parseWebTabInput('example.com');
    expect(parsed).not.toBeNull();
    expect(parsed?.url).toBe('https://example.com/');
    expect(parsed?.title).toBe('example.com');
  });

  it('returns null for invalid web tab input', () => {
    expect(parseWebTabInput('')).toBeNull();
    expect(parseWebTabInput('not a url')).toBeNull();
  });
});
