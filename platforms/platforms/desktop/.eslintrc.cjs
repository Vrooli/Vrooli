/** @type {import("eslint").Linter.Config} */
module.exports = {
  root: true, // Prevent ESLint from searching further up the directory tree
  extends: [
    '../../.eslintrc' // Inherit from the root configuration
  ],
  parserOptions: {
    tsconfigRootDir: __dirname, // Root directory for T\\S config search is this folder
    project: ['./tsconfig.json'], // Specify the tsconfig file relative to this folder
  },
  ignorePatterns: [
      'dist', // Ignore build output
      '.eslintrc.cjs' // Ignore this file itself
    ],
  settings: {
      // Adjust settings if needed, e.g., React version if using React in preload (unlikely)
      // 'react': {
      //   'version': 'detect',
      // },
  },
  // Add any platform-specific rules or overrides here if necessary
  // rules: {
  //   // Example: override a rule from the root config
  //   'no-console': 'warn',
  // },
}; 