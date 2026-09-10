/** Independent task postconditions run on the real final page without repeating task effects. */
import { chromium } from 'rebrowser-playwright';
import { observeTaskResult } from '../../src/ai/vision-agent';

describe('navigation task contract on a real browser', () => {
  it('extracts verified results after one mutation and detects changed results without replay', async () => {
    const browser = await chromium.launch({headless: true});
    try {
      const page = await browser.newPage();
      page.setDefaultTimeout(5000);
      await page.goto('data:text/html,' + encodeURIComponent('<button id="submit" onclick="document.querySelector(\'#count\').textContent=String(Number(document.querySelector(\'#count\').textContent)+1)">Submit once</button><span id="count">0</span><p class="subject">Invoice approved</p>'), {waitUntil: 'load', timeout: 5000});
      await page.locator('#submit').click();
      const contract = {page, postconditions: [{selector: '#count', mode: 'text_equals' as const, expected: '1'}], extraction: [{name: 'subjects', selector: '.subject', limit: 5}]};
      const observed = await observeTaskResult(contract);
      expect(observed).toEqual({verifiedSuccess: true, extractedData: {subjects: ['Invoice approved']}});
      await observeTaskResult(contract);
      expect(await page.locator('#count').textContent()).toBe('1');
      await page.locator('#count').evaluate(element => { element.textContent = '2'; });
      await expect(observeTaskResult(contract)).rejects.toThrow('postcondition_failed');
      expect(await page.locator('#count').textContent()).toBe('2');
    } finally {
      await browser.close();
    }
  }, 30000);
});
