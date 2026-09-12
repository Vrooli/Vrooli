import { createContext, useContext } from "react";

export interface ShellNavigationState {
  sidebarCollapsed: boolean;
  openSidebar: () => void;
}

const unavailable: ShellNavigationState = { sidebarCollapsed: false, openSidebar: () => undefined };
export const ShellNavigationContext = createContext<ShellNavigationState>(unavailable);
export const useShellNavigation = () => useContext(ShellNavigationContext);
