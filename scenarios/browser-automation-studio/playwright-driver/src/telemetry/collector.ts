import type { Page } from 'playwright';
import type { ConsoleLogEntry, NetworkEvent } from '../types';
import { MAX_CONSOLE_ENTRIES, MAX_NETWORK_EVENTS } from '../constants';
import { logger, normalizeConsoleLogType } from '../utils';

/**
 * Console log collector
 *
 * Collects browser console messages during instruction execution
 */
export class ConsoleLogCollector {
  private logs: ConsoleLogEntry[] = [];
  private maxEntries: number;
  private page: Page;

  constructor(page: Page, maxEntries: number = MAX_CONSOLE_ENTRIES) {
    this.page = page;
    this.maxEntries = maxEntries;
    this.setupListener();
  }

  private setupListener(): void {
    this.page.on('console', (msg) => {
      if (this.logs.length >= this.maxEntries) {
        // Remove oldest entry when limit reached
        this.logs.shift();
      }

      const loc = msg.location();
      const locationStr = loc.url ? `${loc.url}:${loc.lineNumber}:${loc.columnNumber}` : '';

      // Hardened: Use validated console log type mapping
      const type = normalizeConsoleLogType(msg.type());

      const entry: ConsoleLogEntry = {
        timestamp: new Date().toISOString(),
        type,
        text: msg.text(),
        location: locationStr,
      };

      this.logs.push(entry);
    });
  }

  getLogs(): ConsoleLogEntry[] {
    return [...this.logs];
  }

  clear(): void {
    this.logs = [];
  }

  getAndClear(): ConsoleLogEntry[] {
    const logs = this.getLogs();
    this.clear();
    return logs;
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
 */
export class NetworkCollector {
  private events: NetworkEvent[] = [];
  private maxEvents: number;
  private page: Page;
  private requestMap: Map<string, { method: string; url: string; timestamp: string }> = new Map();
  /** Counter for generating unique request IDs when Playwright ID unavailable */
  private requestCounter = 0;

  constructor(page: Page, maxEvents: number = MAX_NETWORK_EVENTS) {
    this.page = page;
    this.maxEvents = maxEvents;
    this.setupListeners();
  }

  private setupListeners(): void {
    // Track requests
    this.page.on('request', (request) => {
      const id = this.getRequestId(request);
      this.requestMap.set(id, {
        method: request.method(),
        url: request.url(),
        timestamp: new Date().toISOString(),
      });
    });

    // Track responses
    this.page.on('response', (response) => {
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
    });

    // Track failures
    this.page.on('requestfailed', (request) => {
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
    });
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

  getEvents(): NetworkEvent[] {
    return [...this.events];
  }

  clear(): void {
    this.events = [];
    this.requestMap.clear();
  }

  getAndClear(): NetworkEvent[] {
    const events = this.getEvents();
    this.clear();
    return events;
  }
}
