import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { LogTable } from './LogTable';

vi.mock('react-window', () => ({
  VariableSizeList: ({ children, itemCount, onScroll }: { children: (args: { index: number; style: React.CSSProperties }) => React.ReactNode; itemCount: number; onScroll: (args: { scrollOffset: number }) => void }) => <div><button onClick={() => { onScroll({ scrollOffset: 0 }); onScroll({ scrollOffset: 20 }); }}>{'scroll'}</button>{Array.from({ length: itemCount + 1 }, (_, index) => children({ index, style: {} }))}</div>,
}));

describe('LogTable', () => {
  it('renders empty and virtualized rows and reports scroll position', () => {
    const onScrollTopChange = vi.fn();
    const { rerender } = render(<LogTable entries={[]} onScrollTopChange={onScrollTopChange} />);
    expect(screen.getByText(/No log entries/)).toBeInTheDocument();
    rerender(<LogTable entries={[{ cursor: '1', timestamp: '2026-01-01T00:00:00Z', priority: 3, unit: 'kernel', message: 'oom' } as never]} onScrollTopChange={onScrollTopChange} />);
    expect(screen.getByText('oom')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'scroll' }));
    expect(onScrollTopChange).toHaveBeenNthCalledWith(1, true);
    expect(onScrollTopChange).toHaveBeenNthCalledWith(2, false);
  });

  it('expands long messages with a variable row height and preserves source context', () => {
    const message = 'journalctl detail '.repeat(20);
    render(<LogTable entries={[{ cursor: 'long', timestamp: '2026-01-01T00:00:00Z', priority: 4, unit: 'systemd', message } as never]} />);
    const toggle = screen.getByRole('button', { name: 'Expand message' });
    fireEvent.click(toggle);
    expect(screen.getByRole('button', { name: 'Collapse message' })).toHaveAttribute('aria-expanded', 'true');
    expect(document.querySelector('.log-row__message.is-expanded')).toHaveTextContent(/journalctl detail/);
  });
});
