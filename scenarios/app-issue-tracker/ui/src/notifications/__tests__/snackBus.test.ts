import { describe, expect, it, vi } from 'vitest';
import {
  DEFAULT_SNACK_DURATION,
  clearSnacks,
  dismissSnack,
  patchSnack,
  publishSnack,
  replaceSnack,
  snackBus,
  type SnackEvent,
} from '../snackBus';

describe('snackBus', () => {
  it('publishes show events with sensible defaults', () => {
    const events: SnackEvent[] = [];
    const unsubscribe = snackBus.subscribe((event) => {
      events.push(event);
    });

    const id = publishSnack({ message: 'Hello world' });

    unsubscribe();

    expect(id).toMatch(/^snack-\d+/);
    expect(events).toHaveLength(1);
    const event = events[0];
    expect(event.type).toBe('show');
    if (event.type !== 'show') {
      throw new Error('expected show event');
    }
    expect(event.snack).toMatchObject({
      id,
      message: 'Hello world',
      variant: 'info',
      autoDismiss: true,
      dismissible: true,
      durationMs: DEFAULT_SNACK_DURATION,
    });
  });

  it('emits update events when replacing a snack', () => {
    const events: SnackEvent[] = [];
    const unsubscribe = snackBus.subscribe((event) => {
      events.push(event);
    });

    const id = publishSnack({ message: 'Processing', variant: 'loading', autoDismiss: false });
    replaceSnack(id, { message: 'Done', variant: 'success', durationMs: 1200 });

    unsubscribe();

    const update = events.find((event) => event.type === 'update');
    expect(update).toBeDefined();
    if (update?.type !== 'update') {
      throw new Error('expected update event');
    }
    expect(update.changes).toMatchObject({
      id,
      message: 'Done',
      variant: 'success',
      durationMs: 1200,
      autoDismiss: true,
    });
  });

  it('emits patch events and honors dismiss/clear', () => {
    const handler = vi.fn();
    const unsubscribe = snackBus.subscribe(handler);

    const id = publishSnack({ message: 'Pending' });
    patchSnack(id, { message: 'Still pending', durationMs: 9000 });
    dismissSnack(id);
    clearSnacks();

    unsubscribe();

    expect(handler).toHaveBeenCalledWith(expect.objectContaining({ type: 'patch', id }));
    expect(handler).toHaveBeenCalledWith(expect.objectContaining({ type: 'dismiss', id }));
    expect(handler).toHaveBeenCalledWith(expect.objectContaining({ type: 'clear' }));
  });
});
