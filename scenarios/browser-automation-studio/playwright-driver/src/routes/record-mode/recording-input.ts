/**
 * Recording Input
 *
 * Handles live input forwarding to the browser:
 * - Mouse/pointer events (move, click, down, up)
 * - Keyboard events (key press, text input)
 * - Wheel/scroll events
 * - Viewport size changes
 */

import type { IncomingMessage, ServerResponse } from 'http';
import type { Page } from 'rebrowser-playwright';
import type { SessionManager } from '../../session';
import type { Config } from '../../config';
import { parseJsonBody, sendJson, sendError } from '../../middleware';
import { ResourceLimitError, SessionNotFoundError } from '../../utils';
import { updateFrameStreamViewport } from '../../frame-streaming';
import type { InputRequest, PointerAction, ViewportRequest, ViewportResponse } from './types';
import { recordingOwner } from './recording-ownership';

// =============================================================================
// Input Handlers
// =============================================================================

type PointerModifier = 'Alt' | 'Control' | 'Meta' | 'Shift';
const pointerModifiers = new Set<PointerModifier>(['Alt', 'Control', 'Meta', 'Shift']);
const heldPointerModifiers = new WeakMap<Page, Set<PointerModifier>>();
const pointerDownPages = new WeakSet<Page>();
const MAX_PENDING_INPUTS = 64;
const MAX_COALESCED_INPUT_CALLERS = 128;
const MAX_INPUT_RECEIPTS = 256;

type InputReceipt = { status: 'ok'; applied_sequence: number; coalesced_count: number; input_id?: string };
class InputIDConflictError extends Error {
  constructor() { super('input_id was already used for a different input'); }
}
type CachedInputReceipt = { fingerprint: string; done: boolean; promise: Promise<InputReceipt> };
type InputWaiter = { resolve: (receipt: InputReceipt) => void; reject: (error: unknown) => void };
type QueuedInput = { sequence: number; coalesceKey?: string; apply: () => Promise<void>; waiters: InputWaiter[] };
type PageInputQueue = { nextSequence: number; running: boolean; pending: QueuedInput[] };
const inputQueues = new WeakMap<Page, PageInputQueue>();
const inputReceipts = new WeakMap<Page, Map<string, CachedInputReceipt>>();

function enqueueIdempotentInput(
  page: Page,
  inputId: string | undefined,
  fingerprint: string,
  apply: () => Promise<void>,
  coalesceKey?: string,
): Promise<InputReceipt> {
  if (!inputId) return enqueueInput(page, apply, coalesceKey);
  const receipts = inputReceipts.get(page) ?? new Map<string, CachedInputReceipt>();
  inputReceipts.set(page, receipts);
  const existing = receipts.get(inputId);
  if (existing) {
    if (existing.fingerprint !== fingerprint) {
      return Promise.reject(new InputIDConflictError());
    }
    // Refresh insertion order so recent retries remain inside the bounded window.
    receipts.delete(inputId);
    receipts.set(inputId, existing);
    return existing.promise;
  }

  const promise = enqueueInput(page, apply, coalesceKey).then((receipt) => ({ ...receipt, input_id: inputId }));
  const entry: CachedInputReceipt = { fingerprint, done: false, promise };
  receipts.set(inputId, entry);
  void promise.then(() => {
    entry.done = true;
    while (receipts.size > MAX_INPUT_RECEIPTS) {
      const oldest = [...receipts].find(([, candidate]) => candidate.done);
      if (!oldest) break;
      receipts.delete(oldest[0]);
    }
  }, () => { receipts.delete(inputId); });
  return promise;
}

function enqueueInput(page: Page, apply: () => Promise<void>, coalesceKey?: string): Promise<InputReceipt> {
  const queue = inputQueues.get(page) ?? { nextSequence: 0, running: false, pending: [] };
  inputQueues.set(page, queue);
  const tail = queue.pending[queue.pending.length - 1];
  const coalesceTarget = coalesceKey !== undefined && tail?.coalesceKey === coalesceKey ? tail : undefined;
  if (!coalesceTarget && queue.pending.length >= MAX_PENDING_INPUTS) {
    return Promise.reject(new ResourceLimitError('Live input queue is full', { maxPending: MAX_PENDING_INPUTS }));
  }
  if (coalesceTarget && coalesceTarget.waiters.length >= MAX_COALESCED_INPUT_CALLERS) {
    return Promise.reject(new ResourceLimitError('Live input motion backlog is full', { maxCoalescedCallers: MAX_COALESCED_INPUT_CALLERS }));
  }

  const sequence = queue.nextSequence + 1;
  queue.nextSequence = sequence;
  const promise = new Promise<InputReceipt>((resolve, reject) => {
    const waiter = { resolve, reject };
    if (coalesceTarget) {
      coalesceTarget.sequence = sequence;
      coalesceTarget.apply = apply;
      coalesceTarget.waiters.push(waiter);
      return;
    }
    queue.pending.push({ sequence, coalesceKey, apply, waiters: [waiter] });
  });
  void drainInputQueue(queue);
  return promise;
}

