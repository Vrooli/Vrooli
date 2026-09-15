import { type Config } from "tailwindcss";
import theme from "./src/theme/tailwind.theme.json";

export default {
  content: [
    "./index.html",
    "./src/**/*.{ts,tsx}",
    "../library/components/*/versions/*/*.{ts,tsx}",
    // BRIDGE (remove when the library emits no utility classes — see gate
    // catalog.utility-class). Some published RCL versions still ship Tailwind class strings; without
    // this glob their styling is purged from this app's bundle with no error.
    "./node_modules/@vrooli/react-component-library/dist/**/*.js",
  ],
  theme: {
    screens: {
      sm: "640px",
      md: "768px",
      lg: "1024px",
      xl: "1280px",
      "2xl": "1536px",
    },
    extend: theme,
  },
  plugins: [],
} satisfies Config;
