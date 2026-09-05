import { type Config } from "tailwindcss";
import tailwindcssAnimate from "tailwindcss-animate";

export default {
  content: [
    "./index.html",
    "./src/**/*.{ts,tsx}",
    // BRIDGE (remove when the library emits no utility classes — see gate
    // catalog.utility-class). Some published RCL versions still ship Tailwind class strings; without
    // this glob their styling is purged from this app's bundle with no error.
    "./node_modules/@vrooli/react-component-library/dist/**/*.js"
  ],
  theme: {
    extend: {
      colors: {
        // Uses rgba() with comma-separated variables for legacy browser
        // compatibility (Chromium <78). See styles.css header comment.
        slate: {
          50: "rgba(var(--slate-50), <alpha-value>)",
          100: "rgba(var(--slate-100), <alpha-value>)",
          200: "rgba(var(--slate-200), <alpha-value>)",
          300: "rgba(var(--slate-300), <alpha-value>)",
          400: "rgba(var(--slate-400), <alpha-value>)",
          500: "rgba(var(--slate-500), <alpha-value>)",
          600: "rgba(var(--slate-600), <alpha-value>)",
          700: "rgba(var(--slate-700), <alpha-value>)",
          800: "rgba(var(--slate-800), <alpha-value>)",
          900: "rgba(var(--slate-900), <alpha-value>)",
          950: "rgba(var(--slate-950), <alpha-value>)",
        },
      },
    }
  },
  plugins: [tailwindcssAnimate]
} satisfies Config;
