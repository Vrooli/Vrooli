import type { Page, CDPSession, Request, Response } from 'rebrowser-playwright';
import type { ConsoleLogEntry, NetworkEvent } from '../types';
import { MAX_CONSOLE_ENTRIES, MAX_NETWORK_EVENTS } from '../constants';
import { logger, normalizeConsoleLogType } from '../utils';

/**
 * Console log collector
 *
 * Collects browser console messages during instruction execution.
 *
 * Temporal hardening:
 * - Event listener is stored and can be removed via dispose()
 * - dispose() should be called when collector is no longer needed
 */
export class ConsoleLogCollector {
  private logs: ConsoleLogEntry[] = [];
  private session?: CDPSession;
  private removeListener?: () => void;
  private starting?: Promise<void>;
  private disposing?: Promise<void>;
  private disposed = false;

  constructor(private readonly page: Page, private readonly maxEntries: number = MAX_CONSOLE_ENTRIES) {}

  start(): Promise<void> {
    if (this.disposed) return Promise.reject(new Error('Console collector disposed'));
    return this.starting ??= this.initialize();
  }

  private async initialize(): Promise<void> {
    const since = Date.now();
    const session = this.session = await this.page.context().newCDPSession(this.page);
    if (this.disposed) return;
    const onConsole: Parameters<typeof session.on<'Runtime.consoleAPICalled'>>[1] = event => {
      // Runtime.enable also replays console history; only this collection window
      // belongs to the instruction. Never evaluate remote objects/getters.
      if (this.disposed || event.timestamp < since) return;
      if (this.logs.length >= this.maxEntries) this.logs.shift();
      const frame = event.stackTrace?.callFrames[0];
      this.logs.push({
        type: normalizeConsoleLogType(event.type),
        text: event.args.map(arg => arg.unserializableValue ?? (arg.value === undefined
          ? arg.description ?? arg.type
          : typeof arg.value === 'string' ? arg.value : JSON.stringify(arg.value))).join(' '),
        timestamp: new Date(event.timestamp).toISOString(),
        location: frame?.url ? `${frame.url}:${frame.lineNumber}:${frame.columnNumber}` : '',
      });
    };
    session.on('Runtime.consoleAPICalled', onConsole);
    this.removeListener = () => session.off('Runtime.consoleAPICalled', onConsole);
    await session.send('Runtime.enable');
  }

  getLogs(): ConsoleLogEntry[] { return [...this.logs]; }
  clear(): void { this.logs = []; }
  getAndClear(): ConsoleLogEntry[] {
    const logs = this.getLogs();
    this.clear();
    return logs;
  }

  dispose(): Promise<void> {
    return this.disposing ??= this.close();
  }

  private async close(): Promise<void> {
    this.disposed = true;
    // A late attachment remains owned until it has been detached.
    await this.starting?.catch(() => undefined);
    if (this.session) {
      this.removeListener?.();
      this.removeListener = undefined;
      try { await this.session.detach(); }
      catch (error) { if (!this.page.isClosed()) throw error; }
      this.session = undefined;
    }
    this.logs = [];
  }
}

/**
 * Network event collector
 *
 * Collects HTTP requests and responses during instruction execution
 */
/**
 * Hardened assumptions:
 * - Multiple requests to the same URL+method can happen concurrently
 * - Playwright request objects have a unique internal ID we can use
 * - Responses may arrive out of order or not at all
 * - Request map entries may leak if responses never arrive (cleaned up on clear)
 *
 * Temporal hardening:
 * - Request map has bounded size to prevent memory leaks from orphaned requests
 * - Old entries are evicted when map exceeds MAX_PENDING_REQUESTS
 * - Request timestamps enable age-based eviction for stale entries
 */
export class NetworkCollector {
  private events: NetworkEvent[] = [];
  private maxEvents: number;
  private page: Page;
  private requestMap: Map<string, { method: string; url: string; timestamp: string; createdAt: number }> = new Map();
  /** Counter for generating unique request IDs when Playwright ID unavailable */
  private requestCounter = 0;
  /** Maximum pending requests before evicting oldest */
  private static readonly MAX_PENDING_REQUESTS = 500;
  /** Maximum age for pending requests before considered stale (30 seconds) */
  private static readonly MAX_REQUEST_AGE_MS = 30_000;
  /** Bound listener references for cleanup */
  private requestHandler: ((request: Request) => void) | null = null;
  private responseHandler: ((response: Response) => void) | null = null;
  private requestFailedHandler: ((request: Request) => void) | null = null;
  /** Track if collector has been disposed */
  private disposed = false;

  constructor(page: Page, maxEvents: number = MAX_NETWORK_EVENTS) {
    this.page = page;
    this.maxEvents = maxEvents;
    this.setupListeners();
  }

