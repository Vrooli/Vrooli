import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { SnackStack } from '../components/SnackStack';
import {
  DEFAULT_SNACK_DURATION,
  dismissSnack,
  publishSnack,
  replaceSnack,
  snackBus,
  type SnackDescriptor,
  type SnackPublishOptions,
  type SnackVariant,
  type SnackEvent,
  type SnackUpdateOptions,
} from './snackBus';

interface SnackStackProviderProps {
  children: ReactNode;
  /** future-proof anchor locations (defaults to top-right) */
  anchor?: 'top-right' | 'top-left' | 'bottom-right' | 'bottom-left';
  /** Maximum number of concurrent snacks before we start dropping oldest ones */
  maxSnacks?: number;
}

interface SnackPublisherApi {
  publish(options: SnackPublishOptions): string;
  replace(id: string, options: SnackPublishOptions): void;
  patch(id: string, changes: SnackUpdateOptions): void;
  dismiss(id: string): void;
  clear(): void;
}

const SnackPublisherContext = createContext<SnackPublisherApi | null>(null);

export type SnackState = SnackDescriptor & {
  expiresAt: number | null;
};

function resolveAutoDismissDuration(snack: SnackDescriptor, now: number): number | null {
  if (!snack.autoDismiss || snack.durationMs === null) {
    return null;
  }
  const elapsed = now - snack.updatedAt;
  const remaining = snack.durationMs - elapsed;
  return remaining > 0 ? remaining : 0;
}

function mergeDescriptor(base: SnackState | null, patch: SnackUpdateOptions): SnackState {
  if (!base) {
    throw new Error('Cannot patch unknown snack. Publish it first.');
  }
  const now = Date.now();
  const nextVariant: SnackVariant = patch.variant ?? base.variant;
  const autoDismiss =
    typeof patch.autoDismiss === 'boolean'
      ? patch.autoDismiss
      : typeof patch.durationMs === 'number'
        ? patch.durationMs > 0
        : base.autoDismiss;

  let nextDuration: number | null;
  if (!autoDismiss) {
    nextDuration = null;
  } else if (typeof patch.durationMs === 'number') {
    nextDuration = patch.durationMs > 0 ? patch.durationMs : null;
  } else if (base.durationMs !== null) {
    // keep existing remaining time window relative to the new update timestamp
    const remaining = base.expiresAt ? base.expiresAt - now : base.durationMs;
    nextDuration = remaining > 0 ? remaining : DEFAULT_SNACK_DURATION;
  } else {
    nextDuration = DEFAULT_SNACK_DURATION;
  }

  const expiresAt = nextDuration !== null ? now + nextDuration : null;

  return {
    ...base,
    message: patch.message ?? base.message,
    title: patch.title ?? base.title,
    variant: nextVariant,
    dismissible: patch.dismissible ?? base.dismissible,
    autoDismiss,
    durationMs: nextDuration,
    action: patch.action ?? base.action,
    metadata: patch.metadata ?? base.metadata,
    updatedAt: now,
    expiresAt,
  } satisfies SnackState;
}

function descriptorToState(snack: SnackDescriptor): SnackState {
  const expiresAt = snack.durationMs !== null ? snack.updatedAt + snack.durationMs : null;
  return { ...snack, expiresAt } satisfies SnackState;
}

export function SnackStackProvider({
  children,
  anchor = 'top-right',
  maxSnacks = 4,
}: SnackStackProviderProps) {
  const [snacks, setSnacks] = useState<SnackState[]>([]);
  const timeoutsRef = useRef<Map<string, number>>(new Map());

  useEffect(() => {
    const handleEvent = (event: SnackEvent) => {
      setSnacks((previous) => {
        switch (event.type) {
          case 'show': {
            const nextState = descriptorToState(event.snack);
            const filtered = previous.filter((snack) => snack.id !== nextState.id);
            const truncated = [nextState, ...filtered];
            if (truncated.length <= maxSnacks) {
              return truncated;
            }
            return truncated.slice(0, maxSnacks);
          }
          case 'update': {
            const nextState = descriptorToState(event.changes);
            return previous.map((snack) => (snack.id === event.id ? nextState : snack));
          }
          case 'patch': {
            return previous.map((snack) =>
              snack.id === event.id ? mergeDescriptor(snack, event.changes) : snack,
            );
          }
          case 'dismiss': {
            return previous.filter((snack) => snack.id !== event.id);
          }
          case 'clear':
            return [];
          default:
            return previous;
        }
      });
    };

    const unsubscribe = snackBus.subscribe(handleEvent);
    return unsubscribe;
  }, [maxSnacks]);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return undefined;
    }

    const cleanup: Array<() => void> = [];
    const timeouts = timeoutsRef.current;

    // clear existing timers before recalculating
    for (const timeoutId of timeouts.values()) {
      window.clearTimeout(timeoutId);
    }
    timeouts.clear();

    snacks.forEach((snack) => {
      const duration = resolveAutoDismissDuration(snack, Date.now());
      if (duration === null) {
        return;
      }
      const timeoutId = window.setTimeout(() => {
        dismissSnack(snack.id);
      }, duration);
      timeouts.set(snack.id, timeoutId);
      cleanup.push(() => window.clearTimeout(timeoutId));
    });

    return () => {
      cleanup.forEach((fn) => fn());
      timeouts.clear();
    };
  }, [snacks]);

  useEffect(() => () => {
    if (typeof window === 'undefined') {
      return;
    }
    for (const timeoutId of timeoutsRef.current.values()) {
      window.clearTimeout(timeoutId);
    }
    timeoutsRef.current.clear();
  }, []);

  const publisher = useMemo<SnackPublisherApi>(() => {
    return {
      publish: (options) => publishSnack(options),
      replace: (id, options) => replaceSnack(id, { ...options, id }),
      patch: (id, changes) => snackBus.patch(id, changes),
      dismiss: (id) => dismissSnack(id),
      clear: () => snackBus.clear(),
    } satisfies SnackPublisherApi;
  }, []);

  const handleDismiss = useCallback((id: string) => {
    dismissSnack(id);
  }, []);

  return (
    <SnackPublisherContext.Provider value={publisher}>
      {children}
      <SnackStack snacks={snacks} anchor={anchor} onDismiss={handleDismiss} />
    </SnackPublisherContext.Provider>
  );
}

export function useSnackPublisher(): SnackPublisherApi {
  const context = useContext(SnackPublisherContext);
  if (!context) {
    throw new Error('useSnackPublisher must be used within a SnackStackProvider');
  }
  return context;
}
