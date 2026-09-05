import type { Browser } from 'rebrowser-playwright';

/**
 * Owns browser instances and per-key launch locks. BrowserManager keeps policy
 * (validation, arguments, health); this class owns concurrent pooling only.
 */
export class BrowserPool {
  private readonly browsers = new Map<string, Browser>();
  private readonly launches = new Map<string, Promise<Browser>>();

  get(key: string): Browser | undefined {
    const browser = this.browsers.get(key);
    return browser?.isConnected() ? browser : undefined;
  }

  async getOrLaunch(key: string, launch: () => Promise<Browser>): Promise<Browser> {
    const existing = this.get(key);
    if (existing) return existing;

    const inFlight = this.launches.get(key);
    if (inFlight) {
      try {
        const browser = await inFlight;
        if (browser.isConnected()) return browser;
      } catch {
        // The original caller reports its launch failure; a later caller can retry.
      }
    }

    const promise = launch();
    this.launches.set(key, promise);
    try {
      const browser = await promise;
      this.browsers.set(key, browser);
      return browser;
    } finally {
      this.launches.delete(key);
    }
  }

  async closeAll(close: (key: string, browser: Browser) => Promise<void>): Promise<void> {
    for (const [key, browser] of this.browsers) await close(key, browser);
    this.browsers.clear();
  }
}
