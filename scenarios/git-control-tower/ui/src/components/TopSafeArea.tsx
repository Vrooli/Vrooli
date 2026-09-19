import type { ReactNode } from "react";
import { StatusBarFill } from "@vrooli/react-component-library/ChromeTheme";

export function TopSafeArea({ children, testId = "mobile-chrome" }: { children: ReactNode; testId?: string }) {
  return (
    <div className="gct-mobile-chrome flex shrink-0 flex-col" data-testid={testId}>
      <div data-testid="status-bar-fill" role="region" aria-label="Mobile status bar fill" className="gct-status-bar-fill">
        <StatusBarFill
          testId="status-bar-fill-strip"
          style={{ blockSize: "env(safe-area-inset-top, 0px)" }}
        />
      </div>
      {children}
    </div>
  );
}

export default TopSafeArea;
