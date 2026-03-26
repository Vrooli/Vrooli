import { type Config } from "tailwindcss";

/*
 * Maps CSS design tokens (defined in styles.css) into Tailwind utilities.
 * Usage: `bg-brand-surface-base`, `text-brand-text-muted`, `rounded-token`, etc.
 *
 * Token values use CSS custom properties with RGB triples so Tailwind's
 * opacity modifier (e.g. `bg-brand-surface-base/50`) works correctly.
 */
const tokenColor = (varName: string) => `rgb(var(${varName}))`;

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        brand: {
          text: {
            DEFAULT: tokenColor("--color-text-primary"),
            muted: tokenColor("--color-text-muted"),
            faint: tokenColor("--color-text-faint"),
          },
          surface: {
            base: tokenColor("--color-surface-base"),
            raised: tokenColor("--color-surface-raised"),
            hover: tokenColor("--color-surface-hover"),
          },
          border: {
            DEFAULT: tokenColor("--color-border-default"),
            focus: tokenColor("--color-border-focus"),
          },
          success: tokenColor("--color-success"),
          danger: {
            DEFAULT: tokenColor("--color-danger"),
            text: tokenColor("--color-danger-text"),
          },
        },
      },
      borderRadius: {
        brand: "var(--radius-md)",
        "brand-lg": "var(--radius-lg)",
      },
      transitionDuration: {
        brand: "var(--motion-fast)",
      },
    },
  },
  plugins: [],
} satisfies Config;
