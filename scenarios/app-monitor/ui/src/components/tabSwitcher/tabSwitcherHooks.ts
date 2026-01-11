import { useCallback, useEffect, useMemo, useState } from 'react';
import type { SetURLSearchParams } from 'react-router-dom';
import type { BrowserTabHistoryRecord, BrowserTabRecord } from '@/state/browserTabsStore';
import type { App, Resource } from '@/types';
import { SEGMENT_QUERY_KEY, SEGMENTS, type SegmentId } from './tabSwitcherConstants';
import { matchesResourceSearch, matchesWebTabSearch, normalizeSearchValue } from './tabSwitcherUtils';

export const resolveSegment = (value: string | null): SegmentId => {
  if (value === 'resources' || value === 'web') {
    return value;
  }
  return 'apps';
};

export function useTabSwitcherSegment({
  segmentParam,
  searchParams,
  setSearchParams,
}: {
  segmentParam: string | null;
  searchParams: URLSearchParams;
  setSearchParams: SetURLSearchParams;
}) {
  const [activeSegment, setActiveSegment] = useState<SegmentId>(() => resolveSegment(segmentParam));

  useEffect(() => {
    setActiveSegment(resolveSegment(segmentParam));
  }, [segmentParam]);

  const handleSegmentSelect = useCallback((segmentId: SegmentId) => {
    setActiveSegment(segmentId);
    const nextParams = new URLSearchParams(searchParams);
    nextParams.set(SEGMENT_QUERY_KEY, segmentId);
    setSearchParams(nextParams, { replace: true });
  }, [searchParams, setSearchParams]);

  const activeSegmentLabel = useMemo(
    () => SEGMENTS.find(segment => segment.id === activeSegment)?.label ?? '',
    [activeSegment],
  );

  return {
    activeSegment,
    activeSegmentLabel,
    handleSegmentSelect,
  };
}

export function useTabSwitcherFiltering({
  search,
  browserTabs,
  browserHistory,
  sortedResources,
  recentApps,
}: {
  search: string;
  browserTabs: BrowserTabRecord[];
  browserHistory: BrowserTabHistoryRecord[];
  sortedResources: Resource[];
  recentApps: App[];
}) {
  const normalizedSearch = useMemo(() => normalizeSearchValue(search), [search]);

  const filteredResources = useMemo(() => (
    sortedResources.filter(resource => matchesResourceSearch(resource, normalizedSearch))
  ), [sortedResources, normalizedSearch]);

  const filteredActiveWebTabs = useMemo(() => (
    [...browserTabs]
      .sort((a, b) => b.lastActiveAt - a.lastActiveAt)
      .filter(tab => matchesWebTabSearch(tab, normalizedSearch))
  ), [browserTabs, normalizedSearch]);

  const filteredHistoryTabs = useMemo(() => (
    !normalizedSearch
      ? browserHistory
      : browserHistory.filter(tab => matchesWebTabSearch(tab, normalizedSearch))
  ), [browserHistory, normalizedSearch]);

  const showAppHistory = !normalizedSearch && recentApps.length > 0;
  const showWebHistory = filteredHistoryTabs.length > 0;

  return {
    normalizedSearch,
    filteredResources,
    filteredActiveWebTabs,
    filteredHistoryTabs,
    showAppHistory,
    showWebHistory,
  };
}
