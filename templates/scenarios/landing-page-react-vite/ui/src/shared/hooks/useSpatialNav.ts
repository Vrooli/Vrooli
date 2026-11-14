import { useEffect, useRef } from 'react';
import {
  initSpatialNav,
  type SpatialNavController,
  type SpatialNavBridgeOptions,
} from '@vrooli/iframe-bridge/spatial';

/**
 * Initialises gamepad-driven spatial navigation on mount and disposes on
 * unmount.  Returns a ref to the controller so components can register
 * focus groups via `controllerRef.current?.registerGroup(el, mode)`.
 *
 * ```tsx
 * const spatialNav = useSpatialNav();
 * // Then pass spatialNav to <SpatialGroup controllerRef={spatialNav} mode="spatial">
 * ```
 */
export function useSpatialNav(
  options?: SpatialNavBridgeOptions,
): React.RefObject<SpatialNavController | null> {
  const controllerRef = useRef<SpatialNavController | null>(null);

  useEffect(() => {
    const controller = initSpatialNav(options);
    controllerRef.current = controller;
    return () => {
      controller.dispose();
      controllerRef.current = null;
    };
    // Options are read once at mount — intentionally stable dependency.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return controllerRef;
}
