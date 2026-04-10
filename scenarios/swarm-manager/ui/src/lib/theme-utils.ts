import { useSyncExternalStore } from "react";

export type ThemePreference = "dark" | "light" | "system";
export type ResolvedTheme = "dark" | "light";

const prefersDarkQuery = "(prefers-color-scheme: dark)";
const themeChangeEvent = "vrooli-theme-change";

export function resolveTheme(theme: ThemePreference): ResolvedTheme {
  if (theme === "system" && typeof window !== "undefined") {
    return window.matchMedia(prefersDarkQuery).matches ? "dark" : "light";
  }
  return theme === "light" ? "light" : "dark";
}

export function applyTheme(theme: ThemePreference): ResolvedTheme {
  if (typeof document === "undefined") {
    return resolveTheme(theme);
  }

  const resolved = resolveTheme(theme);
  const root = document.documentElement;
  root.dataset.theme = theme;
  root.dataset.resolvedTheme = resolved;
  root.style.colorScheme = resolved;
  if (typeof window !== "undefined") {
    window.dispatchEvent(new CustomEvent(themeChangeEvent, { detail: resolved }));
  }
  return resolved;
}

export function watchSystemTheme(onChange: (resolved: ResolvedTheme) => void): () => void {
  if (typeof window === "undefined") {
    return () => undefined;
  }

  const media = window.matchMedia(prefersDarkQuery);
  const handler = () => onChange(media.matches ? "dark" : "light");

  if (media.addEventListener) {
    media.addEventListener("change", handler);
  } else {
    media.addListener(handler);
  }

  return () => {
    if (media.removeEventListener) {
      media.removeEventListener("change", handler);
    } else {
      media.removeListener(handler);
    }
  };
}

function getResolvedThemeSnapshot(): ResolvedTheme {
  if (typeof document === "undefined") {
    return "dark";
  }
  return document.documentElement.dataset.resolvedTheme === "light" ? "light" : "dark";
}

function subscribeThemeChange(callback: () => void): () => void {
  if (typeof window === "undefined") {
    return () => undefined;
  }
  const handler = () => callback();
  window.addEventListener(themeChangeEvent, handler);
  return () => window.removeEventListener(themeChangeEvent, handler);
}

export function useResolvedTheme(): ResolvedTheme {
  return useSyncExternalStore(subscribeThemeChange, getResolvedThemeSnapshot, () => "dark");
}
