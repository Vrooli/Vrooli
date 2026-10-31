import type { Page } from 'rebrowser-playwright';
import { createMockPage } from '../../helpers';
import {
  enqueueIdempotentInput,
  resetPageInputState,
} from '../../../src/session/live-input';

describe('session-owned live input lifecycle', () => {
  it('retires idempotency receipts when a retained page is reset', async () => {
    const page = createMockPage() as unknown as Page;
    const applied: string[] = [];

    const first = await enqueueIdempotentInput(
      page,
      'input-1',
      'before-reset',
      () => { applied.push('before-reset'); return Promise.resolve(); },
    );
    await resetPageInputState(page);

    const second = await enqueueIdempotentInput(
      page,
      'input-1',
      'after-reset',
      () => { applied.push('after-reset'); return Promise.resolve(); },
    );

    expect(first.applied_sequence).toBe(1);
    expect(second.applied_sequence).toBe(1);
    expect(applied).toEqual(['before-reset', 'after-reset']);
  });

  it('joins admitted input before retiring the retained page state', async () => {
    const page = createMockPage() as unknown as Page;
    let release!: () => void;
    let started!: () => void;
    const gate = new Promise<void>((resolve) => { release = resolve; });
    const began = new Promise<void>((resolve) => { started = resolve; });

    const first = enqueueIdempotentInput(page, 'input-2', 'before-reset', async () => {
      started();
      await gate;
    });
    await began;
    const reset = resetPageInputState(page);
    const settledBeforeRelease = await Promise.race([
      reset.then(() => false),
      new Promise<boolean>((resolve) => setImmediate(() => resolve(true))),
    ]);
    expect(settledBeforeRelease).toBe(true);

    release();
    await Promise.all([first, reset]);
    await expect(enqueueIdempotentInput(
      page,
      'input-2',
      'after-reset',
      () => Promise.resolve(),
    )).resolves.toMatchObject({ applied_sequence: 1 });
  });
});
