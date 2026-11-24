import { handleSessionAccessibilityTree } from '../../../src/routes/session-accessibility-tree';
import type { SessionManager } from '../../../src/session';
import type { SessionState } from '../../../src/types/session';
import type { AccessibilityTreeObservation } from '../../../src/tracing/accessibility-observation';

const tree: AccessibilityTreeObservation = {
  contract: 'bas-accessibility-tree-observation/v1',
  url: 'https://allowed.example',
  viewport: { width: 800, height: 600, deviceScaleFactor: 1 },
  captured_at: '2026-11-19T00:00:00.000Z',
  node_count: 1,
  truncated: false,
  root: { role: 'RootWebArea', name: 'Current page', children: [] },
};

function fixture(options: { url?: string; view?: boolean; allow?: string[] } = {}) {
  const session = {
    id: 'session-1',
    ownerExecutionId: 'owner-1',
    leaseId: 'lease-1',
    createdAt: new Date(),
    instructionCount: 0,
    phase: 'ready',
    instructionInFlight: false,
    spec: {
      permission_profile: {
        effect: 'observe',
        approval: 'none',
        view: options.view ?? true,
        url_allow: options.allow ?? ['https://allowed.example/*'],
      },
    },
    page: { url: () => options.url ?? 'https://allowed.example/current', isClosed: () => false },
    pageBindings: { getId: () => 'page-1' },
  } as unknown as SessionState;
  const manager = {
    getSessionForLease: (sessionId: string, executionId: string, leaseId: string) => {
      if (sessionId !== 'session-1' || executionId !== 'owner-1' || leaseId !== 'lease-1')
        throw new Error('not found');
      return session;
    },
  } as unknown as SessionManager;
  const headers: Record<string, string> = {
    'x-bas-execution-id': 'owner-1',
    'x-bas-session-lease-id': 'lease-1',
  };
  const req = { headers, url: '/session/session-1/accessibility-tree?expected_page_id=page-1' };
  const res = {
    statusCode: 200,
    headers: {} as Record<string, string>,
    body: '',
    setHeader(name: string, value: string) {
      this.headers[name] = value;
    },
    end(body: string) {
      this.body = body;
    },
  };
  return { req, res, manager, headers, session };
}

describe('handleSessionAccessibilityTree', () => {
  test.each([
    ['wrong caller', 'x-bas-execution-id', 'other-owner'],
    ['wrong lease', 'x-bas-session-lease-id', 'other-lease'],
  ])('rejects %s without exposing the session', async (_name, header, value) => {
    const f = fixture();
    f.headers[header] = value;
    let observed = false;
    await handleSessionAccessibilityTree(
      f.req as never,
      f.res as never,
      'session-1',
      f.manager,
      async () => {
        observed = true;
        return tree;
      }
    );
    expect(f.res.statusCode).toBe(404);
    expect(f.res.body).toContain('session not found');
    expect(observed).toBe(false);
  });

  test('denies when View permission is absent', async () => {
    const f = fixture({ view: false });
    let observed = false;
    await handleSessionAccessibilityTree(
      f.req as never,
      f.res as never,
      'session-1',
      f.manager,
      async () => {
        observed = true;
        return tree;
      }
    );
    expect(f.res.statusCode).toBe(403);
    expect(observed).toBe(false);
  });

  test('denies an out-of-scope current URL', async () => {
    const f = fixture({ url: 'https://outside.example/private' });
    let observed = false;
    await handleSessionAccessibilityTree(
      f.req as never,
      f.res as never,
      'session-1',
      f.manager,
      async () => {
        observed = true;
        return tree;
      }
    );
    expect(f.res.statusCode).toBe(403);
    expect(observed).toBe(false);
  });

  test('returns the current tree for the exact active page and lease', async () => {
    const f = fixture();
    const referencedTree: AccessibilityTreeObservation = {
      ...tree,
      root: { role: 'RootWebArea', name: 'Current page', children: [{ role: 'button', name: 'Continue', ref: 'e123' , children: [] }] },
    };
    Object.defineProperty(referencedTree, 'agentReferenceTargets', {
      value: new Map([['e123', { selector: 'button[data-testid="continue"]', role: 'button', name: 'Continue' }]]),
    });
    await handleSessionAccessibilityTree(
      f.req as never,
      f.res as never,
      'session-1',
      f.manager,
      async () => referencedTree
    );
    expect(f.res.statusCode).toBe(200);
    const body = JSON.parse(f.res.body);
    expect(body.tree.root.children[0].ref).toBe('e123');
    expect(f.res.body).not.toContain('data-testid');
    expect(f.session.accessibilityReferenceTargets?.get('e123')).toMatchObject({ pageId: 'page-1', selector: 'button[data-testid="continue"]' });
    expect(f.session.accessibilityReferenceTargets?.get('e123')?.expiresAt).toBeGreaterThan(Date.now());
  });
});
