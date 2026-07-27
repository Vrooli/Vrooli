import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ErrorBoundary } from './ErrorBoundary';

function Broken({ message = 'Network Error' }: { message?: string }) {
  throw new Error(message);
  return null;
}

describe('ErrorBoundary', () => {
  afterEach(() => { vi.restoreAllMocks(); });

  it('renders sanitized app fallback actions and reports contextual errors', () => {
    const onError = vi.fn();
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    render(<ErrorBoundary level="app" name="checkout" onError={onError}><Broken message="Network Error: token=secret" /></ErrorBoundary>);

    expect(screen.getByText('Network connection issue')).toBeInTheDocument();
    expect(screen.queryByText(/token=secret/)).not.toBeInTheDocument();
    expect(onError).toHaveBeenCalledOnce();
    expect(consoleError).toHaveBeenCalled();
    expect(screen.getByRole('button', { name: 'Refresh Page' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Go Home' })).toBeInTheDocument();
  });

  it('supports route retry and navigation controls', () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    render(<ErrorBoundary level="route"><Broken message="404 Not Found" /></ErrorBoundary>);
    expect(screen.getByText('Resource not found')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Try Again' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Go Back' })).toBeInTheDocument();
    expect(consoleError).toHaveBeenCalled();
  });

  it('uses custom fallback reset callbacks and handles concise component errors', () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const fallback = vi.fn(({ error, resetError }: { error: Error; resetError: () => void }) => <button onClick={resetError}>Reset {error.message}</button>);
    render(<ErrorBoundary fallback={fallback}><Broken message="Safe message" /></ErrorBoundary>);
    expect(screen.getByRole('button', { name: 'Reset Safe message' })).toBeInTheDocument();
    expect(fallback).toHaveBeenCalled();
    expect(consoleError).toHaveBeenCalled();
  });
});
