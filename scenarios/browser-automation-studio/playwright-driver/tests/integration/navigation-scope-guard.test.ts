import { createServer, type Server } from 'http';
import { chromium, type Browser, type BrowserContext, type Page } from 'rebrowser-playwright';
import {
  ensureNavigationScopeForPage,
  installNavigationScopeGuard,
} from '../../src/session/navigation-scope';
import type { BrowserPermissionProfile } from '../../src/types/session';

describe('session URL scope on browser initiated document navigations', () => {
  let browser: Browser;
  let server: Server;
  let origin: string;
  let disallowedHostRequests = 0;
  let redirectRequests = 0;
  let allowedRedirectRequests = 0;
  let insideRequests = 0;
  const profile: BrowserPermissionProfile = {
    effect: 'act',
    approval: 'none',
    control: true,
    view: true,
    url_allow: ['http://localhost/*'],
    url_deny: [],
  };

  beforeAll(async () => {
    browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
    server = createServer((request, response) => {
      if (String(request.headers.host).startsWith('127.0.0.1:')) disallowedHostRequests++;
      if (request.url === '/redirect') {
        redirectRequests++;
        response.writeHead(302, { Location: `http://127.0.0.1:${port}/destination` });
        response.end();
        return;
      }
      if (request.url === '/redirect-allowed') {
        allowedRedirectRequests++;
        response.writeHead(302, { Location: `http://localhost:${port}/inside` });
        response.end();
        return;
      }
      if (request.url === '/inside') insideRequests++;
      if (request.url === '/redirect-chain') {
        response.writeHead(302, { Location: `http://localhost:${port}/redirect` });
        response.end();
        return;
      }
      response.writeHead(200, { 'Content-Type': 'text/html' });
      response.end(
        `<!doctype html><a id="escape" href="${fixtureDisallowedURL()}">escape</a><button id="script" onclick="location.href='${fixtureDisallowedURL()}'">script</button>`
      );
    });
    await new Promise<void>((resolve) => server.listen(0, '0.0.0.0', resolve));
    const address = server.address();
    if (!address || typeof address === 'string') throw new Error('fixture server did not bind TCP');
    port = address.port;
    origin = `http://localhost:${port}`;
  });

  beforeEach(() => {
    deniedRequests.clear();
    disallowedHostRequests = 0;
    redirectRequests = 0;
    allowedRedirectRequests = 0;
    insideRequests = 0;
  });

  afterAll(async () => {
    await browser?.close();
    if (server?.listening)
      await new Promise<void>((resolve, reject) =>
        server.close((error) => (error ? reject(error) : resolve()))
      );
  });

  let port = 0;
  const fixtureDisallowedURL = (): string => `http://127.0.0.1:${port}/destination`;
  const newPage = async (): Promise<{ context: BrowserContext; page: Page }> => {
    const context = await browser.newContext();
    await installNavigationScopeGuard(context, profile);
    context.on('requestfailed', (request) => {
      if (request.url() === fixtureDisallowedURL()) deniedRequests.add(request.url());
    });
    const page = await context.newPage();
    await ensureNavigationScopeForPage(page);
    await page.goto(origin);
    return { context, page };
  };
  const expectDeniedBeforeRequest = async (
    context: BrowserContext,
    page: Page,
    expectFailedRequest = true
  ): Promise<void> => {
    await page.waitForTimeout(30);
    if (expectFailedRequest) expect(deniedRequests.has(fixtureDisallowedURL())).toBe(true);
    expect(disallowedHostRequests).toBe(0);
    await context.close();
  };
  const deniedRequests = new Set<string>();

  it('blocks a link click before the destination request', async () => {
    const { context, page } = await newPage();
    await page
      .locator('#escape')
      .click()
      .catch(() => undefined);
    await expectDeniedBeforeRequest(context, page);
  });

  it('blocks script-driven location navigation before the destination request', async () => {
    const { context, page } = await newPage();
    await page
      .evaluate((target) => {
        location.href = target;
      }, fixtureDisallowedURL())
      .catch(() => undefined);
    await expectDeniedBeforeRequest(context, page);
  });

  it('checks a redirect target before the redirected request', async () => {
    const { context, page } = await newPage();
    await page.goto(`${origin}/redirect`).catch(() => undefined);
    expect(redirectRequests).toBe(1);
    expect(page.url()).toBe(`${origin}/`);
    await expectDeniedBeforeRequest(context, page, false);
  });

  it('preserves an in-scope redirect', async () => {
    const { context, page } = await newPage();
    await page.evaluate((target) => {
      const link = document.createElement('a');
      link.id = 'allowed-redirect';
      link.href = target;
      link.textContent = 'allowed redirect';
      document.body.appendChild(link);
    }, `${origin}/redirect-allowed`);
    await page.locator('#allowed-redirect').click();
    expect(page.url()).toBe(`${origin}/inside`);
    expect(allowedRedirectRequests).toBe(1);
    expect(insideRequests).toBe(1);
    expect(disallowedHostRequests).toBe(0);
    await context.close();
  });

  it('checks every hop in a redirect chain before the first disallowed destination request', async () => {
    const { context, page } = await newPage();
    await page.goto(`${origin}/redirect-chain`).catch(() => undefined);
    expect(page.url()).toBe(`${origin}/`);
    expect(redirectRequests).toBe(1);
    await expectDeniedBeforeRequest(context, page, false);
  });

  it('blocks popup document navigation before the destination request', async () => {
    const { context, page } = await newPage();
    const popupPromise = page.waitForEvent('popup');
    await page.evaluate((target) => window.open(target), fixtureDisallowedURL());
    const popup = await popupPromise;
    await expectDeniedBeforeRequest(context, popup);
  });

  it('applies redirect checks to a newly opened popup page', async () => {
    const { context, page } = await newPage();
    const popupPromise = page.waitForEvent('popup');
    await page.evaluate((target) => window.open(target), `${origin}/redirect-chain`);
    const popup = await popupPromise;
    await popup.waitForTimeout(80);
    expect(redirectRequests).toBe(1);
    expect(disallowedHostRequests).toBe(0);
    expect(popup.url()).not.toContain('127.0.0.1');
    await context.close();
  });

  it('blocks iframe document navigation before the destination request', async () => {
    const { context, page } = await newPage();
    await page.evaluate((target) => {
      const frame = document.createElement('iframe');
      frame.src = target;
      document.body.appendChild(frame);
    }, fixtureDisallowedURL());
    await expectDeniedBeforeRequest(context, page);
  });

  it('permits an in-scope link and its local response', async () => {
    const { context, page } = await newPage();
    await page.evaluate((target) => {
      const link = document.createElement('a');
      link.href = target;
      link.textContent = 'in scope';
      document.body.appendChild(link);
    }, `${origin}/inside`);
    await page.getByText('in scope').click();
    expect(page.url()).toBe(`${origin}/inside`);
    expect(disallowedHostRequests).toBe(0);
    await context.close();
  });
});
