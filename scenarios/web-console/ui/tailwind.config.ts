import { type Config } from "tailwindcss";

/**
 * Design token mapping: CSS custom properties → Tailwind utility classes.
 *
 * Tokens are defined in src/styles.css as `--wc-*` variables.
 * Components use semantic class names (e.g. `bg-wc-surface-base`)
 * instead of raw Tailwind palette classes (e.g. `bg-slate-950`).
 */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        wc: {
          "surface-base": "rgb(var(--wc-surface-base) / <alpha-value>)",
          "surface-raised": "rgb(var(--wc-surface-raised) / <alpha-value>)",
          "surface-header": "rgb(var(--wc-surface-header) / <alpha-value>)",
          "surface-input": "rgb(var(--wc-surface-input) / <alpha-value>)",
          "text-primary": "rgb(var(--wc-text-primary) / <alpha-value>)",
          "text-secondary": "rgb(var(--wc-text-secondary) / <alpha-value>)",
          "text-muted": "rgb(var(--wc-text-muted) / <alpha-value>)",
          "text-faint": "rgb(var(--wc-text-faint) / <alpha-value>)",
          accent: "rgb(var(--wc-accent) / <alpha-value>)",
          "accent-fg": "rgb(var(--wc-accent-fg) / <alpha-value>)",
          "error-text": "rgb(var(--wc-error-text) / <alpha-value>)",
          "error-detail": "rgb(var(--wc-error-detail) / <alpha-value>)",
        },
        // Border/overlay tokens use pre-composed alpha, reference directly
      },
      borderColor: {
        "wc-default": "rgb(var(--wc-border-default))",
        "wc-hover": "rgb(var(--wc-border-hover))",
        "wc-accent": "rgb(var(--wc-accent-border))",
        "wc-error": "rgb(var(--wc-error-border))",
      },
      height: {
        /* h-wc-app: the viewport-aware app height.
           --wc-app-height is set by the useAppViewport hook to
           visualViewport.height (the real visible area, excluding
           browser chrome and virtual keyboard). Use this instead of
           h-screen (100vh) which overshoots on mobile.
           Fallback 100dvh handles pre-hook-initialization (loading/error
           states that render before the Workspace mounts the hook). */
        "wc-app": "var(--wc-app-height, 100dvh)",
      },
      backgroundColor: {
        "wc-backdrop": "rgb(var(--wc-backdrop))",
        "wc-backdrop-heavy": "rgb(var(--wc-backdrop-heavy))",
        "wc-error-surface": "rgb(var(--wc-error-surface))",
        "wc-accent-active": "rgb(var(--wc-accent-active))",
      },
    },
  },
  plugins: [],
} satisfies Config;
