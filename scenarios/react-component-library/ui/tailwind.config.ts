import { type Config } from "tailwindcss";
import { APP_SCREENS } from "./src/styles/breakpoints";
import theme from "./tailwind.theme.json";

export default {
  content: [
    "./index.html",
    "./src/**/*.{ts,tsx}",
    "../library/components/*/versions/*/*.{ts,tsx}"
  ],
  theme: {
    screens: APP_SCREENS,
    extend: theme
  },
  plugins: []
} satisfies Config;
