/* eslint-disable @typescript-eslint/no-explicit-any -- The optional pinned MCP package is deliberately not a BAS dependency, and its private handler registry has no public types. */
/* eslint-disable @typescript-eslint/require-await -- The façade mirrors asynchronous methods required by the official MCP context contract, including methods that reject or return local state. */
/* eslint-disable @typescript-eslint/no-unnecessary-type-assertion -- Upstream task JSON is runtime-validated at the boundary before this narrow cast. */

import { EventEmitter } from 'node:events';

/** The subset of the existing BAS task surface that this adapter can call. */
export interface BrokerTaskTransport {
  runTask(sessionId: string, task: Record<string, unknown>): Promise<unknown>;
  draftTrace?(sessionId: string, request: TraceDraftRequest): Promise<unknown>;
}

export interface TraceDraftRequest extends Record<string, unknown> {
  name: string;
  description?: string;
  assertions: Array<Record<string, unknown>>;
}

/** An owner-held authenticated request closure; it adds identity/grant headers out of band. */
export type AuthorizedPost = (
  url: string,
  body: Record<string, unknown>
) => Promise<{
  status: number;
  json(): Promise<unknown>;
  body?: ReadableStream<Uint8Array> | null;
}>;

const MAX_BROKER_RESPONSE_BYTES = 2 << 20;

/** Read production fetch bodies incrementally so a compromised/misconfigured
 * Broker cannot make an adapter retain an unbounded response. The json-only
 * fallback supports caller-owned transports that do not expose a stream. */
async function readBrokerJSON(response: {
  json(): Promise<unknown>;
  body?: ReadableStream<Uint8Array> | null;
}): Promise<unknown> {
  if (!response.body) return response.json();
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      total += value.byteLength;
      if (total > MAX_BROKER_RESPONSE_BYTES) {
        await reader.cancel();
        throw new Error('BAS response exceeds the size limit');
      }
      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }
  const bytes = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  try {
    return JSON.parse(new TextDecoder().decode(bytes));
  } catch {
    throw new Error('BAS response is not valid JSON');
  }
}

/** HTTP transport for BAS's existing Agent Access task route. Credentials stay
 * in the trusted request closure and never enter MCP tool arguments or task JSON. */
export class BASBrowserTaskHTTPTransport implements BrokerTaskTransport {
  private readonly baseURL: string;
  constructor(
    apiURL: string,
    private readonly sessionId: string,
    private readonly authorizedPost: AuthorizedPost
  ) {
    if (!sessionId.trim()) throw new Error('a pre-existing BAS session ID is required');
    const parsed = new URL(apiURL);
    if (
      !['http:', 'https:'].includes(parsed.protocol) ||
      parsed.username ||
      parsed.password ||
      parsed.pathname !== '/' ||
      parsed.search ||
      parsed.hash
    ) {
      throw new Error(
        'BAS API URL must be an HTTP(S) origin without credentials, path, query, or fragment'
      );
    }
    this.baseURL = parsed.origin;
  }

  async runTask(sessionId: string, task: Record<string, unknown>): Promise<unknown> {
    if (sessionId !== this.sessionId)
      throw new Error('BAS task transport is bound to a different session');
    if (!validBrokerTask(task))
      throw new Error('BAS task is outside the fixed typed adapter contract');
    const response = await this.authorizedPost(
      `${this.baseURL}/api/v1/browser/sessions/${encodeURIComponent(sessionId)}/task`,
      task
    );
    if (response.status < 200 || response.status >= 300) {
      let code = 'BROKER_TASK_DENIED';
      try {
        const payload = (await readBrokerJSON(response)) as { error?: { code?: string } };
        if (payload?.error?.code) code = payload.error.code;
      } catch {
        /* keep the safe generic code */
      }
      throw new Error(code);
    }
    return readBrokerJSON(response);
  }

  async draftTrace(sessionId: string, request: TraceDraftRequest): Promise<unknown> {
    if (sessionId !== this.sessionId)
      throw new Error('BAS draft transport is bound to a different session');
    if (!validTraceDraftRequest(request))
      throw new Error('BAS trace draft is outside the fixed typed contract');
    const response = await this.authorizedPost(
      `${this.baseURL}/api/v1/browser/sessions/${encodeURIComponent(sessionId)}/trace/draft`,
      request
    );
    if (response.status < 200 || response.status >= 300) {
      let code = 'TRACE_DRAFT_DENIED';
      try {
        const payload = (await readBrokerJSON(response)) as { error?: { code?: string } };
        if (payload?.error?.code) code = payload.error.code;
      } catch {
        /* keep the safe generic code */
      }
      throw new Error(code);
    }
    return readBrokerJSON(response);
  }
}

