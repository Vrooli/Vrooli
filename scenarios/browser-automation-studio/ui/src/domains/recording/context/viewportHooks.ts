import { useContext } from 'react';
import { ViewportContext } from './viewportContext';
import type { ViewportContextValue } from './ViewportProvider';

/**
 * Hook to access viewport context.
 * Must be used within a ViewportProvider.
 */
export function useViewport(): ViewportContextValue {
  const context = useContext(ViewportContext);
  if (!context) {
    throw new Error('useViewport must be used within a ViewportProvider');
  }
  return context;
}

/**
 * Hook to access viewport context, returning null if not within provider.
 * Useful for components that may be used outside the viewport context.
 */
export function useViewportOptional(): ViewportContextValue | null {
  return useContext(ViewportContext);
}
