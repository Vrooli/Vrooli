/**
 * Selector Spike Test
 *
 * Phase 0: Validate selector generation reliability across diverse sites.
 * Goal: >70% of selectors should survive a page refresh.
 *
 * Usage:
 *   npx ts-node tests/spike/selector-spike.ts
 *
 * Or run specific test:
 *   npx ts-node tests/spike/selector-spike.ts --site=github
 */

import { chromium, type Browser, type ElementHandle } from 'playwright';

// ============================================================================
// Configuration
// ============================================================================

interface TestSite {
  name: string;
  url: string;
  /** CSS selectors to find test elements */
  elementSelectors: string[];
  /** Expected element types */
  expectedTypes: string[];
}

const TEST_SITES: TestSite[] = [
  {
    name: 'GitHub Login',
    url: 'https://github.com/login',
    elementSelectors: [
      'input[name="login"]',
      'input[name="password"]',
      'input[type="submit"]',
      'a[href="/password_reset"]',
    ],
    expectedTypes: ['input', 'input', 'button', 'link'],
  },
  {
    name: 'Google Search',
    url: 'https://www.google.com',
    elementSelectors: [
      'textarea[name="q"]',
    ],
    expectedTypes: ['textarea'],
  },
  {
    name: 'Wikipedia',
    url: 'https://en.wikipedia.org/wiki/Main_Page',
    elementSelectors: [
      '#searchInput',
      'button[type="submit"]',
      '#p-personal a',
    ],
    expectedTypes: ['input', 'button', 'link'],
  },
  {
    name: 'Hacker News',
    url: 'https://news.ycombinator.com/',
    elementSelectors: [
      '.hnname a',
      'span.pagetop a[href="newest"]',
      'a.storylink, .titleline > a',
    ],
    expectedTypes: ['link', 'link', 'link'],
  },
  {
    name: 'DuckDuckGo',
    url: 'https://duckduckgo.com',
    elementSelectors: [
      'input[name="q"]',
      'button[type="submit"]',
    ],
    expectedTypes: ['input', 'button'],
  },
  {
    name: 'W3Schools',
    url: 'https://www.w3schools.com/',
    elementSelectors: [
      '#search2',
      'a[title="Home"]',
      'a[title="Tutorials"]',
    ],
    expectedTypes: ['input', 'link', 'link'],
  },
  {
    name: 'Example.com',
    url: 'https://example.com/',
    elementSelectors: [
      'h1',
      'p',
      'a',
    ],
    expectedTypes: ['heading', 'paragraph', 'link'],
  },
  {
    name: 'HTTPBin',
    url: 'https://httpbin.org/',
    elementSelectors: [
      'h2',
      'a.btn',
      'img.logo',
    ],
    expectedTypes: ['heading', 'button', 'image'],
  },
  {
    name: 'JSON Placeholder',
    url: 'https://jsonplaceholder.typicode.com/',
    elementSelectors: [
      'h1',
      'a[href="/guide/"]',
      'code',
    ],
    expectedTypes: ['heading', 'link', 'code'],
  },
  {
    name: 'Playwright Docs',
    url: 'https://playwright.dev/',
    elementSelectors: [
      'a.navbar__brand',
      'button.navbar__toggle',
    ],
    expectedTypes: ['link', 'button'],
  },
];

// ============================================================================
// Selector Generation (inline for spike - matches injector.ts)
// ============================================================================

/**
 * This is the selector generation code that runs in browser context.
 * It's inlined here to avoid import complexity during spike testing.
 */
