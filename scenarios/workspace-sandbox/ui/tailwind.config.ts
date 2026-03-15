import { type Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      keyframes: {
        "slide-up": {
          from: { transform: "translateY(100%)" },
          to: { transform: "translateY(0)" },
        },
        "slide-down": {
          from: { transform: "translateY(0)" },
          to: { transform: "translateY(100%)" },
        },
        "fade-in": {
          from: { opacity: "0" },
          to: { opacity: "1" },
        },
        "fade-out": {
          from: { opacity: "1" },
          to: { opacity: "0" },
        },
      },
      animation: {
        "slide-up": "slide-up 0.15s ease-out",
        "slide-down": "slide-down 0.15s ease-out",
        "fade-in": "fade-in 0.1s ease-out",
        "fade-out": "fade-out 0.1s ease-out",
      },
    },
  },
  plugins: [],
} satisfies Config;