/**
 * Bind the adapter to Agent Manager's existing run environment. These values
 * are injected by the run owner (the CLI uses the same names); they are never
 * MCP arguments, and BAS revalidates the identity/grant for every task call.
 */
export function createAgentManagerTaskTransport(sessionId: string): BASBrowserTaskHTTPTransport {
  const apiURL = process.env.BAS_BROWSER_API_URL?.trim();
  const identityToken = process.env.VROOLI_AGENT_IDENTITY_TOKEN?.trim();
  const grantReceipt = process.env.VROOLI_BROWSER_GRANT_RECEIPT?.trim();
  if (!apiURL)
    throw new Error('BAS_BROWSER_API_URL is required from the Agent Manager run context');
  if (!identityToken || !grantReceipt)
    throw new Error('Agent Manager identity and browser grant are required from the run context');

  const authorizedPost: AuthorizedPost = async (url, body) => {
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
        'X-Agent-Identity-Token': identityToken,
        'X-Vrooli-Browser-Grant-Receipt': grantReceipt,
      },
      body: JSON.stringify(body),
      redirect: 'manual',
    });
    return { status: response.status, json: () => response.json() };
  };
  return new BASBrowserTaskHTTPTransport(apiURL, sessionId, authorizedPost);
}

export interface BrokerBackedMcpOptions {
  sessionId: string;
  transport: BrokerTaskTransport;
  /** The pinned @playwright/mcp 0.0.85 `createConnection` entrypoint. */
  createConnection: (
    config: Record<string, unknown>,
    contextGetter: () => Promise<unknown>
  ) => Promise<unknown>;
}

const ALLOWED_TOOLS = new Set([
  'browser_snapshot',
  'browser_click',
  'browser_type',
  'browser_navigate',
  'browser_press_key',
  'browser_receipt',
  'browser_draft_trace',
]);
const REF = /^e[0-9]{1,90}$/;
const PRESSABLE_KEY =
  /^(?:[A-Za-z0-9]|Enter|Tab|Escape|Backspace|Delete|Arrow(?:Left|Right|Up|Down)|Home|End|PageUp|PageDown|Space|F(?:[1-9]|1[0-2]))$/i;

function keyboardTask(key: string): Record<string, unknown> | undefined {
  const parts = key.split('+');
  let pressed: string;
  let modifiers: string[] = [];
  if (parts.length === 1) pressed = parts[0] ?? '';
  else if (
    parts.length === 2 &&
    ['control', 'ctrl', 'meta'].includes((parts[0] ?? '').toLowerCase())
  ) {
    modifiers = [(parts[0] ?? '').toLowerCase() === 'meta' ? 'meta' : 'ctrl'];
    pressed = parts[1] ?? '';
  } else return undefined;
  if (!PRESSABLE_KEY.test(pressed)) return undefined;
  return { operation: 'keyboard', key: pressed, ...(modifiers.length ? { modifiers } : {}) };
}

function validBrokerTask(task: Record<string, unknown>): boolean {
  if (
    !task ||
    typeof task !== 'object' ||
    Array.isArray(task) ||
    typeof task.operation !== 'string'
  )
    return false;
  if (task.operation === 'read' || task.operation === 'receipt')
    return onlyKeys(task, ['operation']);
  if (task.operation === 'click') return onlyKeys(task, ['operation', 'ref']) && isRef(task.ref);
  if (task.operation === 'fill')
    return (
      onlyKeys(task, ['operation', 'ref', 'value']) &&
      isRef(task.ref) &&
      typeof task.value === 'string'
    );
  if (task.operation === 'navigate')
    return (
      onlyKeys(task, ['operation', 'url']) && typeof task.url === 'string' && task.url.trim() !== ''
    );
  if (task.operation === 'keyboard')
    return (
      onlyKeys(task, ['operation', 'key', 'modifiers']) &&
      typeof task.key === 'string' &&
      PRESSABLE_KEY.test(task.key) &&
      (task.modifiers === undefined ||
        (Array.isArray(task.modifiers) &&
          task.modifiers.length === 1 &&
          ['ctrl', 'meta'].includes(String(task.modifiers[0]))))
    );
  return false;
}

