import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ComingSoonPage } from './ComingSoonPage';
import * as api from '../../../shared/api';

vi.mock('../../../shared/api', async () => ({ ...(await vi.importActual('../../../shared/api')), submitWaitlistEmail: vi.fn() }));

const branding = {
  site_name: 'Acme Cloud', tagline: 'Reliable tools for growing teams', logo_url: '/assets/logo.svg',
  theme_primary_color: '#118877', theme_background_color: '#071018', coming_soon_message: 'A better launch is on the way.',
};

describe('ComingSoonPage', () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it('renders the configured brand and launch message', () => {
    render(<ComingSoonPage branding={branding} />);
    expect(screen.getByRole('img', { name: 'Acme Cloud' })).toHaveAttribute('src', '/assets/logo.svg');
    expect(screen.getByText('Reliable tools for growing teams')).toBeInTheDocument();
    expect(screen.getByText('A better launch is on the way.')).toBeInTheDocument();
  });

  it('rejects an empty waitlist form without sending a public request', () => {
    render(<ComingSoonPage branding={branding} />);
    fireEvent.click(screen.getByRole('button', { name: 'Notify me when ready' }));
    expect(screen.getByText('Please enter your email')).toBeInTheDocument();
    expect(api.submitWaitlistEmail).not.toHaveBeenCalled();
  });

  it('submits a trimmed email once and prevents duplicate subscription while successful', async () => {
    vi.mocked(api.submitWaitlistEmail).mockResolvedValue({ success: true });
    render(<ComingSoonPage branding={branding} />);
    fireEvent.change(screen.getByLabelText('Email address'), { target: { value: ' buyer@example.com ' } });
    fireEvent.click(screen.getByRole('button', { name: 'Notify me when ready' }));

    await waitFor(() => { expect(api.submitWaitlistEmail).toHaveBeenCalledWith('buyer@example.com'); });
    expect(await screen.findByText("You're on the list!")).toBeInTheDocument();
    expect(screen.getByLabelText('Email address')).toBeDisabled();
  });

  it('recovers from API failure when the visitor changes the address', async () => {
    vi.mocked(api.submitWaitlistEmail).mockRejectedValue(new Error('This email is already registered'));
    render(<ComingSoonPage branding={{ site_name: 'Acme Cloud' }} />);
    fireEvent.change(screen.getByLabelText('Email address'), { target: { value: 'buyer@example.com' } });
    fireEvent.click(screen.getByRole('button', { name: 'Notify me when ready' }));
    expect(await screen.findByText('This email is already registered')).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText('Email address'), { target: { value: 'another@example.com' } });
    expect(screen.queryByText('This email is already registered')).not.toBeInTheDocument();
    expect(screen.getByText('We are working hard to bring you something amazing. Stay tuned!')).toBeInTheDocument();
  });
});
