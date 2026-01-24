import { type Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        accent: "rgb(var(--color-accent) / <alpha-value>)",
        "accent-secondary": "rgb(var(--color-accent-secondary) / <alpha-value>)",
        "accent-tertiary": "rgb(var(--color-accent-tertiary) / <alpha-value>)",
        "accent-cool": "rgb(var(--color-accent-cool) / <alpha-value>)",
        success: "rgb(var(--color-success) / <alpha-value>)",
        warning: "rgb(var(--color-warning) / <alpha-value>)",
        "bg-base": "rgb(var(--color-bg-base) / <alpha-value>)",
        "surface-primary": "rgb(var(--color-surface-primary) / <alpha-value>)",
        "surface-muted": "rgb(var(--color-surface-muted) / <alpha-value>)",
        "surface-alt": "rgb(var(--color-surface-alt) / <alpha-value>)",
        "surface-deep": "rgb(var(--color-surface-deep) / <alpha-value>)",
        "surface-darker": "rgb(var(--color-surface-darker) / <alpha-value>)",
        "text-primary": "rgb(var(--color-text-primary) / <alpha-value>)",
        "text-secondary": "rgb(var(--color-text-secondary) / <alpha-value>)",
        "text-muted": "rgb(var(--color-text-muted) / <alpha-value>)",
        "flow-bg": "rgb(var(--flow-bg) / <alpha-value>)",
        "flow-bg-secondary": "rgb(var(--flow-bg-secondary) / <alpha-value>)",
        "flow-node": "rgb(var(--flow-node) / <alpha-value>)",
        "flow-node-hover": "rgb(var(--flow-node-hover) / <alpha-value>)",
        "flow-edge": "rgb(var(--flow-edge) / <alpha-value>)",
        "flow-accent": "rgb(var(--flow-accent) / <alpha-value>)",
        "flow-accent-hover": "rgb(var(--flow-accent-hover) / <alpha-value>)",
        "flow-text": "rgb(var(--flow-text) / <alpha-value>)",
        "flow-text-secondary": "rgb(var(--flow-text-secondary) / <alpha-value>)",
        "flow-text-muted": "rgb(var(--flow-text-muted) / <alpha-value>)",
        "flow-border": "rgb(var(--flow-border) / <alpha-value>)",
        "flow-border-light": "rgb(var(--flow-border-light) / <alpha-value>)",
        "flow-surface": "rgb(var(--flow-surface) / <alpha-value>)",
        "flow-surface-hover": "rgb(var(--flow-surface-hover) / <alpha-value>)"
      }
    }
  },
  plugins: []
} satisfies Config;