async function drainInputQueue(queue: PageInputQueue): Promise<void> {
  if (queue.running) return;
  queue.running = true;
  try {
    while (queue.pending.length > 0) {
      const item = queue.pending.shift();
      if (!item) continue;
      try {
        await item.apply();
        const receipt: InputReceipt = {
          status: 'ok', applied_sequence: item.sequence, coalesced_count: item.waiters.length - 1,
        };
        for (const waiter of item.waiters) waiter.resolve(receipt);
      } catch (error) {
        for (const waiter of item.waiters) waiter.reject(error);
      }
    }
  } finally {
    queue.running = false;
    if (queue.pending.length > 0) void drainInputQueue(queue);
  }
}

function getHeldPointerModifiers(page: Page): Set<PointerModifier> {
  let held = heldPointerModifiers.get(page);
  if (!held) {
    held = new Set();
    heldPointerModifiers.set(page, held);
  }
  return held;
}

async function synchronizePointerModifiers(
  page: Page,
  target: Set<PointerModifier>,
  held: Set<PointerModifier>,
  assertOwner: () => void,
): Promise<void> {
  for (const modifier of [...held]) {
    if (target.has(modifier)) continue;
    assertOwner();
    await page.keyboard.up(modifier);
    held.delete(modifier);
  }
  for (const modifier of target) {
    if (held.has(modifier)) continue;
    assertOwner();
    // Add before awaiting because the browser may apply the key before the
    // transport rejects; the caller's finally block must still release it.
    held.add(modifier);
    await page.keyboard.down(modifier);
  }
}

async function releasePointerModifiers(page: Page, held: Set<PointerModifier>): Promise<void> {
  let releaseError: unknown;
  for (const modifier of [...held].reverse()) {
    try {
      await page.keyboard.up(modifier);
      held.delete(modifier);
    } catch (error) {
      releaseError ??= error;
    }
  }
  if (releaseError) throw releaseError;
}

/**
 * Forward live input events to the active Playwright page.
 *
 * POST /session/:id/record/input
 */