const SELECTOR_GENERATION_CODE = `
(function(targetSelector) {
  const element = document.querySelector(targetSelector);
  if (!element) return null;

  const candidates = [];

  // Helper functions
  function isUniqueSelector(selector) {
    try { return document.querySelectorAll(selector).length === 1; }
    catch { return false; }
  }

  function isUniqueXPath(xpath) {
    try {
      const result = document.evaluate(xpath, document, null, XPathResult.ORDERED_NODE_SNAPSHOT_TYPE, null);
      return result.snapshotLength === 1;
    } catch { return false; }
  }

  function isDynamicId(id) {
    const patterns = [/^[a-f0-9]{8,}$/i, /^\\d+$/, /:r[0-9]+:/, /^:r/, /^ember\\d+$/, /_\\d{10,}/];
    return patterns.some(p => p.test(id));
  }

  function escapeCssAttr(str) {
    return str.replace(/"/g, '\\\\"').replace(/\\n/g, '\\\\n');
  }

  function escapeCssSelector(str) {
    return str.replace(/([!"#$%&'()*+,./:;<=>?@[\\\\\\]^\\x60{|}~])/g, '\\\\$1');
  }

  function getStableClasses(el) {
    const unstable = [/^css-[a-z0-9]+$/i, /^sc-[a-zA-Z]+$/, /^_[a-zA-Z0-9]+$/, /^[a-zA-Z]+-[0-9]+$/];
    return Array.from(el.classList).filter(cls => {
      if (!cls.trim()) return false;
      return !unstable.some(p => p.test(cls));
    });
  }

  function getVisibleText(el) {
    if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
      return (el.value || el.placeholder || '').slice(0, 100);
    }
    return (el.textContent || '').trim().slice(0, 100);
  }

  function inferRole(el) {
    // Return CSS tag name (not role name) for use with :has-text()
    const tag = el.tagName.toLowerCase();
    // Only return for elements that commonly have meaningful text
    const textTags = ['button', 'a', 'span', 'div', 'p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'label', 'li', 'td', 'th'];
    return textTags.includes(tag) ? tag : null;
  }

  // Strategy 1: data-testid
  const testIdAttrs = ['data-testid', 'data-test-id', 'data-test', 'data-cy', 'data-qa'];
  for (const attr of testIdAttrs) {
    const value = element.getAttribute(attr);
    if (value) {
      const selector = '[' + attr + '="' + escapeCssAttr(value) + '"]';
      if (isUniqueSelector(selector)) {
        candidates.push({ type: 'data-testid', value: selector, confidence: 0.98, specificity: 100 });
      }
    }
  }

  // Strategy 2: Unique ID
  const id = element.id;
  if (id) {
    const isDynamic = isDynamicId(id);
    const selector = '#' + escapeCssSelector(id);
    if (isUniqueSelector(selector)) {
      candidates.push({ type: 'id', value: selector, confidence: isDynamic ? 0.6 : 0.95, specificity: 95 });
    }
  }

  // Strategy 3: ARIA
  const ariaLabel = element.getAttribute('aria-label');
  if (ariaLabel) {
    const selector = '[aria-label="' + escapeCssAttr(ariaLabel) + '"]';
    if (isUniqueSelector(selector)) {
      candidates.push({ type: 'aria', value: selector, confidence: 0.85, specificity: 80 });
    }
  }

  // Strategy 4: Tag + text (using Playwright's :has-text())
  const tag = inferRole(element);
  const text = getVisibleText(element);
  if (tag && text && text.length > 2 && text.length <= 50) {
    // Truncate text to first 30 chars for selector stability
    const selectorText = text.length > 30 ? text.slice(0, 30) : text;
    const selector = tag + ':has-text("' + selectorText.replace(/"/g, '\\\\"') + '")';
    // Note: Can't easily validate Playwright selectors with querySelectorAll
    // So we only add this as a fallback with lower confidence
    candidates.push({
      type: 'text',
      value: selector,
      confidence: text.length > 15 ? 0.6 : 0.55,
      specificity: 50,
    });
  }

  // Strategy 5: Data attributes
  for (const attr of element.attributes) {
    if (!attr.name.startsWith('data-')) continue;
    const skip = ['testid', 'test-id', 'test', 'cy', 'qa'];
    if (skip.includes(attr.name.slice(5))) continue;

    const selector = '[' + attr.name + '="' + escapeCssAttr(attr.value) + '"]';
    if (isUniqueSelector(selector)) {
      candidates.push({ type: 'data-attr', value: selector, confidence: 0.7, specificity: 65 });
      break;
    }
  }

  // Strategy 6: CSS path
  const parts = [];
  let current = element;
  let depth = 0;
  while (current && current !== document.body && depth < 5) {
    const tag = current.tagName.toLowerCase();
    const stableClasses = getStableClasses(current);
    let part = stableClasses.length > 0 ? tag + '.' + stableClasses.slice(0, 2).join('.') : tag;
    parts.unshift(part);

    const selector = parts.join(' > ');
    if (isUniqueSelector(selector)) {
      candidates.push({ type: 'css', value: selector, confidence: 0.65, specificity: 60 - depth * 5 });
      break;
    }
    current = current.parentElement;
    depth++;
  }

  // Strategy 7: CSS with nth-child
  if (!candidates.find(c => c.type === 'css')) {
    const parts2 = [];
    let curr = element;
    let d = 0;
    while (curr && curr !== document.body && d < 5) {
      const tag = curr.tagName.toLowerCase();
      const parent = curr.parentElement;
      let part = tag;
      if (parent) {
        const siblings = Array.from(parent.children).filter(s => s.tagName.toLowerCase() === tag);
        if (siblings.length > 1) {
          part = tag + ':nth-of-type(' + (siblings.indexOf(curr) + 1) + ')';
        }
      }
      parts2.unshift(part);
      const sel = parts2.join(' > ');
      if (isUniqueSelector(sel)) {
        candidates.push({ type: 'css', value: sel, confidence: 0.5, specificity: 40 - d * 5 });
        break;
      }
      curr = parent;
      d++;
    }
  }

  // Strategy 8: XPath
  if (text && text.length > 0 && text.length <= 50) {
    const xpath = '//' + element.tagName.toLowerCase() + '[contains(text(), "' + text.replace(/"/g, '\\"') + '")]';
    if (isUniqueXPath(xpath)) {
      candidates.push({ type: 'xpath', value: xpath, confidence: 0.55, specificity: 35 });
    }
  }

  // Positional XPath fallback
  const posXPath = (function() {
    const xparts = [];
    let xcurr = element;
    while (xcurr && xcurr !== document.documentElement) {
      const xtag = xcurr.tagName.toLowerCase();
      const xparent = xcurr.parentElement;
      if (xparent) {
        const xsibs = Array.from(xparent.children).filter(s => s.tagName.toLowerCase() === xtag);
        if (xsibs.length > 1) {
          xparts.unshift(xtag + '[' + (xsibs.indexOf(xcurr) + 1) + ']');
        } else {
          xparts.unshift(xtag);
        }
      } else {
        xparts.unshift(xtag);
      }
      xcurr = xparent;
    }
    return '/' + xparts.join('/');
  })();
  candidates.push({ type: 'xpath', value: posXPath, confidence: 0.4, specificity: 25 });

  candidates.sort((a, b) => b.confidence - a.confidence);

  return {
    primary: candidates[0]?.value || posXPath,
    candidates: candidates,
  };
})
`;

