import { FrameHandler } from '../../../src/handlers/frame';
import { getDocument, type HandlerContext } from '../../../src/handlers/base';
import {
  createMockPage,
  createMockFrame,
  createMockContext,
  createTestConfig,
  createTypedInstruction,
} from '../../helpers';
import { logger, metrics } from '../../../src/utils';

describe('session-owned frame selection', () => {
  const handler = new FrameHandler();
  let context: HandlerContext;
  let main: ReturnType<typeof createMockFrame>;
  let left: ReturnType<typeof createMockFrame>;
  let right: ReturnType<typeof createMockFrame>;
  let nested: ReturnType<typeof createMockFrame>;
  beforeEach(() => {
    const page = createMockPage();
    main = createMockFrame({
      page: jest.fn().mockReturnValue(page),
      isDetached: jest.fn().mockReturnValue(false),
    });
    page.mainFrame.mockReturnValue(main);
    const child = (parent: typeof main, url: string) =>
      createMockFrame({
        page: jest.fn().mockReturnValue(page),
        parentFrame: jest.fn().mockReturnValue(parent),
        isDetached: jest.fn().mockReturnValue(false),
        url: jest.fn().mockReturnValue(url),
      });
    left = child(main, 'https://fixture/child');
    right = child(main, 'https://fixture/child');
    nested = child(left, 'https://fixture/nested');
    main.childFrames.mockReturnValue([left, right]);
    left.childFrames.mockReturnValue([nested]);
    context = {
      page,
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'frame-fixture',
      frameStack: [],
    };
  });
  const run = (params: Record<string, unknown>) =>
    handler.execute(createTypedInstruction('frame-switch', params), context);

  it('refuses ambiguous same-URL siblings without changing selection', async () => {
    expect((await run({ action: 'enter', frameUrl: '/child' })).error?.message).toMatch(
      /ambiguous/
    );
    expect(context.frameStack).toEqual([]);
  });
  it('enters the actual selected child, retaining its identity', async () => {
    main.childFrames.mockReturnValue([left]);
    expect((await run({ action: 'enter', frameUrl: '/child' })).success).toBe(true);
    expect(getDocument(context)).toBe(left);
    expect(context.frameStack).toEqual([left]);
  });
  it('reselects the same unambiguous frame without adding depth', async () => {
    context.frameStack = [left];
    const result = await run({ action: 'enter', frameUrl: '/child' });
    expect(result.success).toBe(true);
    expect(result.extracted_data).toMatchObject({ idempotent: true, stackDepth: 1 });
    expect(context.frameStack).toEqual([left]);
  });
  it('enters nested frames relative to the selected document', async () => {
    context.frameStack = [left];
    expect((await run({ action: 'enter', frameUrl: '/nested' })).success).toBe(true);
    expect(getDocument(context)).toBe(nested);
  });
  it('PARENT selects one ancestor and EXIT returns to the main document', async () => {
    context.frameStack = [left, nested];
    expect((await run({ action: 'parent' })).success).toBe(true);
    expect(getDocument(context)).toBe(left);
    context.frameStack.push(nested);
    expect((await run({ action: 'exit' })).success).toBe(true);
    expect(getDocument(context)).toBe(context.page);
    expect(context.frameStack).toEqual([]);
  });
  it.each(['exit', 'parent'])('refuses %s at the main document', async (action) => {
    expect((await run({ action })).error).toMatchObject({ code: 'NOT_IN_FRAME', retryable: false });
  });
  it('refuses detached selection until an explicit EXIT recovers it', async () => {
    context.frameStack = [left];
    left.isDetached.mockReturnValue(true);
    expect(() => getDocument(context)).toThrow(/detached/);
    expect((await run({ action: 'enter', frameUrl: '/nested' })).success).toBe(false);
    expect(context.frameStack).toEqual([left]);
    expect((await run({ action: 'exit' })).success).toBe(true);
    expect(context.frameStack).toEqual([]);
  });
  it('keeps the path intact when PARENT would select a detached ancestor', async () => {
    context.frameStack = [left, nested];
    left.isDetached.mockReturnValue(true);
    expect((await run({ action: 'parent' })).success).toBe(false);
    expect(context.frameStack).toEqual([left, nested]);
  });
  it('rejects frames from another page or broken ancestry', () => {
    context.frameStack = [nested];
    expect(() => getDocument(context)).toThrow();
    context.frameStack = [left];
    left.page.mockReturnValue(createMockPage());
    expect(() => getDocument(context)).toThrow();
  });
  it('reports a missing frame without changing state', async () => {
    expect((await run({ action: 'enter', frameUrl: '/missing' })).error?.code).toBe(
      'FRAME_NOT_FOUND'
    );
    expect(context.frameStack).toEqual([]);
  });
  it('requires a target and rejects unknown operations', async () => {
    expect((await run({ action: 'enter' })).error?.code).toBe('MISSING_PARAM');
    expect((await run({ action: 'invalid-action' })).success).toBe(false);
  });
  it('requires a session owner for frame navigation', async () => {
    delete context.frameStack;
    expect((await run({ action: 'enter', frameUrl: '/child' })).error?.message).toMatch(
      /session-owned/
    );
  });
});
