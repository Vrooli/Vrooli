import { useCallback, useEffect, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useShellOverlayStore } from '@/state/shellOverlayStore';

export type OverlayView = 'tabs' | 'actions';

type OverlayOptions = {
  replace?: boolean;
  params?: Record<string, string | null | undefined>;
  preserve?: string[];
};

const OVERLAY_PARAM = 'overlay';

const isOverlayView = (value: string | null): value is OverlayView => (
  value === 'tabs' || value === 'actions'
);

export function useOverlayRouter() {
  const [searchParams, setSearchParams] = useSearchParams();
  const storeOverlay = useShellOverlayStore(state => state.activeView);
  const openView = useShellOverlayStore(state => state.openView);
  const closeView = useShellOverlayStore(state => state.closeView);

  const overlayFromQuery = useMemo(() => {
    const value = searchParams.get(OVERLAY_PARAM);
    return isOverlayView(value) ? value : null;
  }, [searchParams]);

  useEffect(() => {
    if (overlayFromQuery) {
      if (storeOverlay !== overlayFromQuery) {
        openView(overlayFromQuery);
      }
      return;
    }

    if (storeOverlay !== null) {
      closeView();
    }
  }, [overlayFromQuery, storeOverlay, openView, closeView]);

  const updateOverlay = useCallback((nextOverlay: OverlayView | null, options?: OverlayOptions) => {
    const nextParams = new URLSearchParams(searchParams);

    if (nextOverlay) {
      nextParams.set(OVERLAY_PARAM, nextOverlay);
    } else {
      nextParams.delete(OVERLAY_PARAM);
    }

    if (!(options?.preserve ?? []).includes('segment') && !nextOverlay) {
      nextParams.delete('segment');
    }

    if (options?.params) {
      Object.entries(options.params).forEach(([key, value]) => {
        if (value == null || value === '') {
          nextParams.delete(key);
        } else {
          nextParams.set(key, value);
        }
      });
    }

    setSearchParams(nextParams, { replace: options?.replace ?? false });
  }, [searchParams, setSearchParams]);

  const openOverlay = useCallback((view: OverlayView, options?: OverlayOptions) => {
    updateOverlay(view, options);
  }, [updateOverlay]);

  const closeOverlay = useCallback((options?: OverlayOptions) => {
    updateOverlay(null, options);
  }, [updateOverlay]);

  return {
    overlay: overlayFromQuery,
    openOverlay,
    closeOverlay,
    setOverlay: updateOverlay,
  };
}

export default useOverlayRouter;
