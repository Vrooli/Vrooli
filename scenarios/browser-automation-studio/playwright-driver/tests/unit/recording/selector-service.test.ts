import type { Page } from 'rebrowser-playwright';
import { createTestConfig } from '../../helpers';
import {
  SelectorService,
  isXPathSelector,
  validateSelectorOnPage,
} from '../../../src/recording/validation/selector-service';
import { CONFIDENCE_SCORES, SELECTOR_DEFAULTS, SPECIFICITY_SCORES } from '../../../src/recording/validation/selector-config';

describe('SelectorService', () => {
  it('validates CSS selectors with waitForSelector enabled', async () => {
    const waitFor = jest.fn().mockResolvedValue(undefined);
    const locator = {
      count: jest.fn().mockResolvedValue(1),
      first: jest.fn().mockReturnValue({ waitFor }),
    };

    const page = {
      locator: jest.fn().mockReturnValue(locator),
      evaluate: jest.fn(),
    } as unknown as Page;

    const config = createTestConfig({ execution: { defaultTimeoutMs: 1234 } });
    const service = new SelectorService(page, config);

    const result = await service.validate('.btn-primary', { waitForSelector: true, timeout: 1500 });

    expect(result.valid).toBe(true);
    expect(result.matchCount).toBe(1);
    expect(waitFor).toHaveBeenCalledWith({ timeout: 1500, state: 'attached' });
  });

  it('returns invalid when CSS selector matches zero elements', async () => {
    const locator = {
      count: jest.fn().mockResolvedValue(0),
      first: jest.fn().mockReturnValue({ waitFor: jest.fn() }),
    };

    const page = {
      locator: jest.fn().mockReturnValue(locator),
      evaluate: jest.fn(),
    } as unknown as Page;

    const service = new SelectorService(page, createTestConfig());
    const result = await service.validate('.missing');

    expect(result.valid).toBe(false);
    expect(result.matchCount).toBe(0);
  });

  it('validates XPath selectors and reports invalid syntax', async () => {
    const page = {
      locator: jest.fn(),
      evaluate: jest.fn().mockResolvedValueOnce(-1).mockResolvedValueOnce(2),
    } as unknown as Page;

    const service = new SelectorService(page, createTestConfig());

    const invalid = await service.validate('//bad[');
    expect(invalid.valid).toBe(false);
    expect(invalid.error).toBe('Invalid XPath expression');

    const multiple = await service.validate('//div');
    expect(multiple.valid).toBe(false);
    expect(multiple.matchCount).toBe(2);
  });

  it('checks syntax validity for XPath and CSS selectors', async () => {
    const page = {
      locator: jest.fn().mockImplementation(() => {
        throw new Error('Invalid selector');
      }),
      evaluate: jest.fn().mockResolvedValue(true),
    } as unknown as Page;

    const service = new SelectorService(page, createTestConfig());

    const xpathValid = await service.isValidSyntax('//div');
    const cssValid = await service.isValidSyntax('div[');

    expect(xpathValid).toBe(true);
    expect(cssValid).toBe(false);
  });

  it('calculates confidence using candidate metadata and pattern analysis', () => {
    const page = { locator: jest.fn(), evaluate: jest.fn() } as unknown as Page;
    const service = new SelectorService(page, createTestConfig());

    const boosted = service.calculateConfidence('[data-testid="login"]', [
      { type: 'data-testid', value: '[data-testid="login"]', confidence: 0.5 },
    ]);

    expect(boosted).toBeGreaterThanOrEqual(0.85);

    const fromPattern = service.calculateConfidence('#abcdef12');
    expect(fromPattern).toBe(CONFIDENCE_SCORES.idDynamic);
  });

  it('calculates selector specificity', () => {
    const page = { locator: jest.fn(), evaluate: jest.fn() } as unknown as Page;
    const service = new SelectorService(page, createTestConfig());

    expect(service.calculateSpecificity('[data-testid="login"]')).toBe(SPECIFICITY_SCORES.dataTestId);
    expect(service.calculateSpecificity('.nav-item', 'ariaLabel')).toBe(SPECIFICITY_SCORES.ariaLabel);
  });

  it('detects XPath selectors via utilities', () => {
    const page = { locator: jest.fn(), evaluate: jest.fn() } as unknown as Page;
    const service = new SelectorService(page, createTestConfig());

    expect(service.isXPath('//div')).toBe(true);
    expect(isXPathSelector('(//div)[1]')).toBe(true);
    expect(isXPathSelector('.btn')).toBe(false);
  });

  it('evaluates selector stability helpers and config getters', () => {
    const page = { locator: jest.fn(), evaluate: jest.fn() } as unknown as Page;
    const config = createTestConfig({
      recording: {
        minSelectorConfidence: 0.42,
        selector: {
          maxCssDepth: 7,
          includeXPath: false,
        },
      },
    });
    const service = new SelectorService(page, config);

    expect(service.isDynamicId('123456')).toBe(true);
    expect(service.isUnstableClass('css-abc123')).toBe(true);
    expect(service.isSemanticClass('btn-primary')).toBe(true);
    expect(service.getMinConfidence()).toBe(0.42);
    expect(service.getMaxCssDepth()).toBe(7);
    expect(service.isXPathEnabled()).toBe(false);
    expect(SELECTOR_DEFAULTS.minConfidence).toBeGreaterThan(0);
  });

  it('validates selectors using the standalone helper', async () => {
    const page = {
      locator: jest.fn().mockReturnValue({ count: jest.fn().mockResolvedValue(1) }),
      evaluate: jest.fn().mockResolvedValue(1),
    } as unknown as Page;

    const css = await validateSelectorOnPage(page, '.btn');
    const xpath = await validateSelectorOnPage(page, '//div');

    expect(css.valid).toBe(true);
    expect(xpath.valid).toBe(true);
  });
});