// ============================================================================
// Type Definitions
// ============================================================================

interface SelectorCandidate {
  type: string;
  value: string;
  confidence: number;
  specificity: number;
}

interface SelectorSet {
  primary: string;
  candidates: SelectorCandidate[];
}

interface SelectorResult {
  originalSelector: string;
  generatedSelector: string;
  generatedType: string;
  confidence: number;
  validBeforeRefresh: boolean;
  validAfterRefresh: boolean;
  matchCountBefore: number;
  matchCountAfter: number;
  error?: string;
}

interface SiteResult {
  siteName: string;
  url: string;
  results: SelectorResult[];
  successRate: number;
  totalElements: number;
  successfulElements: number;
  error?: string;
}

// ============================================================================
// Test Execution
// ============================================================================

async function testSite(browser: Browser, site: TestSite): Promise<SiteResult> {
  const page = await browser.newPage();
  const results: SelectorResult[] = [];

  try {
    console.log(`\n${'='.repeat(60)}`);
    console.log(`Testing: ${site.name}`);
    console.log(`URL: ${site.url}`);
    console.log('='.repeat(60));

    // Navigate to site
    await page.goto(site.url, { waitUntil: 'domcontentloaded', timeout: 30000 });
    await page.waitForTimeout(2000); // Wait for dynamic content

    // Test each element
    for (let i = 0; i < site.elementSelectors.length; i++) {
      const originalSelector = site.elementSelectors[i];
      const result: SelectorResult = {
        originalSelector,
        generatedSelector: '',
        generatedType: '',
        confidence: 0,
        validBeforeRefresh: false,
        validAfterRefresh: false,
        matchCountBefore: 0,
        matchCountAfter: 0,
      };

      try {
        // Check if element exists
        const elementHandle: ElementHandle | null = await page.$(originalSelector);
        if (!elementHandle) {
          result.error = 'Element not found';
          results.push(result);
          console.log(`  [${i + 1}] ${originalSelector}: NOT FOUND`);
          continue;
        }

        // Generate selectors in browser context by passing the selector string
        const selectorSet = await page.evaluate(
          `(${SELECTOR_GENERATION_CODE})("${originalSelector.replace(/"/g, '\\"')}")`
        ) as SelectorSet | null;

        if (!selectorSet) {
          result.error = 'Failed to generate selectors';
          results.push(result);
          console.log(`  [${i + 1}] ${originalSelector}: GENERATION FAILED`);
          continue;
        }

        result.generatedSelector = selectorSet.primary;
        result.generatedType = selectorSet.candidates[0]?.type || 'unknown';
        result.confidence = selectorSet.candidates[0]?.confidence || 0;

        // Validate before refresh
        const matchesBefore = await validateSelector(page, result.generatedSelector);
        result.matchCountBefore = matchesBefore;
        result.validBeforeRefresh = matchesBefore === 1;

        // Refresh page
        await page.reload({ waitUntil: 'domcontentloaded', timeout: 30000 });
        await page.waitForTimeout(2000);

        // Validate after refresh
        const matchesAfter = await validateSelector(page, result.generatedSelector);
        result.matchCountAfter = matchesAfter;
        result.validAfterRefresh = matchesAfter === 1;

        // Log result
        const status = result.validAfterRefresh ? '✅' : '❌';
        console.log(
          `  [${i + 1}] ${status} ${result.generatedType} (${(result.confidence * 100).toFixed(0)}%): ${result.generatedSelector.slice(0, 50)}${result.generatedSelector.length > 50 ? '...' : ''}`
        );

        if (!result.validAfterRefresh) {
          console.log(`       Before: ${result.matchCountBefore} matches, After: ${result.matchCountAfter} matches`);
        }
      } catch (err) {
        result.error = err instanceof Error ? err.message : String(err);
        console.log(`  [${i + 1}] ⚠️  Error: ${result.error}`);
      }

      results.push(result);
    }
  } catch (err) {
    console.error(`  Site error: ${err instanceof Error ? err.message : err}`);
    return {
      siteName: site.name,
      url: site.url,
      results,
      successRate: 0,
      totalElements: site.elementSelectors.length,
      successfulElements: 0,
      error: err instanceof Error ? err.message : String(err),
    };
  } finally {
    await page.close();
  }

  const successfulElements = results.filter((r) => r.validAfterRefresh).length;
  const successRate = results.length > 0 ? successfulElements / results.length : 0;

  console.log(`\n  Summary: ${successfulElements}/${results.length} selectors survived refresh (${(successRate * 100).toFixed(1)}%)`);

  return {
    siteName: site.name,
    url: site.url,
    results,
    successRate,
    totalElements: results.length,
    successfulElements,
  };
}

