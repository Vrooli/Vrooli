import { type Config } from "tailwindcss";
import { APP_SCREENS } from "./src/styles/breakpoints";
import theme from "./tailwind.theme.json";

export default {
  content: [
    "./index.html",
    "./src/**/*.{ts,tsx}",
    "../library/components/*/versions/*/*.{ts,tsx}",
    // BRIDGE (remove when the library emits no utility classes — see gate
    // catalog.utility-class). Some published RCL versions still ship Tailwind class strings; without
    // this glob their styling is purged from this app's bundle with no error.
    "./node_modules/@vrooli/react-component-library/dist/**/*.js"
  ],
  theme: {
    screens: APP_SCREENS,
    extend: theme
  },
  plugins: []
} satisfies Config;
