/* eslint-disable @typescript-eslint/no-var-requires -- The exact upstream package is loaded from its temporary admitted install so it stays out of the BAS dependency graph. */
/* eslint-disable @typescript-eslint/no-explicit-any -- Tests exercise the intentionally untyped optional MCP package private-handler seam. */
/* eslint-disable @typescript-eslint/require-await -- Fake Broker methods mirror the async production transport contract. */
/* eslint-disable @typescript-eslint/no-unnecessary-type-assertion -- The exact optional package boundary is resolved dynamically at runtime. */

import {
  BASBrowserTaskHTTPTransport,
  createAgentManagerTaskTransport,
  createBrokerBackedPlaywrightMcp,
  type AuthorizedPost,
  type BrokerTaskTransport,
} from '../../../src/agentaccess/playwright-mcp-adapter';
import * as fs from 'node:fs';
import * as path from 'node:path';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';

const vector = JSON.parse(
  fs.readFileSync(
    path.resolve(__dirname, '../../../../testdata/harness-conformance/v1.json'),
    'utf8'
  )
) as {
  session_id: string;
  owner_id: string;
  grant_id: string;
  steps: Array<{
    tool: string;
    arguments: Record<string, unknown>;
    task: Record<string, unknown>;
    permission_class: string;
    effect_class: string;
  }>;
  expected_final: { url: string; button: string; input: string };
  expected_final_observation: { url: string; button: string; textbox: string };
  expected_trace: Array<{
    sequence: number;
    session_id: string;
    owner_id: string;
    grant_id: string;
    operation: string;
    permission_class: string;
    effect_class: string;
    effect_applied: boolean;
  }>;
  expected_revocation: {
    after_step: number;
    denied_step_index: number;
    denied_operation: string;
    effects_after_revocation: number;
    trace: Array<{
      sequence: number;
      session_id: string;
      owner_id: string;
      grant_id: string;
      operation: string;
      permission_class: string;
      effect_class: string;
      effect_applied: boolean;
    }>;
  };
  expected_scope_denial: {
    url: string;
    trace: Array<{
      sequence: number;
      session_id: string;
      owner_id: string;
      grant_id: string;
      operation: string;
      permission_class: string;
      effect_class: string;
      effect_applied: boolean;
    }>;
  };
  rejected_calls: Array<{
    id: string;
    playwright_mcp_tool: string;
    playwright_mcp_arguments: Record<string, unknown>;
    reason: string;
  }>;
};

const pinnedEntrypoint = process.env.MB5_PLAYWRIGHT_MCP_ENTRYPOINT;
const describePinned = pinnedEntrypoint ? describe : describe.skip;

