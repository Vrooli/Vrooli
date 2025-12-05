import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright test configuration for selector validation tests.
 *
 * Run with:
 *   pnpm test:selectors           - Run in headless mode
 *   pnpm test:selectors:headed    - Run with visible browser
 */
export default defineConfig({
  testDir: './src',
  testMatch: '**/*.test.ts',

  // Fail fast during development
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,

  // Reporter
  reporter: [
    ['list'],
    ['html', { outputFolder: 'playwright-report' }],
  ],

  // Shared settings
  use: {
    // Base URL for relative paths
    baseURL: 'https://example.com',

    // Collect trace on failure
    trace: 'on-first-retry',

    // Screenshot on failure
    screenshot: 'only-on-failure',

    // Timeout settings
    actionTimeout: 10000,
    navigationTimeout: 30000,
  },

  // Project configuration
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Timeout for each test
  timeout: 60000,

  // Output directory
  outputDir: 'test-results',
});
