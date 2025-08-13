/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./src/**/*.{js,jsx,ts,tsx}",
    "./public/index.html",
  ],
  theme: {
    extend: {
      colors: {
        neon: {
          pink: '#ff00ff',
          cyan: '#00ffff',
          green: '#00ff00',
          orange: '#ff8000',
          purple: '#8000ff',
          yellow: '#ffff00',
        },
        retro: {
          dark: '#0a0a0a',
          purple: '#1a0533',
          blue: '#0f1419',
          pink: '#2d1b42',
        }
      },
      fontFamily: {
        'mono': ['Monaco', 'Menlo', 'Consolas', 'monospace'],
        'pixel': ['Orbitron', 'monospace'],
      },
      animation: {
        'pulse-neon': 'pulse-neon 2s ease-in-out infinite alternate',
        'glow': 'glow 2s ease-in-out infinite alternate',
        'scan': 'scan 2s linear infinite',
        'flicker': 'flicker 0.15s infinite linear',
      },
      keyframes: {
        'pulse-neon': {
          'from': {
            textShadow: '0 0 5px #ff00ff, 0 0 10px #ff00ff, 0 0 15px #ff00ff, 0 0 20px #ff00ff',
          },
          'to': {
            textShadow: '0 0 2px #ff00ff, 0 0 5px #ff00ff, 0 0 8px #ff00ff, 0 0 12px #ff00ff',
          }
        },
        'glow': {
          'from': {
            boxShadow: '0 0 5px #00ffff, 0 0 10px #00ffff, 0 0 15px #00ffff',
          },
          'to': {
            boxShadow: '0 0 10px #00ffff, 0 0 20px #00ffff, 0 0 30px #00ffff',
          }
        },
        'scan': {
          '0%': { transform: 'translateY(-100%)' },
          '100%': { transform: 'translateY(100vh)' }
        },
        'flicker': {
          '0%, 100%': { opacity: 1 },
          '50%': { opacity: 0.8 }
        }
      },
      backgroundImage: {
        'grid-pattern': 'linear-gradient(rgba(0,255,255,0.1) 1px, transparent 1px), linear-gradient(90deg, rgba(0,255,255,0.1) 1px, transparent 1px)',
        'retro-gradient': 'linear-gradient(135deg, #1a0533 0%, #0f1419 50%, #2d1b42 100%)',
      },
      backgroundSize: {
        'grid': '20px 20px',
      }
    },
  },
  plugins: [],
}