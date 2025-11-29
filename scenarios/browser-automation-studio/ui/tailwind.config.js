/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: ['selector', '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        // Override gray scale with theme-aware CSS variables
        gray: {
          50: 'rgb(var(--gray-50) / <alpha-value>)',
          100: 'rgb(var(--gray-100) / <alpha-value>)',
          200: 'rgb(var(--gray-200) / <alpha-value>)',
          300: 'rgb(var(--gray-300) / <alpha-value>)',
          400: 'rgb(var(--gray-400) / <alpha-value>)',
          500: 'rgb(var(--gray-500) / <alpha-value>)',
          600: 'rgb(var(--gray-600) / <alpha-value>)',
          700: 'rgb(var(--gray-700) / <alpha-value>)',
          800: 'rgb(var(--gray-800) / <alpha-value>)',
          900: 'rgb(var(--gray-900) / <alpha-value>)',
          950: 'rgb(var(--gray-950) / <alpha-value>)',
        },
        // Override white to be theme-aware (white in dark, dark in light)
        white: 'rgb(var(--color-white) / <alpha-value>)',
        // Override black to be theme-aware
        black: 'rgb(var(--color-black) / <alpha-value>)',
        // Theme-aware colors using CSS variables with opacity support
        'flow-bg': 'rgb(var(--flow-bg) / <alpha-value>)',
        'flow-bg-secondary': 'rgb(var(--flow-bg-secondary) / <alpha-value>)',
        'flow-node': 'rgb(var(--flow-node) / <alpha-value>)',
        'flow-node-hover': 'rgb(var(--flow-node-hover) / <alpha-value>)',
        'flow-edge': 'rgb(var(--flow-edge) / <alpha-value>)',
        'flow-accent': 'rgb(var(--flow-accent) / <alpha-value>)',
        'flow-accent-hover': 'rgb(var(--flow-accent-hover) / <alpha-value>)',
        'flow-text': 'rgb(var(--flow-text) / <alpha-value>)',
        'flow-text-secondary': 'rgb(var(--flow-text-secondary) / <alpha-value>)',
        'flow-text-muted': 'rgb(var(--flow-text-muted) / <alpha-value>)',
        'flow-border': 'rgb(var(--flow-border) / <alpha-value>)',
        'flow-border-light': 'rgb(var(--flow-border-light) / <alpha-value>)',
        'flow-success': 'rgb(var(--flow-success) / <alpha-value>)',
        'flow-error': 'rgb(var(--flow-error) / <alpha-value>)',
        'flow-warning': 'rgb(var(--flow-warning) / <alpha-value>)',
        'flow-surface': 'rgb(var(--flow-surface) / <alpha-value>)',
        'flow-surface-hover': 'rgb(var(--flow-surface-hover) / <alpha-value>)',
        // Static colors (not theme-dependent)
        'terminal-bg': '#0c0d13',
        'terminal-text': '#00ff00',
      },
      fontFamily: {
        'mono': ['JetBrains Mono', 'Fira Code', 'monospace'],
        'sans': ['ui-sans-serif', 'system-ui', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'Helvetica Neue', 'Arial', 'sans-serif'],
        'system': ['system-ui', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'Helvetica Neue', 'Arial', 'sans-serif'],
      },
      fontSize: {
        'xs': 'var(--font-size-xs)',
        'sm': 'var(--font-size-sm)',
        'base': 'var(--font-size-base)',
        'lg': 'var(--font-size-lg)',
        'xl': 'var(--font-size-xl)',
        '2xl': 'var(--font-size-2xl)',
      },
      // Spacing that respects compact mode
      spacing: {
        'compact-1': 'var(--space-1)',
        'compact-2': 'var(--space-2)',
        'compact-3': 'var(--space-3)',
        'compact-4': 'var(--space-4)',
        'compact-5': 'var(--space-5)',
        'compact-6': 'var(--space-6)',
        'compact-8': 'var(--space-8)',
        'card': 'var(--card-padding)',
        'section': 'var(--section-gap)',
        'item': 'var(--item-gap)',
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'slide-in': 'slideIn 0.3s ease-out',
      },
      keyframes: {
        slideIn: {
          '0%': { transform: 'translateX(100%)' },
          '100%': { transform: 'translateX(0)' },
        },
      },
    },
  },
  plugins: [],
}