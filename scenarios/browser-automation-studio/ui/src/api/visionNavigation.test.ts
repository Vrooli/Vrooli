import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createRouterTransport } from '@connectrpc/connect';
import { createClient } from '@connectrpc/connect';
import { VisionNavigationService } from '@vrooli/proto-types/browser-automation-studio/v1/ai/ai_pb';

// We test the proto-Connect wiring in isolation: build a router-transport,
// register a fake handler, and assert that calls hit the expected method
// with the expected request shape. This guards the contract that
// useAINavigation depends on without spinning up the real server.

describe('visionNavigationClient (Connect-RPC)', () => {
  const listNavigators = vi.fn();
  const startNavigation = vi.fn();
  const abortNavigation = vi.fn();
  const resumeNavigation = vi.fn();
  const getNavigationStatus = vi.fn();

  beforeEach(() => {
    listNavigators.mockReset();
    startNavigation.mockReset();
    abortNavigation.mockReset();
    resumeNavigation.mockReset();
    getNavigationStatus.mockReset();
  });

  const buildClient = () => {
    const transport = createRouterTransport(({ service }) => {
      service(VisionNavigationService, {
        listNavigators,
        startNavigation,
        abortNavigation,
        resumeNavigation,
        getNavigationStatus,
      });
    });
    return createClient(VisionNavigationService, transport);
  };

  it('sends StartNavigation with proto-shaped request and returns the response', async () => {
    startNavigation.mockResolvedValue({
      navigationId: 'nav-1',
      status: 'started',
      model: 'local_first',
      maxSteps: 10,
      navigatorType: 'playwright',
    });
    const client = buildClient();

    const resp = await client.startNavigation({
      sessionId: 's-1',
      prompt: 'click login',
      model: 'local_first',
      maxSteps: 10,
    });

    expect(resp.navigationId).toBe('nav-1');
    expect(resp.navigatorType).toBe('playwright');
    expect(startNavigation).toHaveBeenCalledTimes(1);
    const req = startNavigation.mock.calls[0][0];
    expect(req.sessionId).toBe('s-1');
    expect(req.prompt).toBe('click login');
    expect(req.maxSteps).toBe(10);
  });

  it('sends AbortNavigation with the navigation_id', async () => {
    abortNavigation.mockResolvedValue({
      navigationId: 'nav-1',
      status: 'aborting',
      message: 'Abort signal sent.',
    });
    const client = buildClient();

    const resp = await client.abortNavigation({ navigationId: 'nav-1' });
    expect(resp.status).toBe('aborting');
    expect(abortNavigation.mock.calls[0][0].navigationId).toBe('nav-1');
  });

  it('sends ResumeNavigation with the navigation_id', async () => {
    resumeNavigation.mockResolvedValue({
      navigationId: 'nav-1',
      status: 'resumed',
      message: 'Navigation resumed.',
    });
    const client = buildClient();

    const resp = await client.resumeNavigation({ navigationId: 'nav-1' });
    expect(resp.status).toBe('resumed');
    expect(resumeNavigation.mock.calls[0][0].navigationId).toBe('nav-1');
  });

  it('decodes ListNavigators responses including nested credit policy', async () => {
    listNavigators.mockResolvedValue({
      navigators: [
        {
          type: 'playwright',
          available: true,
          description: 'Playwright vision',
          creditPolicy: {
            requiresCredits: true,
            creditsPerStep: 2,
            bypassConditions: ['byok'],
          },
          allowedSources: ['ui', 'cli', 'api'],
          unavailableReason: '',
        },
      ],
      default: 'playwright',
    });
    const client = buildClient();
    const resp = await client.listNavigators({ clientSource: 'ui' });
    expect(resp.default).toBe('playwright');
    expect(resp.navigators).toHaveLength(1);
    expect(resp.navigators[0].creditPolicy?.creditsPerStep).toBe(2);
  });
});
