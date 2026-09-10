import type { ReactNode } from "react";
import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";

interface AppShellProps {
  children: ReactNode;
}

/** The shared shell owns the application landmark and responsive geometry. */
export function AppShell({ children }: AppShellProps) {
  return (
    <LibraryAppShell
      brand="Vrooli"
      brandMark={<span aria-hidden="true">V</span>}
      items={[]}
      mobileNav="tabs"
      mainMode="fill"
      navigationLabel=""
      skipLabel="Skip to main content"
      className="onboarding-rcl-shell min-h-full"
      mainClassName="min-h-full p-0"
    >
      {children}
    </LibraryAppShell>
  );
}
