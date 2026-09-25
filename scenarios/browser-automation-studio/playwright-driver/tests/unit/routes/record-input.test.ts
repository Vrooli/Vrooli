import { handleRecordInput, handleRecordViewport } from '../../../src/routes/record-mode/recording-input';
import { createMockHttpRequest, createMockHttpResponse, createMockPage, createTestConfig } from '../../helpers';
import type { SessionManager } from '../../../src/session';
import { updateFrameStreamViewport } from '../../../src/frame-streaming';
import { SessionNotFoundError } from '../../../src/utils';

jest.mock('../../../src/frame-streaming', () => ({
  updateFrameStreamViewport: jest.fn().mockResolvedValue({ success: true }),
}));

describe('recording input routes', () => {
  const config = createTestConfig();
  let mockPage: ReturnType<typeof createMockPage>;
  let sessionManager: Pick<SessionManager, 'getSession' | 'getSessionForLease' | 'updateActivity'>;

  beforeEach(() => {
    jest.clearAllMocks();
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
        down: jest.fn().mockResolvedValue(undefined),
        up: jest.fn().mockResolvedValue(undefined),
      } as unknown as ReturnType<typeof createMockPage>['keyboard'],
      setViewportSize: jest.fn().mockResolvedValue(undefined),
      viewportSize: jest.fn().mockReturnValue({ width: 800, height: 600 }),
    });
    const session = { phase: 'ready', ownerExecutionId: 'owner', leaseId: 'lease', page: mockPage, pageToIdMap: new WeakMap([[mockPage, 'selected-page']]) } as ReturnType<SessionManager['getSession']>;
    sessionManager = {
      getSession: (): ReturnType<SessionManager['getSession']> => session,
      getSessionForLease: (id: string, owner: string, lease: string): ReturnType<SessionManager['getSession']> => {
        if (owner !== session.ownerExecutionId || lease !== session.leaseId) throw new SessionNotFoundError(id);
        return session;
      },
      updateActivity: jest.fn(),
    };
  });

  it.each(['missing', 'stale', 'released', 'current'])('admits only the current caller lease before input effects: %s [REQ:BAS-RH-J17]', async (envelope) => {
    const session = { phase: 'ready', ownerExecutionId: 'owner', leaseId: 'lease', page: mockPage };
    const manager = {
      getSession: jest.fn(() => session),
      getSessionForLease: (_id: string, owner: string, lease: string) => {
        if (owner !== session.ownerExecutionId || lease !== session.leaseId || envelope === 'released') throw new SessionNotFoundError('input');
        return session;
      },
      updateActivity: jest.fn(),
    } as unknown as SessionManager;
    const lease = envelope === 'missing' ? {} : envelope === 'stale'
      ? { execution_id: 'old-owner', lease_id: 'old-lease' } : { execution_id: 'owner', lease_id: 'lease' };
    const res = createMockHttpResponse();
    await handleRecordInput(createMockHttpRequest({ method: 'POST', body: { ...lease, type: 'pointer', action: 'click', x: 12, y: 34 } }), res, 'input', manager, config);
    expect(res.statusCode).toBe(envelope === 'current' ? 200 : envelope === 'missing' ? 400 : 404);
    expect(mockPage.mouse.click).toHaveBeenCalledTimes(envelope === 'current' ? 1 : 0);
    if (envelope !== 'current') {
      expect(manager.getSession).not.toHaveBeenCalled();
      expect(manager.updateActivity).not.toHaveBeenCalled();
    }
  });

  it.each(['down', 'up'] as const)('rejects a lease handoff before the second pointer %s operation', async (action) => {
    const session = sessionManager.getSession('test');
    jest.mocked(mockPage.mouse.move).mockImplementation(() => { session.leaseId = 'replacement'; return Promise.resolve(); });
    const res = createMockHttpResponse();
    await handleRecordInput(createMockHttpRequest({ method: 'POST', body: { execution_id: 'owner', lease_id: 'lease', type: 'pointer', action, x: 12, y: 34 } }), res, 'test', sessionManager as SessionManager, config);
    expect(res.statusCode).toBe(404);
    expect(mockPage.mouse.down).not.toHaveBeenCalled();
    expect(mockPage.mouse.up).not.toHaveBeenCalled();
  });

  it('rejects input when ownership changes while reading the body', async () => {
    const res = createMockHttpResponse();
    const pending = handleRecordInput(createMockHttpRequest({ method: 'POST', body: { execution_id: 'owner', lease_id: 'lease', type: 'pointer', action: 'click' } }), res, 'test', sessionManager as SessionManager, config);
    sessionManager.getSession('test').leaseId = 'replacement';
    await pending;
    expect(res.statusCode).toBe(404);
    expect(mockPage.mouse.click).not.toHaveBeenCalled();
    expect(sessionManager.updateActivity).not.toHaveBeenCalled();
  });

  it('handles pointer move input', async () => {
    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/input',
      body: { execution_id: 'owner', lease_id: 'lease', type: 'pointer', action: 'move', x: 10, y: 20 },
    });
    const res = createMockHttpResponse();

    await handleRecordInput(req, res, 'test', sessionManager as SessionManager, config);

    expect(mockPage.mouse.move).toHaveBeenCalledWith(10, 20);
    expect(res.statusCode).toBe(200);
  });

  it('returns the original receipt for a retried input ID without applying the click twice', async () => {
    const body = { execution_id: 'owner', lease_id: 'lease', input_id: 'input-retry-1', type: 'pointer', action: 'click', x: 12, y: 34 };
    const first = createMockHttpResponse();
    const retry = createMockHttpResponse();
    await handleRecordInput(createMockHttpRequest({ method: 'POST', body }), first, 'test', sessionManager as SessionManager, config);
    await handleRecordInput(createMockHttpRequest({ method: 'POST', body }), retry, 'test', sessionManager as SessionManager, config);

    expect(mockPage.mouse.click).toHaveBeenCalledTimes(1);
    expect(JSON.parse(retry.getBody())).toEqual(JSON.parse(first.getBody()));
    expect(JSON.parse(retry.getBody())).toMatchObject({ input_id: 'input-retry-1', status: 'ok', applied_sequence: 1 });
  });

  it('rejects reusing an input ID for a different event', async () => {
    const first = createMockHttpResponse();
    const conflict = createMockHttpResponse();
    await handleRecordInput(createMockHttpRequest({ method: 'POST', body: { execution_id: 'owner', lease_id: 'lease', input_id: 'input-conflict-1', type: 'pointer', action: 'click', x: 12, y: 34 } }), first, 'test', sessionManager as SessionManager, config);
    await handleRecordInput(createMockHttpRequest({ method: 'POST', body: { execution_id: 'owner', lease_id: 'lease', input_id: 'input-conflict-1', type: 'pointer', action: 'click', x: 99, y: 34 } }), conflict, 'test', sessionManager as SessionManager, config);

    expect(conflict.statusCode).toBe(409);
    expect(mockPage.mouse.click).toHaveBeenCalledTimes(1);
  });

  it('serializes concurrent pointer transitions so up cannot overtake down', async () => {
    const applied: string[] = [];
    let completeDownMove!: () => void;
    let downMoveStarted!: () => void;
    let upApplied!: () => void;
    const downMoveGate = new Promise<void>((resolve) => { completeDownMove = resolve; });
    const started = new Promise<void>((resolve) => { downMoveStarted = resolve; });
    const up = new Promise<void>((resolve) => { upApplied = resolve; });
    let moveCount = 0;
    jest.mocked(mockPage.mouse.move).mockImplementation(async () => {
      moveCount += 1;
      if (moveCount === 1) {
        downMoveStarted();
        await downMoveGate;
      }
    });
    jest.mocked(mockPage.mouse.down).mockImplementation(() => { applied.push('down'); return Promise.resolve(); });
    jest.mocked(mockPage.mouse.up).mockImplementation(() => { applied.push('up'); upApplied(); return Promise.resolve(); });

    const downCall = handleRecordInput(createMockHttpRequest({ method: 'POST', body: {
      execution_id: 'owner', lease_id: 'lease', type: 'pointer', action: 'down', button: 'left',
    } }), createMockHttpResponse(), 'test', sessionManager as SessionManager, config);
    await started;
    const upCall = handleRecordInput(createMockHttpRequest({ method: 'POST', body: {
      execution_id: 'owner', lease_id: 'lease', type: 'pointer', action: 'up', button: 'left',
    } }), createMockHttpResponse(), 'test', sessionManager as SessionManager, config);

    try {
      const upOvertookDown = await Promise.race([
        up.then(() => true),
        new Promise<boolean>((resolve) => setTimeout(() => resolve(false), 25)),
      ]);
      completeDownMove();
      await Promise.all([downCall, upCall]);
      expect(upOvertookDown).toBe(false);
      expect(applied).toEqual(['down', 'up']);
    } finally {
      completeDownMove();
      await Promise.allSettled([downCall, upCall]);
    }
  });

  it('coalesces queued pointer motion and acknowledges its final applied sequence', async () => {
    let releaseMove!: () => void;
    let moveStarted!: () => void;
    let allAdmitted!: () => void;
    let admissionCount = 0;
    const gate = new Promise<void>((resolve) => { releaseMove = resolve; });
    const started = new Promise<void>((resolve) => { moveStarted = resolve; });
    const admissions = new Promise<void>((resolve) => { allAdmitted = resolve; });
    jest.mocked(sessionManager.updateActivity).mockImplementation(() => {
      admissionCount += 1;
      if (admissionCount === 4) allAdmitted();
    });
    jest.mocked(mockPage.mouse.move).mockImplementationOnce(async () => {
      moveStarted();
      await gate;
    });
    const send = (body: Record<string, unknown>): { response: ReturnType<typeof createMockHttpResponse>; pending: Promise<void> } => {
      const response = createMockHttpResponse();
      const pending = handleRecordInput(createMockHttpRequest({ method: 'POST', body: {
        execution_id: 'owner', lease_id: 'lease', ...body,
      } }), response, 'test', sessionManager as SessionManager, config);
      return { response, pending };
    };
    const down = send({ type: 'pointer', action: 'down', x: 1, y: 1 });
    await started;
    const moves = [10, 20, 30].map((x) => send({ type: 'pointer', action: 'move', x, y: x }));

    try {
      await admissions;
      expect(mockPage.mouse.move).toHaveBeenCalledTimes(1);
      releaseMove();
      await Promise.all([down.pending, ...moves.map((move) => move.pending)]);
      expect(mockPage.mouse.move).toHaveBeenCalledTimes(2);
      expect(mockPage.mouse.move).toHaveBeenLastCalledWith(30, 30);
      for (const move of moves) {
        expect(move.response.statusCode).toBe(200);
        expect(move.response.getJSON()).toMatchObject({ status: 'ok', applied_sequence: 4, coalesced_count: 2 });
      }
    } finally {
      releaseMove();
      await Promise.allSettled([down.pending, ...moves.map((move) => move.pending)]);
    }
  });

  it('rejects discrete input once the bounded page queue is full', async () => {
    let releaseMove!: () => void;
    let moveStarted!: () => void;
    let allAdmitted!: () => void;
    let admissionCount = 0;
    const gate = new Promise<void>((resolve) => { releaseMove = resolve; });
    const started = new Promise<void>((resolve) => { moveStarted = resolve; });
    const admissions = new Promise<void>((resolve) => { allAdmitted = resolve; });
    jest.mocked(sessionManager.updateActivity).mockImplementation(() => {
      admissionCount += 1;
      if (admissionCount === 66) allAdmitted();
    });
    jest.mocked(mockPage.mouse.move).mockImplementationOnce(async () => {
      moveStarted();
      await gate;
    });
    const send = (body: Record<string, unknown>): { response: ReturnType<typeof createMockHttpResponse>; pending: Promise<void> } => {
      const response = createMockHttpResponse();
      const pending = handleRecordInput(createMockHttpRequest({ method: 'POST', body: {
        execution_id: 'owner', lease_id: 'lease', ...body,
      } }), response, 'test', sessionManager as SessionManager, config);
      return { response, pending };
    };
    const down = send({ type: 'pointer', action: 'down', x: 1, y: 1 });
    await started;
    const queued = Array.from({ length: 65 }, () => send({ type: 'wheel', delta_y: 1 }));

    try {
      await admissions;
      await new Promise<void>((resolve) => setImmediate(resolve));
      expect(queued.filter((item) => item.response.statusCode === 429)).toHaveLength(1);
      releaseMove();
      await Promise.all([down.pending, ...queued.map((item) => item.pending)]);
      expect(queued.filter((item) => item.response.statusCode === 200)).toHaveLength(64);
      expect(mockPage.mouse.wheel).toHaveBeenCalledTimes(64);
    } finally {
      releaseMove();
      await Promise.allSettled([down.pending, ...queued.map((item) => item.pending)]);
    }
  });

  it('holds pointer modifiers through down and releases them after up', async () => {
    const downResponse = createMockHttpResponse();
    await handleRecordInput(createMockHttpRequest({ method: 'POST', body: {
      execution_id: 'owner', lease_id: 'lease', type: 'pointer', action: 'down', button: 'left', modifiers: ['Shift'],
    } }), downResponse, 'test', sessionManager as SessionManager, config);

    expect(mockPage.keyboard.down).toHaveBeenCalledWith('Shift');
    expect(mockPage.mouse.down).toHaveBeenCalledWith({ button: 'left' });
    expect(mockPage.keyboard.up).not.toHaveBeenCalled();

    const moveResponse = createMockHttpResponse();
    await handleRecordInput(createMockHttpRequest({ method: 'POST', body: {
      execution_id: 'owner', lease_id: 'lease', type: 'pointer', action: 'move', x: 20, y: 30, modifiers: ['Shift'],
    } }), moveResponse, 'test', sessionManager as SessionManager, config);
    expect(mockPage.keyboard.up).not.toHaveBeenCalled();

    const releaseOrder: string[] = [];
    jest.mocked(mockPage.mouse.up).mockImplementation(() => { releaseOrder.push('pointer-up'); return Promise.resolve(); });
    jest.mocked(mockPage.keyboard.up).mockImplementation(() => { releaseOrder.push('modifier-up'); return Promise.resolve(); });
    const upResponse = createMockHttpResponse();
    await handleRecordInput(createMockHttpRequest({ method: 'POST', body: {
      execution_id: 'owner', lease_id: 'lease', type: 'pointer', action: 'up', button: 'left', modifiers: ['Shift'],
    } }), upResponse, 'test', sessionManager as SessionManager, config);

    expect(mockPage.mouse.up).toHaveBeenCalledWith({ button: 'left' });
    expect(mockPage.keyboard.up).toHaveBeenCalledWith('Shift');
    expect(releaseOrder).toEqual(['pointer-up', 'modifier-up']);
    expect(upResponse.statusCode).toBe(200);
  });

  it('releases pointer modifiers after a failed down action', async () => {
    jest.mocked(mockPage.mouse.down).mockRejectedValueOnce(new Error('synthetic pointer failure'));
    const res = createMockHttpResponse();
    await handleRecordInput(createMockHttpRequest({ method: 'POST', body: {
      execution_id: 'owner', lease_id: 'lease', type: 'pointer', action: 'down', button: 'left', modifiers: ['Control'],
    } }), res, 'test', sessionManager as SessionManager, config);

    expect(mockPage.keyboard.down).toHaveBeenCalledWith('Control');
    expect(mockPage.keyboard.up).toHaveBeenCalledWith('Control');
    expect(mockPage.mouse.up).toHaveBeenCalledWith({ button: 'left' });
    expect(res.statusCode).not.toBe(200);
  });

  it('presses modifiers only for a modified pointer click', async () => {
    const res = createMockHttpResponse();
    await handleRecordInput(createMockHttpRequest({ method: 'POST', body: {
      execution_id: 'owner', lease_id: 'lease', type: 'pointer', action: 'click', button: 'right', x: 20, y: 30, modifiers: ['Alt', 'Meta'],
    } }), res, 'test', sessionManager as SessionManager, config);

    expect(mockPage.keyboard.down).toHaveBeenNthCalledWith(1, 'Alt');
    expect(mockPage.keyboard.down).toHaveBeenNthCalledWith(2, 'Meta');
    expect(mockPage.mouse.click).toHaveBeenCalledWith(20, 30, { button: 'right' });
    expect(mockPage.keyboard.up).toHaveBeenNthCalledWith(1, 'Meta');
    expect(mockPage.keyboard.up).toHaveBeenNthCalledWith(2, 'Alt');
    expect(res.statusCode).toBe(200);
  });

  it('attempts to release every modifier when one release fails', async () => {
    jest.mocked(mockPage.keyboard.up).mockRejectedValueOnce(new Error('synthetic modifier release failure'));
    const res = createMockHttpResponse();
    await handleRecordInput(createMockHttpRequest({ method: 'POST', body: {
      execution_id: 'owner', lease_id: 'lease', type: 'pointer', action: 'click', modifiers: ['Shift', 'Alt'],
    } }), res, 'test', sessionManager as SessionManager, config);

    expect(mockPage.keyboard.up).toHaveBeenCalledTimes(2);
    expect(mockPage.keyboard.up).toHaveBeenNthCalledWith(1, 'Alt');
    expect(mockPage.keyboard.up).toHaveBeenNthCalledWith(2, 'Shift');
    expect(res.statusCode).not.toBe(200);
  });

  it('returns 400 for invalid pointer action', async () => {
    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/input',
      body: { execution_id: 'owner', lease_id: 'lease', type: 'pointer', action: 'tap', x: 1, y: 2 },
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
      body: { execution_id: 'owner', lease_id: 'lease', type: 'keyboard', text: 'hello' },
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
      body: { execution_id: 'owner', lease_id: 'lease', type: 'keyboard', key: 'Enter', modifiers: ['Shift', 'Alt'] },
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
      body: { execution_id: 'owner', lease_id: 'lease', type: 'keyboard' },
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
      body: { execution_id: 'owner', lease_id: 'lease', type: 'wheel', delta_x: 5, delta_y: -3 },
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
      body: { execution_id: 'owner', lease_id: 'lease', expected_page_id: 'selected-page', width: 800.4, height: 600.6 },
    });
    const res = createMockHttpResponse();

    await handleRecordViewport(req, res, 'test', sessionManager as SessionManager, config);

    expect(mockPage.setViewportSize).toHaveBeenCalledWith({ width: 800, height: 601 });
    expect(res.statusCode).toBe(200);
    expect(res.getJSON()).toEqual({ session_id: 'test', driver_page_id: 'selected-page', width: 800, height: 600 });
  });

  it('rejects invalid viewport sizes', async () => {
    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/viewport',
      body: { execution_id: 'owner', lease_id: 'lease', expected_page_id: 'selected-page', width: 0, height: -10 },
    });
    const res = createMockHttpResponse();

    await handleRecordViewport(req, res, 'test', sessionManager as SessionManager, config);

    expect(res.statusCode).toBe(400);
    expect(res.getJSON().error).toBe('INVALID_VIEWPORT');
  });
  describe('viewport admission [REQ:BAS-RH-J03]', () => {
    it.each(['missing lease', 'stale lease', 'closing', 'stale page', 'body handoff'])('rejects %s before changing the page', async kind => {
      const session = sessionManager.getSession('test');
      if (kind === 'closing') session.phase = 'closing';
      const body = {execution_id: kind === 'missing lease' ? undefined : 'owner', lease_id: kind === 'stale lease' ? 'old' : 'lease', expected_page_id: kind === 'stale page' ? 'old-page' : 'selected-page', width: 900, height: 700};
      const res = createMockHttpResponse();
      const pending = handleRecordViewport(createMockHttpRequest({method: 'POST', body}), res, 'test', sessionManager as SessionManager, config);
      if (kind === 'body handoff') session.leaseId = 'replacement';
      await pending;
      expect(res.statusCode).toBe(kind === 'missing lease' ? 400 : kind === 'stale page' ? 409 : 404);
      expect(mockPage.setViewportSize).not.toHaveBeenCalled();
      expect(updateFrameStreamViewport).not.toHaveBeenCalled();
    });
    it.each(['page', 'lease'])('does not acknowledge or refresh capture after a %s handoff', async kind => {
      const session = sessionManager.getSession('test');
      jest.mocked(mockPage.setViewportSize).mockImplementationOnce(() => {
        if (kind === 'page') session.page = createMockPage();else session.leaseId = 'replacement';
        return Promise.resolve();
      });
      const res = createMockHttpResponse();
      await handleRecordViewport(createMockHttpRequest({method: 'POST', body: {execution_id: 'owner', lease_id: 'lease', expected_page_id: 'selected-page', width: 900, height: 700}}), res, 'test', sessionManager as SessionManager, config);
      expect(res.statusCode).toBe(404);
      expect(updateFrameStreamViewport).not.toHaveBeenCalled();
    });
  });

});