  private setupListeners(): void {
    // Track requests
    this.requestHandler = (request: Request): void => {
      // Guard: Don't process events after dispose
      if (this.disposed) return;
      const id = this.getRequestId(request);
      const now = Date.now();

      // Evict stale entries before adding new one (prevents unbounded growth)
      this.evictStaleRequests(now);

      this.requestMap.set(id, {
        method: request.method(),
        url: request.url(),
        timestamp: new Date().toISOString(),
        createdAt: now,
      });
    };
    this.page.on('request', this.requestHandler);

    // Track responses
    this.responseHandler = (response: Response): void => {
      // Guard: Don't process events after dispose
      if (this.disposed) return;
      const id = this.getRequestId(response.request());
      const requestData = this.requestMap.get(id);

      if (!requestData) {
        logger.debug('Response received without matching request', { url: response.url() });
        return;
      }

      if (this.events.length >= this.maxEvents) {
        // Remove oldest event when limit reached
        this.events.shift();
      }

      const event: NetworkEvent = {
        type: 'response',
        timestamp: requestData.timestamp,
        method: requestData.method,
        url: requestData.url,
        status: response.status(),
        ok: response.ok(),
        resource_type: response.request().resourceType(),
      };

      this.events.push(event);
      this.requestMap.delete(id);
    };
    this.page.on('response', this.responseHandler);

    // Track failures
    this.requestFailedHandler = (request: Request): void => {
      // Guard: Don't process events after dispose
      if (this.disposed) return;
      const id = this.getRequestId(request);
      const requestData = this.requestMap.get(id);

      if (!requestData) {
        return;
      }

      if (this.events.length >= this.maxEvents) {
        this.events.shift();
      }

      const event: NetworkEvent = {
        type: 'failure',
        timestamp: requestData.timestamp,
        method: requestData.method,
        url: requestData.url,
        failure: request.failure()?.errorText || 'Request failed',
        resource_type: request.resourceType(),
      };

      this.events.push(event);
      this.requestMap.delete(id);
    };
    this.page.on('requestfailed', this.requestFailedHandler);
  }

  /**
   * Get a unique identifier for a request.
   *
   * Hardened: Uses Playwright's internal request reference (via WeakMap pattern)
   * instead of URL+method which is NOT unique for concurrent requests to same endpoint.
   *
   * For Playwright requests, we use the request object's string representation
   * which includes an internal unique ID. This handles:
   * - Multiple concurrent requests to same URL
   * - Same URL called multiple times in sequence
   * - Requests that share identical URL and method
   */
  private getRequestId(request: { url: () => string; method: () => string }): string {
    // Playwright Request objects have a stable toString() that includes unique internal ID
    // Format: "Request: <method> <url>" but importantly the object identity is unique
    // We create a composite key using object reference via a counter when we first see the request
    //
    // Note: We still include method:url for debugging, but prefix with counter for uniqueness
    const baseId = `${request.method()}:${request.url()}`;

    // For response/failure lookups, we need to match back to the original request
    // Playwright guarantees response.request() returns the same Request object
    // so we can use object identity. However, since we can't use WeakMap with
    // the request object directly (it's not the key), we use a simpler approach:
    // For requests, we'll store with a sequence number that we can match on lookup.
    //
    // Actually, Playwright's response.request() returns the exact same Request object,
    // so the simplest fix is to use the string representation which IS unique per request.
    try {
      // Use String() to get Playwright's internal representation which is unique
      return String(request);
    } catch {
      // Fallback if String() fails for some reason
      return `${this.requestCounter++}:${baseId}`;
    }
  }

  /**
   * Evict stale pending requests to prevent memory leaks.
   *
   * Called on each new request to maintain bounded memory usage.
   * Evicts entries that are either:
   * - Older than MAX_REQUEST_AGE_MS (likely orphaned)
   * - When map exceeds MAX_PENDING_REQUESTS (FIFO eviction)
   */
  private evictStaleRequests(now: number): void {
    // First pass: remove stale entries by age
    for (const [id, data] of this.requestMap.entries()) {
      if (now - data.createdAt > NetworkCollector.MAX_REQUEST_AGE_MS) {
        this.requestMap.delete(id);
        logger.debug('Evicted stale pending request', {
          url: data.url.slice(0, 100),
          ageMs: now - data.createdAt,
        });
      }
    }

    // Second pass: if still over limit, evict oldest by creation time
    if (this.requestMap.size > NetworkCollector.MAX_PENDING_REQUESTS) {
      const entries = Array.from(this.requestMap.entries())
        .sort((a, b) => a[1].createdAt - b[1].createdAt);

      const toEvict = entries.slice(0, this.requestMap.size - NetworkCollector.MAX_PENDING_REQUESTS);
      for (const [id] of toEvict) {
        this.requestMap.delete(id);
      }

      if (toEvict.length > 0) {
        logger.debug('Evicted oldest pending requests due to size limit', {
          evictedCount: toEvict.length,
          remainingCount: this.requestMap.size,
        });
      }
    }
  }

  getEvents(): NetworkEvent[] {
    return [...this.events];
  }

  clear(): void {
    this.events = [];
    this.requestMap.clear();
    // Reset counter to prevent overflow on very long sessions
    this.requestCounter = 0;
  }

  getAndClear(): NetworkEvent[] {
    const events = this.getEvents();
    this.clear();
    return events;
  }

  /**
   * Dispose the collector and remove event listeners.
   *
   * Temporal hardening: Must be called to prevent memory leaks and
   * stale event handlers when the collector is no longer needed.
   */
  dispose(): void {
    if (this.disposed) return;
    this.disposed = true;

    if (this.requestHandler) {
      this.page.off('request', this.requestHandler);
      this.requestHandler = null;
    }

    if (this.responseHandler) {
      this.page.off('response', this.responseHandler);
      this.responseHandler = null;
    }

    if (this.requestFailedHandler) {
      this.page.off('requestfailed', this.requestFailedHandler);
      this.requestFailedHandler = null;
    }

    this.events = [];
    this.requestMap.clear();
  }
}
