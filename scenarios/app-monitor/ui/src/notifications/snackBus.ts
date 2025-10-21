/*
 * Centralized pub/sub singleton for app-monitor snack notifications.
 * Components publish events; the SnackStackProvider subscribes and renders them.
 */

export type SnackVariant = 'info' | 'success' | 'warning' | 'error' | 'loading';

export interface SnackAction {
  label: string;
  handler: () => void;
  /** When false, keep the snack visible after invoking the action. Defaults to true. */
  dismissOnAction?: boolean;
}

export interface SnackPublishOptions {
  id?: string;
  message: string;
  variant?: SnackVariant;
  title?: string;
  /**
   * Auto-dismiss duration in milliseconds. Use 0 or a negative number for persistent snacks.
   * When omitted, defaults to 5000ms except for `loading`, which defaults to persistent.
   */
  durationMs?: number;
  autoDismiss?: boolean;
  dismissible?: boolean;
  action?: SnackAction;
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

  replace(id: string, options: SnackPublishOptions) {
    const snack = normalizePublishOptions({ ...options, id });
    this.emit({ type: 'update', id, changes: snack });
  }

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
