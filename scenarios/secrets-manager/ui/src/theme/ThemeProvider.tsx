import { useEffect, useMemo, useState, type ReactNode } from "react";
import { ThemeContext, type ThemeChoice } from "./ThemeContext";

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [choice, setChoice] = useState<ThemeChoice>(() => {
    const stored = typeof window === "undefined" ? null : window.localStorage.getItem("secrets-manager.theme");
    return stored === "light" || stored === "dark" ? stored : "system";
  });
  useEffect(() => {
    const resolved = choice === "system" && typeof window !== "undefined" && window.matchMedia?.("(prefers-color-scheme: dark)").matches ? "dark" : choice === "system" ? "light" : choice;
    document.documentElement.dataset.theme = resolved;
    document.documentElement.style.colorScheme = resolved;
    if (typeof window !== "undefined") window.localStorage.setItem("secrets-manager.theme", choice);
  }, [choice]);
  const value = useMemo(() => ({ choice, setTheme: setChoice }), [choice]);
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}