export async function handleRecordInput(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const body = await parseJsonBody(req, config);
    const request = body as unknown as InputRequest;

    if (!request?.type) {
      sendJson(res, 400, {
        error: 'MISSING_TYPE',
        message: 'type field is required',
      });
      return;
    }

    const ownedSession = recordingOwner(body, sessionId, sessionManager);
    const session = ownedSession();
    sessionManager.updateActivity(sessionId);
    const page = session.page;
    const modifiers = request.modifiers || [];

    if (request.type === 'pointer' && !['move', 'down', 'up', 'click'].includes(request.action || 'move')) {
      sendJson(res, 400, { error: 'INVALID_ACTION', message: 'Unsupported pointer action' });
      return;
    }
    if (request.type === 'keyboard' && !request.text && !request.key) {
      sendJson(res, 400, { error: 'MISSING_KEYBOARD_DATA', message: 'Provide key or text for keyboard input' });
      return;
    }
    if (!['pointer', 'keyboard', 'wheel'].includes(request.type)) {
      sendJson(res, 400, { error: 'INVALID_TYPE', message: 'Unsupported input type' });
      return;
    }

    if (request.input_id !== undefined && (typeof request.input_id !== 'string' || request.input_id.length === 0 || request.input_id.length > 128)) {
      sendJson(res, 400, { error: 'INVALID_INPUT_ID', message: 'input_id must be a non-empty string of at most 128 characters' });
      return;
    }

    const action = request.action || 'move';
    const coalesceKey = request.type === 'pointer' && action === 'move'
      ? JSON.stringify((request.modifiers ?? []).filter((modifier) => pointerModifiers.has(modifier as PointerModifier)).sort())
      : undefined;
    const inputPayload = Object.fromEntries(Object.entries(request).filter(
      ([key]) => key !== 'execution_id' && key !== 'lease_id' && key !== 'input_id',
    ));
    const receipt = await enqueueIdempotentInput(page, request.input_id, JSON.stringify(inputPayload), async () => {
      ownedSession();
      switch (request.type) {
        case 'pointer': {
          const x = request.x ?? 0;
          const y = request.y ?? 0;
          const button = request.button || 'left';
          const pointerAction: PointerAction = request.action || 'move';
          const held = getHeldPointerModifiers(page);
          const requested = new Set((request.modifiers ?? []).filter(
            (modifier): modifier is PointerModifier => pointerModifiers.has(modifier as PointerModifier),
          ));
          let downAttempted = false;
          let downCompleted = false;

          try {
            await synchronizePointerModifiers(page, requested, held, ownedSession);
            if (pointerAction === 'move') {
              await page.mouse.move(x, y);
            } else if (pointerAction === 'down') {
              await page.mouse.move(x, y);
              ownedSession();
              pointerDownPages.add(page);
              downAttempted = true;
              await page.mouse.down({ button });
              downCompleted = true;
            } else if (pointerAction === 'up') {
              await page.mouse.move(x, y);
              ownedSession();
              await page.mouse.up({ button });
              pointerDownPages.delete(page);
            } else {
              await page.mouse.click(x, y, { button });
            }
          } catch (error) {
            if (pointerAction === 'down' && downAttempted && !downCompleted) {
              await page.mouse.up({ button }).catch(() => undefined);
              pointerDownPages.delete(page);
            }
            throw error;
          } finally {
            const keepModifiersHeld = (pointerAction === 'down' && downCompleted)
              || ((pointerAction === 'move' || pointerAction === 'up') && pointerDownPages.has(page));
            if (!keepModifiersHeld) await releasePointerModifiers(page, held);
          }
          break;
        }
        case 'wheel':
          await page.mouse.wheel(request.delta_x ?? 0, request.delta_y ?? 0);
          break;
        case 'keyboard':
          if (request.text) await page.keyboard.type(request.text);
          else {
            const key = request.key;
            if (!key) throw new Error('Keyboard key was not supplied after validation');
            const combo = modifiers.length > 0 ? `${modifiers.join('+')}+${key}` : key;
            await page.keyboard.press(combo);
          }
          break;
      }
      ownedSession();
    }, coalesceKey);
    sendJson(res, 200, receipt);
  } catch (error) {
    if (error instanceof InputIDConflictError) {
      sendJson(res, 409, { error: 'INPUT_ID_CONFLICT', message: error.message });
      return;
    }
    sendError(res, error as Error, `/session/${sessionId}/record/input`);
  }
}

/**
 * Update viewport size for the active recording page.
 * Also updates the frame streaming viewport to restart screencast at new dimensions.
 *
 * POST /session/:id/record/viewport
 */
export async function handleRecordViewport(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const body = await parseJsonBody(req, config);
    const ownedSession = recordingOwner(body, sessionId, sessionManager);
    const session = ownedSession();
    const page = session.page;
    const pageId = session.pageToIdMap.get(page);
    if (!pageId || body.expected_page_id !== pageId) {
      sendJson(res, 409, {error: 'PAGE_CHANGED', message: 'The selected recording tab changed before resize'});
      return;
    }
    const ownedPage = (): void => {
      if (ownedSession().page !== page || session.pageToIdMap.get(page) !== pageId) throw new SessionNotFoundError(sessionId);
    };
    const {width, height} = body as unknown as ViewportRequest;
    if (typeof width !== 'number' || typeof height !== 'number' || !Number.isFinite(width) || !Number.isFinite(height) || Math.round(width) <= 0 || Math.round(height) <= 0) {
      sendJson(res, 400, {error: 'INVALID_VIEWPORT', message: 'width and height must be positive finite numbers'});
      return;
    }
    await page.setViewportSize({width: Math.round(width), height: Math.round(height)});
    ownedPage();
    await updateFrameStreamViewport(sessionId, page);
    ownedPage();
    const viewport = page.viewportSize();
    if (!viewport) throw new Error('Applied viewport is unavailable');
    const response: ViewportResponse = {session_id: sessionId, driver_page_id: pageId, ...viewport};

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/viewport`);
  }
}
