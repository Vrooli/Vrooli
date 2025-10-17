import { createRoot, type Root } from 'react-dom/client';
import { act } from 'react';
import type { ReactNode } from 'react';

interface RenderHookResult<T> {
  result: { current: T };
  rerender: (callback: () => T) => void;
  unmount: () => void;
}

function TestHarness<T>({ callback, onUpdate }: { callback: () => T; onUpdate: (value: T) => void }) {
  const value = callback();
  onUpdate(value);
  return null;
}

export function renderHook<T>(callback: () => T): RenderHookResult<T> {
  const container = document.createElement('div');
  let root: Root | null = null;
  const result = { current: undefined as unknown as T };

  const render = (cb: () => T) => {
    const element: ReactNode = (
      <TestHarness
        callback={cb}
        onUpdate={(value) => {
          result.current = value;
        }}
      />
    );

    act(() => {
      if (!root) {
        root = createRoot(container);
      }
      root.render(element);
    });
  };

  render(callback);

  return {
    result,
    rerender: (nextCallback) => render(nextCallback),
    unmount: () => {
      if (!root) {
        return;
      }
      act(() => {
        root?.unmount();
      });
      root = null;
    },
  };
}