function validTraceDraftRequest(request: TraceDraftRequest): boolean {
  if (!request || typeof request !== 'object' || Array.isArray(request)) return false;
  if (
    !onlyKeys(request as unknown as Record<string, unknown>, ['name', 'description', 'assertions'])
  )
    return false;
  if (
    typeof request.name !== 'string' ||
    !request.name.trim() ||
    (request.description !== undefined && typeof request.description !== 'string') ||
    !Array.isArray(request.assertions) ||
    request.assertions.length === 0
  )
    return false;
  return request.assertions.every(
    (assertion) => assertion && typeof assertion === 'object' && !Array.isArray(assertion)
  );
}

/**
 * Build the official Playwright MCP server around a façade that has no
 * Playwright Page, Browser, CDP, script, or lifecycle capability. This pins
 * the package's registered MCP handlers to the bounded BAS task family and
 * forwards only typed tasks through the caller-supplied authenticated Broker
 * transport. There is intentionally no local launch or fallback path.
 */
export async function createBrokerBackedPlaywrightMcp(
  options: BrokerBackedMcpOptions
): Promise<{ server: any; context: BrokerContextFacade }> {
  if (!options.sessionId.trim()) throw new Error('a pre-existing BAS session ID is required');
  const context = new BrokerContextFacade(options.sessionId, options.transport);
  const server = await options.createConnection(
    {
      browser: { isolated: false },
      capabilities: ['core-input', 'core-navigation'],
      sharedBrowserContext: true,
      codegen: 'none',
    },
    async () => context
  );
  installToolPolicy(server, options);
  return { server, context };
}

/**
 * @playwright/mcp 0.0.85 has no public tool allowlist on createConnection.
 * Its returned MCP Server stores handlers in `_requestHandlers`; this pinned
 * adapter wraps those already-registered handlers before any transport can
 * expose them. The exact pin and focused contract test guard this seam.
 */
function installToolPolicy(server: any, options: BrokerBackedMcpOptions): void {
  const handlers: Map<string, (...args: any[]) => Promise<any>> | undefined =
    server?._requestHandlers;
  if (!(handlers instanceof Map))
    throw new Error('pinned Playwright MCP Server handler registry is unavailable');
  const list = handlers.get('tools/list');
  const call = handlers.get('tools/call');
  if (!list || !call) throw new Error('pinned Playwright MCP tools handlers are unavailable');
  handlers.set('tools/list', async (...args: any[]) => {
    const result = await list(...args);
    const tools = (result.tools ?? []).filter(
      (tool: { name?: string }) =>
        ALLOWED_TOOLS.has(tool.name ?? '') &&
        tool.name !== 'browser_receipt' &&
        tool.name !== 'browser_draft_trace'
    );
    tools.push({
      name: 'browser_receipt',
      description: 'Read the LPBS settlement result for this exact BAS session after it closes.',
      inputSchema: { type: 'object', properties: {}, additionalProperties: false },
    });
    tools.push({
      name: 'browser_draft_trace',
      description:
        'Compile the accepted trace of this existing BAS session into a deterministic, unsaved workflow draft. Does not write a project workflow.',
      inputSchema: {
        type: 'object',
        required: ['name', 'assertions'],
        additionalProperties: false,
        properties: {
          name: { type: 'string', minLength: 1 },
          description: { type: 'string' },
          assertions: { type: 'array', minItems: 1, items: { type: 'object' } },
        },
      },
    });
    return { ...result, tools };
  });
  handlers.set('tools/call', async (request: any, ...args: any[]) => {
    const name = request?.params?.name;
    const input = request?.params?.arguments ?? {};
    if (!ALLOWED_TOOLS.has(name))
      return denied(`Playwright MCP tool is outside the BAS task adapter: ${String(name)}`);
    if (name === 'browser_receipt') {
      if (
        !input ||
        typeof input !== 'object' ||
        Array.isArray(input) ||
        Object.keys(input).length !== 0
      )
        return denied('BAS receipt accepts no caller-supplied account or reservation fields');
      try {
        const receipt = await options.transport.runTask(options.sessionId, {
          operation: 'receipt',
        });
        return { content: [{ type: 'text', text: JSON.stringify(receipt) }] };
      } catch {
        return denied('BAS session receipt is unavailable or not authorized');
      }
    }
    if (name === 'browser_draft_trace') {
      if (!validTraceDraftRequest(input as TraceDraftRequest) || !options.transport.draftTrace)
        return denied(
          'BAS trace draft requires a supported typed draft transport and valid assertions'
        );
      try {
        const draft = await options.transport.draftTrace(
          options.sessionId,
          input as TraceDraftRequest
        );
        return { content: [{ type: 'text', text: JSON.stringify(draft) }] };
      } catch {
        return denied('BAS trace draft is unavailable or not authorized for this session');
      }
    }
    if (!validToolArguments(name, input))
      return denied('Playwright MCP arguments exceed the BAS typed task contract');
    return call(request, ...args);
  });
}

