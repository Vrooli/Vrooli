import { createContext, useContext } from "react";

export interface AppShellContextValue {
  openSidebar: () => void;
  closeSidebar: () => void;
  toggleSidebar: () => void;
}

const noop = () => {};

export const AppShellContext = createContext<AppShellContextValue>({
  openSidebar: noop,
  closeSidebar: noop,
  toggleSidebar: noop,
});

export function useAppShell(): AppShellContextValue {
  return useContext(AppShellContext);
}
