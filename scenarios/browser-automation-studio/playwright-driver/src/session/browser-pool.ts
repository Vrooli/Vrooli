import type { Browser } from 'rebrowser-playwright';

/**
 * Owns browser instances and per-key launch locks. BrowserManager keeps policy
 * (validation, arguments, health); this class owns concurrent pooling only.
 */
export class BrowserPool {
  private readonly browsers = new Map<string, Browser>();
  private readonly launches = new Map<string, Promise<Browser>>();
  private closing = false;
  private generation = 0;
  private closePromise?: Promise<void>;

  get(key: string): Browser | undefined {
    if (this.closing) return undefined;
    const browser = this.browsers.get(key);
    return browser?.isConnected() ? browser : undefined;
  }

  async getOrLaunch(key: string, launch: () => Promise<Browser>): Promise<Browser> {
    if (this.closing) throw new Error('Browser pool is shutting down');
    const generation = this.generation;
    const existing = this.get(key);
    if (existing) return existing;

    const inFlight = this.launches.get(key);
    if (inFlight) {
      const browser = await inFlight;
      if (generation !== this.generation) throw new Error('Browser pool shut down during launch');
      if (browser.isConnected()) return browser;
      return this.getOrLaunch(key, launch);
    }

    const promise = this.launchWithSingleRetry(launch, generation);
    this.launches.set(key, promise);
    try {
      const browser = await promise;
      if (generation !== this.generation) throw new Error('Browser pool shut down during launch');
      if (browser.isConnected()) this.browsers.set(key, browser);
      return browser;
    } finally {
      if (this.launches.get(key) === promise) this.launches.delete(key);
    }
  }

  async closeAll(close: (key: string, browser: Browser) => Promise<void>): Promise<void> {
    if (this.closePromise) return this.closePromise;

    this.closing = true;
    this.generation += 1;
    const launches = [...this.launches.entries()];
    const closing = this.closeOwnedBrowsers(launches, close);
    this.closePromise = closing;
    try {
      await closing;
    } finally {
      if (this.closePromise === closing) this.closePromise = undefined;
      this.closing = false;
    }
  }

  private async closeOwnedBrowsers(
    launches: Array<[string, Promise<Browser>]>,
    close: (key: string, browser: Browser) => Promise<void>
  ): Promise<void> {
    const launchResults = await Promise.allSettled(launches.map(([, launch]) => launch));
    const owned = new Map<Browser, string>();

    for (const [key, browser] of this.browsers) owned.set(browser, key);
    launchResults.forEach((result, index) => {
      const launch = launches[index];
      if (result.status === 'fulfilled' && launch) owned.set(result.value, launch[0]);
    });

    this.browsers.clear();
    this.launches.clear();

    const failures: unknown[] = [];
    for (const [browser, key] of owned) {
      try {
        await close(key, browser);
      } catch (error) {
        failures.push(error);
      }
    }
    if (failures.length > 0) {
      throw new Error(
        `Failed to close ${failures.length} pooled browser(s): ${String(failures[0])}`
      );
    }
  }

  private async launchWithSingleRetry(
    launch: () => Promise<Browser>,
    generation: number
  ): Promise<Browser> {
    try {
      return await launch();
    } catch (error) {
      if (this.closing || generation !== this.generation) throw error;
      return launch();
    }
  }
}