describe('BAS Agent Access task HTTP transport', () => {
  test('exposes bounded keyboard actions through the Broker task route', async () => {
    const calls: Array<{ sessionId: string; task: Record<string, unknown> }> = [];
    const transport: BrokerTaskTransport = {
      runTask: async (sessionId, task) => {
        calls.push({ sessionId, task });
        return { accepted: true };
      },
    };
    const createConnection = async (
      _config: Record<string, unknown>,
      getContext: () => Promise<unknown>
    ) => {
      const context = (await getContext()) as {
        page: { keyboard: { press(key: string): Promise<void> } };
      };
      const handlers = new Map<string, (...args: any[]) => Promise<any>>();
      handlers.set('tools/list', async () => ({ tools: [{ name: 'browser_press_key' }] }));
      handlers.set('tools/call', async (request) => {
        await context.page.keyboard.press(request.params.arguments.key);
        return { content: [{ type: 'text', text: 'accepted' }] };
      });
      return { _requestHandlers: handlers, close: async () => undefined };
    };
    const { server } = await createBrokerBackedPlaywrightMcp({
      sessionId: 'session-1',
      transport,
      createConnection,
    });
    try {
      const list = await server._requestHandlers.get('tools/list')!({});
      expect(list.tools.map((tool: { name: string }) => tool.name)).toContain('browser_press_key');
      const call = server._requestHandlers.get('tools/call')!;
      const accepted = await call({
        params: { name: 'browser_press_key', arguments: { key: 'Control+Enter' } },
      });
      expect(accepted.isError).not.toBe(true);
      expect(calls).toEqual([
        {
          sessionId: 'session-1',
          task: { operation: 'keyboard', key: 'Enter', modifiers: ['ctrl'] },
        },
      ]);
      const denied = await call({
        params: { name: 'browser_press_key', arguments: { key: 'Alt+F4' } },
      });
      expect(denied.isError).toBe(true);
      expect(calls).toHaveLength(1);
    } finally {
      await server.close();
    }
  });

  test('requires the same exact BAS origin contract as Browser Use', () => {
    for (const apiURL of [
      'https://user:secret@bas.example',
      'https://bas.example/api',
      'https://bas.example/?token=secret',
      'https://bas.example/#session',
    ]) {
      expect(
        () =>
          new BASBrowserTaskHTTPTransport(apiURL, 'session-1', async () => ({
            status: 200,
            json: async () => ({}),
          }))
      ).toThrow(/HTTP\(S\) origin/);
    }
  });

  test('uses the existing authenticated task route without serializing authority material', async () => {
    let url = '';
    let body: Record<string, unknown> = {};
    const transport = new BASBrowserTaskHTTPTransport(
      'https://bas.example/',
      'session/1',
      async (requestURL, requestBody) => {
        url = requestURL;
        body = requestBody;
        return { status: 200, json: async () => ({ success: true }) };
      }
    );
    await expect(
      transport.runTask('session/1', { operation: 'click', ref: 'e123' })
    ).resolves.toEqual({ success: true });
    expect(url).toBe('https://bas.example/api/v1/browser/sessions/session%2F1/task');
    expect(body).toEqual({ operation: 'click', ref: 'e123' });
    expect(body).not.toHaveProperty('identity_token');
    expect(body).not.toHaveProperty('grant_receipt');
    await expect(transport.runTask('other-session', { operation: 'read' })).rejects.toThrow(
      /bound to a different session/
    );
  });

  test('bounds production Broker response streaming before JSON parsing', async () => {
    let cancelled = false;
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(new Uint8Array((2 << 20) + 1));
      },
      cancel() {
        cancelled = true;
      },
    });
    const json = jest.fn(async () => ({ oversized: true }));
    const transport = new BASBrowserTaskHTTPTransport(
      'https://bas.example',
      'session-1',
      async () => ({ status: 200, json, body })
    );
    await expect(transport.runTask('session-1', { operation: 'read' })).rejects.toThrow(
      'BAS response exceeds the size limit'
    );
    expect(json).not.toHaveBeenCalled();
    expect(cancelled).toBe(true);
  });

  test('sends draft compilation to the existing session owner without project authority', async () => {
    let url = '';
    let body: Record<string, unknown> = {};
    const transport = new BASBrowserTaskHTTPTransport(
      'https://bas.example',
      'session/1',
      async (requestURL, requestBody) => {
        url = requestURL;
        body = requestBody;
        return {
          status: 200,
          json: async () => ({ draft_digest: 'sha256:fixture', workflow_id: '' }),
        };
      }
    );
    const request = {
      name: 'Account flow',
      assertions: [
        {
          type: 'ACTION_TYPE_ASSERT',
          assert: { selector: '#done', mode: 'ASSERTION_MODE_VISIBLE' },
        },
      ],
    };
    await expect(transport.draftTrace('session/1', request)).resolves.toEqual({
      draft_digest: 'sha256:fixture',
      workflow_id: '',
    });
    expect(url).toBe('https://bas.example/api/v1/browser/sessions/session%2F1/trace/draft');
    expect(body).toEqual(request);
    expect(body).not.toHaveProperty('project_id');
    expect(body).not.toHaveProperty('grant_receipt');
    await expect(transport.draftTrace('other-session', request)).rejects.toThrow(
      /bound to a different session/
    );
    await expect(
      transport.draftTrace('session/1', { ...request, project_id: 'forged' } as any)
    ).rejects.toThrow(/fixed typed contract/);
  });

  test('rejects task operations and caller authority fields before authenticated transport', async () => {
    let calls = 0;
    const transport = new BASBrowserTaskHTTPTransport(
      'https://bas.example',
      'session-1',
      async () => {
        calls += 1;
        return { status: 200, json: async () => ({ accepted: true }) };
      }
    );
    for (const task of [
      { operation: 'evaluate', code: 'document.cookie' },
      { operation: 'click', ref: 'e123', grant_receipt: 'forged' },
      { operation: 'click', ref: '#continue' },
      { operation: 'fill', ref: 'e123' },
      { operation: 'navigate', url: ' ' },
    ]) {
      await expect(transport.runTask('session-1', task)).rejects.toThrow(
        /fixed typed adapter contract/
      );
    }
    expect(calls).toBe(0);
  });

  test('builds the production task binding from Agent Manager run environment and revalidates through BAS', async () => {
    const envKeys = [
      'BAS_BROWSER_API_URL',
      'VROOLI_AGENT_IDENTITY_TOKEN',
      'VROOLI_BROWSER_GRANT_RECEIPT',
    ] as const;
    const previous = envKeys.map((key) => process.env[key]);
    const originalFetch = global.fetch;
    process.env.BAS_BROWSER_API_URL = 'https://bas.example/';
    process.env.VROOLI_AGENT_IDENTITY_TOKEN = 'am-signed-run-token';
    process.env.VROOLI_BROWSER_GRANT_RECEIPT = 'browser-grant-1';
    const requestURLs: string[] = [];
    const requestInits: RequestInit[] = [];
    global.fetch = (async (input: URL | RequestInfo, init?: RequestInit) => {
      requestURLs.push(String(input));
      requestInits.push(init ?? {});
      return new Response(JSON.stringify({ accepted: true }), { status: 200 });
    }) as typeof global.fetch;
    try {
      const transport = createAgentManagerTaskTransport('session-1');
      await expect(transport.runTask('session-1', { operation: 'read' })).resolves.toEqual({
        accepted: true,
      });
      const draft = {
        name: 'Account flow',
        assertions: [
          {
            type: 'ACTION_TYPE_ASSERT',
            assert: { selector: '#done', mode: 'ASSERTION_MODE_VISIBLE' },
          },
        ],
      };
      await expect(transport.draftTrace('session-1', draft)).resolves.toEqual({ accepted: true });
      expect(requestURLs).toEqual([
        'https://bas.example/api/v1/browser/sessions/session-1/task',
        'https://bas.example/api/v1/browser/sessions/session-1/trace/draft',
      ]);
      expect(requestInits.map((init) => init.headers)).toEqual([
        expect.objectContaining({
          'X-Agent-Identity-Token': 'am-signed-run-token',
          'X-Vrooli-Browser-Grant-Receipt': 'browser-grant-1',
        }),
        expect.objectContaining({
          'X-Agent-Identity-Token': 'am-signed-run-token',
          'X-Vrooli-Browser-Grant-Receipt': 'browser-grant-1',
        }),
      ]);
      expect(requestInits.map((init) => init.redirect)).toEqual(['manual', 'manual']);
      expect(requestInits[0]?.body).toBe('{"operation":"read"}');
      expect(requestInits[1]?.body).toBe(JSON.stringify(draft));
      expect(requestInits[1]?.body).not.toContain('project_id');
      expect(requestInits[1]?.body).not.toContain('am-signed-run-token');
      expect(requestInits[1]?.body).not.toContain('browser-grant-1');
    } finally {
      global.fetch = originalFetch;
      envKeys.forEach((key, index) => {
        const value = previous[index];
        if (value === undefined) delete process.env[key];
        else process.env[key] = value;
      });
    }
  });

  test('does not follow a BAS task redirect with Agent Manager credentials', async () => {
    const envKeys = [
      'BAS_BROWSER_API_URL',
      'VROOLI_AGENT_IDENTITY_TOKEN',
      'VROOLI_BROWSER_GRANT_RECEIPT',
    ] as const;
    const previous = envKeys.map((key) => process.env[key]);
    const originalFetch = global.fetch;
    process.env.BAS_BROWSER_API_URL = 'https://bas.example';
    process.env.VROOLI_AGENT_IDENTITY_TOKEN = 'am-signed-run-token';
    process.env.VROOLI_BROWSER_GRANT_RECEIPT = 'browser-grant-1';
    let calls = 0;
    let requestInit: RequestInit | undefined;
    global.fetch = (async (_input: URL | RequestInfo, init?: RequestInit) => {
      calls += 1;
      requestInit = init;
      return new Response(JSON.stringify({ error: { code: 'BAS_REDIRECT_DENIED' } }), {
        status: 307,
        headers: { location: 'https://redirect-target.example/collect' },
      });
    }) as typeof global.fetch;
    try {
      const transport = createAgentManagerTaskTransport('session-1');
      await expect(transport.runTask('session-1', { operation: 'read' })).rejects.toThrow(
        /BAS_REDIRECT_DENIED/
      );
      expect(requestInit?.redirect).toBe('manual');
      expect(calls).toBe(1);
    } finally {
      global.fetch = originalFetch;
      envKeys.forEach((key, index) => {
        const value = previous[index];
        if (value === undefined) delete process.env[key];
        else process.env[key] = value;
      });
    }
  });

  test('fails closed when Agent Manager run authority is absent', () => {
    const apiURL = process.env.BAS_BROWSER_API_URL;
    const token = process.env.VROOLI_AGENT_IDENTITY_TOKEN;
    const grant = process.env.VROOLI_BROWSER_GRANT_RECEIPT;
    process.env.BAS_BROWSER_API_URL = 'https://bas.example';
    delete process.env.VROOLI_AGENT_IDENTITY_TOKEN;
    delete process.env.VROOLI_BROWSER_GRANT_RECEIPT;
    try {
      expect(() => createAgentManagerTaskTransport('session-1')).toThrow(
        /identity and browser grant/
      );
    } finally {
      if (apiURL !== undefined) process.env.BAS_BROWSER_API_URL = apiURL;
      else delete process.env.BAS_BROWSER_API_URL;
      if (token !== undefined) process.env.VROOLI_AGENT_IDENTITY_TOKEN = token;
      if (grant !== undefined) process.env.VROOLI_BROWSER_GRANT_RECEIPT = grant;
    }
  });
});

