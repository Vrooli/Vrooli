import { type Page } from 'rebrowser-playwright';
import { ownBrowser } from './support/browser-fixture';
jest.setTimeout(30000);

const DELAYED_FIXTURE = `data:text/html,${encodeURIComponent(`
<!doctype html>
<html>
  <body>
    <main id="snapshot-root"><h1>Ready shell</h1></main>
    <script>
      setTimeout(() => {
        const late = document.createElement('p');
        late.id = 'late-content';
        late.textContent = 'Late content';
        document.getElementById('snapshot-root').appendChild(late);
      }, 1500);
    </script>
  </body>
</html>`)};`;
describe('DOM snapshot readiness', () => {
  const browserOwner = ownBrowser({ headless: true }, true);
  async function newPage(): Promise<Page> {
    if (!browserOwner.browser) {
      throw new Error('Chromium unavailable');
    }
    return browserOwner.browser.newPage();
  }

  it('misses late content without wait_for and captures it with wait_for', async () => {
    if (!browserOwner.browser) return;

    const earlyPage = await newPage();
    try {
      await earlyPage.goto(DELAYED_FIXTURE, { waitUntil: 'load' });
      expect(await earlyPage.locator('#late-content').count()).toBe(0);
    } finally {
      await earlyPage.close();
    }

    const settledPage = await newPage();
    try {
      await settledPage.goto(DELAYED_FIXTURE, { waitUntil: 'load' });
      await settledPage.waitForSelector('#late-content', { timeout: 5000 });
      expect(await settledPage.locator('#late-content').innerText()).toBe('Late content');
    } finally {
      await settledPage.close();
    }
  });
});
