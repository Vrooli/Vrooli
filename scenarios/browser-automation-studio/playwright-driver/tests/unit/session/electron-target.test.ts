import type { Page } from 'rebrowser-playwright';
import {
  selectElectronPage,
  validateElectronTargetCapabilities,
  validateElectronTargetSpec,
  verifyElectronRenderer,
} from '../../../src/session/electron-target';

const target = {
  target_id: 'target-1',
  cdp_endpoint: 'http://127.0.0.1:43123',
  renderer_id: 'renderer-1',
  renderer_url: 'file:///controlled/index.html',
  renderer_title: 'Controlled Desktop',
  scenario_name: 'controlled-scenario',
  artifact_digest: 'sha256:controlled',
  context_id: 'ctx-1',
  cdp_transport: 'loopback-authenticated' as const,
};

describe('Electron target admission', () => {
  afterEach(() => jest.restoreAllMocks());

  it('rejects non-loopback endpoints and embedded credentials', () => {
    expect(() => validateElectronTargetSpec({ ...target, cdp_endpoint: 'http://example.test:9222' })).toThrow(
      'loopback'
    );
    expect(() => validateElectronTargetSpec({ ...target, cdp_endpoint: 'http://user:pass@127.0.0.1:9222' })).toThrow(
      'credentials'
    );
  });

  it('refuses evidence capabilities that require context creation control', () => {
    expect(() => validateElectronTargetCapabilities({ video: true })).toThrow('video');
    expect(() => validateElectronTargetCapabilities({ har: true, performance_trace: true })).toThrow('har');
    expect(() => validateElectronTargetCapabilities(undefined)).not.toThrow();
  });

  it('requires the target to carry scenario and artifact identity', () => {
    expect(() => validateElectronTargetSpec({ ...target, scenario_name: '' })).toThrow('scenario_name');
    expect(() => validateElectronTargetSpec({ ...target, artifact_digest: '' })).toThrow('artifact_digest');
  });

  it('requires the admitted renderer id and origin', async () => {
    const response = (): Response => new Response(
      JSON.stringify([{ id: 'renderer-1', type: 'page', url: target.renderer_url, title: target.renderer_title }]),
      { status: 200, headers: { 'content-type': 'application/json' } }
    );
    jest.spyOn(global, 'fetch').mockImplementation(() => Promise.resolve(response()));
    await expect(verifyElectronRenderer(target)).resolves.toBeUndefined();
    await expect(verifyElectronRenderer({ ...target, renderer_id: 'wrong' })).rejects.toThrow('missing');
    const transitioned = (): Response => new Response(
      JSON.stringify([{ id: 'renderer-1', type: 'page', url: 'file:///controlled/route', title: target.renderer_title }]),
      { status: 200, headers: { 'content-type': 'application/json' } }
    );
    jest.spyOn(global, 'fetch').mockImplementation(() => Promise.resolve(transitioned()));
    await expect(verifyElectronRenderer(target)).resolves.toBeUndefined();
  });

  it('does not treat a normal renderer title transition as target replacement', async () => {
    const response = (): Response => new Response(
      JSON.stringify([{ id: target.renderer_id, type: 'page', url: target.renderer_url, title: 'Hello Desktop' }]),
      { status: 200, headers: { 'content-type': 'application/json' } }
    );
    jest.spyOn(global, 'fetch').mockImplementation(() => Promise.resolve(response()));
    await expect(verifyElectronRenderer(target)).resolves.toBeUndefined();

    const page = ({ url: () => target.renderer_url, title: jest.fn().mockResolvedValue('Hello Desktop') }) as unknown as Page;
    await expect(selectElectronPage([page], target)).resolves.toBe(page);
  });

  it('refuses a stale or unavailable CDP endpoint', async () => {
    jest.spyOn(global, 'fetch').mockRejectedValue(new TypeError('fetch failed'));
    await expect(verifyElectronRenderer(target)).rejects.toThrow('fetch failed');
  });

  it('refuses ambiguous or missing Playwright page identity', async () => {
    const page = (url: string): Page => ({ url: () => url }) as unknown as Page;
    await expect(selectElectronPage([page(target.renderer_url), page(target.renderer_url)], target)).rejects.toThrow('ambiguous');
    await expect(selectElectronPage([page('file:///controlled/route')], target)).resolves.toBeTruthy();
    await expect(selectElectronPage([page('about:blank')], target)).rejects.toThrow('missing');
    const matchingPage = page(target.renderer_url);
    matchingPage.title = jest.fn().mockResolvedValue(target.renderer_title);
    await expect(selectElectronPage([matchingPage], target)).resolves.toBe(matchingPage);
  });
});
