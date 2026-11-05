import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ErrorBoundary } from './ErrorBoundary';

let shouldThrow = true;

function ThrowingChild() {
  if (shouldThrow) throw new Error('render failed');
  return <div>recovered content</div>;
}

describe('ErrorBoundary', () => {
  beforeEach(() => {
    shouldThrow = true;
  });

  it('captures errors, copies a report, and retries the child', async () => {
    const onError = vi.fn();
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } });
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    render(<ErrorBoundary onError={onError}><ThrowingChild /></ErrorBoundary>);
    expect(screen.getByText('System Error')).toBeInTheDocument();
    expect(screen.getByText('render failed')).toBeInTheDocument();
    expect(onError).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole('button', { name: 'Copy Error' }));
    await waitFor(() => expect(screen.getByRole('button', { name: 'Copied!' })).toBeInTheDocument());
    expect(writeText).toHaveBeenCalledWith(expect.stringContaining('render failed'));
    shouldThrow = false;
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(screen.getByText('recovered content')).toBeInTheDocument();
    consoleError.mockRestore();
  });

  it('uses a caller-provided fallback and does not render the error panel', () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    render(<ErrorBoundary fallback={<div>custom recovery</div>}><ThrowingChild /></ErrorBoundary>);
    expect(screen.getByText('custom recovery')).toBeInTheDocument();
    expect(screen.queryByText('System Error')).not.toBeInTheDocument();
    consoleError.mockRestore();
  });

  it('renders development diagnostics and tolerates clipboard failure', async () => {
    vi.stubEnv('NODE_ENV', 'development');
    const writeText = vi.fn().mockRejectedValue(new Error('clipboard blocked'));
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } });
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    render(<ErrorBoundary><ThrowingChild /></ErrorBoundary>);
    expect(screen.getByText('Development Details')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Copy Error' }));
    await waitFor(() => { expect(writeText).toHaveBeenCalled(); });
    expect(screen.getByRole('button', { name: 'Copy Error' })).toBeInTheDocument();
    consoleError.mockRestore();
    vi.unstubAllEnvs();
  });
});