function validToolArguments(name: string, input: Record<string, unknown>): boolean {
  if (!input || typeof input !== 'object' || Array.isArray(input)) return false;
  if (name === 'browser_snapshot')
    return (
      onlyKeys(input, ['target', 'depth', 'boxes']) &&
      (input.target === undefined || isRef(input.target))
    );
  if (name === 'browser_click') {
    return (
      onlyKeys(input, ['target', 'element', 'button', 'modifiers', 'doubleClick']) &&
      isRef(input.target) &&
      input.doubleClick !== true &&
      (input.button === undefined || input.button === 'left') &&
      (input.modifiers === undefined ||
        (Array.isArray(input.modifiers) && input.modifiers.length === 0))
    );
  }
  if (name === 'browser_type') {
    return (
      onlyKeys(input, ['target', 'element', 'text', 'submit', 'slowly']) &&
      isRef(input.target) &&
      typeof input.text === 'string' &&
      input.submit !== true &&
      input.slowly !== true
    );
  }
  if (name === 'browser_navigate')
    return onlyKeys(input, ['url']) && typeof input.url === 'string' && input.url.trim() !== '';
  if (name === 'browser_press_key')
    return (
      onlyKeys(input, ['key']) &&
      typeof input.key === 'string' &&
      keyboardTask(input.key) !== undefined
    );
  return false;
}

function onlyKeys(input: Record<string, unknown>, allowed: string[]): boolean {
  return Object.keys(input).every((key) => allowed.includes(key));
}

function isRef(value: unknown): value is string {
  return typeof value === 'string' && REF.test(value);
}

function denied(message: string): {
  content: Array<{ type: 'text'; text: string }>;
  isError: true;
} {
  return { content: [{ type: 'text', text: message }], isError: true };
}

interface AccessibilityNode {
  role?: string;
  name?: string;
  description?: string;
  states?: string[];
  ref?: string;
  children?: AccessibilityNode[];
}

interface AccessibilityTreeDocument {
  url?: string;
  current_url?: string;
  root: AccessibilityNode;
}

export class BrokerContextFacade extends EventEmitter {
  readonly page: BrokerPageFacade;
  readonly debugger = { pausedDetails: () => null };

  constructor(
    readonly sessionId: string,
    transport: BrokerTaskTransport
  ) {
    super();
    this.page = new BrokerPageFacade(this, transport);
  }

  pages(): BrokerPageFacade[] {
    return [this.page];
  }
  async settlementReceipt(): Promise<unknown> {
    return this.page.settlementReceipt();
  }
  browser(): undefined {
    return undefined;
  }
  isClosed(): boolean {
    return false;
  }
  checkUrlAllowed(url: string): void {
    if (/^(file|javascript|data):/i.test(url))
      throw new Error('URL scheme is outside the BAS browser task contract');
  }
  route(): never {
    throw new Error('network routing is outside the BAS browser task contract');
  }
  unroute(): never {
    throw new Error('network routing is outside the BAS browser task contract');
  }
  addInitScript(): never {
    throw new Error('page scripts are outside the BAS browser task contract');
  }
  newPage(): never {
    throw new Error('BAS owns browser lifecycle; a new page cannot be created here');
  }
  close(): never {
    throw new Error('BAS owns browser lifecycle; this façade cannot close the session');
  }
  cookies(): never {
    throw new Error('cookies are outside the BAS browser task contract');
  }
  addCookies(): never {
    throw new Error('cookies are outside the BAS browser task contract');
  }
}

