/**
 * @libraryId react-component-library:FocusTrapPanel
 * @displayName FocusTrapPanel
 * @description Small overlay panel demonstrating a pinned reusable focus-trap hook.
 * @version 1.0.1
 * @tags ["overlay","accessibility","focus"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";
import { useRef, type ReactNode } from "react";

import { useFocusTrap } from "@vrooli/react-component-library/DrawerShell/1.0.0";

export interface FocusTrapPanelProps {
  open?: boolean;
  children: ReactNode;
}

export const FocusTrapPanel = withClassName(function FocusTrapPanel({
  open = true,
  children,
}: FocusTrapPanelProps) {
  const panelRef = useRef<HTMLDivElement>(null);
  useFocusTrap(open, panelRef);
  if (!open) return null;
  return (
    <div ref={panelRef} role="dialog" aria-modal="true">
      {children}
    </div>
  );
});
