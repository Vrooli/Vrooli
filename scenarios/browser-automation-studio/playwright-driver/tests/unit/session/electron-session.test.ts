import { playwrightProvider } from '../../../src/playwright';
import { SessionManager } from '../../../src/session/manager';
import { createMockBrowser, createMockContext, createMockPage, createTestConfig } from '../../helpers';

describe('Electron-backed SessionManager sessions', () => {
  afterEach(() => jest.restoreAllMocks());

  it('attaches to the exact renderer and detaches without closing its page/context', async () => {
    const page = createMockPage({
      url: jest.fn().mockReturnValue('file:///controlled/index.html'),
      title: jest.fn().mockResolvedValue('Controlled Desktop'),
    });
    const context = createMockContext({
      pages: jest.fn().mockReturnValue([page]),
      setExtraHTTPHeaders: jest.fn().mockResolvedValue(undefined),
    });
    const browser = createMockBrowser({ contexts: jest.fn().mockReturnValue([context]) });
    jest.spyOn(playwrightProvider.chromium, 'connectOverCDP').mockResolvedValue(browser);
    const response = (): Response => new Response(
      JSON.stringify([{ id: 'renderer-1', type: 'page', url: 'file:///controlled/index.html', title: 'Controlled Desktop' }]),
      { status: 200, headers: { 'content-type': 'application/json' } }
    );
    jest.spyOn(global, 'fetch').mockImplementation(() => Promise.resolve(response()));

    const manager = new SessionManager(createTestConfig());
    const result = await manager.startSession({
      execution_id: 'electron-exec',
      workflow_id: 'adhoc-index-uuid',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      browser_profile: { extra_headers: { 'X-Vrooli-Test-Mode': '1' } },
      app_target: {
        target_id: 'target-1',
        cdp_endpoint: 'http://127.0.0.1:43123',
        renderer_id: 'renderer-1',
        renderer_url: 'file:///controlled/index.html',
        renderer_title: 'Controlled Desktop',
        scenario_name: 'controlled-scenario',
        artifact_digest: 'sha256:controlled',
        context_id: 'ctx-1',
        cdp_transport: 'loopback-authenticated',
      },
      validation_context: {
        context_id: 'ctx-1',
        scenario_name: 'controlled-scenario',
        artifact_digest: 'sha256:controlled',
        target_id: 'target-1',
        workflow_id: 'secrets-manager:bas/cases/01-foundation/dashboard-smoke.json',
        profile_id: 'normal',
        isolation_lease_id: 'lease-1',
      },
    });

    expect(result.sessionId).toBeTruthy();
    expect(context.setExtraHTTPHeaders).toHaveBeenCalledWith({ 'X-Vrooli-Test-Mode': '1' });
    await manager.closeSession(result.sessionId);
    expect(browser.close).toHaveBeenCalledTimes(1);
    expect(context.close).not.toHaveBeenCalled();
    expect(page.close).not.toHaveBeenCalled();
    await manager.shutdown();
  });
});