export class BrokerPageFacade extends EventEmitter {
  private currentUrl = 'about:blank';
  private pageTitle = '';
  private readonly mainFrameFacade = {
    url: () => this.currentUrl,
    webmcp: { tools: async () => [] as never[] },
  };
  readonly keyboard = {
    press: async (key: string): Promise<void> => {
      const task = keyboardTask(key);
      if (!task) throw new Error('key is outside the bounded BAS keyboard contract');
      await this.transport.runTask(this.browserContext.sessionId, task);
    },
  };

  constructor(
    private readonly browserContext: BrokerContextFacade,
    private readonly transport: BrokerTaskTransport
  ) {
    super();
  }

  context(): BrokerContextFacade {
    return this.browserContext;
  }
  url(): string {
    return this.currentUrl;
  }
  frames(): Array<typeof this.mainFrameFacade> {
    return [this.mainFrameFacade];
  }
  mainFrame(): typeof this.mainFrameFacade {
    return this.mainFrameFacade;
  }
  async title(): Promise<string> {
    return this.pageTitle;
  }
  async settlementReceipt(): Promise<unknown> {
    return this.transport.runTask(this.browserContext.sessionId, { operation: 'receipt' });
  }
  isClosed(): boolean {
    return false;
  }
  async goto(url: string): Promise<void> {
    this.browserContext.checkUrlAllowed(url);
    const result = await this.transport.runTask(this.browserContext.sessionId, {
      operation: 'navigate',
      url,
    });
    this.currentUrl = resultString(result, ['url', 'current_url', 'final_url']) || url;
  }
  async waitForLoadState(): Promise<void> {
    /* Broker action receipts are the completion boundary. */
  }
  async ariaSnapshot(): Promise<string> {
    const tree = await this.readTree();
    this.currentUrl = resultString(tree, ['url', 'current_url']) || this.currentUrl;
    this.pageTitle = tree.root?.name ?? this.pageTitle;
    return formatTree(tree.root);
  }
  async ariaSnapshotJSON(): Promise<unknown> {
    return (await this.readTree()).root;
  }
  getByRef(ref: string): BrokerLocatorFacade {
    if (!isRef(ref)) throw new Error('only BAS opaque accessibility refs are accepted');
    return new BrokerLocatorFacade(this, ref);
  }
  locator(selector: string): BrokerLocatorFacade {
    const match = /^aria-ref=(e[0-9]{1,90})$/.exec(selector);
    if (!match?.[1]) throw new Error('only BAS opaque accessibility refs are accepted');
    return this.getByRef(match[1] ?? '');
  }
  $(selector: string): never {
    void selector;
    throw new Error('CSS/DOM element handles are not exposed by the BAS façade');
  }
  evaluate(): never {
    throw new Error('page script execution is not exposed by the BAS façade');
  }
  evaluateHandle(): never {
    throw new Error('page script execution is not exposed by the BAS façade');
  }
  close(): never {
    throw new Error('BAS owns browser lifecycle; this façade cannot close the session');
  }
  newPage(): never {
    throw new Error('BAS owns browser lifecycle; this façade cannot create a page');
  }
  setViewportSize(): never {
    throw new Error('viewport changes are outside the BAS task contract');
  }
  bringToFront(): never {
    throw new Error('page selection is owned by the BAS session');
  }
  async consoleMessages(): Promise<never[]> {
    return [];
  }
  async pageErrors(): Promise<never[]> {
    return [];
  }
  async requests(): Promise<never[]> {
    return [];
  }
  async clearConsoleMessages(): Promise<void> {}
  async clearPageErrors(): Promise<void> {}

  async screenshot(): Promise<never> {
    throw new Error('screenshots are not enabled in this Playwright MCP adapter');
  }
  async reload(): Promise<never> {
    throw new Error('reload is outside the BAS task contract');
  }
  async goBack(): Promise<never> {
    throw new Error('history navigation is outside the BAS task contract');
  }
  async goForward(): Promise<never> {
    throw new Error('history navigation is outside the BAS task contract');
  }

