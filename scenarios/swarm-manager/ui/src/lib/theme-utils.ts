export type ThemePreference = "dark" | "light" | "system";
export type ResolvedTheme = "dark" | "light";

const prefersDarkQuery = "(prefers-color-scheme: dark)";

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
