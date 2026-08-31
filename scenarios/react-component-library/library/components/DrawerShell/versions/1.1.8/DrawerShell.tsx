/**
 * @libraryId react-component-library:DrawerShell
 * @displayName DrawerShell
 * @description The drawer presentation occupying the full application viewport while retaining source context, navigation continuity, safe areas, and route-compatible dismissal.
 * @version 1.1.8
 * @tags ["overlay","drawer","layout","reviewed","accessibility"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode } from "react";
import { BottomSheet } from "@vrooli/react-component-library/BottomSheet/1";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";

export interface DrawerShellProps {
  open?: boolean;
  onClose?: () => void;
  closeAriaLabel?: string;
  title?: ReactNode;
  headerActions?: ReactNode;
  headerExtra?: ReactNode;
  panelTestId?: string;
  size?: "full" | "compact";
  avoidKeyboard?: boolean;
  children: ReactNode;
}

export function DrawerShell({
  open = true,
  onClose = () => {},
  closeAriaLabel = "Close drawer",
  title = "Drawer",
  headerActions,
  headerExtra,
  panelTestId,
  size = "full",
  avoidKeyboard = false,
  children,
}: DrawerShellProps) {
  if (size === "compact") {
    return (
      <BottomSheet
        open={open}
        onClose={onClose}
        title={title}
        ariaLabel={typeof title === "string" ? title : closeAriaLabel}
        headerActions={headerActions}
        closeLabel={closeAriaLabel}
        avoidKeyboard={avoidKeyboard}
        testId={panelTestId}
      >
        {headerExtra}
        {children}
      </BottomSheet>
    );
  }

  return (
    <FullPageDrawer
      open={open}
      onClose={onClose}
      title={title}
      ariaLabel={typeof title === "string" ? title : closeAriaLabel}
      headerActions={headerActions}
      headerExtra={headerExtra}
      closeLabel={closeAriaLabel}
      avoidKeyboard={avoidKeyboard}
      testId={panelTestId}
    >
      {children}
    </FullPageDrawer>
  );
}

export default DrawerShell;
