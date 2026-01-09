import { useCallback, useEffect, useMemo, useState, type Dispatch, type SetStateAction } from 'react';
import type { Issue } from '../types/issue';

interface SearchSyncHelpers {
  getParam: (key: string) => string | null;
  setParams: (updater: (params: URLSearchParams) => void) => void;
  subscribe: (listener: (params: URLSearchParams) => void) => () => void;
}

interface UseIssueFocusOptions {
  issues: Issue[];
  search: SearchSyncHelpers;
  onOpen: () => void;
  onClose: () => void;
  setOpen: (value: boolean) => void;
}

interface UseIssueFocusResult {
  focusedIssueId: string | null;
  setFocusedIssueId: Dispatch<SetStateAction<string | null>>;
  selectIssue: (issueId: string) => void;
  clearFocus: () => void;
  selectedIssue: Issue | null;
}

export function useIssueFocus({
  issues,
  search,
  onOpen,
  onClose,
  setOpen,
}: UseIssueFocusOptions): UseIssueFocusResult {
  const [focusedIssueId, setFocusedIssueId] = useState<string | null>(() => {
    const param = search.getParam('issue');
    return param && param.trim() ? param.trim() : null;
  });

  useEffect(() => {
    return search.subscribe((params) => {
      const nextIssueId = params.get('issue');
      setFocusedIssueId(nextIssueId && nextIssueId.trim() ? nextIssueId.trim() : null);
    });
  }, [search]);

  useEffect(() => {
    search.setParams((params) => {
      if (focusedIssueId) {
        params.set('issue', focusedIssueId);
      } else {
        params.delete('issue');
      }
    });
  }, [focusedIssueId, search]);

  useEffect(() => {
    setOpen(Boolean(focusedIssueId));
  }, [focusedIssueId, setOpen]);

  useEffect(() => {
    if (!focusedIssueId) {
      return;
    }
    const issueExists = issues.some((issue) => issue.id === focusedIssueId);
    if (!issueExists) {
      return;
    }
    if (typeof window === 'undefined' || typeof document === 'undefined') {
      return;
    }
    const element = document.querySelector<HTMLElement>(`[data-issue-id="${focusedIssueId}"]`);
    if (element) {
      element.focus({ preventScroll: true });
      window.setTimeout(() => {
        element.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }, 50);
    }
  }, [focusedIssueId, issues]);

  const selectIssue = useCallback(
    (issueId: string) => {
      setFocusedIssueId((current) => {
        if (current === issueId) {
          onClose();
          return null;
        }
        onOpen();
        return issueId;
      });
    },
    [onClose, onOpen, setFocusedIssueId],
  );

  const clearFocus = useCallback(() => {
    onClose();
    setFocusedIssueId(null);
  }, [onClose, setFocusedIssueId]);

  const selectedIssue = useMemo(() => {
    return issues.find((issue) => issue.id === focusedIssueId) ?? null;
  }, [issues, focusedIssueId]);

  return {
    focusedIssueId,
    setFocusedIssueId,
    selectIssue,
    clearFocus,
    selectedIssue,
  };
}
