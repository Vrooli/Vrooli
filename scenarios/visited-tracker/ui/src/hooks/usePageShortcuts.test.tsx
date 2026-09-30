import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { usePageShortcuts } from './usePageShortcuts';

function Page({ refresh }: { refresh: () => void }) {
  usePageShortcuts({ r: refresh });
  return <><input aria-label="Search" /><div contentEditable data-testid="editor"><span>Editable text</span></div><div role="dialog"><button>Dialog action</button></div></>;
}

describe('page shortcut ownership', () => {
  it('uses the latest action, handles uppercase keys, and unregisters on unmount', () => {
    const oldAction = vi.fn();
    const latest = vi.fn();
    const view = render(<Page refresh={oldAction} />);
    view.rerender(<Page refresh={latest} />);
    expect(fireEvent.keyDown(window, { key: 'R' })).toBe(false);
    expect(oldAction).not.toHaveBeenCalled();
    expect(latest).toHaveBeenCalledTimes(1);
    view.unmount();
    fireEvent.keyDown(window, { key: 'r' });
    expect(latest).toHaveBeenCalledTimes(1);
  });

  it('yields to editing, dialog controls, browser shortcuts, composition and handled events', () => {
    const refresh = vi.fn();
    render(<Page refresh={refresh} />);
    for (const target of [screen.getByRole('textbox'), screen.getByText('Editable text'), screen.getByRole('button')]) {
      fireEvent.keyDown(target, { key: 'r' });
    }
    for (const option of ['ctrlKey', 'metaKey', 'altKey', 'repeat', 'isComposing']) {
      fireEvent.keyDown(window, { key: 'r', [option]: true });
    }
    const handled = new KeyboardEvent('keydown', { key: 'r', cancelable: true });
    handled.preventDefault();
    window.dispatchEvent(handled);
    fireEvent.keyDown(window, { key: 'q' });
    expect(refresh).not.toHaveBeenCalled();
  });
});
