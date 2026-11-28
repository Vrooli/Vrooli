import type { ReactNode } from "react";

import "./tokens.css";

export function ThemeProvider({ children }: { children: ReactNode }) {
  return <>{children}</>;
}
