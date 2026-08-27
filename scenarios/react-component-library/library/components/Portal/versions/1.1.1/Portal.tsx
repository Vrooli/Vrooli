/**
 * @libraryId react-component-library:Portal
 * @displayName Portal
 * @description Hydration-safe shared overlay portal owned by the layer manager.
 * @version 1.1.1
 * @tags ["runtime","overlay","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { createPortal } from "react-dom";
import type { ReactNode } from "react";
import { useHydrated } from "@vrooli/react-component-library/useHydrated/1.0.0";
import { getLayerPortalContainer } from "@vrooli/react-component-library/LayerManager/2.0.0";

export interface PortalProps {
  children: ReactNode;
  container?: Element | null;
}

export function Portal({ children, container }: PortalProps) {
  const hydrated = useHydrated();
  if (!hydrated) return null;
  const target = container ?? getLayerPortalContainer();
  return target ? createPortal(children, target) : null;
}

export default Portal;
