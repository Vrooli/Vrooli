import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useFocusTrap } from '../useFocusTrap';

describe('useFocusTrap', () => {
  let container: HTMLDivElement;
  let firstButton: HTMLButtonElement;
  let lastButton: HTMLButtonElement;
  let initialFocusButton: HTMLButtonElement;

  beforeEach(() => {
    container = document.createElement('div');
    firstButton = document.createElement('button');
    firstButton.textContent = 'First';
    lastButton = document.createElement('button');
    lastButton.textContent = 'Last';
    initialFocusButton = document.createElement('button');
    initialFocusButton.textContent = 'Initial';

    container.appendChild(firstButton);
    container.appendChild(initialFocusButton);
    container.appendChild(lastButton);
    document.body.appendChild(container);
  });

  afterEach(() => {
    document.body.removeChild(container);
  });

  it('focuses initial element on activation', async () => {
    const onEscape = vi.fn();
    const containerRef = { current: container };
    const initialRef = { current: initialFocusButton };

    renderHook(() => useFocusTrap(containerRef, initialRef, true, onEscape));

    // Focus is set via setTimeout(0)
    await new Promise((r) => setTimeout(r, 10));
    expect(document.activeElement).toBe(initialFocusButton);
  });

  it('calls onEscape when Escape is pressed', () => {
    const onEscape = vi.fn();
    const containerRef = { current: container };
    const initialRef = { current: initialFocusButton };

    renderHook(() => useFocusTrap(containerRef, initialRef, true, onEscape));

    container.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
    expect(onEscape).toHaveBeenCalledOnce();
  });

  it('wraps focus from last to first on Tab', async () => {
    const onEscape = vi.fn();
    const containerRef = { current: container };
    const initialRef = { current: initialFocusButton };

    renderHook(() => useFocusTrap(containerRef, initialRef, true, onEscape));

    lastButton.focus();
    container.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true }));
    expect(document.activeElement).toBe(firstButton);
  });

  it('wraps focus from first to last on Shift+Tab', async () => {
    const onEscape = vi.fn();
    const containerRef = { current: container };
    const initialRef = { current: initialFocusButton };

    renderHook(() => useFocusTrap(containerRef, initialRef, true, onEscape));

    firstButton.focus();
    container.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true }));
    expect(document.activeElement).toBe(lastButton);
  });

  it('does not trap when inactive', () => {
    const onEscape = vi.fn();
    const containerRef = { current: container };
    const initialRef = { current: initialFocusButton };

    renderHook(() => useFocusTrap(containerRef, initialRef, false, onEscape));

    container.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
    expect(onEscape).not.toHaveBeenCalled();
  });
});
