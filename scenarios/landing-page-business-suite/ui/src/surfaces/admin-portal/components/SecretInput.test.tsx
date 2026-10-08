import { screen } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { SecretInput } from './SecretInput';

describe('SecretInput', () => {
  it('reveals an existing secret and can hide it again', async () => {
    const user = userEvent.setup();
    const onReveal = vi.fn().mockResolvedValue('pk_live_revealed');
    render(<SecretInput isSet value="" onChange={vi.fn()} onReveal={onReveal} />);

    await user.click(screen.getByRole('button', { name: 'Reveal secret' }));
    expect(await screen.findByDisplayValue('pk_live_revealed')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Hide secret' }));
    expect(screen.getByRole('button', { name: 'Reveal secret' })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Reveal secret' }));
    expect(onReveal).toHaveBeenCalledOnce();
  });

  it('allows a saved secret to be replaced and forwards the new value', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<SecretInput isSet value="" onChange={onChange} onReveal={vi.fn()} placeholder="pk_live_..." />);

    await user.click(screen.getByRole('button', { name: 'Replace secret' }));
    await user.type(screen.getByPlaceholderText('pk_live_...'), 'pk_live_rotated');

    expect(onChange).toHaveBeenCalled();
  });

  it('shows a safe error when revealing a secret fails', async () => {
    const user = userEvent.setup();
    render(<SecretInput isSet value="" onChange={vi.fn()} onReveal={vi.fn().mockRejectedValue(new Error('Access denied'))} />);

    await user.click(screen.getByRole('button', { name: 'Reveal secret' }));
    expect(await screen.findByText('Access denied')).toBeInTheDocument();
  });

  it('renders a normal input when no secret has been stored', () => {
    render(<SecretInput isSet={false} value="draft" onChange={vi.fn()} onReveal={vi.fn()} placeholder="secret" disabled />);
    expect(screen.getByDisplayValue('draft')).toBeDisabled();
    expect(screen.queryByRole('button', { name: 'Reveal secret' })).not.toBeInTheDocument();
  });
});
