import { useCallback } from 'react';

type Unsubscribe = () => void;

function ensureParams(): URLSearchParams {
  if (typeof window === 'undefined') {
    return new URLSearchParams();
  }
  return new URLSearchParams(window.location.search);
}

export function useSearchParamSync() {
  const getParams = useCallback(() => ensureParams(), []);

  const getParam = useCallback(
    (key: string): string | null => {
      const params = ensureParams();
      const value = params.get(key);
      if (!value) {
        return null;
      }
      const trimmed = value.trim();
      return trimmed.length > 0 ? trimmed : null;
    },
    [],
  );

  const setParams = useCallback((updater: (params: URLSearchParams) => void) => {
    if (typeof window === 'undefined') {
      return;
    }
    const url = new URL(window.location.href);
    const params = new URLSearchParams(url.search);
    updater(params);
    const nextSearch = params.toString();
    const nextUrl = `${url.pathname}${nextSearch ? `?${nextSearch}` : ''}${url.hash}`;
    window.history.replaceState({}, '', nextUrl);
  }, []);

  const subscribe = useCallback((listener: (params: URLSearchParams) => void): Unsubscribe => {
    if (typeof window === 'undefined') {
      return () => {};
    }

    const handler = () => {
      listener(new URLSearchParams(window.location.search));
    };

    window.addEventListener('popstate', handler);
    return () => {
      window.removeEventListener('popstate', handler);
    };
  }, []);

  return {
    getParams,
    getParam,
    setParams,
    subscribe,
  };
}

