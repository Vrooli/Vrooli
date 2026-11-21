import { afterAll, beforeAll } from '@jest/globals';
import { chromium, type Browser } from 'rebrowser-playwright';

type Options = Parameters<typeof chromium.launch>[0]; type Launcher = (options: Options) => Promise<Browser>; type Hook = (callback: () => void | Promise<void>) => void; type Hooks = { beforeAll: Hook; afterAll: Hook };

export function ownBrowser(options: Options, tolerateFailure = false, hooks: Hooks = { beforeAll, afterAll }, launch: Launcher = (value) => chromium.launch(value)): { browser?: Browser } {
  let browser: Browser | undefined;
  hooks.beforeAll(async () => { try { browser = await launch(options); } catch (error) { if (!tolerateFailure) throw error; } });
  hooks.afterAll(async () => { await browser?.close(); });
  return { get browser() { return browser; } };
}

export async function withBrowser<T>(options: Options, run: (browser: Browser) => Promise<T>, launch: Launcher = (value) => chromium.launch(value)): Promise<T> {
  const browser = await launch(options);
  try { return await run(browser); } finally { await browser.close(); }
}

export function itWithBrowser<T>(name: string, options: Options, run: (browser: Browser) => Promise<T>, timeout?: number): void {
  it(name, () => withBrowser(options, run), timeout);
}
