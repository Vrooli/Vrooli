import { useEffect } from 'react';

type ThemeMode = 'light' | 'dark';

export function useThemeClass(theme: ThemeMode) {
  useEffect(() => {
    if (typeof document === 'undefined') {
      return undefined;
    }

    const body = document.body;
    const root = document.documentElement;

    body.classList.remove('light', 'dark');
    root.classList.remove('light', 'dark');

    body.classList.add(theme);
    root.classList.add(theme);
    root.style.setProperty('color-scheme', theme);

    return () => {
      body.classList.remove(theme);
      root.classList.remove(theme);
    };
  }, [theme]);
}
