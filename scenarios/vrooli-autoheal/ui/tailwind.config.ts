import { type Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        text: {
          primary: "rgb(var(--color-text-primary) / <alpha-value>)",
          muted: "rgb(var(--color-text-muted) / <alpha-value>)",
          inverse: "rgb(var(--color-text-inverse) / <alpha-value>)",
          danger: "rgb(var(--color-text-danger) / <alpha-value>)",
        },
        surface: {
          base: "rgb(var(--color-surface-base) / <alpha-value>)",
          elevated: "rgb(var(--color-surface-elevated) / <alpha-value>)",
          overlay: "rgb(var(--color-surface-overlay) / <alpha-value>)",
        },
        border: {
          default: "rgb(var(--color-border-default) / <alpha-value>)",
          subtle: "rgb(var(--color-border-subtle) / <alpha-value>)",
          strong: "rgb(var(--color-border-strong) / <alpha-value>)",
        },
        accent: {
          primary: "rgb(var(--color-accent-primary) / <alpha-value>)",
          success: "rgb(var(--color-accent-success) / <alpha-value>)",
          warning: "rgb(var(--color-accent-warning) / <alpha-value>)",
          danger: "rgb(var(--color-accent-danger) / <alpha-value>)",
        },
      },
      borderRadius: {
        sm: "var(--radius-sm)",
        md: "var(--radius-md)",
        lg: "var(--radius-lg)",
        pill: "var(--radius-pill)",
      },
      boxShadow: {
        panel: "var(--shadow-panel)",
      },
      transitionDuration: {
        fast: "var(--motion-fast)",
        normal: "var(--motion-normal)",
        slow: "var(--motion-slow)",
      },
    }
  },
  plugins: []
} satisfies Config;
