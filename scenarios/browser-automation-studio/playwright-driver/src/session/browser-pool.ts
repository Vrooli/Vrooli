import type { Browser } from 'rebrowser-playwright';

export interface BrowserPoolUsage {
  browserCount: number;
  contextCount: number;
  idleBrowserCount: number;
  maxIdleSeconds: number;
}

/**
 * Owns browser instances and per-key launch locks. BrowserManager keeps policy
 * (validation, arguments, health); this class owns concurrent pooling only.
 */
export class BrowserPool {
  private readonly browsers = new Map<string, Browser>();
  private readonly launches = new Map<string, Promise<Browser>>();
  private readonly lastUsedAt = new Map<string, number>();
  private readonly evictions = new Map<string, Promise<boolean>>();
  private closing = false;
  private generation = 0;
  private closePromise?: Promise<void>;

  get(key: string): Browser | undefined {
    if (this.closing || this.evictions.has(key)) return undefined;
    const browser = this.browsers.get(key);
    return browser?.isConnected() ? browser : undefined;
  }

  async getOrLaunch(key: string, launch: () => Promise<Browser>): Promise<Browser> {
    const eviction = this.evictions.get(key);
    if (eviction) await eviction.catch(() => false);
    if (this.closing) throw new Error('Browser pool is shutting down');
    const generation = this.generation;
    const existing = this.get(key);
    if (existing) {
      this.lastUsedAt.set(key, Date.now());
      return existing;
    }

    const inFlight = this.launches.get(key);
    if (inFlight) {
      const browser = await inFlight;
      if (generation !== this.generation) throw new Error('Browser pool shut down during launch');
      if (browser.isConnected()) {
        this.lastUsedAt.set(key, Date.now());
        return browser;
      }
      return this.getOrLaunch(key, launch);
    }

    const promise = this.launchWithSingleRetry(launch, generation);
    this.launches.set(key, promise);
    try {
      const browser = await promise;
      if (generation !== this.generation) throw new Error('Browser pool shut down during launch');
      if (browser.isConnected()) {
        this.browsers.set(key, browser);
        this.lastUsedAt.set(key, Date.now());
      }
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

  /**
   * Close browsers that have been idle long enough and have no open contexts.
   * Reserving each key before awaiting prevents a new session from obtaining a
   * browser while its idle eviction is in progress. A failed close keeps a
   * still-connected browser available for reuse.
   */
  async closeIdle(
    idleForMs: number,
    close: (key: string, browser: Browser) => Promise<void>,
    now = Date.now()
  ): Promise<number> {
    if (this.closing || idleForMs < 0) return 0;
    let closed = 0;
    for (const [key, browser] of this.browsers) {
      if (this.evictions.has(key)) continue;
      const lastUsed = this.lastUsedAt.get(key) ?? now;
      if (now - lastUsed < idleForMs) continue;

      const eviction = Promise.resolve().then(async () => {
        if (this.closing || this.browsers.get(key) !== browser) return false;
        if (!browser.isConnected()) {
          this.browsers.delete(key);
          this.lastUsedAt.delete(key);
          return false;
        }
        const currentLastUsed = this.lastUsedAt.get(key) ?? now;
        if (now - currentLastUsed < idleForMs || browser.contexts().length !== 0) return false;

        this.browsers.delete(key);
        this.lastUsedAt.delete(key);
        try {
          await close(key, browser);
          return true;
        } catch (error) {
          if (browser.isConnected() && !this.closing) {
            this.browsers.set(key, browser);
            this.lastUsedAt.set(key, currentLastUsed);
          }
          throw error;
        }
      });
      this.evictions.set(key, eviction);
      try {
        if (await eviction) closed++;
      } finally {
        if (this.evictions.get(key) === eviction) this.evictions.delete(key);
      }
    }
    return closed;
  }

  usageSnapshot(now = Date.now()): BrowserPoolUsage {
    const usage: BrowserPoolUsage = {
      browserCount: 0,
      contextCount: 0,
      idleBrowserCount: 0,
      maxIdleSeconds: 0,
    };
    for (const [key, browser] of this.browsers) {
      if (!browser.isConnected()) continue;
      usage.browserCount++;
      let contexts = 0;
      try {
        contexts = browser.contexts().length;
      } catch {
        // Metrics must not interfere with browser ownership or session work.
        continue;
      }
      usage.contextCount += contexts;
      if (contexts === 0) {
        usage.idleBrowserCount++;
        const lastUsed = this.lastUsedAt.get(key) ?? now;
        usage.maxIdleSeconds = Math.max(usage.maxIdleSeconds, Math.floor((now - lastUsed) / 1_000));
      }
    }
    return usage;
  }

  /** Begin a fresh idle grace period after session/capture activity ends. */
  markUsed(now = Date.now()): void {
    for (const [key, browser] of this.browsers) {
      if (browser.isConnected()) this.lastUsedAt.set(key, now);
    }
  }

  private async closeOwnedBrowsers(
    launches: Array<[string, Promise<Browser>]>,
    close: (key: string, browser: Browser) => Promise<void>
  ): Promise<void> {
    await Promise.allSettled([...this.evictions.values()]);
    const launchResults = await Promise.allSettled(launches.map(([, launch]) => launch));
    const owned = new Map<Browser, string>();

    for (const [key, browser] of this.browsers) owned.set(browser, key);
    launchResults.forEach((result, index) => {
      const launch = launches[index];
      if (result.status === 'fulfilled' && launch) owned.set(result.value, launch[0]);
    });

    this.browsers.clear();
    this.lastUsedAt.clear();
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
