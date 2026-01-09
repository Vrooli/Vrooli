import type { Config } from 'tailwindcss';
import animate from 'tailwindcss-animate';

const config: Config = {
  darkMode: ['class'],
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    container: {
      center: true,
      padding: '2rem',
      screens: {
        '2xl': '1280px'
      }
    },
    extend: {
      fontFamily: {
        sans: ['"Inter"', 'system-ui', 'sans-serif'],
        heading: ['"Manrope"', 'system-ui', 'sans-serif']
      },
      colors: {
        background: 'hsl(210 20% 98%)',
        foreground: 'hsl(222.2 47.4% 11.2%)',
        muted: {
          DEFAULT: 'hsl(210 16% 93%)',
          foreground: 'hsl(215 16.3% 46.9%)'
        },
        border: 'hsl(214 32% 91%)',
        card: {
          DEFAULT: '#ffffff',
          foreground: 'hsl(222.2 47.4% 11.2%)'
        },
        brand: {
          50: '#f3faf6',
          100: '#d6f3e3',
          200: '#b2e8cd',
          300: '#87d8b3',
          400: '#5cc996',
          500: '#35b57b',
          600: '#269265',
          700: '#1f7353',
          800: '#1b5c43',
          900: '#12402e'
        }
      },
      backgroundImage: {
        'hero-texture': 'radial-gradient(circle at 20% 20%, rgba(53,181,123,0.45), transparent 45%), radial-gradient(circle at 80% 10%, rgba(17,94,89,0.35), transparent 40%), radial-gradient(circle at 50% 80%, rgba(12,74,110,0.3), transparent 40%)'
      },
      borderRadius: {
        lg: '0.85rem',
        md: '0.6rem',
        sm: '0.45rem'
      },
      keyframes: {
        shimmer: {
          '0%': { backgroundPosition: '-700px 0' },
          '100%': { backgroundPosition: '700px 0' }
        },
        'fade-up': {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        }
      },
      animation: {
        shimmer: 'shimmer 1.8s ease-in-out infinite',
        'fade-up': 'fade-up 0.45s ease forwards'
      }
    }
  },
  plugins: [animate]
};

export default config;
