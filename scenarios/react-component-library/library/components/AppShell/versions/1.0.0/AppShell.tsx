/** @vrooliComponentSource react-component-library:AppShell */
import type { ReactNode } from "react";
export function AppShell({
  navigation,
  children,
}: {
  navigation?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div
      data-app-shell
      style={{
        minHeight: 480,
        background: "var(--color-background, #f8fafc)",
        color: "var(--color-foreground, #0f172a)",
        padding: 24,
      }}
    >
      {navigation}
      {children}
    </div>
  );
}
