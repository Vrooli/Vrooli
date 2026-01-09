import type { Config } from 'tailwindcss';
import animate from 'tailwindcss-animate';

const config: Config = {
  darkMode: ['class'],
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        background: '#050608',
        foreground: '#e3ffe8',
        accent: '#5effc4',
        accentMuted: '#0a3626',
        panel: 'rgba(8, 32, 24, 0.82)',
        panelBorder: '#3bff9d',
        panelGlow: 'rgba(20, 255, 153, 0.18)',
        danger: '#ff6b93'
      },
      fontFamily: {
        display: ['"VT323"', 'monospace'],
        sans: ['"Space Grotesk"', 'monospace']
      },
      boxShadow: {
        glow: '0 20px 45px rgba(20, 255, 153, 0.22)'
      },
      keyframes: {
        flicker: {
          '0%, 100%': { opacity: '1' },
          '41.99%': { opacity: '1' },
          '42%': { opacity: '0.8' },
          '43%': { opacity: '1' },
          '70%': { opacity: '0.97' },
          '71%': { opacity: '1' }
        },
        pulseSoft: {
          '0%, 100%': { opacity: '0.82' },
          '50%': { opacity: '1' }
        }
      },
      animation: {
        flicker: 'flicker 3s infinite steps(1)',
        pulseSoft: 'pulseSoft 4s ease-in-out infinite'
      }
    }
  },
  plugins: [animate]
};

export default config;