  private async readTree(): Promise<AccessibilityTreeDocument> {
    const response = (await this.transport.runTask(this.browserContext.sessionId, {
      operation: 'read',
    })) as any;
    const tree = response?.tree ?? response?.Tree ?? response?.observation?.tree;
    if (!tree?.root || typeof tree.root !== 'object')
      throw new Error('BAS did not return an accessibility tree');
    return { ...tree, url: response.url ?? tree.url };
  }

  async act(operation: 'click' | 'fill' | 'type', ref: string, value?: string): Promise<void> {
    if (!isRef(ref)) throw new Error('only BAS opaque accessibility refs are accepted');
    const payload: Record<string, unknown> = { operation, ref };
    if (value !== undefined) payload.value = value;
    await this.transport.runTask(this.browserContext.sessionId, payload);
  }
}

export class BrokerLocatorFacade {
  constructor(
    private readonly page: BrokerPageFacade,
    private readonly ref: string
  ) {}
  describe(): BrokerLocatorFacade {
    return this;
  }
  async normalize(): Promise<{ toString(): string; selectorForQuery(): string }> {
    if (!isRef(this.ref)) throw new Error('invalid BAS accessibility ref');
    return { toString: () => this.ref, selectorForQuery: () => `aria-ref=${this.ref}` };
  }
  async ariaSnapshot(): Promise<string> {
    const tree = await this.page.ariaSnapshot();
    const lines = tree.split('\n');
    const marker = `[ref=${this.ref}]`;
    const index = lines.findIndex((line) => line.includes(marker));
    return index < 0 ? '' : (lines[index] ?? '');
  }
  async ariaSnapshotJSON(): Promise<unknown> {
    return await this.page.ariaSnapshotJSON();
  }
  async click(options: Record<string, unknown> = {}): Promise<void> {
    assertSimpleClick(options);
    await this.page.act('click', this.ref);
  }
  async dblclick(): Promise<never> {
    throw new Error('double click is outside the BAS task contract');
  }
  async fill(value: string): Promise<void> {
    await this.page.act('fill', this.ref, value);
  }
  async pressSequentially(value: string): Promise<void> {
    await this.page.act('type', this.ref, value);
  }
  async press(): Promise<never> {
    throw new Error('key presses are outside the BAS task contract');
  }
  async check(): Promise<never> {
    throw new Error('checkbox actions are outside the BAS task contract');
  }
  async uncheck(): Promise<never> {
    throw new Error('checkbox actions are outside the BAS task contract');
  }
  async selectOption(): Promise<never> {
    throw new Error('selection actions are outside the BAS task contract');
  }
  async hover(): Promise<never> {
    throw new Error('hover actions are outside the BAS task contract');
  }
  async setInputFiles(): Promise<never> {
    throw new Error('file upload is outside the BAS task contract');
  }
  async evaluate(): Promise<never> {
    throw new Error('page script execution is not exposed by the BAS façade');
  }
}

function assertSimpleClick(options: Record<string, unknown>): void {
  if (options.button !== undefined && options.button !== 'left')
    throw new Error('only a left click is supported');
  if (
    options.modifiers !== undefined &&
    (!Array.isArray(options.modifiers) || options.modifiers.length)
  )
    throw new Error('modified clicks are outside the BAS task contract');
  if (options.force === true) throw new Error('forced clicks are outside the BAS task contract');
}

function resultString(value: unknown, keys: string[]): string {
  if (!value || typeof value !== 'object') return '';
  const source = value as Record<string, unknown>;
  for (const key of keys) if (typeof source[key] === 'string') return source[key] as string;
  return '';
}

function formatTree(node: AccessibilityNode | undefined, depth = 0): string {
  if (!node) return '';
  const lines: string[] = [];
  const visit = (current: AccessibilityNode, level: number) => {
    const indent = '  '.repeat(level);
    let line = `${indent}- ${current.role ?? 'generic'}`;
    if (current.name) line += ` "${current.name.replace(/["\\\n\r]/g, ' ')}"`;
    if (current.states?.length) line += ` ${current.states.map((state) => `[${state}]`).join(' ')}`;
    if (current.ref) line += ` [ref=${current.ref}]`;
    lines.push(line);
    for (const child of current.children ?? []) visit(child, level + 1);
  };
  visit(node, depth);
  return lines.join('\n');
}
