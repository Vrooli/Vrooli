import { type Config } from "tailwindcss";

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
    extend: {}
  },
  plugins: []
} satisfies Config;
