/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        'flow-bg': '#0d0e14',
        'flow-node': '#1a1d29',
        'flow-node-hover': '#252837',
        'flow-edge': '#4a5568',
        'flow-accent': '#3b82f6',
        'flow-success': '#10b981',
        'flow-error': '#ef4444',
        'flow-warning': '#f59e0b',
        'terminal-bg': '#0c0d13',
        'terminal-text': '#00ff00',
      },
      fontFamily: {
        'mono': ['JetBrains Mono', 'Fira Code', 'monospace'],
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