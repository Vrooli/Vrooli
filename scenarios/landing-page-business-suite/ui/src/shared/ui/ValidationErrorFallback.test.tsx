import { describe, expect, it, vi } from 'vitest';
import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import { ValidationErrorFallback, ValidationGuard } from './ValidationErrorFallback';

describe('ValidationErrorFallback', () => {
  it('renders minimal contextual recovery UI and invokes retry', () => {
    const onRetry = vi.fn();
    render(<ValidationErrorFallback error="schema mismatch" context="profiles" variant="minimal" onRetry={onRetry} />);
    expect(screen.getByText('Failed to load profiles')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it('renders inline details and card recovery actions without leaking raw data by default', () => {
    const onRetry = vi.fn();
    const onReport = vi.fn();
    const { rerender } = render(<ValidationErrorFallback error="invalid payload" context="billing" variant="inline" showDetails onRetry={onRetry} />);
    expect(screen.getByText('Unable to display billing')).toBeInTheDocument();
    expect(screen.getByText('Technical details')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(onRetry).toHaveBeenCalledOnce();

    rerender(<ValidationErrorFallback error="invalid payload" context="billing" rawData={{ id: 7 }} onRetry={onRetry} onReport={onReport} showDetails />);
    expect(screen.getByText('Unable to load billing')).toBeInTheDocument();
    expect(screen.getByText('Raw Data')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Try Again' }));
    fireEvent.click(screen.getByRole('button', { name: 'Report Issue' }));
    expect(onRetry).toHaveBeenCalledTimes(2);
    expect(onReport).toHaveBeenCalledOnce();
  });

  it('renders guarded children only for validated data and an error fallback otherwise', () => {
    const child = vi.fn((value: { name: string }) => <span>{value.name}</span>);
    const { rerender } = render(<ValidationGuard result={{ success: true, data: { name: 'Ready' } }}>{child}</ValidationGuard>);
    expect(screen.getByText('Ready')).toBeInTheDocument();
    expect(child).toHaveBeenCalledWith({ name: 'Ready' });

    rerender(<ValidationGuard result={{ success: false, error: 'bad response', raw: { bad: true } }} context="configuration" variant="minimal">{child}</ValidationGuard>);
    expect(screen.getByText('Failed to load configuration')).toBeInTheDocument();
    expect(child).toHaveBeenCalledOnce();
  });
});
