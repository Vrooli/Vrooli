import { handleRecordInput, handleRecordViewport } from '../../../src/routes/record-mode/recording-input';
import { createMockHttpRequest, createMockHttpResponse, createMockPage, createTestConfig } from '../../helpers';
import type { SessionManager } from '../../../src/session';

jest.mock('../../../src/frame-streaming', () => ({
  updateFrameStreamViewport: jest.fn().mockResolvedValue({ success: true }),
}));

describe('recording input routes', () => {
  const config = createTestConfig();
  let mockPage: ReturnType<typeof createMockPage>;
  let sessionManager: Pick<SessionManager, 'getSession'>;

  beforeEach(() => {
    mockPage = createMockPage({
      mouse: {
        move: jest.fn().mockResolvedValue(undefined),
        down: jest.fn().mockResolvedValue(undefined),
        up: jest.fn().mockResolvedValue(undefined),
        click: jest.fn().mockResolvedValue(undefined),
        wheel: jest.fn().mockResolvedValue(undefined),
      } as unknown as ReturnType<typeof createMockPage>['mouse'],
      keyboard: {
        type: jest.fn().mockResolvedValue(undefined),
        press: jest.fn().mockResolvedValue(undefined),
      } as unknown as ReturnType<typeof createMockPage>['keyboard'],
      setViewportSize: jest.fn().mockResolvedValue(undefined),
      viewportSize: jest.fn().mockReturnValue({ width: 800, height: 600 }),
    });
    sessionManager = {
      getSession: () => ({ page: mockPage } as ReturnType<SessionManager['getSession']>),
    };
  });

  it('handles pointer move input', async () => {
    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/input',
      body: { type: 'pointer', action: 'move', x: 10, y: 20 },
    });
    const res = createMockHttpResponse();

    await handleRecordInput(req, res, 'test', sessionManager as SessionManager, config);

    expect(mockPage.mouse.move).toHaveBeenCalledWith(10, 20);
    expect(res.statusCode).toBe(200);
  });

  it('returns 400 for invalid pointer action', async () => {
    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/input',
      body: { type: 'pointer', action: 'tap', x: 1, y: 2 },
    });
    const res = createMockHttpResponse();

    await handleRecordInput(req, res, 'test', sessionManager as SessionManager, config);

    expect(res.statusCode).toBe(400);
    expect(res.getJSON().error).toBe('INVALID_ACTION');
  });

  it('handles keyboard text input', async () => {
    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/input',
      body: { type: 'keyboard', text: 'hello' },
    });
    const res = createMockHttpResponse();

    await handleRecordInput(req, res, 'test', sessionManager as SessionManager, config);

    expect(mockPage.keyboard.type).toHaveBeenCalledWith('hello');
    expect(res.statusCode).toBe(200);
  });

  it('handles keyboard key input with modifiers', async () => {
    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/input',
      body: { type: 'keyboard', key: 'Enter', modifiers: ['Shift', 'Alt'] },
    });
    const res = createMockHttpResponse();

    await handleRecordInput(req, res, 'test', sessionManager as SessionManager, config);

    expect(mockPage.keyboard.press).toHaveBeenCalledWith('Shift+Alt+Enter');
    expect(res.statusCode).toBe(200);
  });

  it('returns 400 when keyboard input lacks key and text', async () => {
    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/input',
      body: { type: 'keyboard' },
    });
    const res = createMockHttpResponse();

    await handleRecordInput(req, res, 'test', sessionManager as SessionManager, config);

    expect(res.statusCode).toBe(400);
    expect(res.getJSON().error).toBe('MISSING_KEYBOARD_DATA');
  });

  it('handles wheel input', async () => {
    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/input',
      body: { type: 'wheel', delta_x: 5, delta_y: -3 },
    });
    const res = createMockHttpResponse();

    await handleRecordInput(req, res, 'test', sessionManager as SessionManager, config);

    expect(mockPage.mouse.wheel).toHaveBeenCalledWith(5, -3);
    expect(res.statusCode).toBe(200);
  });

  it('updates viewport and responds with current size', async () => {
    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/viewport',
      body: { width: 800.4, height: 600.6 },
    });
    const res = createMockHttpResponse();

    await handleRecordViewport(req, res, 'test', sessionManager as SessionManager, config);

    expect(mockPage.setViewportSize).toHaveBeenCalledWith({ width: 800, height: 601 });
    expect(res.statusCode).toBe(200);
    expect(res.getJSON()).toEqual({ session_id: 'test', width: 800, height: 600 });
  });

  it('rejects invalid viewport sizes', async () => {
    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/viewport',
      body: { width: 0, height: -10 },
    });
    const res = createMockHttpResponse();

    await handleRecordViewport(req, res, 'test', sessionManager as SessionManager, config);

    expect(res.statusCode).toBe(400);
    expect(res.getJSON().error).toBe('INVALID_VIEWPORT');
  });
});