/**
 * Validate a selector and return match count.
 */
async function validateSelector(page: Awaited<ReturnType<Browser['newPage']>>, selector: string): Promise<number> {
  // Handle Playwright-specific selectors vs CSS/XPath
  if (selector.includes(':has-text(')) {
    // Playwright selector - use locator
    try {
      return await page.locator(selector).count();
    } catch {
      return 0;
    }
  } else if (selector.startsWith('/')) {
    // XPath selector - evaluate in browser context
    try {
      const count = await page.evaluate(`
        (function() {
          try {
            const result = document.evaluate(
              "${selector.replace(/"/g, '\\"')}",
              document,
              null,
              XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
              null
            );
            return result.snapshotLength;
          } catch (e) {
            return 0;
          }
        })()
      `);
      return count as number;
    } catch {
      return 0;
    }
  } else {
    // CSS selector
    try {
      const elements = await page.$$(selector);
      return elements.length;
    } catch {
      return 0;
    }
  }
}

async function runSpike() {
  console.log('🚀 Selector Spike Test');
  console.log('='.repeat(60));
  console.log('Goal: >70% of selectors should survive a page refresh');
  console.log('='.repeat(60));

  // Parse command line args
  const args = process.argv.slice(2);
  const siteArg = args.find((a) => a.startsWith('--site='));
  const siteName = siteArg?.split('=')[1];

  // Filter sites if specified
  let sites = TEST_SITES;
  if (siteName) {
    sites = TEST_SITES.filter(
      (s) => s.name.toLowerCase().includes(siteName.toLowerCase())
    );
    if (sites.length === 0) {
      console.error(`No sites matching: ${siteName}`);
      console.log('Available sites:', TEST_SITES.map((s) => s.name).join(', '));
      process.exit(1);
    }
  }

  const browser = await chromium.launch({
    headless: true,
  });

  const allResults: SiteResult[] = [];

  try {
    for (const site of sites) {
      const result = await testSite(browser, site);
      allResults.push(result);
    }
  } finally {
    await browser.close();
  }

  // Print summary
  console.log('\n' + '='.repeat(60));
  console.log('FINAL SUMMARY');
  console.log('='.repeat(60));

  let totalElements = 0;
  let totalSuccessful = 0;

  for (const result of allResults) {
    const status = result.successRate >= 0.7 ? '✅' : '❌';
    console.log(
      `${status} ${result.siteName}: ${result.successfulElements}/${result.totalElements} (${(result.successRate * 100).toFixed(1)}%)`
    );
    totalElements += result.totalElements;
    totalSuccessful += result.successfulElements;
  }

  const overallRate = totalElements > 0 ? totalSuccessful / totalElements : 0;
  console.log('─'.repeat(60));
  console.log(
    `OVERALL: ${totalSuccessful}/${totalElements} (${(overallRate * 100).toFixed(1)}%)`
  );

  const passed = overallRate >= 0.7;
  console.log('─'.repeat(60));
  console.log(passed ? '✅ SPIKE PASSED: >70% reliability achieved!' : '❌ SPIKE FAILED: <70% reliability');
  console.log('='.repeat(60));

  // Detailed breakdown by selector type
  console.log('\nBREAKDOWN BY SELECTOR TYPE:');
  const byType: Record<string, { total: number; successful: number }> = {};

  for (const site of allResults) {
    for (const result of site.results) {
      if (!result.generatedType) continue;
      if (!byType[result.generatedType]) {
        byType[result.generatedType] = { total: 0, successful: 0 };
      }
      byType[result.generatedType].total++;
      if (result.validAfterRefresh) {
        byType[result.generatedType].successful++;
      }
    }
  }

  for (const [type, stats] of Object.entries(byType)) {
    const rate = stats.total > 0 ? stats.successful / stats.total : 0;
    console.log(`  ${type}: ${stats.successful}/${stats.total} (${(rate * 100).toFixed(1)}%)`);
  }

  process.exit(passed ? 0 : 1);
}

// Run the spike
runSpike().catch((err) => {
  console.error('Spike failed:', err);
  process.exit(1);
});
