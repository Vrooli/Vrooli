import type { Config } from 'tailwindcss';
import animate from 'tailwindcss-animate';

const config: Config = {
  darkMode: ['class'],
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    container: {
      center: true,
      padding: '1.25rem',
      screens: {
        '2xl': '1280px',
      },
    },
    extend: {
      colors: {
        background: 'hsl(222 47% 11%)',
        foreground: 'hsl(216 33% 97%)',
        accent: {
          DEFAULT: 'hsl(200 88% 60%)',
          foreground: 'hsl(216 33% 97%)',
        },
        muted: {
          DEFAULT: 'hsl(222 37% 18%)',
          foreground: 'hsl(219 18% 65%)',
        },
        border: 'hsl(217 33% 17%)',
        ring: 'hsl(200 88% 60%)',
        card: {
          DEFAULT: 'hsl(223 35% 14%)',
          foreground: 'hsl(216 33% 97%)',
        },
        success: {
          DEFAULT: 'hsl(151 55% 45%)',
          foreground: 'hsl(168 44% 15%)',
        },
        warning: {
          DEFAULT: 'hsl(38 92% 50%)',
          foreground: 'hsl(27 36% 12%)',
        },
        danger: {
          DEFAULT: 'hsl(0 84% 60%)',
          foreground: 'hsl(0 0% 100%)',
        },
      },
      fontFamily: {
        sans: ['"Inter"', 'system-ui', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'SFMono-Regular', 'monospace'],
      },
      borderRadius: {
        lg: '0.9rem',
        md: '0.65rem',
        sm: '0.5rem',
      },
      backgroundImage: {
        'grid-overlay': 'radial-gradient(circle at 1px 1px, rgba(56, 189, 248, 0.12) 0, rgba(2, 6, 23, 0.1) 1px)',
      },
      keyframes: {
        "pulse-soft": {
          '0%, 100%': { opacity: '0.45' },
          '50%': { opacity: '1' },
        },
      },
      animation: {
        'pulse-soft': 'pulse-soft 2.5s ease-in-out infinite',
      },
    },
  },
  plugins: [animate],
};

export default config;
