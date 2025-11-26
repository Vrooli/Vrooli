import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright configuration for chart-generator integration tests
 * See https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: './test',
  testMatch: '**/*.spec.ts',

  // Timeout per test
  timeout: 30000,

  // Fail the build on CI if tests were accidentally marked as test.only
  forbidOnly: !!process.env.CI,

  // Retry on CI only
  retries: process.env.CI ? 2 : 0,

  // Reporter
  reporter: [
    ['list'],
    ['json', { outputFile: 'test/artifacts/playwright-results.json' }],
  ],

  use: {
    // Base URL for tests
    baseURL: `http://localhost:${process.env.UI_PORT || '37957'}`,

    // Collect trace when retrying failed test
    trace: 'on-first-retry',

    // Screenshot on failure
    screenshot: 'only-on-failure',

    // Video on failure
    video: 'retain-on-failure',
  },

  // Configure projects for different browsers (focus on chromium for now)
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Run local dev server before tests
  webServer: undefined, // Scenario is already running via lifecycle
});