describe('shared v1 Broker task vector through Playwright MCP façade', () => {
  test('runs Browser Use then Playwright MCP through one shared BAS HTTP session state', async () => {
    const state = {
      url: 'https://allowed.example/start',
      clicked: false,
      value: '',
      trace: [] as Array<Record<string, unknown>>,
      sessionIDs: new Set<string>(),
      routes: [] as string[],
    };
    const server = createServer(async (request, response) => {
      state.routes.push(request.url ?? '');
      if (
        request.method !== 'POST' ||
        request.url !== `/api/v1/browser/sessions/${vector.session_id}/task`
      ) {
        response.writeHead(404).end();
        return;
      }
      if (
        request.headers['x-agent-identity-token'] !== 'shared-test-identity' ||
        request.headers['x-vrooli-browser-grant-receipt'] !== vector.grant_id
      ) {
        response
          .writeHead(401, { 'content-type': 'application/json' })
          .end(JSON.stringify({ error: { code: 'AM_CONTEXT_DENIED' } }));
        return;
      }
      let raw = '';
      for await (const chunk of request) raw += chunk.toString();
      const task = JSON.parse(raw) as Record<string, unknown>;
      const stepIndex = state.trace.length;
      const step = vector.steps[stepIndex];
      if (!step) {
        response
          .writeHead(409, { 'content-type': 'application/json' })
          .end(JSON.stringify({ error: { code: 'UNEXPECTED_EXTRA_TASK' } }));
        return;
      }
      expect(task).toEqual(step.task);
      state.sessionIDs.add(vector.session_id);
      const operation = String(task.operation);
      const entry: Record<string, unknown> = {
        sequence: stepIndex + 1,
        session_id: vector.session_id,
        owner_id: vector.owner_id,
        grant_id: vector.grant_id,
        operation,
        permission_class: step.permission_class,
        effect_class: step.effect_class,
        effect_applied: false,
      };
      state.trace.push(entry);
      const json = (value: unknown, status = 200) => {
        response.writeHead(status, { 'content-type': 'application/json' });
        response.end(JSON.stringify(value));
      };
      if (operation === 'read') {
        json({
          url: state.url,
          tree: {
            root: {
              role: 'RootWebArea',
              name: 'Shared HTTP fixture',
              children: [
                { role: 'button', name: state.clicked ? 'Opened' : 'Continue', ref: 'e123' },
                { role: 'textbox', name: 'Search', value: state.value, ref: 'e456' },
              ],
            },
          },
        });
        return;
      }
      if (operation === 'click' && task.ref === 'e123') {
        state.clicked = true;
        entry.effect_applied = true;
        json({ success: true });
        return;
      }
      if (operation === 'fill' && task.ref === 'e456' && typeof task.value === 'string') {
        state.value = task.value;
        entry.effect_applied = true;
        json({ success: true });
        return;
      }
      if (operation === 'navigate' && task.url === vector.expected_final.url) {
        state.url = String(task.url);
        entry.effect_applied = true;
        json({ url: state.url });
        return;
      }
      json({ error: { code: 'BROWSER_TASK_DENIED' } }, 403);
    });

    await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
    const address = server.address();
    if (!address || typeof address === 'string')
      throw new Error('shared BAS test server has no TCP address');
    const apiOrigin = `http://127.0.0.1:${address.port}`;
    const apiRoot = path.resolve(__dirname, '../../../../api');
    try {
      const pythonOutput = await new Promise<string>((resolve, reject) => {
        const child = spawn(
          'python3',
          ['-m', 'automation.harness.cross_language_browser_use_prefix'],
          {
            cwd: apiRoot,
            env: {
              PATH: process.env.PATH,
              BAS_BROWSER_API_URL: apiOrigin,
              VROOLI_AGENT_IDENTITY_TOKEN: 'shared-test-identity',
              VROOLI_BROWSER_GRANT_RECEIPT: vector.grant_id,
            },
          }
        );
        let stdout = '';
        let stderr = '';
        child.stdout.setEncoding('utf8').on('data', (chunk: string) => (stdout += chunk));
        child.stderr.setEncoding('utf8').on('data', (chunk: string) => (stderr += chunk));
        child.on('error', reject);
        child.on('close', (code) => {
          if (code === 0) resolve(stdout);
          else reject(new Error(`Browser Use conformance child exited ${code}: ${stderr}`));
        });
      });
      expect(JSON.parse(pythonOutput)).toEqual({
        status: 'completed',
        tools: [
          'browser_click',
          'browser_draft_trace',
          'browser_navigate',
          'browser_press_key',
          'browser_receipt',
          'browser_snapshot',
          'browser_type',
          'finish',
        ],
      });

      const authorizedPost: AuthorizedPost = async (url, body) => {
        const result = await fetch(url, {
          method: 'POST',
          headers: {
            'content-type': 'application/json',
            'x-agent-identity-token': 'shared-test-identity',
            'x-vrooli-browser-grant-receipt': vector.grant_id,
          },
          body: JSON.stringify(body),
          redirect: 'manual',
        });
        return { status: result.status, json: () => result.json() };
      };
      const transport = new BASBrowserTaskHTTPTransport(
        apiOrigin,
        vector.session_id,
        authorizedPost
      );
      const createConnection = async (
        _config: Record<string, unknown>,
        getContext: () => Promise<unknown>
      ) => {
        const context = (await getContext()) as any;
        const handlers = new Map<string, (...args: any[]) => Promise<any>>();
        handlers.set('tools/list', async () => ({
          tools: vector.steps.map((item) => ({ name: item.tool })),
        }));
        handlers.set('tools/call', async (call) => {
          const { name, arguments: args } = call.params;
          if (name === 'browser_navigate') await context.page.goto(args.url);
          else if (name === 'browser_snapshot')
            return { content: [{ type: 'text', text: await context.page.ariaSnapshot() }] };
          else throw new Error(`unexpected Playwright MCP continuation tool ${name}`);
          return { content: [{ type: 'text', text: 'ok' }] };
        });
        return { _requestHandlers: handlers, close: async () => undefined };
      };
      const { server: mcpServer, context } = await createBrokerBackedPlaywrightMcp({
        sessionId: vector.session_id,
        transport,
        createConnection,
      });
      try {
        const call = mcpServer._requestHandlers.get('tools/call')!;
        let finalSnapshot = '';
        for (const step of vector.steps.slice(3)) {
          const result = await call({ params: { name: step.tool, arguments: step.arguments } });
          expect(result.isError).not.toBe(true);
          if (step.task.operation === 'read') finalSnapshot = result.content[0].text;
        }
        expect({
          url: context.page.url(),
          button: finalSnapshot.match(/- button "([^"]+)"/)?.[1],
          textbox: finalSnapshot.match(/- textbox "([^"]+)"/)?.[1],
        }).toEqual(vector.expected_final_observation);
        expect({
          clicked: state.clicked,
          value: state.value,
        }).toEqual({ clicked: true, value: vector.expected_final.input });
        expect(state.trace).toEqual(vector.expected_trace);
        expect([...state.sessionIDs]).toEqual([vector.session_id]);
        expect(state.routes).toEqual(
          vector.steps.map(() => `/api/v1/browser/sessions/${vector.session_id}/task`)
        );
        expect(() => context.newPage()).toThrow(/owns browser lifecycle/);
        expect(() => context.close()).toThrow(/owns browser lifecycle/);
      } finally {
        await mcpServer.close();
      }
    } finally {
      await new Promise<void>((resolve, reject) =>
        server.close((error) => (error ? reject(error) : resolve()))
      );
    }
  });

  test('publishes an unsaved trace draft tool bound to this existing session', async () => {
    const calls: Array<{ sessionId: string; request: Record<string, unknown> }> = [];
    const transport: BrokerTaskTransport = {
      async runTask() {
        return {};
      },
      async draftTrace(sessionId, request) {
        calls.push({ sessionId, request });
        return { draft_digest: 'sha256:fixture', session_id: sessionId, workflow_id: '' };
      },
    };
    const createConnection = async (
      _config: Record<string, unknown>,
      getContext: () => Promise<unknown>
    ) => {
      await getContext();
      const handlers = new Map<string, (...args: any[]) => Promise<any>>();
      handlers.set('tools/list', async () => ({
        tools: [{ name: 'browser_snapshot' }, { name: 'browser_draft_trace' }],
      }));
      handlers.set('tools/call', async () => ({ content: [] }));
      return { _requestHandlers: handlers, close: async () => undefined };
    };
    const { server } = await createBrokerBackedPlaywrightMcp({
      sessionId: vector.session_id,
      transport,
      createConnection,
    });
    try {
      const tools = await server._requestHandlers.get('tools/list')!(
        { method: 'tools/list', params: {} },
        {}
      );
      expect(tools.tools.map((tool: { name: string }) => tool.name)).toContain(
        'browser_draft_trace'
      );
      const call = server._requestHandlers.get('tools/call')!;
      const draftRequest = {
        name: 'Account flow',
        assertions: [
          {
            type: 'ACTION_TYPE_ASSERT',
            assert: { selector: '#done', mode: 'ASSERTION_MODE_VISIBLE' },
          },
        ],
      };
      const result = await call(
        { params: { name: 'browser_draft_trace', arguments: draftRequest } },
        {}
      );
      expect(result.isError).not.toBe(true);
      expect(JSON.parse(result.content[0].text)).toEqual({
        draft_digest: 'sha256:fixture',
        session_id: vector.session_id,
        workflow_id: '',
      });
      expect(calls).toEqual([{ sessionId: vector.session_id, request: draftRequest }]);
      const forged = await call(
        {
          params: {
            name: 'browser_draft_trace',
            arguments: { ...draftRequest, project_id: 'other-project' },
          },
        },
        {}
      );
      expect(forged.isError).toBe(true);
      expect(calls).toHaveLength(1);
    } finally {
      await server.close();
    }
  });

  test('exposes session receipt via the same transport and rejects caller-selected account or reservation', async () => {
    const calls: Array<{ sessionId: string; task: Record<string, unknown> }> = [];
    const transport: BrokerTaskTransport = {
      async runTask(sessionId, task) {
        calls.push({ sessionId, task });
        return {
          settlement_status: 'settled',
          receipt: { reservation_id: 'lpbs-1', status: 'settled', browser_session_minutes: 2 },
        };
      },
    };
    const createConnection = async (
      _config: Record<string, unknown>,
      getContext: () => Promise<unknown>
    ) => {
      await getContext();
      const handlers = new Map<string, (...args: any[]) => Promise<any>>();
      handlers.set('tools/list', async () => ({ tools: [] }));
      handlers.set('tools/call', async () => ({ content: [] }));
      return { _requestHandlers: handlers, close: async () => undefined };
    };
    const { server } = await createBrokerBackedPlaywrightMcp({
      sessionId: vector.session_id,
      transport,
      createConnection,
    });
    try {
      const listed = await server._requestHandlers.get('tools/list')!(
        { method: 'tools/list', params: {} },
        {}
      );
      expect(listed.tools.map((tool: { name: string }) => tool.name)).toEqual([
        'browser_receipt',
        'browser_draft_trace',
      ]);
      const handler = server._requestHandlers.get('tools/call')!;
      const receipt = await handler({ params: { name: 'browser_receipt', arguments: {} } }, {});
      expect(receipt).toEqual(
        expect.objectContaining({
          content: [expect.objectContaining({ text: expect.stringContaining('lpbs-1') })],
        })
      );
      expect(calls).toHaveLength(1);
      expect(JSON.parse(receipt.content[0].text)).toEqual({
        settlement_status: 'settled',
        receipt: { reservation_id: 'lpbs-1', status: 'settled', browser_session_minutes: 2 },
      });
      expect(calls).toEqual([{ sessionId: vector.session_id, task: { operation: 'receipt' } }]);
      const forged = await handler(
        { params: { name: 'browser_receipt', arguments: { business_account_id: 'attacker' } } },
        {}
      );
      expect(forged.isError).toBe(true);
      expect(calls).toHaveLength(1);
    } finally {
      await server.close();
    }
  });

  test('rejects shared malformed-ref and caller-script cases before Broker transport', async () => {
    const calls: Array<{ sessionId: string; task: Record<string, unknown> }> = [];
    const transport: BrokerTaskTransport = {
      async runTask(sessionId, task) {
        calls.push({ sessionId, task });
        return { success: true };
      },
    };
    const createConnection = async (
      _config: Record<string, unknown>,
      getContext: () => Promise<unknown>
    ) => {
      const context = (await getContext()) as any;
      const handlers = new Map<string, (...args: any[]) => Promise<any>>();
      handlers.set('tools/list', async () => ({
        tools: vector.rejected_calls.map((item) => ({ name: item.playwright_mcp_tool })),
      }));
      handlers.set('tools/call', async (request) => {
        const { name, arguments: args } = request.params;
        if (name === 'browser_click') await context.page.getByRef(args.target).click();
        return { content: [{ type: 'text', text: 'ok' }] };
      });
      return { _requestHandlers: handlers, close: async () => undefined };
    };
    const { server } = await createBrokerBackedPlaywrightMcp({
      sessionId: vector.session_id,
      transport,
      createConnection,
    });
    try {
      const handler = server._requestHandlers.get('tools/call')!;
      for (const rejected of vector.rejected_calls) {
        const result = await handler({
          params: {
            name: rejected.playwright_mcp_tool,
            arguments: rejected.playwright_mcp_arguments,
          },
        });
        expect(result.isError).toBe(true);
        expect(calls).toEqual([]);
      }
    } finally {
      await server.close();
    }
  });

  test('preserves ordered operations, permission/effect classes, trace and one-session binding', async () => {
    const state = {
      url: 'https://allowed.example/start',
      clicked: false,
      value: '',
      trace: [] as Array<Record<string, unknown>>,
    };
    const transport: BrokerTaskTransport = {
      async runTask(sessionId, task) {
        const operation = String(task.operation);
        const entry: Record<string, unknown> = {
          sequence: state.trace.length + 1,
          session: sessionId,
          owner: vector.owner_id,
          grant: vector.grant_id,
          operation,
          permissionClass: { read: 'observe', click: 'act', fill: 'act', navigate: 'navigate' }[
            operation as 'read' | 'click' | 'fill' | 'navigate'
          ],
          effectClass: { read: 'none', click: 'ui', fill: 'ui', navigate: 'navigation' }[
            operation as 'read' | 'click' | 'fill' | 'navigate'
          ],
          effectApplied: false,
        };
        state.trace.push(entry);
        if (sessionId !== vector.session_id) throw new Error('session mismatch');
        if (operation === 'read')
          return {
            url: state.url,
            tree: {
              root: {
                role: 'RootWebArea',
                name: 'Broker fixture',
                children: [
                  { role: 'button', name: state.clicked ? 'Opened' : 'Continue', ref: 'e123' },
                  { role: 'textbox', name: 'Search', value: state.value, ref: 'e456' },
                ],
              },
            },
          };
        if (operation === 'click' && task.ref === 'e123') {
          state.clicked = true;
          entry.effectApplied = true;
          return { success: true };
        }
        if (operation === 'fill' && task.ref === 'e456') {
          state.value = String(task.value);
          entry.effectApplied = true;
          return { success: true };
        }
        if (operation === 'navigate' && task.url === vector.expected_final.url) {
          state.url = String(task.url);
          entry.effectApplied = true;
          return { url: state.url };
        }
        throw new Error('unsupported vector task');
      },
    };
    const createConnection = async (
      _config: Record<string, unknown>,
      getContext: () => Promise<unknown>
    ) => {
      const context = (await getContext()) as any;
      const handlers = new Map<string, (...args: any[]) => Promise<any>>();
      handlers.set('tools/list', async () => ({
        tools: vector.steps.map((step) => ({ name: step.tool })),
      }));
      handlers.set('tools/call', async (request) => {
        const { name, arguments: args } = request.params;
        if (name === 'browser_snapshot')
          return { content: [{ type: 'text', text: await context.page.ariaSnapshot() }] };
        if (name === 'browser_click') {
          try {
            await context.page.getByRef(args.target).click();
          } catch (error) {
            return { isError: true, content: [{ type: 'text', text: String(error) }] };
          }
        } else if (name === 'browser_type')
          await context.page.getByRef(args.target).fill(args.text);
        else if (name === 'browser_navigate') await context.page.goto(args.url);
        return { content: [{ type: 'text', text: 'ok' }] };
      });
      return { _requestHandlers: handlers, close: async () => undefined };
    };
    const { server, context } = await createBrokerBackedPlaywrightMcp({
      sessionId: vector.session_id,
      transport,
      createConnection,
    });
    try {
      const handler = server._requestHandlers.get('tools/call')!;
      let finalObservation = '';
      for (const step of vector.steps) {
        const result = await handler({ params: { name: step.tool, arguments: step.arguments } });
        expect(result.isError).not.toBe(true);
        if (step.task.operation === 'read') finalObservation = JSON.stringify(result);
      }
      expect(state.url).toBe(vector.expected_final.url);
      expect(state.clicked).toBe(true);
      expect(state.value).toBe(vector.expected_final.input);
      expect(finalObservation).toContain(vector.expected_final.button);
      const snapshotText = (JSON.parse(finalObservation) as { content: Array<{ text: string }> })
        .content[0]!.text;
      expect({
        url: context.page.url(),
        button: snapshotText.match(/- button "([^"]+)"/)?.[1],
        textbox: snapshotText.match(/- textbox "([^"]+)"/)?.[1],
      }).toEqual(vector.expected_final_observation);
      expect(state.trace.map(({ operation }) => operation)).toEqual(
        vector.steps.map((step) => step.task.operation)
      );
      expect(
        state.trace.map((row) => ({
          sequence: row.sequence,
          session_id: row.session,
          owner_id: row.owner,
          grant_id: row.grant,
          operation: row.operation,
          permission_class: row.permissionClass,
          effect_class: row.effectClass,
          effect_applied: row.effectApplied,
        }))
      ).toEqual(vector.expected_trace);
      expect(() => context.newPage()).toThrow(/owns browser lifecycle/);
      expect(() => context.close()).toThrow(/owns browser lifecycle/);
    } finally {
      await server.close();
    }
  });

  test('uses the vector revocation point and records a denied click without an effect', async () => {
    let active = true;
    const trace: Array<{ operation: string; effectApplied: boolean }> = [];
    const transport: BrokerTaskTransport = {
      async runTask(_sessionId, task) {
        const operation = String(task.operation);
        const entry = { operation, effectApplied: false };
        trace.push(entry);
        if (!active) throw new Error('BROWSER_PERMISSION_DENIED: grant revoked');
        if (operation === 'read')
          return {
            tree: {
              root: {
                role: 'RootWebArea',
                children: [{ role: 'button', name: 'Continue', ref: 'e123' }],
              },
            },
          };
        if (operation === 'click' && task.ref === 'e123') {
          entry.effectApplied = true;
          return { success: true };
        }
        throw new Error('unsupported task');
      },
    };
    const createConnection = async (
      _config: Record<string, unknown>,
      getContext: () => Promise<unknown>
    ) => {
      const context = (await getContext()) as any;
      const handlers = new Map<string, (...args: any[]) => Promise<any>>();
      handlers.set('tools/list', async () => ({
        tools: vector.steps.map((step) => ({ name: step.tool })),
      }));
      handlers.set('tools/call', async (request) => {
        const { name, arguments: args } = request.params;
        if (name === 'browser_snapshot')
          return { content: [{ type: 'text', text: await context.page.ariaSnapshot() }] };
        if (name === 'browser_click') {
          try {
            await context.page.getByRef(args.target).click();
          } catch (error) {
            return { isError: true, content: [{ type: 'text', text: String(error) }] };
          }
        }
        return { content: [{ type: 'text', text: 'ok' }] };
      });
      return { _requestHandlers: handlers, close: async () => undefined };
    };
    const { server } = await createBrokerBackedPlaywrightMcp({
      sessionId: vector.session_id,
      transport,
      createConnection,
    });
    try {
      const handler = server._requestHandlers.get('tools/call')!;
      const first = vector.steps[0]!;
      const denied = vector.steps[vector.expected_revocation.denied_step_index]!;
      expect(
        (await handler({ params: { name: first.tool, arguments: first.arguments } })).isError
      ).not.toBe(true);
      active = false;
      expect(
        (await handler({ params: { name: denied.tool, arguments: denied.arguments } })).isError
      ).toBe(true);
      expect(trace.map((item) => item.operation)).toEqual(['read', 'click']);
      expect(trace[1]?.effectApplied).toBe(false);
      expect(
        trace.map((item, index) => ({
          sequence: index + 1,
          session_id: vector.session_id,
          owner_id: vector.owner_id,
          grant_id: vector.grant_id,
          operation: item.operation,
          permission_class: item.operation === 'read' ? 'observe' : 'act',
          effect_class: item.operation === 'read' ? 'none' : 'ui',
          effect_applied: item.effectApplied,
        }))
      ).toEqual(vector.expected_revocation.trace);
      expect(trace.slice(1).filter((item) => item.effectApplied)).toHaveLength(
        vector.expected_revocation.effects_after_revocation
      );
    } finally {
      await server.close();
    }
  });

  test('routes out-of-scope navigation to the existing owner and records no effect', async () => {
    const trace: Array<{
      sequence: number;
      session: string;
      owner: string;
      grant: string;
      operation: string;
      permissionClass: string;
      effectClass: string;
      effectApplied: boolean;
    }> = [];
    const transport: BrokerTaskTransport = {
      async runTask(sessionId, task) {
        trace.push({
          sequence: trace.length + 1,
          session: sessionId,
          owner: vector.owner_id,
          grant: vector.grant_id,
          operation: String(task.operation),
          permissionClass: 'navigate',
          effectClass: 'navigation',
          effectApplied: false,
        });
        if (new URL(String(task.url)).origin !== 'https://allowed.example')
          throw new Error('BROWSER_PERMISSION_DENIED: URL is outside the permitted scope');
        trace[trace.length - 1]!.effectApplied = true;
        return { url: task.url };
      },
    };
    const createConnection = async (
      _config: Record<string, unknown>,
      getContext: () => Promise<unknown>
    ) => {
      const context = (await getContext()) as any;
      const handlers = new Map<string, (...args: any[]) => Promise<any>>();
      handlers.set('tools/list', async () => ({ tools: [{ name: 'browser_navigate' }] }));
      handlers.set('tools/call', async (request) => {
        try {
          await context.page.goto(request.params.arguments.url);
          return { content: [{ type: 'text', text: 'ok' }] };
        } catch (error) {
          return { isError: true, content: [{ type: 'text', text: String(error) }] };
        }
      });
      return { _requestHandlers: handlers, close: async () => undefined };
    };
    const { server } = await createBrokerBackedPlaywrightMcp({
      sessionId: vector.session_id,
      transport,
      createConnection,
    });
    try {
      const callTool = server._requestHandlers.get('tools/call')!;
      const result = await callTool({
        params: { name: 'browser_navigate', arguments: { url: vector.expected_scope_denial.url } },
      });
      expect(result.isError).toBe(true);
      expect(
        trace.map((row) => ({
          sequence: row.sequence,
          session_id: row.session,
          owner_id: row.owner,
          grant_id: row.grant,
          operation: row.operation,
          permission_class: row.permissionClass,
          effect_class: row.effectClass,
          effect_applied: row.effectApplied,
        }))
      ).toEqual(vector.expected_scope_denial.trace);
    } finally {
      await server.close();
    }
  });
});

