import { useRef, type ReactNode } from "react";

import { useFocusTrap } from "../../../DrawerShell/versions/1.0.0/useFocusTrap";

export interface FocusTrapPanelProps {
  open?: boolean;
  children: ReactNode;
}

export function FocusTrapPanel({ open = true, children }: FocusTrapPanelProps) {
  const panelRef = useRef<HTMLDivElement>(null);
  useFocusTrap(open, panelRef);
  if (!open) return null;
  return <div ref={panelRef} role="dialog" aria-modal="true">{children}</div>;
}
