import { test, expect } from '@playwright/test';

test('loads main app without runtime React crash', async ({ page }) => {
  const errors: string[] = [];
  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      errors.push(msg.text());
    }
  });

  await page.goto('/');
  await page.waitForTimeout(2000);

  const hasReactContextError = errors.some((msg) => msg.includes('createContext'));
  expect(hasReactContextError, errors.join('\n')).toBeFalsy();

  await expect(page.locator('body')).toContainText('Workflows', { timeout: 5000 });
});
