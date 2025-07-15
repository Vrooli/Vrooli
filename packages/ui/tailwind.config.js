/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      // Use the same font family as MUI
      fontFamily: {
        sans: ['Roboto', 'Helvetica', 'Arial', 'sans-serif'],
      },
      // Custom animations for spinners and dialogs
      animation: {
        'spin': 'spin 1s linear infinite',
        'pulse': 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'float': 'float 3s ease-in-out infinite',
        'float-delayed': 'float-delayed 3s ease-in-out infinite 1.5s',
        'scale-in': 'scale-in 0.2s ease-out',
        'success-pop': 'success-pop 0.5s cubic-bezier(0.68, -0.55, 0.265, 1.55)',
        'slide-infinite': 'slide-infinite 3s linear infinite',
      },
      keyframes: {
        float: {
          '0%, 100%': { transform: 'translateY(0) scale(1)' },
          '50%': { transform: 'translateY(-20px) scale(1.1)' },
        },
        'float-delayed': {
          '0%, 100%': { transform: 'translateX(0) scale(1)' },
          '50%': { transform: 'translateX(20px) scale(1.1)' },
        },
        'scale-in': {
          from: { transform: 'scale(0)', opacity: '0' },
          to: { transform: 'scale(1)', opacity: '1' },
        },
        'success-pop': {
          '0%': { transform: 'scale(0)', opacity: '0' },
          '50%': { transform: 'scale(1.2)', opacity: '1' },
          '100%': { transform: 'scale(1)', opacity: '1' },
        },
        'slide-infinite': {
          '0%': { transform: 'translateX(-100%)' },
          '100%': { transform: 'translateX(100%)' },
        },
      },
      // Sync with MUI theme colors
      colors: {
        primary: {
          50: '#e3f2fd',
          100: '#bbdefb',
          200: '#90caf9',
          300: '#64b5f6',
          400: '#42a5f5',
          500: '#2196f3',
          600: '#1e88e5',
          700: '#1976d2',
          800: '#1565c0',
          900: '#0d47a1',
          main: 'var(--primary-main)',
          dark: 'var(--primary-dark)',
          light: 'var(--primary-light)',
        },
        secondary: {
          main: 'var(--secondary-main)',
          dark: 'var(--secondary-dark)',
          light: 'var(--secondary-light)',
        },
        background: {
          default: 'var(--background-default)',
          paper: 'var(--background-paper)',
        },
        text: {
          primary: 'var(--text-primary)',
          secondary: 'var(--text-secondary)',
        },
        danger: {
          main: 'var(--danger-main)',
          dark: 'var(--danger-dark)',
          light: 'var(--danger-light)',
        },
        success: {
          main: 'var(--success-main)',
          dark: 'var(--success-dark)',
          light: 'var(--success-light)',
        },
        warning: {
          main: 'var(--warning-main)',
          dark: 'var(--warning-dark)',
          light: 'var(--warning-light)',
        },
        info: {
          main: 'var(--info-main)',
          dark: 'var(--info-dark)',
          light: 'var(--info-light)',
        }
      },
      fontSize: {
        // Dynamic font sizes that sync with theme
        'dynamic-xs': ['var(--font-size-xs)', '1.25'],
        'dynamic-sm': ['var(--font-size-sm)', '1.375'],
        'dynamic-base': ['var(--font-size-base)', '1.5'],
        'dynamic-lg': ['var(--font-size-lg)', '1.75'],
        'dynamic-xl': ['var(--font-size-xl)', '1.75'],
      },
      spacing: {
        // Add spacing that matches MUI's spacing function
        '0.5': '0.125rem', // theme.spacing(0.5)
        '1': '0.25rem',    // theme.spacing(1)
        '2': '0.5rem',     // theme.spacing(2)
        '3': '0.75rem',    // theme.spacing(3)
        '3.5': '0.875rem',
        '4': '1rem',       // theme.spacing(4)
        '4.5': '1.125rem',
        '5': '1.25rem',    // theme.spacing(5)
        '5.5': '1.375rem',
        '6': '1.5rem',     // theme.spacing(6)
        '7': '1.75rem',    // theme.spacing(7)
        '8': '2rem',       // theme.spacing(8)
        '9': '2.25rem',    // theme.spacing(9)
        '10': '2.5rem',    // theme.spacing(10)
      }
    },
  },
  plugins: [
    // Add support for indeterminate pseudo-class modifier
    function({ addVariant }) {
      addVariant('peer-indeterminate', ':merge(.peer):indeterminate ~ &')
    }
  ],
  // Important: Add prefix to avoid conflicts during transition
  prefix: 'tw-',
  corePlugins: {
    // Disable Tailwind's CSS reset to avoid conflicts with MUI
    preflight: false,
  }
}