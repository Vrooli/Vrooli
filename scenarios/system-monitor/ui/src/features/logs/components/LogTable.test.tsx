import { createElement } from 'react';
import { fireEvent, screen } from '@testing-library/react';
import axe from 'axe-core';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import type { LogEntry } from '../types';
import { LogTable } from './LogTable';

vi.mock('react-window', () => ({
  VariableSizeList: ({ children, itemCount, onScroll, itemData }: { children: React.ComponentType<{ index: number; style: React.CSSProperties; data: unknown }>; itemCount: number; onScroll: (args: { scrollOffset: number }) => void; itemData: unknown }) => <div><button onClick={() => { onScroll({ scrollOffset: 0 }); onScroll({ scrollOffset: 20 }); }}>{'scroll'}</button>{Array.from({ length: itemCount + 1 }, (_, index) => createElement(children, { key: index, index, style: {}, data: itemData }))}</div>,
}));

const logEntry = (overrides: Partial<LogEntry> & Pick<LogEntry, 'message'>): LogEntry => ({
  timestamp: '2026-01-01T00:00:00Z',
  realtime: 0,
  priority: 6,
  unit: 'systemd',
  ...overrides,
});

describe('LogTable', () => {
  it('renders empty and virtualized rows and reports scroll position', () => {
    const onScrollTopChange = vi.fn();
    const { rerender } = render(<LogTable entries={[]} onScrollTopChange={onScrollTopChange} />);
    expect(screen.getByText(/No matching entries for these filters/)).toBeInTheDocument();
    rerender(<LogTable entries={[logEntry({ cursor: '1', priority: 3, unit: 'kernel', message: 'oom' })]} onScrollTopChange={onScrollTopChange} />);
    expect(screen.getByText('oom')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'scroll' }));
    expect(onScrollTopChange).toHaveBeenNthCalledWith(1, true);
    expect(onScrollTopChange).toHaveBeenNthCalledWith(2, false);
  });

  it('preserves focused row controls when its parent rerenders', () => {
    const entries = [logEntry({ cursor: 'focus', message: 'ready' })];
    const { rerender } = render(<LogTable entries={entries} onSelect={() => {}} />);
    const button = screen.getByRole('button', { name: /Read entry/ });
    button.focus();
    rerender(<LogTable entries={entries} onSelect={() => {}} />);
    expect(button).toHaveFocus();
  });

  it('exposes a long expanded message in a keyboard-accessible scroll region', () => {
    const message = 'journalctl detail '.repeat(25).slice(0, 445);
    render(<LogTable entries={[logEntry({ cursor: 'long', priority: 4, message })]} />);
    const toggle = screen.getByRole('button', { name: 'Expand message' });
    fireEvent.click(toggle);
    expect(screen.getByRole('button', { name: 'Collapse message' })).toHaveAttribute('aria-expanded', 'true');
    const messageRegion = screen.getByRole('region', { name: 'Expanded log message' });
    expect(messageRegion).toHaveAttribute('tabindex', '0');
    expect(messageRegion.textContent).toBe(message);
  });

  it('has no automated accessibility violations with a long message expanded', async () => {
    const message = 'journalctl detail '.repeat(25).slice(0, 445);
    const { container } = render(<LogTable entries={[logEntry({ cursor: 'long', priority: 4, message })]} />);
    fireEvent.click(screen.getByRole('button', { name: 'Expand message' }));
    const results = await axe.run(container);
    expect(results.violations).toEqual([]);
  });

  it('keeps short messages collapsed without an expansion control', () => {
    const message = 'routine service started successfully'.padEnd(75, '.');
    render(<LogTable entries={[logEntry({ cursor: 'short', message })]} />);
    expect(screen.getByText(message)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Expand message|Collapse message/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('region', { name: 'Expanded log message' })).not.toBeInTheDocument();
  });
});
