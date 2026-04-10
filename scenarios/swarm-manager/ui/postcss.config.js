export default {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
    // Converts modern CSS color syntax (e.g. `rgb(34 211 238 / 0.5)`)
    // to legacy `rgba(34, 211, 238, 0.5)` for Chromium <78 compatibility
    // (Google TV and other embedded browsers).
    'postcss-preset-env': {
      features: {
        'color-functional-notation': true,
      },
      // Disable all other features — autoprefixer and Tailwind handle the rest.
      stage: false,
    },
  }
};
