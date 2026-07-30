import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../../../test-utils/renderWithProviders';
import { UserLogin } from './UserLogin';
import * as api from '../../../shared/api';

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return { ...actual, requestMagicLink: vi.fn() };
});

const requestMagicLink = vi.mocked(api.requestMagicLink);

function renderLogin(route = '/auth/login') {
  return renderWithProviders(<UserLogin />, { route });
}

describe('UserLogin', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
  });

  it('validates missing and malformed email without calling the API', async () => {
    renderLogin();
    const form = screen.getByTestId('submit-button').closest('form');
    expect(form).not.toBeNull();
    fireEvent.submit(form!);
    expect(await screen.findByText('Email is required')).toBeInTheDocument();

    fireEvent.change(screen.getByTestId('email-input'), { target: { value: 'invalid-address' } });
    fireEvent.submit(form!);
    expect(await screen.findByText('Please enter a valid email address')).toBeInTheDocument();
    expect(requestMagicLink).not.toHaveBeenCalled();
  });

  it('requests a normalized magic link and retains callback parameters for secure verification', async () => {
    requestMagicLink.mockResolvedValue({ message: 'Request sent' });
    renderLogin('/auth/login?redirect_uri=vrooli%3A%2F%2Fcallback&app=Desktop&state=nonce');
    fireEvent.change(screen.getByTestId('email-input'), { target: { value: ' Buyer@Example.COM ' } });
    fireEvent.click(screen.getByTestId('submit-button'));

    await waitFor(() => {
      expect(requestMagicLink).toHaveBeenCalledWith('buyer@example.com');
    });
    expect(await screen.findByText('Check your email')).toBeInTheDocument();
    expect(sessionStorage.getItem('auth_callback_params')).toBe(JSON.stringify({ redirect_uri: 'vrooli://callback', app: 'Desktop', state: 'nonce' }));
  });

  it('maps rate limiting to a non-enumerating customer-safe message', async () => {
    requestMagicLink.mockRejectedValue(new api.ApiError('too many attempts', 'rate_limited'));
    renderLogin();
    fireEvent.change(screen.getByTestId('email-input'), { target: { value: 'buyer@example.com' } });
    fireEvent.click(screen.getByTestId('submit-button'));

    expect(await screen.findByText('Too many login attempts. Please wait a moment and try again.')).toBeInTheDocument();
  });

  it.each([
    ['validation', 'Please enter a valid email address.'],
    ['network', 'Unable to reach the server. Please check your connection.'],
    ['unexpected', 'Something went wrong. Please try again.'],
  ] as const)('maps %s request failures to a customer-safe message', async (type, expectedMessage) => {
    requestMagicLink.mockRejectedValue(
      type === 'unexpected' ? new Error('unexpected failure') : new api.ApiError('request failed', type),
    );
    renderLogin();
    fireEvent.change(screen.getByTestId('email-input'), { target: { value: 'buyer@example.com' } });
    fireEvent.click(screen.getByTestId('submit-button'));

    expect(await screen.findByText(expectedMessage)).toBeInTheDocument();
  });
});
