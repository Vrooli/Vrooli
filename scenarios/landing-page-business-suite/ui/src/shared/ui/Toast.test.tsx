import { useContext } from 'react';
import { act, fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ToastProvider } from './Toast';
import { ToastContext } from './ToastContext';

function ToastControls() {
  const toast = useContext(ToastContext)!;
  return <>
    <button onClick={() => toast.success('Saved successfully', 'Saved')}>Success</button>
    <button onClick={() => toast.error('Could not save', 'Failure')}>Error</button>
    <button onClick={() => toast.warning('Check your settings')}>Warning</button>
    <button onClick={() => toast.info('Deployment started')}>Info</button>
    <button onClick={() => toast.addToast({ type: 'info', message: 'Manual toast', duration: 0 })}>Manual</button>
  </>;
}

describe('ToastProvider', () => {
  afterEach(() => { vi.useRealTimers(); });

  it('shows every toast type, honors max capacity, and supports manual dismissal', () => {
    render(<ToastProvider maxToasts={3}><ToastControls /></ToastProvider>);
    fireEvent.click(screen.getByRole('button', { name: 'Success' }));
    fireEvent.click(screen.getByRole('button', { name: 'Error' }));
    fireEvent.click(screen.getByRole('button', { name: 'Warning' }));
    fireEvent.click(screen.getByRole('button', { name: 'Info' }));

    expect(screen.queryByText('Saved successfully')).not.toBeInTheDocument();
    expect(screen.getByText('Could not save')).toBeInTheDocument();
    expect(screen.getByText('Check your settings')).toBeInTheDocument();
    expect(screen.getByText('Deployment started')).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'Notifications' })).toBeInTheDocument();
    fireEvent.click(screen.getAllByRole('button', { name: 'Dismiss notification' })[0]!);
    expect(screen.queryByText('Could not save')).not.toBeInTheDocument();
  });

  it('auto-dismisses timed toasts but keeps duration-zero notifications until dismissed', () => {
    vi.useFakeTimers();
    render(<ToastProvider defaultDuration={100}><ToastControls /></ToastProvider>);
    fireEvent.click(screen.getByRole('button', { name: 'Success' }));
    fireEvent.click(screen.getByRole('button', { name: 'Manual' }));
    expect(screen.getByText('Saved successfully')).toBeInTheDocument();
    act(() => { vi.advanceTimersByTime(100); });
    expect(screen.queryByText('Saved successfully')).not.toBeInTheDocument();
    expect(screen.getByText('Manual toast')).toBeInTheDocument();
  });
});
