import { useCallback, useEffect, useRef, useState, type RefObject } from 'react';
import type { PreviewMode } from './usePreviewMode';

type LiveState = 'idle' | 'loading' | 'loaded' | 'blocked';

interface UseLivePreviewOptions {
  mode: PreviewMode;
  url: string | null;
  onBlocked: () => void;
}

interface UseLivePreviewResult {
  liveState: LiveState;
  iframeKey: number;
  iframeRef: RefObject<HTMLIFrameElement>;
  handleIframeLoad: () => void;
  handleIframeError: () => void;
  reload: () => void;
}

export function useLivePreview({ mode, url, onBlocked }: UseLivePreviewOptions): UseLivePreviewResult {
  const [liveState, setLiveState] = useState<LiveState>('idle');
  const [iframeKey, setIframeKey] = useState(0);
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const loadTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const blockedRef = useRef(false);

  const clearTimer = () => {
    if (loadTimerRef.current) {
      clearTimeout(loadTimerRef.current);
      loadTimerRef.current = null;
    }
  };

  const markBlocked = useCallback(() => {
    blockedRef.current = true;
    setLiveState('blocked');
    onBlocked();
  }, [onBlocked]);

  useEffect(() => {
    blockedRef.current = false;
    if (mode === 'live' && url) {
      setLiveState('loading');
      clearTimer();
      loadTimerRef.current = setTimeout(() => {
        markBlocked();
      }, 4500);
    } else {
      setLiveState('idle');
      clearTimer();
    }
    return clearTimer;
  }, [mode, url, markBlocked]);

  const inspectIframe = (): 'blocked' | 'ok' | 'unknown' => {
    const iframe = iframeRef.current;
    if (!iframe) return 'unknown';
    try {
      const doc = iframe.contentDocument;
      if (!doc || !doc.body) return 'unknown';
      const text = (doc.body.innerText || '').toLowerCase();
      const childCount = doc.body.childElementCount || 0;
      const refused = text.includes('refused to connect') || text.includes('denied') || text.includes('blocked');
      if (refused) return 'blocked';
      if (childCount > 0 || text.trim().length > 0) return 'ok';
      return 'unknown';
    } catch {
      return 'ok';
    }
  };

  const handleIframeLoad = useCallback(() => {
    clearTimer();
    blockedRef.current = false;
    setLiveState('loading');

    const runChecks = () => {
      const status = inspectIframe();
      if (status === 'blocked') {
        markBlocked();
        return true;
      }
      if (status === 'ok') {
        setLiveState('loaded');
        return true;
      }
      return false;
    };

    if (runChecks()) return;
    setTimeout(() => {
      if (blockedRef.current) return;
      if (runChecks()) return;
      setTimeout(() => {
        if (blockedRef.current) return;
        if (!runChecks()) {
          const isLocal =
            url?.includes('localhost') ||
            url?.includes('127.0.0.1') ||
            url?.includes('0.0.0.0');
          if (isLocal) {
            setLiveState('loaded');
          } else {
            markBlocked();
          }
        }
      }, 600);
    }, 400);
  }, [markBlocked, url]);

  const handleIframeError = useCallback(() => {
    clearTimer();
    markBlocked();
  }, [markBlocked]);

  const reload = useCallback(() => {
    setIframeKey((k) => k + 1);
  }, []);

  useEffect(() => clearTimer, []);

  return { liveState, iframeKey, iframeRef, handleIframeLoad, handleIframeError, reload };
}