describePinned('@playwright/mcp 0.0.85 Broker façade through official entrypoint', () => {
  const allowedTools = [
    'browser_click',
    'browser_draft_trace',
    'browser_navigate',
    'browser_press_key',
    'browser_receipt',
    'browser_snapshot',
    'browser_type',
  ];

  function setup() {
    // Test invocation supplies the path from the separately admitted exact-pin
    // installation. The package remains out of the BAS dependency graph.
    if (!pinnedEntrypoint) throw new Error('exact pinned Playwright MCP entrypoint is required');
    const official = require(pinnedEntrypoint) as {
      createConnection: (config: any, contextGetter: any) => Promise<any>;
    };
    const state = {
      owner: 'agent-1',
      active: true,
      inScope: true,
      url: 'https://allowed.example/start',
      pageEpoch: 0,
      refsExpired: false,
      buttonClicked: false,
      inputValue: '',
      trace: [] as Array<{
        sequence: number;
        session: string;
        owner: string;
        grant: string;
        operation: string;
        permissionClass: string;
        effectClass: string;
        effectApplied: boolean;
      }>,
      calls: [] as Array<{ session: string; operation: string; ref?: string }>,
    };
    const transport: BrokerTaskTransport = {
      async runTask(sessionId, task) {
        state.calls.push({
          session: sessionId,
          operation: String(task.operation),
          ref: task.ref as string | undefined,
        });
        const permissionClass =
          { read: 'observe', click: 'act', fill: 'act', navigate: 'navigate' }[
            String(task.operation)
          ] ?? 'unknown';
        const traceEntry = {
          sequence: state.trace.length + 1,
          session: sessionId,
          owner: state.owner,
          grant: vector.grant_id,
          operation: String(task.operation),
          permissionClass,
          effectClass:
            { read: 'none', click: 'ui', fill: 'ui', navigate: 'navigation' }[
              String(task.operation)
            ] ?? 'unknown',
          effectApplied: false,
        };
        state.trace.push(traceEntry);
        if (sessionId !== 'session-1' || state.owner !== 'agent-1' || !state.active)
          throw new Error('BROWSER_PERMISSION_DENIED: session owner revoked');
        if (task.operation === 'receipt')
          return {
            settlement_status: 'settled',
            receipt: { reservation_id: 'hold-1', status: 'settled', browser_session_minutes: 2 },
          };
        if (!state.inScope && task.operation !== 'read')
          throw new Error('BROWSER_PERMISSION_DENIED: URL is outside the permitted scope');
        if (task.operation === 'read') {
          const buttonRef = state.pageEpoch === 0 ? 'e123' : 'e223';
          const inputRef = state.pageEpoch === 0 ? 'e456' : 'e256';
          return {
            url: state.url,
            tree: {
              url: new URL(state.url).origin,
              root: {
                role: 'RootWebArea',
                name: 'Broker fixture',
                children: [
                  {
                    role: 'button',
                    name: state.buttonClicked ? 'Opened' : 'Continue',
                    ref: buttonRef,
                    children: [],
                  },
                  { role: 'textbox', name: 'Search', ref: inputRef, children: [] },
                ],
              },
            },
          };
        }
        if (task.operation === 'navigate') {
          const target = new URL(String(task.url));
          if (target.origin !== 'https://allowed.example')
            throw new Error('BROWSER_PERMISSION_DENIED: URL is outside the permitted scope');
          state.url = target.toString();
          state.pageEpoch += 1;
          traceEntry.effectApplied = true;
          return { url: state.url };
        }
        const buttonRef = state.pageEpoch === 0 ? 'e123' : 'e223';
        const inputRef = state.pageEpoch === 0 ? 'e456' : 'e256';
        if (state.refsExpired) throw new Error('ACCESSIBILITY_REFERENCE_STALE: reference expired');
        if (![buttonRef, inputRef].includes(String(task.ref)))
          throw new Error('ACCESSIBILITY_REFERENCE_STALE');
        if (task.ref === buttonRef && task.operation === 'click') state.buttonClicked = true;
        if (task.ref === inputRef && ['fill', 'type'].includes(String(task.operation)))
          state.inputValue = String(task.value ?? '');
        traceEntry.effectApplied = true;
        return { url: state.url, success: true };
      },
      async draftTrace(sessionId, request) {
        state.calls.push({ session: sessionId, operation: 'trace/draft' });
        return {
          draft_digest: 'sha256:fixture',
          session_id: sessionId,
          workflow_id: '',
          name: request.name,
        };
      },
    };
    return { official, state, transport };
  }

  async function call(server: any, name: string, args: Record<string, unknown> = {}) {
    const handler = server._requestHandlers.get('tools/call');
    return handler(
      { method: 'tools/call', params: { name, arguments: args } },
      { signal: new AbortController().signal }
    );
  }

  function unrefTestTimers(): () => void {
    const original = global.setTimeout;
    global.setTimeout = ((...args: any[]) => {
      const timer = original(...(args as Parameters<typeof setTimeout>));
      (timer as NodeJS.Timeout).unref?.();
      return timer;
    }) as typeof global.setTimeout;
    return () => {
      global.setTimeout = original;
    };
  }

  test('consumes the shared v1 task vector through the official MCP entrypoint on one session', async () => {
    const restoreTimers = unrefTestTimers();
    const { official, state, transport } = setup();
    const { server, context } = await createBrokerBackedPlaywrightMcp({
      sessionId: 'session-1',
      transport,
      createConnection: official.createConnection,
    });
    try {
      const listed = await server._requestHandlers.get('tools/list')(
        { method: 'tools/list', params: {} },
        {}
      );
      expect(listed.tools.map((tool: { name: string }) => tool.name).sort()).toEqual(allowedTools);
      expect(listed.tools.map((tool: { name: string }) => tool.name)).not.toContain(
        'browser_evaluate'
      );
      expect(listed.tools.map((tool: { name: string }) => tool.name)).not.toContain(
        'browser_close'
      );

      let finalObservation = '';
      for (const step of vector.steps) {
        const result = await call(server, step.tool, step.arguments);
        expect(result.isError).not.toBe(true);
        if (step.task.operation === 'read') finalObservation = JSON.stringify(result);
      }
      expect(state.buttonClicked).toBe(true);
      expect(state.inputValue).toBe(vector.expected_final.input);
      expect(state.url).toBe(vector.expected_final.url);
      expect(finalObservation).toContain(vector.expected_final.button);
      const snapshotText = (JSON.parse(finalObservation) as { content: Array<{ text: string }> })
        .content[0]!.text;
      expect({
        url: context.page.url(),
        button: snapshotText.match(/- button "([^"]+)"/)?.[1],
        textbox: snapshotText.match(/- textbox "([^"]+)"/)?.[1],
      }).toEqual(vector.expected_final_observation);
      expect(new Set(state.calls.map((entry) => entry.session))).toEqual(
        new Set([vector.session_id])
      );
      expect(
        state.calls.map(({ operation, ref }) => ({ operation, ...(ref ? { ref } : {}) }))
      ).toEqual(
        vector.steps.map((step) => ({
          operation: step.task.operation,
          ...('ref' in step.task ? { ref: step.task.ref } : {}),
        }))
      );
      expect(
        state.trace.map((row) => ({
          sequence: row.sequence,
          session_id: row.session,
          owner_id: row.owner,
          grant_id: row.grant,
          operation: row.operation,
          permission_class: row.permissionClass,
          effect_class: row.effectClass,
          effect_applied: row.effectApplied,
        }))
      ).toEqual(vector.expected_trace);

      expect(context.pages()).toHaveLength(1);
      expect(context.browser()).toBeUndefined();
      expect(() => context.newPage()).toThrow(/owns browser lifecycle/);
      expect(() => context.close()).toThrow(/owns browser lifecycle/);
      expect(() => context.page.evaluate('document.cookie')).toThrow(
        /script execution is not exposed/
      );
      expect(() => context.page.locator('button#continue')).toThrow(/opaque accessibility refs/);
      expect(() => context.page.close()).toThrow(/owns browser lifecycle/);
    } finally {
      await server.close();
      restoreTimers();
    }
  }, 30000);

  test.each([
    ['evaluate', 'browser_evaluate', { code: 'document.cookie' }],
    ['close lifecycle', 'browser_close', {}],
    ['arbitrary selector', 'browser_click', { target: 'button#continue' }],
    ['snapshot file write', 'browser_snapshot', { filename: '/tmp/leak.md' }],
    ['click keyboard modifier', 'browser_click', { target: 'e123', modifiers: ['Control'] }],
    ['type submit side effect', 'browser_type', { target: 'e456', text: 'x', submit: true }],
    ['script URL', 'browser_navigate', { url: 'javascript:document.cookie' }],
  ])('denies %s before any Broker effect', async (_label, name, args) => {
    const { official, state, transport } = setup();
    const { server } = await createBrokerBackedPlaywrightMcp({
      sessionId: 'session-1',
      transport,
      createConnection: official.createConnection,
    });
    try {
      const before = state.calls.length;
      const result = await call(server, name, args);
      expect(result.isError).toBe(true);
      expect(state.calls).toHaveLength(before);
    } finally {
      await server.close();
    }
  });

  test('exposes the exact settled receipt through the authenticated session transport only', async () => {
    const { official, state, transport } = setup();
    const { server, context } = await createBrokerBackedPlaywrightMcp({
      sessionId: 'session-1',
      transport,
      createConnection: official.createConnection,
    });
    try {
      const receipt = await call(server, 'browser_receipt');
      expect(JSON.parse(receipt.content[0].text)).toEqual({
        settlement_status: 'settled',
        receipt: { reservation_id: 'hold-1', status: 'settled', browser_session_minutes: 2 },
      });
      expect(state.calls).toEqual([{ session: 'session-1', operation: 'receipt' }]);
      const forged = await call(server, 'browser_receipt', {
        business_account_id: 'attacker',
        reservation_id: 'other',
      });
      expect(forged.isError).toBe(true);
      expect(await context.settlementReceipt()).toEqual({
        settlement_status: 'settled',
        receipt: { reservation_id: 'hold-1', status: 'settled', browser_session_minutes: 2 },
      });
    } finally {
      await server.close();
    }
  });

  test('exposes an unsaved draft through the same authenticated session adapter', async () => {
    const { official, state, transport } = setup();
    const { server } = await createBrokerBackedPlaywrightMcp({
      sessionId: 'session-1',
      transport,
      createConnection: official.createConnection,
    });
    try {
      const draftRequest = {
        name: 'Account flow',
        assertions: [
          {
            type: 'ACTION_TYPE_ASSERT',
            assert: { selector: '#done', mode: 'ASSERTION_MODE_VISIBLE' },
          },
        ],
      };
      const draft = await call(server, 'browser_draft_trace', draftRequest);
      expect(draft.isError).not.toBe(true);
      expect(JSON.parse(draft.content[0].text)).toEqual({
        draft_digest: 'sha256:fixture',
        session_id: 'session-1',
        workflow_id: '',
        name: 'Account flow',
      });
      expect(state.calls).toEqual([{ session: 'session-1', operation: 'trace/draft' }]);
      const forged = await call(server, 'browser_draft_trace', {
        ...draftRequest,
        project_id: 'forged',
      });
      expect(forged.isError).toBe(true);
      expect(state.calls).toHaveLength(1);
    } finally {
      await server.close();
    }
  });

  test.each([
    [
      'owner revoked',
      (state: ReturnType<typeof setup>['state']) => {
        state.active = false;
      },
    ],
    [
      'scope lost',
      (state: ReturnType<typeof setup>['state']) => {
        state.inScope = false;
      },
    ],
  ])('rechecks Broker authority on each official tool call after %s', async (_label, revoke) => {
    const restoreTimers = unrefTestTimers();
    const { official, state, transport } = setup();
    const { server } = await createBrokerBackedPlaywrightMcp({
      sessionId: 'session-1',
      transport,
      createConnection: official.createConnection,
    });
    try {
      const initialStep = vector.steps[0]!;
      const deniedStep = vector.steps[vector.expected_revocation.denied_step_index]!;
      const initial = await call(server, initialStep.tool, initialStep.arguments);
      expect(initial.isError).not.toBe(true);
      const actionCount = state.calls.filter((entry) => entry.operation === 'click').length;
      revoke(state);
      const click = await call(server, deniedStep.tool, deniedStep.arguments);
      expect(click.isError).toBe(true);
      expect(state.calls.filter((entry) => entry.operation === 'click')).toHaveLength(
        actionCount + 1
      );
      expect(state.buttonClicked).toBe(false);
      expect(state.trace.at(-1)?.effectApplied).toBe(false);
    } finally {
      await server.close();
      restoreTimers();
    }
  });

  test('navigation rotates page refs; old refs and out-of-scope navigation are denied', async () => {
    const restoreTimers = unrefTestTimers();
    const { official, state, transport } = setup();
    const { server } = await createBrokerBackedPlaywrightMcp({
      sessionId: 'session-1',
      transport,
      createConnection: official.createConnection,
    });
    try {
      const initial = await call(server, 'browser_snapshot');
      expect(JSON.stringify(initial)).toContain('[ref=e123]');
      const navigated = await call(server, 'browser_navigate', {
        url: 'https://allowed.example/next',
      });
      expect(navigated.isError).not.toBe(true);
      const current = await call(server, 'browser_snapshot');
      expect(JSON.stringify(current)).toContain('[ref=e223]');
      const oldRef = await call(server, 'browser_click', { target: 'e123' });
      expect(oldRef.isError).toBe(true);
      expect(state.buttonClicked).toBe(false);
      state.inScope = false;
      const deniedNavigation = await call(server, 'browser_navigate', {
        url: 'https://outside.example/',
      });
      expect(deniedNavigation.isError).toBe(true);
      expect(state.url).toBe('https://allowed.example/next');
    } finally {
      await server.close();
      restoreTimers();
    }
  });

  test('rejects an expired ref before clicking', async () => {
    const restoreTimers = unrefTestTimers();
    const { official, state, transport } = setup();
    const { server } = await createBrokerBackedPlaywrightMcp({
      sessionId: 'session-1',
      transport,
      createConnection: official.createConnection,
    });
    try {
      const initial = await call(server, 'browser_snapshot');
      expect(JSON.stringify(initial)).toContain('[ref=e123]');
      state.refsExpired = true;
      const click = await call(server, 'browser_click', { target: 'e123' });
      expect(click.isError).toBe(true);
      expect(state.buttonClicked).toBe(false);
    } finally {
      await server.close();
      restoreTimers();
    }
  });
});
