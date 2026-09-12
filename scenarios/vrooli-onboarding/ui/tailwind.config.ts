import { type Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        surface: "var(--color-surface)",
        "surface-muted": "var(--color-surface-muted)",
        "surface-subtle": "var(--color-surface-subtle)",
        "surface-elevated": "var(--color-surface-elevated)",
        foreground: "var(--color-foreground)",
        "foreground-strong": "var(--color-foreground-strong)",
        muted: "var(--color-muted)",
        border: "var(--color-border)",
        "border-muted": "var(--color-border-muted)",
        focus: "var(--color-focus)",
        primary: "var(--color-primary)",
        "primary-hover": "var(--color-primary-hover)",
        "primary-soft": "var(--color-primary-soft)",
        "on-primary": "var(--color-on-primary)",
        danger: "var(--color-danger)",
        "danger-surface": "var(--color-danger-surface)",
        warning: "var(--color-warning)",
        "warning-surface": "var(--color-warning-surface)",
      },
    }
  },
  plugins: []
} satisfies Config;
