/**
 * @libraryId react-component-library:Portal
 * @displayName Portal
 * @description Reusable portal component implementation for the React component library.
 * @version 1.1.3
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { createPortal } from "react-dom";
import type { ReactNode } from "react";
import { useHydrated } from "@vrooli/react-component-library/useHydrated/1";
import { getLayerPortalContainer } from "@vrooli/react-component-library/LayerManager/2";

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
