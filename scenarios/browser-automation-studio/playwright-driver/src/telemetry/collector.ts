import type { Page } from 'playwright';
import type { ConsoleLogEntry, NetworkEvent } from '../types';
import { MAX_CONSOLE_ENTRIES, MAX_NETWORK_EVENTS } from '../constants';
import { logger } from '../utils';

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

      // Map Playwright console types to our types
      const msgType = msg.type();
      const type: 'log' | 'warn' | 'error' | 'info' | 'debug' =
        msgType === 'warning' ? 'warn' :
        ['log', 'warn', 'error', 'info', 'debug'].includes(msgType) ? msgType as any : 'log';

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
export class NetworkCollector {
  private events: NetworkEvent[] = [];
  private maxEvents: number;
  private page: Page;
  private requestMap: Map<string, { method: string; url: string; timestamp: string }> = new Map();

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

  private getRequestId(request: { url: () => string; method: () => string }): string {
    // Use URL + method as unique identifier
    return `${request.method()}:${request.url()}`;
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
