/**
 * Compatibility adapter for existing Swarm Manager drawer call sites.
 *
 * Overlay behavior belongs to the React Component Library. ResponsiveDialog
 * supplies the centered desktop dialog, mobile sheet, safe-area insets, and
 * swipe-to-dismiss gesture. Keeping this small adapter lets the existing
 * feature components migrate without each one reimplementing overlay chrome.
 */

import type { ReactNode } from "react";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1.3.1";

export interface DrawerProps {
  /** Whether the drawer is visible */
  isOpen: boolean;
  /** Callback to close the drawer */
  onClose: () => void;
  /** Drawer title */
  title: string;
  /** Optional subtitle below the title */
  description?: string;
  /** Drawer content */
  children: ReactNode;
  /** Optional footer (sticky at the bottom) */
  footer?: ReactNode;
  /** Additional CSS classes for the shared overlay panel */
  className?: string;
  /** data-testid value */
  testId?: string;
}

export function Drawer({
  isOpen,
  onClose,
  title,
  description,
  children,
  footer,
  className,
  testId,
}: DrawerProps) {
  return (
    <ResponsiveDialog
      open={isOpen}
      onClose={onClose}
      title={title}
      subheader={description ? <p className="px-4 py-3 text-xs text-slate-400">{description}</p> : undefined}
      footer={footer}
      closeLabel="Close drawer"
      grabberLabel="Close drawer"
      size="md"
      contentPadding="none"
      panelClassName={className}
      testId={testId}
    >
      {children}
    </ResponsiveDialog>
  );
}
