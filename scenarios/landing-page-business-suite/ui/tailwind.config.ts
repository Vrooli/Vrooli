import { type Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
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
