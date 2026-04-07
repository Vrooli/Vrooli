import { createContext, useContext, type ReactNode } from 'react';
import type { SpatialNavController } from '@vrooli/iframe-bridge/spatial';

const SpatialNavContext = createContext<React.RefObject<SpatialNavController | null> | null>(null);

/**
 * Provides the SpatialNavController ref to the component tree.
 * Wrap your app (or the section that uses spatial nav) with this provider,
 * then any descendant can call `useSpatialNavContext()` to access the controller
 * (e.g., for `<SpatialGroup>` or modal scope management).
 */
export function SpatialNavProvider({
  controllerRef,
  children,
}: {
  controllerRef: React.RefObject<SpatialNavController | null>;
  children: ReactNode;
}) {
  return (
    <SpatialNavContext.Provider value={controllerRef}>
      {children}
    </SpatialNavContext.Provider>
  );
}

/**
 * Access the SpatialNavController ref from context.
 * Returns `null` if no provider is mounted (spatial nav not initialized).
 */
export function useSpatialNavContext(): React.RefObject<SpatialNavController | null> | null {
  return useContext(SpatialNavContext);
}
