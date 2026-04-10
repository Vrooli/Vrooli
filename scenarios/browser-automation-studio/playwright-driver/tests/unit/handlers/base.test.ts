import { BaseHandler } from '../../../src/handlers/base';
import { createMockPage, createMockContext, createTestConfig } from '../../helpers';
import type { HandlerContext } from '../../../src/handlers/base';

class TestHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return [];
  }

  execute(): Promise<{ success: boolean }> {
    return Promise.resolve({ success: true });
  }

  public waitForElementPublic(page: Parameters<BaseHandler['waitForElement']>[0], selector: string, timeout?: number) {
    return this.waitForElement(page, selector, timeout);
  }

  public getBoundingBoxPublic(page: Parameters<BaseHandler['getBoundingBox']>[0], selector: string) {
    return this.getBoundingBox(page, selector);
  }

  public extractTextPublic(page: Parameters<BaseHandler['extractText']>[0], selector?: string) {
    return this.extractText(page, selector);
  }

  public extractAttributePublic(page: Parameters<BaseHandler['extractAttribute']>[0], selector: string, attribute: string) {
    return this.extractAttribute(page, selector, attribute);
  }

  public requireTypedParamsPublic<T>(params: T | undefined, handlerType: string, nodeId: string): T {
    return this.requireTypedParams(params, handlerType, nodeId);
  }

  public missingParamErrorPublic(handlerType: string, paramName: string) {
    return this.missingParamError(handlerType, paramName);
  }
}

describe('BaseHandler utilities', () => {
  let handler: TestHandler;
  let context: HandlerContext;

  beforeEach(() => {
    handler = new TestHandler();
    context = {
      page: createMockPage(),
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger: { info: jest.fn(), warn: jest.fn(), error: jest.fn(), debug: jest.fn() } as never,
      metrics: { instructionErrors: { inc: jest.fn() } } as never,
      sessionId: 'session-1',
    };
  });

  it('waitForElement delegates to page.waitForSelector with visible state', async () => {
    await handler.waitForElementPublic(context.page, '#selector', 1234);
    expect(context.page.waitForSelector).toHaveBeenCalledWith('#selector', {
      timeout: 1234,
      state: 'visible',
    });
  });

  it('getBoundingBox returns normalized box when available', async () => {
    const box = await handler.getBoundingBoxPublic(context.page, '#target');
    expect(box).toEqual({ x: 0, y: 0, width: 100, height: 50 });
  });

  it('getBoundingBox returns null when element has no box', async () => {
    const locator = context.page.locator('#target');
    locator.first().boundingBox = jest.fn().mockResolvedValue(null);
    const box = await handler.getBoundingBoxPublic(context.page, '#target');
    expect(box).toBeNull();
  });

  it('extractText uses selector when provided', async () => {
    const text = await handler.extractTextPublic(context.page, '#target');
    expect(text).toBe('test text');
  });

  it('extractText falls back to body when selector missing', async () => {
    context.page.textContent = jest.fn().mockResolvedValue('body text');
    const text = await handler.extractTextPublic(context.page);
    expect(context.page.textContent).toHaveBeenCalledWith('body');
    expect(text).toBe('body text');
  });

  it('extractAttribute delegates to locator.getAttribute', async () => {
    context.page.locator = jest.fn().mockReturnValue({
      first: jest.fn().mockReturnValue({
        getAttribute: jest.fn().mockResolvedValue('test-value'),
      }),
    });

    const value = await handler.extractAttributePublic(context.page, '#target', 'data-test');
    expect(value).toBe('test-value');
  });

  it('requireTypedParams throws with clear message when missing', () => {
    expect(() => handler.requireTypedParamsPublic(undefined, 'click', 'node-1')).toThrow(
      '[click] Missing typed action params for node node-1.'
    );
  });

  it('missingParamError returns structured error result', () => {
    const result = handler.missingParamErrorPublic('click', 'selector');
    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('MISSING_PARAM');
    expect(result.error?.message).toContain('click instruction missing selector parameter');
  });
});
