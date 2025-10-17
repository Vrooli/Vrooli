/*
 * Centralized pub/sub singleton for UI snack notifications.
 * Consumers publish events; the SnackStackProvider subscribes and renders them.
 */

export type SnackVariant = 'info' | 'success' | 'warning' | 'error' | 'loading';

export interface SnackAction {
  label: string;
  handler: () => void;
  /** When false, the snack persists after invoking the action. Defaults to true. */
  dismissOnAction?: boolean;
}

export interface SnackPublishOptions {
  id?: string;
  message: string;
  variant?: SnackVariant;
  /** Optional human friendly title; allows future richer UI without changing API */
  title?: string;
  /**
   * Auto-dismiss duration in milliseconds. Use 0 or a negative number for persistent snacks.
   * When omitted, defaults to 5000ms except for `loading`, which defaults to persistent.
   */
  durationMs?: number;
  /** Force auto-dismiss behaviour off/on regardless of duration defaults. */
  autoDismiss?: boolean;
  /** Whether the snack renders a close affordance. Defaults true unless explicitly disabled. */
  dismissible?: boolean;
  /** Optional call-to-action rendered beside the message. */
  action?: SnackAction;
  /** Arbitrary metadata so producers can stash structured context if needed. */
  metadata?: Record<string, unknown>;
}

export interface SnackUpdateOptions extends Partial<Omit<SnackPublishOptions, 'id'>> {
  message?: string;
}

export interface SnackDescriptor {
  id: string;
  message: string;
  title?: string;
  variant: SnackVariant;
  dismissible: boolean;
  /** `null` signals persistent snack */
  durationMs: number | null;
  autoDismiss: boolean;
  action?: SnackAction;
  metadata?: Record<string, unknown>;
  createdAt: number;
  updatedAt: number;
}

export type SnackEvent =
  | { type: 'show'; snack: SnackDescriptor }
  | { type: 'update'; id: string; changes: SnackDescriptor }
  | { type: 'patch'; id: string; changes: SnackUpdateOptions }
  | { type: 'dismiss'; id: string }
  | { type: 'clear' };

type SnackListener = (event: SnackEvent) => void;

export const DEFAULT_SNACK_DURATION = 5000;

let counter = 0;
function nextId() {
  counter += 1;
  return `snack-${counter}`;
}

function normalizePublishOptions(options: SnackPublishOptions): SnackDescriptor {
  const now = Date.now();
  const variant = options.variant ?? 'info';
  const autoDismiss = (() => {
    if (typeof options.autoDismiss === 'boolean') {
      return options.autoDismiss;
    }
    if (variant === 'loading') {
      return false;
    }
    if (typeof options.durationMs === 'number') {
      return options.durationMs > 0;
    }
    return true;
  })();

  const resolvedDuration = (() => {
    if (!autoDismiss) {
      return null;
    }
    if (typeof options.durationMs === 'number') {
      return options.durationMs > 0 ? options.durationMs : null;
    }
    return DEFAULT_SNACK_DURATION;
  })();

  const dismissible = options.dismissible ?? true;

  return {
    id: options.id ?? nextId(),
    message: options.message,
    title: options.title,
    variant,
    dismissible,
    durationMs: resolvedDuration,
    autoDismiss,
    action: options.action,
    metadata: options.metadata,
    createdAt: now,
    updatedAt: now,
  } satisfies SnackDescriptor;
}

class SnackBus {
  private listeners = new Set<SnackListener>();

  subscribe(listener: SnackListener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  private emit(event: SnackEvent) {
    for (const listener of this.listeners) {
      listener(event);
    }
  }

  publish(options: SnackPublishOptions): string {
    const snack = normalizePublishOptions(options);
    this.emit({ type: 'show', snack });
    return snack.id;
  }

  /**
   * Replaces an existing snack by id with a new payload. Mainly used when callers
   * want to update message, duration, or intent in one shot while keeping timestamps fresh.
   */
  replace(id: string, options: SnackPublishOptions) {
    const snack = normalizePublishOptions({ ...options, id });
    this.emit({ type: 'update', id, changes: snack });
  }

  /**
   * Partially updates a snack in-place. Consumers can extend lifetime by adjusting duration or
   * switch variants without rewriting the full descriptor.
   */
  patch(id: string, changes: SnackUpdateOptions) {
    this.emit({ type: 'patch', id, changes });
  }

  dismiss(id: string) {
    this.emit({ type: 'dismiss', id });
  }

  clear() {
    this.emit({ type: 'clear' });
  }
}

export const snackBus = new SnackBus();

export function publishSnack(options: SnackPublishOptions): string {
  return snackBus.publish(options);
}

export function replaceSnack(id: string, options: SnackPublishOptions) {
  snackBus.replace(id, options);
}

export function patchSnack(id: string, changes: SnackUpdateOptions) {
  snackBus.patch(id, changes);
}

export function dismissSnack(id: string) {
  snackBus.dismiss(id);
}

export function clearSnacks() {
  snackBus.clear();
}
