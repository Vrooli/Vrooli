/**
 * [REQ:CHART-P0-001-BAR,CHART-P0-001-LINE,CHART-P0-001-PIE,CHART-P0-001-SCATTER,CHART-P0-001-AREA]
 * [REQ:CHART-P0-003-LIGHT,CHART-P0-003-MINIMAL]
 * [REQ:CHART-P0-007-UI-PREVIEW]
 *
 * Integration tests for chart-generator UI workflows
 * Validates chart generation, style application, and UI interactions
 */

import { test, expect } from '@playwright/test';

const UI_PORT = process.env.UI_PORT || '37957';
const BASE_URL = `http://localhost:${UI_PORT}`;

// Selectors (matching ui/src/consts/selectors.ts and actual UI implementation)
const SELECTORS = {
  CHART_PREVIEW_CONTAINER: '[data-testid="chart-preview"]',
  CHART_SVG: '[data-testid="chart-svg"]',
  CHART_TYPE_BAR: '[data-testid="chart-type-bar"]',
  CHART_TYPE_LINE: '[data-testid="chart-type-line"]',
  CHART_TYPE_PIE: '[data-testid="chart-type-pie"]',
  CHART_TYPE_SCATTER: '[data-testid="chart-type-scatter"]',
  CHART_TYPE_AREA: '[data-testid="chart-type-area"]',
  TAB_CHARTS: '[data-testid="tab-charts"]',
  TAB_STYLES: '[data-testid="tab-styles"]',
  STYLE_PROFESSIONAL: '[data-testid="style-professional"]',
  STYLE_MINIMAL: '[data-testid="style-minimal"]',
  STYLE_VIBRANT: '[data-testid="style-vibrant"]',
};

test.describe('Chart Generator Integration Tests', () => {

  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    // Wait for UI to be ready
    await page.waitForLoadState('networkidle');
  });

  // [REQ:CHART-P0-007-UI-PREVIEW]
  test('UI loads successfully', async ({ page }) => {
    // Verify chart preview container is visible
    await expect(page.locator(SELECTORS.CHART_PREVIEW_CONTAINER)).toBeVisible({ timeout: 5000 });

    // Verify page title
    await expect(page).toHaveTitle(/Chart Generator/i);
  });

  // [REQ:CHART-P0-001-BAR]
  test('Bar chart generation', async ({ page }) => {
    // Wait for chart preview container
    await page.waitForSelector(SELECTORS.CHART_PREVIEW_CONTAINER, { timeout: 5000 });

    // Click bar chart type selector
    await page.click(SELECTORS.CHART_TYPE_BAR);

    // Wait for chart to render
    await page.waitForTimeout(500);

    // Verify SVG is visible
    await expect(page.locator(SELECTORS.CHART_SVG)).toBeVisible();
  });

  // [REQ:CHART-P0-001-LINE]
  test('Line chart generation', async ({ page }) => {
    await page.waitForSelector(SELECTORS.CHART_PREVIEW_CONTAINER, { timeout: 5000 });
    await page.click(SELECTORS.CHART_TYPE_LINE);
    await page.waitForTimeout(500);
    await expect(page.locator(SELECTORS.CHART_SVG)).toBeVisible();
  });

  // [REQ:CHART-P0-001-PIE]
  test('Pie chart generation', async ({ page }) => {
    await page.waitForSelector(SELECTORS.CHART_PREVIEW_CONTAINER, { timeout: 5000 });
    await page.click(SELECTORS.CHART_TYPE_PIE);
    await page.waitForTimeout(500);
    await expect(page.locator(SELECTORS.CHART_SVG)).toBeVisible();
  });

  // [REQ:CHART-P0-001-SCATTER]
  test('Scatter chart generation', async ({ page }) => {
    await page.waitForSelector(SELECTORS.CHART_PREVIEW_CONTAINER, { timeout: 5000 });
    await page.click(SELECTORS.CHART_TYPE_SCATTER);
    await page.waitForTimeout(500);
    await expect(page.locator(SELECTORS.CHART_SVG)).toBeVisible();
  });

  // [REQ:CHART-P0-001-AREA]
  test('Area chart generation', async ({ page }) => {
    await page.waitForSelector(SELECTORS.CHART_PREVIEW_CONTAINER, { timeout: 5000 });
    await page.click(SELECTORS.CHART_TYPE_AREA);
    await page.waitForTimeout(500);
    await expect(page.locator(SELECTORS.CHART_SVG)).toBeVisible();
  });

  // [REQ:CHART-P0-003-LIGHT]
  test('Apply professional theme', async ({ page }) => {
    await page.waitForSelector(SELECTORS.CHART_PREVIEW_CONTAINER, { timeout: 5000 });

    // Navigate to Styles tab
    await page.click(SELECTORS.TAB_STYLES);
    await page.waitForTimeout(300);

    // Apply professional style
    await page.click(SELECTORS.STYLE_PROFESSIONAL);
    await page.waitForTimeout(300);

    // Navigate back to Charts tab to see chart with new style
    await page.click(SELECTORS.TAB_CHARTS);
    await page.waitForTimeout(300);

    // Verify chart is visible with new style
    await expect(page.locator(SELECTORS.CHART_SVG)).toBeVisible();
  });

  // [REQ:CHART-P0-003-MINIMAL]
  test('Apply minimal theme', async ({ page }) => {
    await page.waitForSelector(SELECTORS.CHART_PREVIEW_CONTAINER, { timeout: 5000 });

    // Navigate to Styles tab
    await page.click(SELECTORS.TAB_STYLES);
    await page.waitForTimeout(300);

    // Apply minimal style
    await page.click(SELECTORS.STYLE_MINIMAL);
    await page.waitForTimeout(300);

    // Navigate back to Charts tab
    await page.click(SELECTORS.TAB_CHARTS);
    await page.waitForTimeout(300);

    await expect(page.locator(SELECTORS.CHART_SVG)).toBeVisible();
  });

  // Vibrant theme test
  test('Apply vibrant theme', async ({ page }) => {
    await page.waitForSelector(SELECTORS.CHART_PREVIEW_CONTAINER, { timeout: 5000 });

    // Navigate to Styles tab
    await page.click(SELECTORS.TAB_STYLES);
    await page.waitForTimeout(300);

    // Apply vibrant style
    await page.click(SELECTORS.STYLE_VIBRANT);
    await page.waitForTimeout(300);

    // Navigate back to Charts tab
    await page.click(SELECTORS.TAB_CHARTS);
    await page.waitForTimeout(300);

    await expect(page.locator(SELECTORS.CHART_SVG)).toBeVisible();
  });

});
