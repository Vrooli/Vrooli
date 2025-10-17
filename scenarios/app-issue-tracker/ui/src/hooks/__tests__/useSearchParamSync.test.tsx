import { describe, it, expect, beforeEach } from 'vitest';
import { act } from 'react';
import { renderHook } from '../../test-utils/renderHook';
import { useSearchParamSync } from '../useSearchParamSync';

describe('useSearchParamSync', () => {
  beforeEach(() => {
    window.history.replaceState({}, '', 'http://localhost/');
  });

  it('reads and writes query parameters', () => {
    const { result } = renderHook(() => useSearchParamSync());

    act(() => {
      result.current.setParams((params) => {
        params.set('issue', 'ISSUE-100');
        params.set('app_id', 'designer');
      });
    });

    expect(result.current.getParam('issue')).toBe('ISSUE-100');
    expect(window.location.search).toContain('app_id=designer');

    act(() => {
      result.current.setParams((params) => {
        params.set('issue', '');
      });
    });

    expect(result.current.getParam('issue')).toBeNull();
  });

  it('invokes subscribers on popstate events', () => {
    const { result } = renderHook(() => useSearchParamSync());
    const captured: Array<string | null> = [];

    const unsubscribe = result.current.subscribe((params) => {
      captured.push(params.get('issue'));
    });

    act(() => {
      window.history.pushState({}, '', 'http://localhost/?issue=ISSUE-200');
      window.dispatchEvent(new PopStateEvent('popstate'));
    });

    unsubscribe();

    expect(captured).toContain('ISSUE-200');
  });
});
