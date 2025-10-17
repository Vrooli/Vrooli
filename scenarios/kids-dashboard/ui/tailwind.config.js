/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'kid-red': '#FF6B6B',
        'kid-teal': '#4ECDC4',
        'kid-yellow': '#FFE66D',
        'kid-green': '#95E77E',
        'kid-purple': '#B19CD9',
        'kid-orange': '#FFB347',
        'kid-pink': '#FFB6C1',
        'kid-blue': '#87CEEB',
      },
      fontFamily: {
        'fredoka': ['Fredoka', 'cursive'],
        'comic': ['Comic Neue', 'cursive'],
      },
      animation: {
        'bounce-slow': 'bounce 2s infinite',
        'wiggle': 'wiggle 1s ease-in-out infinite',
        'float': 'float 3s ease-in-out infinite',
      },
      keyframes: {
        wiggle: {
          '0%, 100%': { transform: 'rotate(-3deg)' },
          '50%': { transform: 'rotate(3deg)' },
        },
        float: {
          '0%, 100%': { transform: 'translateY(0px)' },
          '50%': { transform: 'translateY(-20px)' },
        },
      },
    },
  },
  plugins: [],
}