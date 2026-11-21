import type { Page } from 'rebrowser-playwright';
import { ResourceLimitError } from '../utils';

type PointerModifier = 'Alt' | 'Control' | 'Meta' | 'Shift';
export type InputReceipt = {
  status: 'ok';
  applied_sequence: number;
  coalesced_count: number;
  input_id?: string;
};
type InputWaiter = { resolve: (receipt: InputReceipt) => void; reject: (error: unknown) => void };
type QueuedInput = {
  sequence: number;
  coalesceKey?: string;
  apply: () => Promise<void>;
  waiters: InputWaiter[];
};
type PageInputQueue = {
  nextSequence: number;
  running: boolean;
  pending: QueuedInput[];
  drain?: Promise<void>;
};
type CachedInputReceipt = { fingerprint: string; done: boolean; promise: Promise<InputReceipt> };

const MAX_PENDING_INPUTS = 64;
const MAX_COALESCED_INPUT_CALLERS = 128;
const MAX_INPUT_RECEIPTS = 256;
const inputQueues = new WeakMap<Page, PageInputQueue>();
const inputReceipts = new WeakMap<Page, Map<string, CachedInputReceipt>>();
const heldPointerModifiers = new WeakMap<Page, Set<PointerModifier>>();
const pointerDownPages = new WeakSet<Page>();

export class InputIDConflictError extends Error {
  constructor() {
    super('input_id was already used for a different input');
  }
}

export function enqueueIdempotentInput(
  page: Page,
  inputId: string | undefined,
  fingerprint: string,
  apply: () => Promise<void>,
  coalesceKey?: string
): Promise<InputReceipt> {
  if (!inputId) return enqueueInput(page, apply, coalesceKey);
  const receipts = inputReceipts.get(page) ?? new Map<string, CachedInputReceipt>();
  inputReceipts.set(page, receipts);
  const existing = receipts.get(inputId);
  if (existing) {
    if (existing.fingerprint !== fingerprint) return Promise.reject(new InputIDConflictError());
    receipts.delete(inputId);
    receipts.set(inputId, existing);
    return existing.promise;
  }
  const promise = enqueueInput(page, apply, coalesceKey).then((receipt) => ({
    ...receipt,
    input_id: inputId,
  }));
  const entry: CachedInputReceipt = { fingerprint, done: false, promise };
  receipts.set(inputId, entry);
  void promise.then(
    () => {
      entry.done = true;
      while (receipts.size > MAX_INPUT_RECEIPTS) {
        const oldest = [...receipts].find(([, candidate]) => candidate.done);
        if (!oldest) break;
        receipts.delete(oldest[0]);
      }
    },
    () => {
      receipts.delete(inputId);
    }
  );
  return promise;
}

/** Serialize a non-idempotent page mutation with live input and page reset. */
export function enqueuePageMutation(page: Page, apply: () => Promise<void>): Promise<void> {
  return enqueueInput(page, apply).then(() => undefined);
}

function enqueueInput(
  page: Page,
  apply: () => Promise<void>,
  coalesceKey?: string
): Promise<InputReceipt> {
  const queue = inputQueues.get(page) ?? { nextSequence: 0, running: false, pending: [] };
  inputQueues.set(page, queue);
  const tail = queue.pending[queue.pending.length - 1];
  const coalesceTarget =
    coalesceKey !== undefined && tail?.coalesceKey === coalesceKey ? tail : undefined;
  if (!coalesceTarget && queue.pending.length >= MAX_PENDING_INPUTS) {
    return Promise.reject(
      new ResourceLimitError('Live input queue is full', { maxPending: MAX_PENDING_INPUTS })
    );
  }
  if (coalesceTarget && coalesceTarget.waiters.length >= MAX_COALESCED_INPUT_CALLERS) {
    return Promise.reject(
      new ResourceLimitError('Live input motion backlog is full', {
        maxCoalescedCallers: MAX_COALESCED_INPUT_CALLERS,
      })
    );
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
  scheduleInputDrain(queue);
  return promise;
}

function scheduleInputDrain(queue: PageInputQueue): void {
  if (queue.drain) return;
  const drain = drainInputQueue(queue);
  queue.drain = drain;
  void drain
    .finally(() => {
      if (queue.drain === drain) queue.drain = undefined;
      if (queue.pending.length > 0) scheduleInputDrain(queue);
    })
    .catch(() => undefined);
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
          status: 'ok',
          applied_sequence: item.sequence,
          coalesced_count: item.waiters.length - 1,
        };
        for (const waiter of item.waiters) waiter.resolve(receipt);
      } catch (error) {
        for (const waiter of item.waiters) waiter.reject(error);
      }
    }
  } finally {
    queue.running = false;
  }
}

/** Join admitted input before a session owner mutates or disposes a page. */
export function settlePageInput(page: Page): Promise<void> {
  return inputQueues.get(page)?.drain ?? Promise.resolve();
}

export function getHeldPointerModifiers(page: Page): Set<PointerModifier> {
  let held = heldPointerModifiers.get(page);
  if (!held) {
    held = new Set();
    heldPointerModifiers.set(page, held);
  }
  return held;
}

export function setPointerDown(page: Page, down: boolean): void {
  if (down) pointerDownPages.add(page);
  else pointerDownPages.delete(page);
}

export function isPointerDown(page: Page): boolean {
  return pointerDownPages.has(page);
}

export async function resetPageInputState(page: Page): Promise<void> {
  // Reset may be called directly by an owner test or a future lifecycle path;
  // join admitted work before retiring the retained page's input identity.
  await settlePageInput(page);
  const held = heldPointerModifiers.get(page);
  if (held) await releasePointerModifiers(page, held);
  inputQueues.delete(page);
  inputReceipts.delete(page);
  pointerDownPages.delete(page);
  heldPointerModifiers.delete(page);
}

export async function synchronizePointerModifiers(
  page: Page,
  target: Set<PointerModifier>,
  held: Set<PointerModifier>,
  assertOwner: () => void
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
    held.add(modifier);
    await page.keyboard.down(modifier);
  }
}

export async function releasePointerModifiers(
  page: Page,
  held: Set<PointerModifier>
): Promise<void> {
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
