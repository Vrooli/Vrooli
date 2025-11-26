import { type Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {},
    screens: {
      'sm': '640px',   // Mobile landscape / small tablets
      'md': '768px',   // Tablets
      'lg': '1024px',  // Desktop / laptop
      'xl': '1280px',  // Large desktop
      '2xl': '1536px', // Extra large desktop
    }
  },
  plugins: []
} satisfies Config;
