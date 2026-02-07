/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useEffect, useMemo, useRef, type MutableRefObject, type ReactNode } from 'react';

type KeyboardScopeHandler = (event: KeyboardEvent) => boolean;

type KeyboardScopeRegistration = {
  id: string;
  priority: number;
  enabledRef: MutableRefObject<boolean>;
  handlerRef: MutableRefObject<KeyboardScopeHandler>;
};

type KeyboardScopesContextValue = {
  registerScope: (scope: KeyboardScopeRegistration) => () => void;
};

const KeyboardScopesContext = createContext<KeyboardScopesContextValue | null>(null);

type StoredScope = KeyboardScopeRegistration & {
  sequence: number;
};

export function KeyboardScopeProvider({ children }: { children: ReactNode }) {
  const scopesRef = useRef<Map<number, StoredScope>>(new Map());
  const sequenceRef = useRef(0);

  const contextValue = useMemo<KeyboardScopesContextValue>(() => ({
    registerScope: (scope) => {
      const token = sequenceRef.current + 1;
      sequenceRef.current = token;
      scopesRef.current.set(token, { ...scope, sequence: token });

      return () => {
        scopesRef.current.delete(token);
      };
    },
  }), []);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      const orderedScopes = Array.from(scopesRef.current.values())
        .filter(scope => scope.enabledRef.current)
        .sort((a, b) => {
          if (a.priority !== b.priority) {
            return b.priority - a.priority;
          }
          return b.sequence - a.sequence;
        });

      for (const scope of orderedScopes) {
        const handled = scope.handlerRef.current(event);
        if (handled || event.defaultPrevented) {
          return;
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown, { capture: true });
    return () => {
      window.removeEventListener('keydown', handleKeyDown, { capture: true });
    };
  }, []);

  return (
    <KeyboardScopesContext.Provider value={contextValue}>
      {children}
    </KeyboardScopesContext.Provider>
  );
}

export function useKeyboardScope({
  id,
  priority,
  enabled = true,
  onKeyDown,
}: {
  id: string;
  priority: number;
  enabled?: boolean;
  onKeyDown: KeyboardScopeHandler;
}) {
  const context = useContext(KeyboardScopesContext);
  const enabledRef = useRef(enabled);
  const handlerRef = useRef(onKeyDown);

  enabledRef.current = enabled;
  handlerRef.current = onKeyDown;

  useEffect(() => {
    if (!context) {
      return;
    }

    return context.registerScope({
      id,
      priority,
      enabledRef,
      handlerRef,
    });
  }, [context, id, priority]);
}
